# Kilonova images. Two targets share one builder:
#
#   docker build --target platform -t kilonova .          # kn main
#   docker build --target grader   -t kilonova-grader .   # kn grader-serve
#
# The platform carries no language toolchains and cannot sandbox in-process, so
# it always runs with KN_EVAL_MODE=remote against a grader. See docs/deployment.md.

# ---------------------------------------------------------------------------
# Builder: generation, then assets, then compile. The order is load-bearing:
# `go generate` writes _translations.json (embedded into the binary) and
# web/assets/chroma.css (an input to the Vite CSS bundle), and web/assets_test.go
# fails if the Vite manifests point at files that never got embedded.
# ---------------------------------------------------------------------------
# Node comes from the official image, not apt, for a later version than what is in apt,
# and Vite 8 / rolldown need a newer one (node:util styleText).
FROM node:24-trixie AS node

FROM golang:1.27-trixie AS builder

COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -sf /usr/local/lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -sf /usr/local/lib/node_modules/corepack/dist/corepack.js /usr/local/bin/corepack \
    && corepack enable pnpm

WORKDIR /src

# Dependency manifests first so these layers survive source edits.
COPY go.mod go.sum ./
RUN go mod download
COPY web/assets/package.json web/assets/pnpm-lock.yaml ./web/assets/
RUN pnpm -C web/assets install --frozen-lockfile

COPY . .

RUN go generate ./...
RUN pnpm -C web/assets build
RUN CGO_ENABLED=0 go build -o /out/kn ./cmd/kn

# ---------------------------------------------------------------------------
# isolate, built from source. isolate-cg-keeper is deliberately not installed:
# kn grader-serve does that work itself (KN_SANDBOX_ENSURE_CG_KEEPER).
# ---------------------------------------------------------------------------
FROM debian:trixie-slim AS isolate-builder

ARG ISOLATE_VERSION=v2.7

RUN apt-get update && apt-get install -y --no-install-recommends \
        build-essential git libcap-dev libsystemd-dev libseccomp-dev pkg-config ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN git clone --depth 1 --branch "${ISOLATE_VERSION}" https://github.com/ioi/isolate.git /src/isolate
WORKDIR /src/isolate
# Binary only: `make install` would also build man pages, dragging in asciidoc.
RUN make isolate

# ---------------------------------------------------------------------------
# Toolchains for the grader, downloaded rather than apt-installed.
# ---------------------------------------------------------------------------
FROM debian:trixie-slim AS toolchain-downloader

