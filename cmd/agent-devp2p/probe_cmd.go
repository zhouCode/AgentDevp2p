package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
	"time"

	"AgentDevp2p/internal/cliout"
	"AgentDevp2p/internal/runner"
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
	ArgsUsage: "[targets...]",
	Flags: []cli.Flag{
		&cli.StringSliceFlag{Name: "targets", Usage: "Target descriptors (repeatable, comma-separated supported)"},
		&cli.StringFlag{Name: "targets-file", Usage: "Read targets from file (one per line, '-' for stdin)"},
		&cli.DurationFlag{Name: "timeout", Value: 3 * time.Second, Usage: "Per-target dial timeout"},
		&cli.IntFlag{Name: "concurrency", Value: 6, Usage: "Max concurrent sessions"},
		&cli.IntFlag{Name: "retries", Value: 0, Usage: "Retries per target on dial_error/timeout"},
		&cli.StringFlag{Name: "run-id", Usage: "Run identifier (auto if empty)"},
		&cli.StringFlag{Name: "case-id", Value: "probe.tcp", Usage: "Case identifier"},
		&cli.StringFlag{Name: "record-dir", Value: "runs", Usage: "Directory for per-target records"},
	},
	Action: probeTCPAction,
}

func probeTCPAction(ctx *cli.Context) error {
	out := cliout.New(ctx)
	inputs, err := probeTargetsInputs(ctx)
	if err != nil {
		return err
	}
	if len(inputs) == 0 {
		return errors.New("need at least one target (args, --targets, or --targets-file)")
	}

	runID := ctx.String("run-id")
	if runID == "" {
		runID = runner.NewRunID()
	}
	caseID := ctx.String("case-id")
	p := runner.TCPProbeParams{
		RunID:       runID,
		CaseID:      caseID,
		Timeout:     ctx.Duration("timeout"),
		Concurrency: ctx.Int("concurrency"),
		Retries:     ctx.Int("retries"),
		RecordDir:   ctx.String("record-dir"),
	}

	_ = out.Event("run.start", map[string]any{
		"run_id":  runID,
		"case_id": caseID,
		"targets": len(inputs),
	})

	var probeTargets []runner.TCPProbeTarget
	for _, in := range inputs {
		t, err := targets.ParseAny(in)
		if err != nil {
			r := runner.TCPSessionRecord{
				RunID:      runID,
				CaseID:     caseID,
				TargetID:   "local_" + runner.SafePathComponent(in),
				TargetRaw:  in,
				Endpoint:   "",
				StartTime:  time.Now().UTC().Format(time.RFC3339Nano),
				DialStart:  time.Now().UTC().Format(time.RFC3339Nano),
				DialEnd:    time.Now().UTC().Format(time.RFC3339Nano),
				EndTime:    time.Now().UTC().Format(time.RFC3339Nano),
				LatencyMS:  0,
				OK:         false,
				ErrorClass: string(runner.ErrorClassLocalError),
				Error:      err.Error(),
			}
			if p.RecordDir != "" {
				_ = runner.WriteTCPRecord(p.RecordDir, runID, caseID, r.TargetID, r)
			}
			_ = out.Event("session", r)
			continue
		}
		if t.TCPEndpoint == "" {
			r := runner.TCPSessionRecord{
				RunID:      runID,
				CaseID:     caseID,
				TargetID:   t.TargetID,
				TargetRaw:  t.Raw,
				Endpoint:   "",
				StartTime:  time.Now().UTC().Format(time.RFC3339Nano),
				DialStart:  time.Now().UTC().Format(time.RFC3339Nano),
				DialEnd:    time.Now().UTC().Format(time.RFC3339Nano),
				EndTime:    time.Now().UTC().Format(time.RFC3339Nano),
				LatencyMS:  0,
				OK:         false,
				ErrorClass: string(runner.ErrorClassLocalError),
				Error:      "node has no TCP endpoint",
			}
			if p.RecordDir != "" {
				_ = runner.WriteTCPRecord(p.RecordDir, runID, caseID, r.TargetID, r)
			}
			_ = out.Event("session", r)
			continue
		}
		probeTargets = append(probeTargets, runner.TCPProbeTarget{ID: t.TargetID, Raw: in, Endpoint: t.TCPEndpoint})
	}

	_ = runner.RunTCPProbe(ctx.Context, p, probeTargets, func(r runner.TCPSessionRecord) {
		_ = out.Event("session", r)
	})

	_ = out.Event("run.end", map[string]any{
		"run_id":  runID,
		"case_id": caseID,
	})
	return nil
}

func probeTargetsInputs(ctx *cli.Context) ([]string, error) {
	var inputs []string
	inputs = append(inputs, splitTargets(ctx.StringSlice("targets"))...)
	inputs = append(inputs, splitTargets(ctx.Args().Slice())...)

	filePath := strings.TrimSpace(ctx.String("targets-file"))
	if filePath == "" {
		return normalizeNonEmpty(inputs), nil
	}

	var fd *os.File
	var err error
	if filePath == "-" {
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

func splitTargets(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		parts := strings.Split(s, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
	}
	return out
}
