module github.com/KiloProjects/kilonova

go 1.25.0

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/benbjohnson/hashfs v0.2.2
	github.com/davecgh/go-spew v1.1.1
	github.com/go-chi/cors v1.2.2
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/gorilla/css v1.0.1 // indirect
	github.com/gorilla/schema v1.4.1
	github.com/gosimple/slug v1.15.0
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jordan-wright/email v4.0.1-0.20210109023952-943e75fe5223+incompatible
	github.com/microcosm-cc/bluemonday v1.0.27
	github.com/yuin/goldmark v1.7.13
	golang.org/x/crypto v0.47.0
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0
	golang.org/x/text v0.33.0
)

require (
	github.com/Yiling-J/theine-go v0.6.1
	github.com/alecthomas/chroma/v2 v2.20.0
	github.com/antchfx/xmlquery v1.4.4
	github.com/disintegration/gift v1.2.1
	github.com/dop251/goja v0.0.0-20250630131328-58d95d85e994
	github.com/evanw/esbuild v0.25.9
	github.com/go-chi/chi/v5 v5.2.4
	github.com/jackc/pgx-shopspring-decimal v0.0.0-20220624020537-1d36b5a1853e
	github.com/jackc/pgx/v5 v5.7.5
	github.com/klauspost/compress v1.18.3
	github.com/shopspring/decimal v1.4.0
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
)

require github.com/dchest/captcha v1.1.0

require github.com/dustin/go-humanize v1.0.1

require (
	connectrpc.com/connect v1.19.1
	github.com/Masterminds/squirrel v1.5.4
	github.com/a-h/templ v0.3.977
	github.com/bwmarrin/discordgo v0.29.0
	github.com/danielgtaylor/huma/v2 v2.34.1
	github.com/dominikbraun/graph v0.23.0
	github.com/exaring/otelpgx v0.9.3
	github.com/go-jose/go-jose/v4 v4.1.2
	github.com/go-logr/logr v1.4.3
	github.com/gohugoio/hugo-goldmark-extensions/passthrough v0.3.1
	github.com/google/uuid v1.6.0
	github.com/joho/godotenv v1.5.1
	github.com/lmittmann/tint v1.1.2
	github.com/mattn/go-isatty v0.0.20
	github.com/oschwald/maxminddb-golang/v2 v2.0.0-beta.8
	github.com/prometheus/client_golang v1.23.2
	github.com/riandyrn/otelchi v0.12.1
	github.com/samber/slog-multi v1.4.1
	github.com/sashabaranov/go-openai v1.41.1
	github.com/skip2/go-qrcode v0.0.0-20200617195104-da1b6568686e
	github.com/spf13/afero v1.15.0
	github.com/yuin/goldmark-highlighting/v2 v2.0.0-20230729083705-37449abec8cc
	github.com/zitadel/oidc/v3 v3.44.0
	go.opentelemetry.io/contrib/bridges/otelslog v0.12.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.64.0
	go.opentelemetry.io/otel v1.39.0
	go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc v0.13.0
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.37.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.37.0
	go.opentelemetry.io/otel/log v0.13.0
	go.opentelemetry.io/otel/sdk v1.39.0
	go.opentelemetry.io/otel/sdk/log v0.13.0
	go.opentelemetry.io/otel/sdk/metric v1.39.0
	go.opentelemetry.io/otel/trace v1.39.0
	golang.org/x/oauth2 v0.33.0
	google.golang.org/protobuf v1.36.11
)

