package main

import "github.com/chann44/ikyk/internals"

func main() {
	internals.StartServer(internals.ServerConfig{
		Name:        "ikyk",
		Port:        "8080",
		Environment: "development",
	}, internals.SetupGateway)
}
