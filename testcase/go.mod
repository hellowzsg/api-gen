module github.com/hellowzsg/api-gen/testcase

go 1.25.0

require (
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.29.0
	github.com/hellowzsg/api-gen/testcase/fixtures/book v0.0.0-00010101000000-000000000000
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11
)

require (
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260715232425-e75dac1f907d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
)

replace (
	github.com/hellowzsg/api-gen/testcase/fixtures/book => ./fixtures/book
	github.com/hellowzsg/api-gen/testcase/fixtures/edge => ./fixtures/edge
	github.com/hellowzsg/api-gen/testcase/fixtures/simple => ./fixtures/simple
)
