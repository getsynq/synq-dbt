module github.com/getsynq/synq-dbt

go 1.22

replace github.com/getsynq/cloud/api => ./gen/

require (
	buf.build/gen/go/getsynq/api/grpc/go v1.5.1-20241009130006-b2ed3af4a469.1
	buf.build/gen/go/getsynq/api/protocolbuffers/go v1.35.1-20241009130006-b2ed3af4a469.1
	github.com/getsynq/cloud/api v0.0.0-20241009145912-9106b17e788b
	github.com/json-iterator/go v1.1.12
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	github.com/t-tomalak/logrus-easy-formatter v0.0.0-20190827215021-c074f06c5816
	golang.org/x/oauth2 v0.23.0
	google.golang.org/grpc v1.67.1
	google.golang.org/protobuf v1.35.1

)

require (
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20241007155032-5fefd90f89a9 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
