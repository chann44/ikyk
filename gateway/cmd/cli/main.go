package main

import (
	"os"

	"github.com/chann44/ikyk/gateway/internals"
)

func main() {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	internals.StartServer(internals.ServerConfig{
		Name:        "ikyk",
		Port:        port,
		Environment: env,
	}, internals.SetupGateway)
}
