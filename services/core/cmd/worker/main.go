package main

import (
	"log/slog"
	"os"

	"github.com/documi/documi/services/core/internal/platform/config"
	"github.com/documi/documi/services/core/internal/platform/logging"
	"github.com/documi/documi/services/core/internal/platform/temporalruntime"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	configValue, err := config.Load()
	if err != nil {
		slog.Error("load configuration", "error", err)
		os.Exit(1)
	}
	logger := logging.New(configValue.LogLevel)
	temporalClient, err := client.Dial(client.Options{
		HostPort:  configValue.TemporalAddress,
		Namespace: configValue.TemporalNamespace,
		Logger:    temporalLogger{logger: logger},
	})
	if err != nil {
		logger.Error("connect to Temporal", "error", err)
		os.Exit(1)
	}
	defer temporalClient.Close()

	registry := worker.New(temporalClient, temporalruntime.DevelopmentTaskQueue, worker.Options{})
	temporalruntime.RegisterDevelopment(registry)
	logger.Info("Temporal worker starting", "task_queue", temporalruntime.DevelopmentTaskQueue)
	if err := registry.Run(worker.InterruptCh()); err != nil {
		logger.Error("Temporal worker stopped", "error", err)
		os.Exit(1)
	}
}

type temporalLogger struct{ logger *slog.Logger }

func (adapter temporalLogger) Debug(message string, keyvals ...interface{}) {
	adapter.logger.Debug(message, keyvals...)
}
func (adapter temporalLogger) Info(message string, keyvals ...interface{}) {
	adapter.logger.Info(message, keyvals...)
}
func (adapter temporalLogger) Warn(message string, keyvals ...interface{}) {
	adapter.logger.Warn(message, keyvals...)
}
func (adapter temporalLogger) Error(message string, keyvals ...interface{}) {
	adapter.logger.Error(message, keyvals...)
}
