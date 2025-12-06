package main

import (
	"os"

	"github.com/chann44/ikyk/apps/internals"
)

func main() {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}

	internals.StartServer(internals.ServerConfig{
		Name:        "ikyk-management-api",
		Port:        port,
		Environment: env,
	}, internals.SetupManagementAPI)
}
