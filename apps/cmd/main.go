package main

import "github.com/chann44/ikyk/apps/internals"

func main() {
	internals.StartServer(internals.ServerConfig{
		Name:        "ikyk-management-api",
		Port:        "9090",
		Environment: "development",
	}, internals.SetupManagementAPI)
}
