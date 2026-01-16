package cliout

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"
)

type Output struct {
	json bool
	w    io.Writer
}

func New(ctx *cli.Context) *Output {
	return &Output{json: ctx.Bool("json"), w: os.Stdout}
}

func (o *Output) Event(name string, payload any) error {
	if o.json {
		b, err := json.Marshal(map[string]any{
			"ts":    time.Now().UTC().Format(time.RFC3339Nano),
			"event": name,
			"data":  payload,
		})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(o.w, string(b))
		return err
	}
	_, err := fmt.Fprintf(o.w, "%s %s %v\n", time.Now().UTC().Format(time.RFC3339Nano), name, payload)
	return err
}
