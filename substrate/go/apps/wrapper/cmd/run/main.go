// Command run starts the trusted agent wrapper: it connects to embedded NATS
// with a minted trusted-wrapper credential and drives the agent command given
// after "--" over structured stream-json stdio.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/lagz0ne/tinkabot/substrate/go/apps/wrapper"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	natsURL := fs.String("nats", "", "embedded NATS client URL")
	creds := fs.String("creds", "", "trusted-wrapper creds file path")
	session := fs.String("session", "", "session id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if *natsURL == "" || *creds == "" || *session == "" || len(rest) == 0 {
		return errors.New("usage: run --nats <url> --creds <file> --session <id> -- <agent command...>")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cmd := exec.Command(rest[0], rest[1:]...)
	cmd.Stderr = os.Stderr
	h, err := wrapper.StartWrapper(ctx, wrapper.WrapperConfig{
		NATSUrl:   *natsURL,
		CredsFile: *creds,
		SessionID: *session,
		Cmd:       cmd,
	})
	if err != nil {
		return err
	}
	return h.Wait(ctx)
}
