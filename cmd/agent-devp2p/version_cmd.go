package main

import (
	"runtime/debug"

	"AgentDevp2p/internal/cliout"
	"github.com/urfave/cli/v2"
)

var versionCommand = &cli.Command{
	Name:   "version",
	Usage:  "Prints version information",
	Action: versionAction,
}

func versionAction(ctx *cli.Context) error {
	out := cliout.New(ctx)
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return out.Event("version", map[string]any{"buildInfo": "unavailable"})
	}
	return out.Event("version", map[string]any{
		"goVersion": bi.GoVersion,
		"path":      bi.Path,
		"main":      bi.Main,
	})
}
