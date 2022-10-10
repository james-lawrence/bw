module github.com/james-lawrence/bw

go 1.16

require (
	cloud.google.com/go/compute v1.7.0
	github.com/0xAX/notificator v0.0.0-20220220101646-ee9b8921e557
	github.com/akutz/memconn v0.1.0
	github.com/alecthomas/kingpin v2.2.6+incompatible
	github.com/alecthomas/kong v0.6.1
	github.com/anacrolix/stm v0.3.0
	github.com/aws/aws-sdk-go v1.44.37
	github.com/bits-and-blooms/bloom/v3 v3.2.0
	github.com/coreos/go-systemd/v22 v22.3.2
	github.com/corpix/uarand v0.1.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/fsnotify/fsnotify v1.5.4
	github.com/go-acme/lego/v4 v4.7.0
	github.com/gofrs/uuid v4.2.0+incompatible
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/schema v1.2.0
	github.com/gorilla/websocket v1.5.0
	github.com/grantae/certinfo v0.0.0-20170412194111-59d56a35515b
	github.com/gutengo/fil v0.0.0-20150411104140-6109b2e0b5cf
	github.com/hashicorp/go-sockaddr v1.0.2
	github.com/hashicorp/memberlist v0.3.1
	github.com/hashicorp/raft v1.3.9
	github.com/hashicorp/raft-boltdb v0.0.0-20220329195025-15018e9b97e0
	github.com/hashicorp/vault/api v1.7.2
	github.com/icrowley/fake v0.0.0-20180203215853-4178557ae428
	github.com/james-lawrence/torrent v0.0.0-20210617021023-f831c663b447
	github.com/joho/godotenv v1.4.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/logrusorgru/aurora v2.0.3+incompatible
	github.com/manifoldco/promptui v0.9.0
	github.com/mattn/go-isatty v0.0.16
	github.com/miekg/dns v1.1.49
	github.com/naoina/go-stringutil v0.1.0 // indirect
	github.com/naoina/toml v0.1.1
	github.com/onsi/ginkgo/v2 v2.1.4
	github.com/onsi/gomega v1.19.0
	github.com/pkg/errors v0.9.1
	github.com/pkg/profile v1.6.0 // indirect
	github.com/posener/complete v1.2.3
	github.com/pterm/pterm v0.12.37
	github.com/subosito/gotenv v1.2.0
	github.com/traefik/yaegi v0.9.8
	github.com/willabides/kongplete v0.3.0
	golang.org/x/crypto v0.0.0-20220525230936-793ad666bf5e
	golang.org/x/oauth2 v0.0.0-20220608161450-d0670ef3b1eb
	golang.org/x/time v0.0.0-20220609170525-579cf78fd858
	golang.org/x/tools v0.1.11
	google.golang.org/api v0.84.0
	google.golang.org/grpc v1.47.0
	google.golang.org/protobuf v1.28.0
	gopkg.in/yaml.v2 v2.4.0
)

exclude (
	golang.org/x/net v0.0.0-20210726213435-c6fcb2dbf985
	golang.org/x/net v0.0.0-20211015210444-4f30a5c0130f
	golang.org/x/net v0.0.0-20211112202133-69e39bad7dc2
	golang.org/x/net v0.0.0-20211216030914-fe4d6282115f
	golang.org/x/net v0.0.0-20220127200216-cd36cc0744dd
)