require (
	buf.build/gen/go/bufbuild/bufplugin/protocolbuffers/go v1.36.11-20250718181942-e35f9b667443.1 // indirect
	buf.build/gen/go/bufbuild/protodescriptor/protocolbuffers/go v1.36.11-20250109164928-1da0de137947.1 // indirect
	buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20251209175733-2a1774d88802.1 // indirect
	buf.build/gen/go/bufbuild/registry/connectrpc/go v1.19.1-20251202164234-62b14f0b533c.2 // indirect
	buf.build/gen/go/bufbuild/registry/protocolbuffers/go v1.36.11-20251202164234-62b14f0b533c.1 // indirect
	buf.build/gen/go/pluginrpc/pluginrpc/protocolbuffers/go v1.36.11-20241007202033-cf42259fcbfc.1 // indirect
	buf.build/go/app v0.2.0 // indirect
	buf.build/go/bufplugin v0.9.0 // indirect
	buf.build/go/bufprivateusage v0.1.0 // indirect
	buf.build/go/interrupt v1.1.0 // indirect
	buf.build/go/protovalidate v1.1.0 // indirect
	buf.build/go/protoyaml v0.6.0 // indirect
	buf.build/go/spdx v0.2.0 // indirect
	buf.build/go/standard v0.1.0 // indirect
	cel.dev/expr v0.25.1 // indirect
	connectrpc.com/otelconnect v0.9.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/a-h/parse v0.0.0-20250122154542-74294addb73e // indirect
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.9.1 // indirect
	github.com/bufbuild/buf v1.64.0 // indirect
	github.com/bufbuild/protocompile v0.14.2-0.20260114160500-16922e24f2b6 // indirect
	github.com/bufbuild/protoplugin v0.0.0-20250218205857-750e09ce93e1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cenkalti/backoff/v5 v5.0.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/clbanning/mxj/v2 v2.7.0 // indirect
	github.com/cli/browser v1.3.0 // indirect
	github.com/containerd/errdefs v1.0.0 // indirect
	github.com/containerd/errdefs/pkg v0.3.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.18.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.7 // indirect
	github.com/distribution/reference v0.6.0 // indirect
	github.com/docker/cli v29.1.5+incompatible // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/docker v28.5.2+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.9.5 // indirect
	github.com/docker/go-connections v0.6.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/google/cel-go v0.26.1 // indirect
	github.com/google/go-containerregistry v0.20.7 // indirect
	github.com/gorilla/securecookie v1.1.2 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jdx/go-netrc v1.0.0 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/morikuni/aec v1.1.0 // indirect
	github.com/muhlemmer/gu v0.3.1 // indirect
	github.com/muhlemmer/httpforwarded v0.1.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/natefinch/atomic v1.0.1 // indirect
	github.com/nicksnyder/go-i18n/v2 v2.6.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/petermattis/goid v0.0.0-20260113132338-7c7de50cc741 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/quic-go/qpack v0.6.0 // indirect
	github.com/quic-go/quic-go v0.59.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rs/cors v1.11.1 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/samber/lo v1.51.0 // indirect
	github.com/samber/slog-common v0.19.0 // indirect
	github.com/segmentio/asm v1.2.1 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/sirupsen/logrus v1.9.4 // indirect
	github.com/spf13/cobra v1.10.2 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/stoewer/go-strcase v1.3.1 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
	github.com/tetratelabs/wazero v1.11.0 // indirect
	github.com/tidwall/btree v1.8.1 // indirect
	github.com/tomwright/dasel/v2 v2.8.2-0.20241008211502-e96f281f05a1 // indirect
	github.com/vbatts/tar-split v0.12.2 // indirect
	github.com/zitadel/logging v0.6.2 // indirect
	github.com/zitadel/schema v1.3.1 // indirect
	go.lsp.dev/jsonrpc2 v0.10.0 // indirect
	go.lsp.dev/pkg v0.0.0-20210717090340-384b27a52fb2 // indirect
	go.lsp.dev/protocol v0.12.0 // indirect
	go.lsp.dev/uri v0.3.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/proto/otlp v1.9.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.1 // indirect
	go.yaml.in/yaml/v2 v2.4.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/exp v0.0.0-20260112195511-716be5621a96 // indirect
	golang.org/x/mod v0.32.0 // indirect
	golang.org/x/term v0.39.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260114163908-3f89685c29c3 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260114163908-3f89685c29c3 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	mvdan.cc/xurls/v2 v2.6.0 // indirect
	pluginrpc.com/pluginrpc v0.5.0 // indirect
)

require (
	github.com/antchfx/xpath v1.3.4 // indirect
	github.com/dlclark/regexp2 v1.11.5 // indirect
	github.com/go-sourcemap/sourcemap v2.1.4+incompatible // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/pprof v0.0.0-20250630185457-6e76a2b096b5 // indirect
	github.com/gosimple/unidecode v1.0.1 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/sys v0.40.0
	vimagination.zapto.org/dos2unix v1.0.1
)

tool (
	connectrpc.com/connect/cmd/protoc-gen-connect-go
	github.com/a-h/templ/cmd/templ
	github.com/bufbuild/buf/cmd/buf
	github.com/nicksnyder/go-i18n/v2/goi18n
	github.com/tomwright/dasel/v2/cmd/dasel
	google.golang.org/protobuf/cmd/protoc-gen-go
)
