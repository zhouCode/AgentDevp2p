package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
)

var app = &cli.App{
	Name:  "agent-devp2p",
	Usage: "AgentDevp2p devp2p differential fuzzing tool",
	Flags: []cli.Flag{
		&cli.BoolFlag{Name: "json", Usage: "Output JSON lines"},
	},
}

func init() {
	app.CommandNotFound = func(ctx *cli.Context, cmd string) {
		fmt.Fprintf(os.Stderr, "No such command: %s\n", cmd)
		os.Exit(1)
	}

	app.Commands = []*cli.Command{
		versionCommand,
		targetsCommand,
		probeCommand,
		handshakeCommand,
	}
}

func main() {
	exit(app.Run(os.Args))
}

func exit(err interface{}) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
