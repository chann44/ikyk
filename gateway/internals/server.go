package internals

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chann44/ikyk/logger"
)

type ServerConfig struct {
	Name        string
	Port        string
	Environment string
}

type GatewaySetupFunc func(log *logger.Logger) http.Handler

func StartServer(config ServerConfig, setupRouter GatewaySetupFunc) {
	loggerConfig := logger.LoggerConfig{
		Environment: config.Environment,
		AppName:     config.Name,
	}
	log := logger.NewLogger(loggerConfig)

	router := setupRouter(log)

	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	go func() {
		log.Info(fmt.Sprintf("%s starting on :%s", config.Name, config.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error(fmt.Sprintf("Server failed to start: %v", err))
		}
	}()

	GracefulShutdown(server, 30*time.Second)
}

func GetEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func GracefulShutdown(server *http.Server, timeout time.Duration) {
	config := logger.LoggerConfig{
		Environment: "development",
		LokiURL:     "http://localhost:3100",
		AppName:     "myapp",
	}

	log := logger.NewLogger(config)

	quit := make(chan os.Signal, 1)

	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error(fmt.Sprintf("Server forced to shutdown: %v", err))
	}
	log.Info("Server exited gracefully")

}
