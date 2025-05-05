module github.com/stackitcloud/terraform-provider-stackit

go 1.24

require (
	github.com/google/go-cmp v0.7.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/hashicorp/terraform-plugin-framework v1.14.1
	github.com/hashicorp/terraform-plugin-framework-validators v0.17.0
	github.com/hashicorp/terraform-plugin-go v0.26.0
	github.com/hashicorp/terraform-plugin-log v0.9.0
	github.com/hashicorp/terraform-plugin-testing v1.12.0
	github.com/stackitcloud/stackit-sdk-go/core v0.17.1
	github.com/stackitcloud/stackit-sdk-go/services/cdn v1.0.0
	github.com/stackitcloud/stackit-sdk-go/services/dns v0.13.2
	github.com/stackitcloud/stackit-sdk-go/services/iaas v0.22.1
	github.com/stackitcloud/stackit-sdk-go/services/loadbalancer v1.0.2
	github.com/stackitcloud/stackit-sdk-go/services/logme v0.22.1
	github.com/stackitcloud/stackit-sdk-go/services/mariadb v0.22.1
	github.com/stackitcloud/stackit-sdk-go/services/modelserving v0.2.2
	github.com/stackitcloud/stackit-sdk-go/services/mongodbflex v1.0.0
	github.com/stackitcloud/stackit-sdk-go/services/objectstorage v1.1.2
	github.com/stackitcloud/stackit-sdk-go/services/observability v0.5.1
	github.com/stackitcloud/stackit-sdk-go/services/opensearch v0.21.1
	github.com/stackitcloud/stackit-sdk-go/services/postgresflex v1.0.3
	github.com/stackitcloud/stackit-sdk-go/services/rabbitmq v0.22.1
	github.com/stackitcloud/stackit-sdk-go/services/redis v0.22.1
	github.com/stackitcloud/stackit-sdk-go/services/resourcemanager v0.13.2
	github.com/stackitcloud/stackit-sdk-go/services/secretsmanager v0.11.3
	github.com/stackitcloud/stackit-sdk-go/services/serverbackup v1.0.2
	github.com/stackitcloud/stackit-sdk-go/services/serverupdate v1.0.2
	github.com/stackitcloud/stackit-sdk-go/services/serviceaccount v0.6.2
	github.com/stackitcloud/stackit-sdk-go/services/serviceenablement v1.0.2
	github.com/stackitcloud/stackit-sdk-go/services/ske v0.22.2
	github.com/stackitcloud/stackit-sdk-go/services/sqlserverflex v1.0.2
	github.com/teambition/rrule-go v1.8.2
	golang.org/x/mod v0.23.0
)

require github.com/hashicorp/go-retryablehttp v0.7.7 // indirect

require (
	github.com/ProtonMail/go-crypto v1.1.3 // indirect
	github.com/agext/levenshtein v1.2.2 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/fatih/color v1.16.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-checkpoint v0.5.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-cty v1.5.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-plugin v1.6.2 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/hashicorp/hc-install v0.9.1 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/hashicorp/logutils v1.0.0 // indirect
	github.com/hashicorp/terraform-exec v0.22.0 // indirect
	github.com/hashicorp/terraform-json v0.24.0 // indirect
	github.com/hashicorp/terraform-plugin-sdk/v2 v2.36.1 // indirect
	github.com/hashicorp/terraform-registry-address v0.2.4 // indirect
	github.com/hashicorp/terraform-svchost v0.1.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
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
	github.com/stackitcloud/stackit-sdk-go/services/authorization v0.6.2
	github.com/stretchr/testify v1.8.4 // indirect
	github.com/vmihailenco/msgpack v4.0.4+incompatible // indirect
	github.com/vmihailenco/msgpack/v5 v5.4.1 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/zclconf/go-cty v1.16.2 // indirect
	golang.org/x/crypto v0.36.0 // indirect
	golang.org/x/net v0.38.0 // indirect
	golang.org/x/sync v0.12.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.23.0 // indirect
	golang.org/x/tools v0.22.0 // indirect
	google.golang.org/appengine v1.6.8 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241015192408-796eee8c2d53 // indirect
	google.golang.org/grpc v1.69.4 // indirect
	google.golang.org/protobuf v1.36.3 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
