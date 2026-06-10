// Command tinkabot is the v1 product entry surface: one binary embedding
// NATS in operator/JWT mode, the frontend shell, and the script materializer
// loop (docs/manual/v1.md "Starting the binary").
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/lagz0ne/tinkabot/substrate/go/tinkabot"
)

func main() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	if err := run(os.Args[1:], os.Stdout, sig); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run starts the assembly, prints the served posture, and drains on the
// first signal. Split from main so the entry point is testable.
func run(args []string, out io.Writer, sig <-chan os.Signal) error {
	fs := flag.NewFlagSet("tinkabot", flag.ContinueOnError)
	store := fs.String("store", "", "durable store directory (operator key, JetStream state, role creds)")
	shell := fs.String("shell", "127.0.0.1:8419", "loopback address for the embedded shell")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *store == "" {
		return errors.New("--store is required")
	}
	app, err := tinkabot.Start(tinkabot.Config{
		StoreDir:  *store,
		Exposure:  embednats.Loopback(),
		ShellAddr: *shell,
	})
	if err != nil {
		return err
	}
	p := app.Posture()
	fmt.Fprintf(out, "nats   %s\n", p.NATS.ClientURL)
	fmt.Fprintf(out, "shell  %s\n", p.Shell.URL)
	for _, role := range []string{tinkabot.RoleCaller, tinkabot.RoleObserver, tinkabot.RoleAuthor} {
		fmt.Fprintf(out, "creds  %s\n", app.CredsFile(role))
	}
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return app.Stop(ctx)
}
