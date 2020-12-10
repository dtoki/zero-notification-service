/*
 * Zero Notification Service
 *
 * No description provided (generated by Openapi Generator https://github.com/openapitools/openapi-generator)
 *
 * API version: 1.0.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/commitdev/zero-notification-service/internal/config"
	"github.com/commitdev/zero-notification-service/internal/log"
	"github.com/commitdev/zero-notification-service/internal/server"
	"github.com/commitdev/zero-notification-service/internal/service"
	"go.uber.org/zap"
)

var (
	appVersion = "SNAPSHOT"
	appBuild   = "SNAPSHOT"
)

func main() {
	config := config.GetConfig()

	log.Init(config)
	defer zap.S().Sync() // Flush logs when the process ends

	zap.S().Infow("zero-notification-service", "version", appVersion, "build", appBuild)

	// Heartbeat for liveness check
	go heartbeat()

	EmailApiService := service.NewEmailApiService(config)
	EmailApiController := server.NewEmailApiController(EmailApiService)

	HealthApiService := service.NewHealthApiService(config)
	HealthApiController := server.NewHealthApiController(HealthApiService)

	NotificationApiService := service.NewNotificationApiService(config)
	NotificationApiController := server.NewNotificationApiController(NotificationApiService)

	router := server.NewRouter(EmailApiController, HealthApiController, NotificationApiController)

	serverAddress := fmt.Sprintf("0.0.0.0:%d", config.Port)
	server := &http.Server{Addr: serverAddress, Handler: router}

	// Watch for signals to handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	// Run the server in a goroutine
	go func() {
		zap.S().Infof("Serving at http://%s/", serverAddress)
		err := server.ListenAndServe()
		if err != http.ErrServerClosed {
			zap.S().Fatalf("Fatal error while serving HTTP: %v\n", err)
			close(stop)
		}
	}()

	// Block while reading from the channel until we receive a signal
	sig := <-stop
	zap.S().Infof("Received signal %s, starting graceful shutdown", sig)

	// Give connections some time to drain
	ctx, cancel := context.WithTimeout(context.Background(), config.GracefulShutdownTimeout*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		zap.S().Fatalf("Error during shutdown, client requests have been terminated: %v\n", err)
	} else {
		zap.S().Infof("Graceful shutdown complete")
	}
}

func heartbeat() {
	for range time.Tick(4 * time.Second) {
		fh, err := os.Create("/tmp/service-alive")
		if err != nil {
			zap.S().Warnf("Unable to write file for liveness check!")
		} else {
			fh.Close()
		}
	}
}
