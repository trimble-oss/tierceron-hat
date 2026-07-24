module github.com/trimble-oss/tierceron-hat/brimfeather

go 1.26.4

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
	github.com/quic-go/quic-go v0.60.0 // indirect
	golang.org/x/crypto v0.52.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260414002931-afd174a4e478 // indirect
	google.golang.org/grpc v1.82.1 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace (
	github.com/trimble-oss/tierceron-hat => ../
	go.opentelemetry.io/otel => go.opentelemetry.io/otel v1.43.0
	go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v1.43.0
	go.opentelemetry.io/otel/sdk => go.opentelemetry.io/otel/sdk v1.43.0
	go.opentelemetry.io/otel/sdk/metric => go.opentelemetry.io/otel/sdk/metric v1.43.0
	go.opentelemetry.io/otel/trace => go.opentelemetry.io/otel/trace v1.43.0
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20251202230838-ff82c1b0f217
)
