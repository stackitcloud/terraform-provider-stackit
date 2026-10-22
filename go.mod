module github.com/stackitcloud/terraform-provider-stackit

go 1.26.10

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/terraform-plugin-framework v1.19.0
	github.com/hashicorp/terraform-plugin-framework-jsontypes v0.2.0
	github.com/hashicorp/terraform-plugin-framework-timeouts v0.7.0
	github.com/hashicorp/terraform-plugin-framework-validators v0.19.0
	github.com/hashicorp/terraform-plugin-go v0.31.0
	github.com/hashicorp/terraform-plugin-log v0.11.0
	github.com/hashicorp/terraform-plugin-testing v1.16.0
	github.com/stackitcloud/stackit-sdk-go/core v0.27.1
	github.com/stackitcloud/stackit-sdk-go/services/alb v0.19.0
	github.com/stackitcloud/stackit-sdk-go/services/albwaf v0.13.5
	github.com/stackitcloud/stackit-sdk-go/services/automation v0.2.1
	github.com/stackitcloud/stackit-sdk-go/services/cdn v1.22.0
	github.com/stackitcloud/stackit-sdk-go/services/certificates v1.10.0
	github.com/stackitcloud/stackit-sdk-go/services/dns v0.25.0
	github.com/stackitcloud/stackit-sdk-go/services/dremio v0.7.0
	github.com/stackitcloud/stackit-sdk-go/services/edge v0.14.0
	github.com/stackitcloud/stackit-sdk-go/services/git v0.15.0
	github.com/stackitcloud/stackit-sdk-go/services/iaas v1.15.1
	github.com/stackitcloud/stackit-sdk-go/services/intake v0.12.0
	github.com/stackitcloud/stackit-sdk-go/services/kms v1.14.0
	github.com/stackitcloud/stackit-sdk-go/services/loadbalancer v1.16.0
	github.com/stackitcloud/stackit-sdk-go/services/logme v1.5.0
	github.com/stackitcloud/stackit-sdk-go/services/logs v0.11.0
	github.com/stackitcloud/stackit-sdk-go/services/mariadb v1.4.0
	github.com/stackitcloud/stackit-sdk-go/services/modelexperiments v0.3.2
	github.com/stackitcloud/stackit-sdk-go/services/modelserving v0.14.0
	github.com/stackitcloud/stackit-sdk-go/services/mongodbflex v1.13.0
	github.com/stackitcloud/stackit-sdk-go/services/objectstorage v1.12.0
	github.com/stackitcloud/stackit-sdk-go/services/observability v0.26.0
	github.com/stackitcloud/stackit-sdk-go/services/opensearch v1.5.0
	github.com/stackitcloud/stackit-sdk-go/services/postgresflex v1.16.0
	github.com/stackitcloud/stackit-sdk-go/services/rabbitmq v1.5.0
	github.com/stackitcloud/stackit-sdk-go/services/redis v1.6.0
	github.com/stackitcloud/stackit-sdk-go/services/resourcemanager v0.27.0
	github.com/stackitcloud/stackit-sdk-go/services/scf v0.11.0
	github.com/stackitcloud/stackit-sdk-go/services/secretsmanager v0.20.0
	github.com/stackitcloud/stackit-sdk-go/services/serverbackup v1.8.0
	github.com/stackitcloud/stackit-sdk-go/services/serverupdate v1.6.0
	github.com/stackitcloud/stackit-sdk-go/services/serviceaccount v0.22.0
	github.com/stackitcloud/stackit-sdk-go/services/serviceenablement v1.8.0
	github.com/stackitcloud/stackit-sdk-go/services/sfs v0.13.0
	github.com/stackitcloud/stackit-sdk-go/services/ske v1.22.0
	github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex v1.19.0
	github.com/stackitcloud/stackit-sdk-go/services/telemetrylink v0.6.0
	github.com/stackitcloud/stackit-sdk-go/services/telemetryrouter v0.6.1
	github.com/stackitcloud/stackit-sdk-go/services/ufw v0.1.1
	github.com/stackitcloud/stackit-sdk-go/services/valkey v0.4.0
	github.com/stackitcloud/stackit-sdk-go/services/vpn v0.15.3
	github.com/teambition/rrule-go v1.8.2
	go.uber.org/mock v0.6.0
	golang.org/x/mod v0.42.0
)

require (
	github.com/hashicorp/go-retryablehttp v0.7.8 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/stretchr/testify v1.11.1 // indirect
)

require (
	github.com/ProtonMail/go-crypto v1.4.1 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/cloudflare/circl v1.6.3 // indirect
	github.com/fatih/color v1.19.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.3.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.7.0 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/hc-install v0.9.4 // indirect
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.25.1 // indirect
	github.com/hashicorp/terraform-json v0.29.0
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.40.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.4.0 // indirect
	github.com/hashicorp/terraform-svchost v0.2.1 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.2.0 // indirect
	github.com/stackitcloud/stackit-sdk-go/services/authorization v0.16.0
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/zclconf/go-cty v1.18.1 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/net v0.59.0 // indirect
	golang.org/x/sync v0.24.0
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	golang.org/x/tools v0.51.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/grpc v1.83.2 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)
