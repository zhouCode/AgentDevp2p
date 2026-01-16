package main

import (
	"errors"

	"AgentDevp2p/internal/cliout"
	"AgentDevp2p/internal/targets"
	"github.com/urfave/cli/v2"
)

var handshakeCommand = &cli.Command{
	Name:  "handshake",
	Usage: "Handshake test commands",
	Subcommands: []*cli.Command{
		handshakeRLPxCommand,
	},
}

var handshakeRLPxCommand = &cli.Command{
	Name:      "rlpx",
	Usage:     "RLPx handshake test (skeleton)",
	ArgsUsage: "<node>",
	Action:    handshakeRLPxAction,
}

func handshakeRLPxAction(ctx *cli.Context) error {
	out := cliout.New(ctx)
	if ctx.NArg() < 1 {
		return errors.New("missing node")
	}
	input := ctx.Args().First()
	t, err := targets.Parse(input)
	if err != nil {
		return err
	}
	return out.Event("handshake.rlpx", map[string]any{
		"target": input,
		"node":   t.NodeString,
		"tcp":    t.TCPEndpoint,
		"status": "not_implemented",
	})
}
