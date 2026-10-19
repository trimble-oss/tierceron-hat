module github.com/trimble-oss/tierceron-hat/brimfeather

go 1.27.2

require (
	github.com/trimble-oss/tierceron-hat v0.0.0-00010101000000-000000000000
	github.com/vbauerster/mpb/v8 v8.12.0
)

require (
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/lafriks/go-shamir v1.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.20 // indirect
	github.com/orcaman/concurrent-map/v2 v2.0.1 // indirect
	github.com/quic-go/quic-go v0.61.0 // indirect
	golang.org/x/crypto v0.55.0 // indirect
	golang.org/x/net v0.58.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260706201446-f0a921348800 // indirect
	google.golang.org/grpc v1.84.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace (
	github.com/trimble-oss/tierceron-hat => ../
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp => go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.68.0
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.43.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/metric => go.opentelemetry.io/otel/sdk/metric v1.43.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.43.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20251202230838-ff82c1b0f217
)
