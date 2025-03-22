module github.com/james-lawrence/bw

go 1.23.0

toolchain go1.24.1

require (
	cloud.google.com/go/compute/metadata v0.6.0
	github.com/0xAX/notificator v0.0.0-20220220101646-ee9b8921e557
	github.com/akutz/memconn v0.1.0
	github.com/alecthomas/kingpin v2.2.6+incompatible
	github.com/alecthomas/kong v1.9.0
	github.com/aws/aws-sdk-go v1.55.6
	github.com/bits-and-blooms/bloom/v3 v3.7.0
	github.com/coreos/go-systemd/v22 v22.5.1-0.20231103132048-7d375ecc2b09
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/fsnotify/fsnotify v1.8.0
	github.com/go-acme/lego/v4 v4.22.2
	github.com/go-git/go-git/v5 v5.13.2
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/schema v1.4.1
	github.com/gorilla/websocket v1.5.3
	github.com/grantae/certinfo v0.0.0-20170412194111-59d56a35515b
	github.com/gutengo/fil v0.0.0-20150411104140-6109b2e0b5cf
	github.com/hashicorp/go-sockaddr v1.0.7
	github.com/hashicorp/memberlist v0.5.3
	github.com/hashicorp/raft v1.7.2
	github.com/hashicorp/raft-boltdb v0.0.0-20250113192317-e8660f88bcc9
	github.com/hashicorp/vault/api v1.15.0
	github.com/icrowley/fake v0.0.0-20240710202011-f797eb4a99c0
	github.com/james-lawrence/torrent v0.0.0-20240404141246-7571207604a7
	github.com/joho/godotenv v1.5.1
	github.com/logrusorgru/aurora v2.0.3+incompatible
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-isatty v0.0.20
	github.com/miekg/dns v1.1.64
	github.com/naoina/toml v0.1.1
	github.com/onsi/ginkgo/v2 v2.23.3
	github.com/onsi/gomega v1.36.3
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.7.0
	github.com/posener/complete v1.2.3
	github.com/pterm/pterm v0.12.80
	github.com/subosito/gotenv v1.6.0
	github.com/traefik/yaegi v0.9.8
	github.com/willabides/kongplete v0.4.0
	golang.org/x/crypto v0.36.0
	golang.org/x/oauth2 v0.28.0
	golang.org/x/time v0.11.0
	golang.org/x/tools v0.31.0
	google.golang.org/api v0.227.0
	google.golang.org/grpc v1.71.0
	google.golang.org/protobuf v1.36.5
	gopkg.in/yaml.v2 v2.4.0
)

require (
	atomicgo.dev/cursor v0.2.0 // indirect
	atomicgo.dev/keyboard v0.2.9 // indirect
	atomicgo.dev/schedule v0.1.0 // indirect
	cloud.google.com/go/auth v0.15.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.7 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/ProtonMail/go-crypto v1.1.5 // indirect
	github.com/RoaringBitmap/roaring v1.9.1 // indirect
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751 // indirect
	github.com/alecthomas/units v0.0.0-20190924025748-f65c72e2690d // indirect
	github.com/anacrolix/generics v0.0.0-20230113004304-d6428d516633 // indirect
	github.com/anacrolix/log v0.15.2 // indirect
	github.com/anacrolix/missinggo v1.3.0 // indirect
	github.com/anacrolix/missinggo/v2 v2.6.0 // indirect
	github.com/anacrolix/multiless v0.3.0 // indirect
	github.com/anacrolix/stm v0.2.0 // indirect
	github.com/anacrolix/upnp v0.1.4 // indirect
	github.com/armon/go-metrics v0.4.1 // indirect
	github.com/aws/aws-sdk-go-v2 v1.32.7 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.17.48 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.16.22 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.26 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.26 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/route53 v1.46.4 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.24.8 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.28.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.33.3 // indirect
	github.com/aws/smithy-go v1.22.1 // indirect
	github.com/benbjohnson/immutable v0.2.0 // indirect
	github.com/bits-and-blooms/bitset v1.12.0 // indirect
	github.com/boltdb/bolt v1.3.1 // indirect
	github.com/bradfitz/iter v0.0.0-20191230175014-e8f45d346db8 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/chzyer/readline v1.5.1 // indirect
	github.com/cloudflare/circl v1.6.0 // indirect
	github.com/containerd/console v1.0.3 // indirect
	github.com/corpix/uarand v0.0.0-20170723150923-031be390f409 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/edsrzf/mmap-go v1.1.0 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/felixge/fgprof v0.9.3 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-git/go-billy/v5 v5.6.2 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/godbus/dbus/v5 v5.1.1-0.20241109141217-c266b19b28e9 // indirect
	github.com/golang/groupcache v0.0.0-20241129210726-2c02b8208cf8 // indirect
	github.com/google/btree v1.1.2 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20250317173921-a4b03ec1a45e // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.14.1 // indirect
	github.com/gookit/color v1.5.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0 // indirect
	github.com/hashicorp/go-metrics v0.5.4 // indirect
	github.com/hashicorp/go-msgpack v0.5.5 // indirect
	github.com/hashicorp/go-msgpack/v2 v2.1.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.7 // indirect
	github.com/hashicorp/go-rootcerts v1.0.2 // indirect
	github.com/hashicorp/go-secure-stdlib/parseutil v0.1.6 // indirect
	github.com/hashicorp/go-secure-stdlib/strutil v0.1.2 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/lithammer/fuzzysearch v1.1.8 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/pjbgf/sha1cd v0.3.2 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/riywo/loginshell v0.0.0-20200815045211-7d26008be1ab // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/ryszard/goskiplist v0.0.0-20150312221310-2dfbae5fcf46 // indirect
	github.com/sean-/seed v0.0.0-20170313163322-e2103e2c3529 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/skeema/knownhosts v1.3.1 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
	go.etcd.io/bbolt v1.3.9 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.59.0 // indirect
	go.opentelemetry.io/otel v1.34.0 // indirect
	go.opentelemetry.io/otel/metric v1.34.0 // indirect
	go.opentelemetry.io/otel/trace v1.34.0 // indirect
	golang.org/x/exp v0.0.0-20250305212735-054e65f0b394 // indirect
	golang.org/x/mod v0.24.0 // indirect
	golang.org/x/net v0.37.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/term v0.30.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250313205543-e70fdf4c4cb4 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
