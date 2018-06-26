package main

import (
	"fmt"
	"log"

	"github.com/aporeto-inc/trireme-example/configuration"
	"github.com/aporeto-inc/trireme-example/logger"
	"github.com/aporeto-inc/trireme-example/triremecli"
	"github.com/spf13/cobra"
)

func banner(version, revision string) {
	fmt.Printf(`


	  _____     _
	 |_   _| __(_)_ __ ___ _ __ ___   ___
	   | || '__| | '__/ _ \ '_'' _ \ / _ \
	   | || |  | | | |  __/ | | | | |  __/
	   |_||_|  |_|_|  \___|_| |_| |_|\___|


_______________________________________________________________
             %s - %s
                                                 🚀  by Aporeto

`, version, revision)
}

func main() {
	var err error
	var app *cobra.Command

	// initialize the CLI
	app = configuration.InitCLI(
		triremecli.ProcessRun,
		triremecli.ProcessRun,
		triremecli.ProcessRun,
		triremecli.ProcessDaemon,
		logger.SetLogs,
		func() {
			banner("14", "20")
		},
	)

	// now run the app
	err = app.Execute()
	if err != nil {
		log.Fatalf("runtime error: %s", err.Error())
	}
}
