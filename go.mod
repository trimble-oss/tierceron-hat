module github.com/trimble-oss/tierceron-hat

go 1.26.1

require (
	github.com/lafriks/go-shamir v1.2.0
	github.com/orcaman/concurrent-map/v2 v2.0.1
	golang.org/x/sys v0.41.0
	google.golang.org/grpc v1.79.3
	google.golang.org/protobuf v1.36.10
)

require (
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/klauspost/reedsolomon v1.12.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/tjfoc/gmsm v1.4.1 // indirect
	golang.org/x/time v0.14.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251202230838-ff82c1b0f217 // indirect
)

require (
	github.com/xtaci/kcp-go/v5 v5.6.70
	golang.org/x/crypto v0.48.0
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

replace (
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.40.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.40.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.40.0
	go.opentelemetry.io/otel/sdk/metric => go.opentelemetry.io/otel/sdk/metric v1.40.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.40.0
)
