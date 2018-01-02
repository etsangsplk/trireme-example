package configuration

import (
	docopt "github.com/docopt/docopt-go"
)

// AuthType is a type that holds an Authentication method
type AuthType int

const (
	// PSK is the Authentication method that relies on a preshared Key (unsecure)
	PSK AuthType = iota + 1
	// PKI is the Authentication methid that relies on a PKI
	PKI
)

// Configuration holds the whole configuration for Trireme-Example
type Configuration struct {
	// Arguments is the retrocompatible format used to define the parameters//process to run
	Arguments map[string]interface{}
	// AuthType defines if Trireme uses PSK or PKI
	Auth AuthType
	// PSK is the PSK used for Trireme (if using PSK)
	PSK string

	// Set of Policies to be used with this example.
	PolicyFile string

	// Launch Trireme-Example with support for CustomExtractor
	CustomExtractor string

	// KeyPath is the path to the Key in PEM encoded format
	KeyPath string
	// CertPath is the path to the Cert in PEM encoded format
	CertPath string
	// CaCertPath is the path to the CaCert in PEM encoded format
	CaCertPath string
	// CaKeyPath is the path to the CaKey in PEM encoded format
	CaKeyPath string

	TriremeNetworks       string
	ParsedTriremeNetworks []string

	LogFormat string
	LogLevel  string

	// RemoteEnforcer defines if the enforcer is spawned into each POD namespace
	// or into the host default namespace.
	RemoteEnforcer bool

	DockerEnforcement bool
	// LinuxProcesses defines if we activate//police LinuxProcesses
	LinuxProcessesEnforcement bool

	// Launch Trireme-Example with support for Swarm
	SwarmMode bool

	// Enforce defines if this process is an enforcer process (spawned into POD namespaces)
	Enforce bool `mapstructure:"Enforce"`
	// Run defines if this process is used to run a command
	Run bool
}

// getArguments return the whole set of arguments for Trireme-Example
func getArguments() (map[string]interface{}, error) {

	usage := `Command for launching programs with Trireme policy.

  Usage:
    trireme-example -h | --help
    trireme-example --version
    trireme-example run
      [--service-name=<sname>]
      [[--label=<keyvalue>]...]
      [--ports=<ports>]
      <command> [--] [<params>...]
    trireme-example daemon
      [--target-networks=<networks>...]
      [--policy=<policyFile>]
      [--usePKI]
      [--hybrid|--remote|--local]
      [--swarm|--extractor <metadatafile>]
      [--keyFile=<keyFile>]
      [--certFile=<certFile>]
      [--caCertFile=<caCertFile>]
      [--caKeyFile=<caKeyFile>]
      [--log-level=<log-level>]
      [--log-level-remote=<log-level>]
      [--log-to-console]
    trireme-example enforce --log-id=<log-id>
      [--log-level=<log-level>]
    trireme-example <cgroup>

  Options:
    -h --help                              Show this help message and exit.
    --version                              show version and exit.
    --service-name=<sname>                 The name of the service to be launched.
    --label=<keyvalue>                     The metadata/labels associated with a service.
    --usePKI                               Use PKI for Trireme [default: false].
    --certFile=<certFile>                  Certificate file [default: certs/cert.pem].
    --keyFile=<keyFile>                    Key file [default: certs/cert-key.pem].
    --caCertFile=<caCertFile>              CA certificate [default: certs/ca.pem].
    --caKeyFile=<caKeyFile>                CA key [default: certs/ca-key.pem].
    --hybrid                               Hybrid mode of deployment [default: false].
    --remote                               Remote mode of deployment [default: false].
    --local                                Local mode of deployment [default: true].
    --swarm                                Deploy Doccker Swarm metadata extractor [default: false].
    --extractor                            External metadata extractor [default: ].
    --policy=<policyFile>                  Policy file [default: policy.json].
    --target-networks=<networks>...        The target networks that Trireme should apply authentication [default: ]
    <cgroup>                               cgroup of process that are terminated.

Logging Options:
    --log-level=<log-level>                Log level [default: info].
    --log-level-remote=<log-level>         Log level for remote enforcers [default: info].
    --log-id=<log-id>                      Log identifier.
    --log-to-console                       Log to console [default: true].
  `

	return docopt.Parse(usage, nil, true, "1.0.0rc2", false)
}

// LoadConfig returns a Configuration struct ready to use.
// TODO: It uses DocOpt as the end config manager. Eventually move everything in Viper.
func LoadConfig() (*Configuration, error) {
	config := &Configuration{}

	oldArgs, err := getArguments()
	if err != nil {
		return nil, err
	}
	config.Arguments = oldArgs

	if oldArgs["run"].(bool) || oldArgs["<cgroup>"] != nil {
		// Execute a command or process a cgroup cleanup and exit
		config.Run = true
	}

	if oldArgs["enforce"].(bool) {
		// Execute a command or process a cgroup cleanup and exit
		config.Enforce = true
	}

	if len(oldArgs["--target-networks"].([]string)) > 0 {
		config.ParsedTriremeNetworks = oldArgs["--target-networks"].([]string)
	}

	config.PolicyFile = oldArgs["--policy"].(string)

	if oldArgs["--usePKI"].(bool) {
		config.Auth = PKI
		config.CertPath = oldArgs["--certFile"].(string)
		config.KeyPath = oldArgs["--keyFile"].(string)
		config.CaCertPath = oldArgs["--caCertFile"].(string)
		config.CaKeyPath = oldArgs["--caKeyFile"].(string)

	} else {
		config.Auth = PSK
		config.PSK = "BADPASS"
	}

	config.DockerEnforcement = true
	if oldArgs["--hybrid"].(bool) {
		config.LinuxProcessesEnforcement = true
	}

	if oldArgs["--remote"].(bool) {
		config.RemoteEnforcer = true
	}

	if oldArgs["--local"].(bool) {
		config.RemoteEnforcer = false
	}

	if oldArgs["--swarm"].(bool) {
		config.SwarmMode = true
	}

	if oldArgs["--extractor"].(bool) {
		config.CustomExtractor = oldArgs["metadatafile"].(string)
	}

	config.LogLevel = oldArgs["--log-level"].(string)
	config.LogFormat = oldArgs["--log-format"].(string)

	return config, nil
}