ARG KOTLIN_VERSION=2.4.20
ARG UV_VERSION=0.12.17

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates curl unzip \
    && rm -rf /var/lib/apt/lists/*

# Unpacked under /usr/local, not /opt: isolate mounts /usr into the sandbox by
# default but not /opt, so a compiler in /opt is invisible to the thing that
# has to run it.
RUN curl -fsSL -o /tmp/kotlin.zip \
        "https://github.com/JetBrains/kotlin/releases/download/v${KOTLIN_VERSION}/kotlin-compiler-${KOTLIN_VERSION}.zip" \
    && unzip -q /tmp/kotlin.zip -d /usr/local \
    && rm /tmp/kotlin.zip

RUN arch="$(dpkg --print-architecture)" \
    && case "$arch" in \
         amd64) uv_target=x86_64-unknown-linux-gnu ;; \
         arm64) uv_target=aarch64-unknown-linux-gnu ;; \
         *) echo "unsupported architecture $arch" >&2; exit 1 ;; \
       esac \
    && curl -fsSL -o /tmp/uv.tar.gz \
        "https://github.com/astral-sh/uv/releases/download/${UV_VERSION}/uv-${uv_target}.tar.gz" \
    && mkdir -p /opt/uv \
    && tar -xzf /tmp/uv.tar.gz -C /opt/uv --strip-components=1 \
    && rm /tmp/uv.tar.gz

# ---------------------------------------------------------------------------
# Grader runtime. Deliberately fat: languages are discovered by probing for
# their binaries, so a toolchain missing here silently disappears from the
# platform's language list.
#
# Runs as root and needs --privileged --cgroupns=host: isolate creates and
# manages cgroup v2 subtrees per sandbox.
# ---------------------------------------------------------------------------
FROM debian:trixie-slim AS grader

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates tzdata curl openssl libcap2 libseccomp2 \
        build-essential mold \
        fp-compiler \
        golang \
        default-jdk \
        python3 python3-venv \
        nodejs \
        php-cli \
        rustc \
    && rm -rf /var/lib/apt/lists/*

COPY --from=isolate-builder /src/isolate/isolate /usr/local/bin/isolate
# Upstream diagnostic script, handy when a host refuses to grade.
COPY --from=isolate-builder /src/isolate/isolate-check-environment /usr/local/bin/isolate-check-environment
COPY --from=toolchain-downloader /usr/local/kotlinc /usr/local/kotlinc
COPY --from=toolchain-downloader /opt/uv/uv /usr/local/bin/uv
COPY --from=toolchain-downloader /opt/uv/uvx /usr/local/bin/uvx
COPY --from=builder /out/kn /usr/local/bin/kn

RUN ln -s /usr/local/kotlinc/bin/kotlinc /usr/local/bin/kotlinc \
    && ln -s /usr/local/kotlinc/bin/kotlin /usr/local/bin/kotlin

# Explicit uid/gid range instead of subid_user, so the image needs no
# /etc/subuid bookkeeping. cg_root is the `auto:` form: kn writes the resolved
# cgroup path there at startup and isolate reads it back.
RUN mkdir -p /var/local/lib/isolate /run/isolate/locks /usr/local/etc \
    && printf '%s\n' \
        'box_root = /var/local/lib/isolate' \
        'lock_root = /run/isolate/locks' \
        'cg_root = auto:/run/isolate/cgroup' \
        'first_uid = 60000' \
        'first_gid = 60000' \
        'num_boxes = 1000' \
        > /usr/local/etc/isolate

COPY docker/grader-entrypoint.sh /usr/local/bin/grader-entrypoint
RUN chmod +x /usr/local/bin/grader-entrypoint

ENV KN_DATA_DIR=/var/lib/kilonova/data \
    KN_GRADER_LISTEN=0.0.0.0:9000 \
    KN_GRADER_TLS_CERT=/etc/kilonova/grader.crt \
    KN_GRADER_TLS_KEY=/etc/kilonova/grader.key \
    KN_SANDBOX_ENSURE_CG_KEEPER=true \
    KN_LOG_FILE=false

VOLUME ["/var/lib/kilonova/data"]
EXPOSE 9000

HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -fsSk https://localhost:9000/healthz || exit 1

ENTRYPOINT ["grader-entrypoint"]
CMD ["grader-serve"]

# ---------------------------------------------------------------------------
# Platform runtime. Last stage, so a plain `docker build .` yields the platform.
# ---------------------------------------------------------------------------
FROM debian:trixie-slim AS platform

# diffutils is not optional: the built-in output checker shells out to `diff`
# on the platform side (eval/checkers/diff.go), even in remote eval mode.
# curl is here for HEALTHCHECK only.
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        diffutils \
        tzdata \
        curl \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /var/lib/kilonova/data

COPY --from=builder /out/kn /usr/local/bin/kn

ENV KN_DATA_DIR=/var/lib/kilonova/data \
    KN_LISTEN=0.0.0.0:8070 \
    KN_EVAL_MODE=remote \
    KN_LOG_FILE=false

WORKDIR /var/lib/kilonova
VOLUME ["/var/lib/kilonova/data"]
EXPOSE 8070

HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=3 \
    CMD curl -fsS http://localhost:8070/healthz || exit 1

ENTRYPOINT ["kn"]
CMD ["main"]
