package main

import (
	"errors"
	"net"
	"time"

	"AgentDevp2p/internal/cliout"
	"AgentDevp2p/internal/targets"
	"github.com/urfave/cli/v2"
)

var probeCommand = &cli.Command{
	Name:  "probe",
	Usage: "Connectivity probes",
	Subcommands: []*cli.Command{
		probeTCPCommand,
	},
}

var probeTCPCommand = &cli.Command{
	Name:      "tcp",
	Usage:     "TCP connect probe",
	ArgsUsage: "<target>",
	Flags: []cli.Flag{
		&cli.DurationFlag{Name: "timeout", Value: 3 * time.Second, Usage: "Dial timeout"},
	},
	Action: probeTCPAction,
}

func probeTCPAction(ctx *cli.Context) error {
	out := cliout.New(ctx)
	if ctx.NArg() < 1 {
		return errors.New("missing target")
	}
	input := ctx.Args().First()
	timeout := ctx.Duration("timeout")

	endpoint, endpointErr := targets.ResolveTCPEndpoint(input)
	start := time.Now()
	if endpointErr != nil {
		return out.Event("probe.tcp", map[string]any{
			"target": input,
			"ok":     false,
			"error":  endpointErr.Error(),
		})
	}

	conn, err := net.DialTimeout("tcp", endpoint, timeout)
	lat := time.Since(start)
	if err != nil {
		return out.Event("probe.tcp", map[string]any{
			"target":   input,
			"endpoint": endpoint,
			"ok":       false,
			"latency":  lat.String(),
			"error":    err.Error(),
		})
	}
	_ = conn.Close()
	return out.Event("probe.tcp", map[string]any{
		"target":   input,
		"endpoint": endpoint,
		"ok":       true,
		"latency":  lat.String(),
	})
}
