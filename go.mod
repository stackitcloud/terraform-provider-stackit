module github.com/stackitcloud/terraform-provider-stackit

go 1.24

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/terraform-plugin-framework v1.15.0
	github.com/hashicorp/terraform-plugin-framework-validators v0.18.0
	github.com/hashicorp/terraform-plugin-go v0.28.0
	github.com/hashicorp/terraform-plugin-log v0.9.0
	github.com/hashicorp/terraform-plugin-testing v1.13.2
	github.com/stackitcloud/stackit-sdk-go/core v0.17.2
	github.com/stackitcloud/stackit-sdk-go/services/cdn v1.3.0
	github.com/stackitcloud/stackit-sdk-go/services/dns v0.17.0
	github.com/stackitcloud/stackit-sdk-go/services/git v0.6.0
	github.com/stackitcloud/stackit-sdk-go/services/iaas v0.26.0
	github.com/stackitcloud/stackit-sdk-go/services/loadbalancer v1.4.0
	github.com/stackitcloud/stackit-sdk-go/services/logme v0.25.0
	github.com/stackitcloud/stackit-sdk-go/services/mariadb v0.25.0
	github.com/stackitcloud/stackit-sdk-go/services/modelserving v0.5.0
	github.com/stackitcloud/stackit-sdk-go/services/mongodbflex v1.2.1
	github.com/stackitcloud/stackit-sdk-go/services/objectstorage v1.3.0
	github.com/stackitcloud/stackit-sdk-go/services/observability v0.8.0
	github.com/stackitcloud/stackit-sdk-go/services/opensearch v0.24.0
	github.com/stackitcloud/stackit-sdk-go/services/postgresflex v1.2.0
	github.com/stackitcloud/stackit-sdk-go/services/rabbitmq v0.25.0
	github.com/stackitcloud/stackit-sdk-go/services/redis v0.25.0
	github.com/stackitcloud/stackit-sdk-go/services/resourcemanager v0.17.0
	github.com/stackitcloud/stackit-sdk-go/services/secretsmanager v0.13.0
	github.com/stackitcloud/stackit-sdk-go/services/serverbackup v1.3.0
	github.com/stackitcloud/stackit-sdk-go/services/serverupdate v1.2.0
	github.com/stackitcloud/stackit-sdk-go/services/serviceaccount v0.9.0
	github.com/stackitcloud/stackit-sdk-go/services/serviceenablement v1.2.1
	github.com/stackitcloud/stackit-sdk-go/services/ske v0.27.0
	github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex v1.3.0
	github.com/teambition/rrule-go v1.8.2
	golang.org/x/mod v0.25.0
)

require github.com/hashicorp/go-retryablehttp v0.7.7 // indirect

require (
	github.com/ProtonMail/go-crypto v1.1.6 // indirect
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/cloudflare/circl v1.6.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.6.3 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/hc-install v0.9.2 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.23.0 // indirect
	github.com/hashicorp/terraform-json v0.25.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.37.0 // indirect
	github.com/hashicorp/terraform-registry-address v0.2.5 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.2 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/mitchellh/go-wordwrap v1.0.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/oklog/run v1.1.0 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	github.com/stackitcloud/stackit-sdk-go/services/authorization v0.8.0
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/zclconf/go-cty v1.16.3 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.34.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc v1.73.0 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)

tool golang.org/x/tools/cmd/goimports
