package main

import "github.com/chann44/ikyk/gateway/internals"

func main() {
	internals.StartServer(internals.ServerConfig{
		Name:        "ikyk",
		Port:        "8080",
		Environment: "development",
	}, int)
}
