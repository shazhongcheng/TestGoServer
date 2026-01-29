module game-server

go 1.24.0

require (
	github.com/gorilla/websocket v1.5.1
	github.com/redis/go-redis/v9 v9.17.3
	go.uber.org/zap v0.0.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	golang.org/x/net v0.17.0 // indirect
)

replace go.uber.org/zap => ./internal/third_party/zap
