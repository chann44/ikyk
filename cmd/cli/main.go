package main

import "github.com/chann44/ikyk/internals"

func main() {
	StartServer(ServerConfig{
		Name:        "ikyk",
		Port:        "8080",
		Environment: "development",
	}, internals.SetupGateway)
}
