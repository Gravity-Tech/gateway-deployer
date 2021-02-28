package main

import (
	"log"
	"os"

	"github.com/Gravity-Tech/gateway-deployer/ethereum/cmd"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "Gravity test framework CLI",
		Usage: "the gravity test framework command line interface",
		Commands: []*cli.Command{
			cmd.EthereumCommand,
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
