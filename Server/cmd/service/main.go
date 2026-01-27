// cmd/service/main.go
package main

import (
	"game-server/internal/service"
	"game-server/internal/service/modules/chat"
	"game-server/internal/service/modules/login"
	"log"
)

func main() {
	srv := service.NewServer()

	if err := srv.RegisterModule(&login.Module{}); err != nil {
		log.Fatal(err)
	}
	if err := srv.RegisterModule(&chat.Module{}); err != nil {
		log.Fatal(err)
	}

	log.Println("Service started")
	// TODO: 启动 RPC / TCP / MQ
}
