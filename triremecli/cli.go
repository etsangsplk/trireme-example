package triremecli

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/aporeto-inc/trireme-example/configuration"
	"github.com/aporeto-inc/trireme-example/extractors"
	"github.com/aporeto-inc/trireme-example/policyexample"
	"github.com/aporeto-inc/trireme-example/utils"

	"github.com/aporeto-inc/trireme-lib/cmd/systemdutil"
	"github.com/aporeto-inc/trireme-lib/collector"
	"github.com/aporeto-inc/trireme-lib/controller"
	"github.com/aporeto-inc/trireme-lib/controller/pkg/secrets"
	"github.com/aporeto-inc/trireme-lib/monitor"
)

// KillContainerOnError defines if the Container is getting killed if the policy Application resulted in an error
const KillContainerOnError = true

// ProcessArgs handles all commands options for trireme
func ProcessArgs(config *configuration.Configuration) (err error) {

	if config.Enforce {
		return ProcessEnforce(config)
	}

	if config.Run {
		// Execute a command or process a cgroup cleanup and exit
		return ProcessRun(config)
	}

	// Trireme Daemon Commands
	return ProcessDaemon(config)
}

// ProcessEnforce is called if the application is run as remote enforcer
func ProcessEnforce(config *configuration.Configuration) (err error) {
	// Run enforcer and exit
	if err := controller.LaunchRemoteEnforcer(nil); err != nil {
		zap.L().Fatal("Unable to start enforcer", zap.Error(err))
	}
	return nil
}

// ProcessRun is called when the application is either adding or removing
// Trireme to a cgroup, or if an application is wrapped with trireme ("run")
func ProcessRun(config *configuration.Configuration) (err error) {
	return systemdutil.ExecuteCommandFromArguments(config.Arguments)
}

// ProcessDaemon is called when trireme-example is called to start the daemon
func ProcessDaemon(config *configuration.Configuration) (err error) {

	// Setting up Secret Auth type based on user config.
	var triremesecret secrets.Secrets
	if config.Auth == configuration.PSK {
		zap.L().Info("Initializing Trireme with PSK Auth. Should NOT be used in production")
		triremesecret = secrets.NewPSKSecrets([]byte(config.PSK))
	} else if config.Auth == configuration.PKI {
		zap.L().Info("Initializing Trireme with PKI Auth")
		triremesecret, err = utils.LoadCompactPKI(config.KeyPath, config.CertPath, config.CaCertPath, config.CaKeyPath)
		if err != nil {
			zap.L().Fatal("error creating PKI Secret for Trireme", zap.Error(err))
		}
	} else {
		zap.L().Fatal("No Authentication option given")
	}

	collectorInstance := collector.NewDefaultCollector()

	controllerOptions := []controller.Option{
		controller.OptionSecret(triremesecret),
		controller.OptionCollector(collectorInstance),
		controller.OptionEnforceLinuxProcess(),
		controller.OptionTargetNetworks(config.ParsedTriremeNetworks),
		controller.OptionProcMountPoint("/proc"),
	}
	if config.LogLevel == "trace" {
		controllerOptions = append(controllerOptions, controller.OptionPacketLogs())
	}

	// Docker options
	dockerOptions := []monitor.DockerMonitorOption{}
	if config.SwarmMode {
		dockerOptions = append(dockerOptions, monitor.SubOptionMonitorDockerExtractor(extractors.SwarmExtractor))
	}

	// Setting up extractor and monitor
	monitorOptions := []monitor.Options{
		monitor.OptionMonitorLinuxProcess(),
		monitor.OptionCollector(collectorInstance),
		monitor.OptionMonitorDocker(dockerOptions...),
		monitor.OptionMonitorUID(),
	}

	// Initialize the controllers
	ctrl := controller.New("ExampleNode", controllerOptions...)
	if ctrl == nil {
		zap.L().Fatal("Unable to initialize trireme")
	}

	// Initialize the policy resolver
	policyEngine := policyexample.NewCustomPolicyResolver(ctrl, config.ParsedTriremeNetworks, config.PolicyFile)

	// Initialize the monitors
	monitorOptions = append(monitorOptions, monitor.OptionPolicyResolver(policyEngine))
	m, err := monitor.NewMonitors(monitorOptions...)
	if err != nil {
		zap.L().Fatal("Unable to initialize monitor: %s", zap.Error(err))
	}

	// Start all the go routines.
	ctx, cancel := context.WithCancel(context.Background())

	if err := ctrl.Run(ctx); err != nil {
		zap.L().Fatal("Failed to start controller")
	}

	if err := m.Run(ctx); err != nil {
		zap.L().Fatal("Failed to start monitor")
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	zap.L().Info("Everything started. Waiting for Stop signal")
	// Waiting for a Signal
	<-c
	zap.L().Debug("Stop signal received")
	ctrl.CleanUp()
	cancel()
	zap.L().Info("Everything stopped. Bye Trireme-Example!")

	return nil
}
