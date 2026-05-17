package main

import (
	"context"
	_ "expvar"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"github.com/mqtt-home/miele-to-mqtt-gw/config"
	"github.com/mqtt-home/miele-to-mqtt-gw/metrics"
	"github.com/mqtt-home/miele-to-mqtt-gw/miele/login"
	"github.com/mqtt-home/miele-to-mqtt-gw/version"
	"github.com/philipparndt/go-logger"
	"github.com/philipparndt/mqtt-gateway/mqtt"
)

// pprofAddr is the bind address for the diagnostic listener. Empty disables.
// We default it on to match `hue-to-mqtt-gw` (which always exposes :6060).
const pprofAddr = ":6060"

func initPprof() {
	if pprofAddr == "" {
		return
	}
	go func() {
		if err := http.ListenAndServe(pprofAddr, nil); err != nil {
			logger.Error("pprof listener failed", "error", err)
		}
	}()
}

func main() {
	logger.Init("info", logger.Logger())
	metrics.Init()
	logger.Info("miele-to-mqtt-gw",
		"version", version.Version,
		"commit", version.GitCommit,
		"built", version.BuildTime,
	)

	if len(os.Args) != 2 {
		logger.Error("Expected config file as argument.")
		os.Exit(1)
	}
	configFile := os.Args[1]
	logger.Info("Loading config", "file", configFile)

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Error("Failed to load config", "error", err)
		os.Exit(1)
	}
	logger.SetLevel(cfg.LogLevel)

	initPprof()

	mgr := login.NewManager()
	mgr.RecoverFromConfig(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := mgr.Login(ctx); err != nil {
		logger.Error("Initial login failed", "error", err)
		os.Exit(1)
	}

	// MQTT connection. mqtt-gateway handles `bridge/state` (online/offline)
	// via its own last-will and on-connect publish.
	mqtt.Start(cfg.MQTT.ToGatewayConfig(), "miele2mqtt")

	app := newApp(cfg, mgr)
	app.start(ctx)

	logger.Info("Application is now ready.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down")
	cancel()
	app.stop()
}
