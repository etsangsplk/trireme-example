package main

import (
	"fmt"
	"net/http"
	"time"

	_ "net/http/pprof"

	"github.com/aporeto-inc/trireme-example/logger"
	"github.com/aporeto-inc/trireme-example/remotebuilder/configuration"
	"github.com/aporeto-inc/trireme-lib/controller"
	"go.uber.org/zap"
)

func main() {

	cfg := configuration.NewConfiguration()
	fmt.Println(cfg)
	time.Local = time.UTC

	if cfg.Enforce {
		_, _, cfg.LogLevel, cfg.LogFormat = controller.GetLogParameters()

		err := logger.SetLogs(cfg.LogFormat, cfg.LogLevel)
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
