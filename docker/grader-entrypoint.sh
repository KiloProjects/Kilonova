#!/bin/sh
# Grader container entrypoint: make the two things the grader needs but cannot
# create for itself, then hand over to kn.
set -e

# uv submissions get <KN_DATA_DIR>/uv bind-mounted into the sandbox as /mnt/uv
# for their XDG directories, and it has to be writable.
mkdir -p "${KN_DATA_DIR}/uv" "${KN_DATA_DIR}/scratch"

# kn grader-serve refuses to start without TLS. Generate a self-signed
# certificate when none was mounted, so a local stack needs no manual step.
# Real deployments mount a proper certificate over these paths; a self-signed
# one requires KN_SANDBOX_ALLOW_INSECURE on the platform and is not for
# production (see docs/deployment.md).
if [ ! -f "${KN_GRADER_TLS_CERT}" ] || [ ! -f "${KN_GRADER_TLS_KEY}" ]; then
    echo "grader: no TLS certificate at ${KN_GRADER_TLS_CERT}, generating a self-signed one" >&2
    mkdir -p "$(dirname "${KN_GRADER_TLS_CERT}")" "$(dirname "${KN_GRADER_TLS_KEY}")"
    cn="${KN_GRADER_CERT_HOST:-$(hostname)}"
    openssl req -x509 -newkey rsa:2048 -nodes -days 825 \
        -subj "/CN=${cn}" \
        -addext "subjectAltName=DNS:${cn},DNS:localhost,IP:127.0.0.1" \
        -keyout "${KN_GRADER_TLS_KEY}" \
        -out "${KN_GRADER_TLS_CERT}" 2>/dev/null
fi

# exec so kn is PID 1 and receives SIGTERM directly.
exec kn "$@"
