module github.com/trimble-oss/tierceron-hat

go 1.26.4

require (
	github.com/lafriks/go-shamir v1.2.0
	github.com/orcaman/concurrent-map/v2 v2.0.1
	golang.org/x/sys v0.47.0
	google.golang.org/grpc v1.82.1
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
)

require (
	github.com/quic-go/quic-go v0.61.0
	golang.org/x/crypto v0.54.0
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/text v0.40.0 // indirect
)

replace (
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.43.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/metric => go.opentelemetry.io/otel/sdk/metric v1.43.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.43.0
)
