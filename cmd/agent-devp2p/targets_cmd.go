package main

import (
	"bufio"
	"errors"
	"os"
	"strings"

	"AgentDevp2p/internal/cliout"
	"AgentDevp2p/internal/targets"
	"github.com/urfave/cli/v2"
)

var targetsCommand = &cli.Command{
	Name:  "targets",
	Usage: "Target utilities",
	Subcommands: []*cli.Command{
		targetsParseCommand,
	},
}

var targetsParseCommand = &cli.Command{
	Name:      "parse",
	Usage:     "Parses and validates target descriptors",
	ArgsUsage: "[targets...]",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "file", Usage: "Read targets from file (one per line, '-' for stdin)"},
	},
	Action: targetsParseAction,
}

func targetsParseAction(ctx *cli.Context) error {
	out := cliout.New(ctx)
	inputs, err := targetsParseInputs(ctx)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errors.New("need at least one target (arg or --file)")
	}

	for _, in := range inputs {
		t, err := targets.ParseAny(in)
		if err != nil {
			_ = out.Event("target", map[string]any{"raw": in, "ok": false, "error": err.Error()})
			continue
		}
		_ = out.Event("target", map[string]any{
			"raw":       in,
			"ok":        true,
			"target_id": t.TargetID,
			"node":      t.NodeString,
			"tcp":       t.TCPEndpoint,
			"udp":       t.UDPEndpoint,
		})
	}
	return nil
}

func targetsParseInputs(ctx *cli.Context) ([]string, error) {
	var inputs []string
	inputs = append(inputs, ctx.Args().Slice()...)
	filePath := ctx.String("file")
	if filePath == "" {
		return normalizeNonEmpty(inputs), nil
	}

	var fd *os.File
	var err error
	if strings.TrimSpace(filePath) == "-" {
		fd = os.Stdin
	} else {
		fd, err = os.Open(filePath)
		if err != nil {
			return nil, err
		}
		defer fd.Close()
	}

	s := bufio.NewScanner(fd)
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" {
			continue
		}
		inputs = append(inputs, line)
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return normalizeNonEmpty(inputs), nil
}

func normalizeNonEmpty(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
