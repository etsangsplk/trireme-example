package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "net/http/pprof"

	"github.com/aporeto-inc/trireme-example/remotebuilder/configuration"
	"github.com/aporeto-inc/trireme-lib/controller"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// setLogs setups Zap to the correct log level and correct output format.
func setLogs(logFormat, logLevel string) error {
	var zapConfig zap.Config

	switch logFormat {
	case "json":
		zapConfig = zap.NewProductionConfig()
		zapConfig.DisableStacktrace = true
	default:
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.DisableStacktrace = true
		zapConfig.DisableCaller = true
		zapConfig.EncoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {}
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set the logger
	switch logLevel {
	case "trace":
		// TODO: Set the level correctly
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "debug":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	case "fatal":
		zapConfig.Level = zap.NewAtomicLevelAt(zap.FatalLevel)
	default:
		zapConfig.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return err
	}

	go func(config zap.Config) {

		defaultLevel := config.Level
		var elevated bool

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGUSR1)
		for s := range c {
			if s == syscall.SIGINT {
				return
			}
			elevated = !elevated

			if elevated {
				config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
				zap.L().Info("Log level elevated to debug")
			} else {
				zap.L().Info("Log level restored to original configuration", zap.String("level", logLevel))
				config.Level = defaultLevel
			}
		}
	}(zapConfig)

	zap.ReplaceGlobals(logger)

	return nil
}

func main() {

	cfg := configuration.NewConfiguration()
	fmt.Println(cfg)
	time.Local = time.UTC

	if cfg.Enforce {
		_, _, cfg.LogLevel, cfg.LogFormat = controller.GetLogParameters()

		err := setLogs(cfg.LogFormat, cfg.LogLevel)
		if err != nil {
			zap.L().Error("Error setting up logs", zap.Error(err))
		}
	}

	if cfg.EnableProfiling {
		go func() {
			fmt.Println(http.ListenAndServe("localhost:6061", nil))
		}()
	}

	if cfg.Enforce {
		if err := controller.LaunchRemoteEnforcer(nil); err != nil {
			zap.L().Fatal("Unable to start enforcer", zap.Error(err))
		}
	}

	zap.L().Debug("Enforcerd stopped")
}
