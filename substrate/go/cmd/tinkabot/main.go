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
	"strings"
	"syscall"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
	"github.com/lagz0ne/tinkabot/substrate/go/tinkabot"
)

var (
	version = "dev"
	commit  = ""
	builtAt = ""
)

type participantSpec struct {
	appID string
	id    string
}

type participantFlags []participantSpec

func (p *participantFlags) String() string {
	if p == nil {
		return ""
	}
	vals := make([]string, 0, len(*p))
	for _, spec := range *p {
		vals = append(vals, spec.appID+":"+spec.id)
	}
	return strings.Join(vals, ",")
}

func (p *participantFlags) Set(raw string) error {
	appID, id, ok := strings.Cut(raw, ":")
	if !ok || appID == "" || id == "" {
		return fmt.Errorf("participant must be <app>:<id>")
	}
	*p = append(*p, participantSpec{appID: appID, id: id})
	return nil
}

type watcherSpec struct {
	name   string
	scope  string
	target string
}

type watcherFlags []watcherSpec

func (w *watcherFlags) String() string {
	if w == nil {
		return ""
	}
	vals := make([]string, 0, len(*w))
	for _, spec := range *w {
		vals = append(vals, spec.name+":"+spec.scope+":"+spec.target)
	}
	return strings.Join(vals, ",")
}

func (w *watcherFlags) Set(raw string) error {
	name, rest, ok := strings.Cut(raw, ":")
	if !ok || name == "" {
		return fmt.Errorf("watcher must be <name>:<item|prefix>:<target>")
	}
	scope, target, ok := strings.Cut(rest, ":")
	if !ok || scope == "" || target == "" {
		return fmt.Errorf("watcher must be <name>:<item|prefix>:<target>")
	}
	*w = append(*w, watcherSpec{name: name, scope: scope, target: target})
	return nil
}

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
	bundle := fs.String("bundle", "", "bundle directory served as an ephemeral app for this run")
	bundleSandbox := fs.String("bundle-sandbox", "", `bundle sandbox tier: "" (default, bwrap, fail-closed) or "none" (trusted, UNSANDBOXED — explicit opt-in)`)
	var participants participantFlags
	fs.Var(&participants, "participant", "admit participant profile as <app>:<id>; repeat for multiple users")
	var watchers watcherFlags
	fs.Var(&watchers, "watcher", "admit watcher profile as <name>:<item|prefix>:<target>; repeat for multiple watchers")
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *showVersion {
		fmt.Fprintln(out, versionString())
		return nil
	}
	if *store == "" {
		return errors.New("--store is required")
	}
	app, err := tinkabot.Start(tinkabot.Config{
		StoreDir:      *store,
		Exposure:      embednats.Loopback(),
		ShellAddr:     *shell,
		DemoSession:   os.Getenv("TB_DEMO_SESSION"),
		BundleDir:     *bundle,
		BundleSandbox: *bundleSandbox,
	})
	if err != nil {
		return err
	}
	stopApp := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.Stop(ctx)
	}
	p := app.Posture()
	fmt.Fprintf(out, "nats   %s\n", p.NATS.ClientURL)
	fmt.Fprintf(out, "shell  %s\n", p.Shell.URL)
	for _, role := range []string{tinkabot.RoleCaller, tinkabot.RoleObserver, tinkabot.RoleAuthor} {
		fmt.Fprintf(out, "creds  %s\n", app.CredsFile(role))
	}
	for _, spec := range participants {
		prof, err := app.AdmitParticipant(spec.appID, spec.id)
		if err != nil {
			stopApp()
			return err
		}
		fmt.Fprintf(out, "participant %s %s %s\n", prof.AppID, prof.ParticipantID, prof.StoreDir)
	}
	for _, spec := range watchers {
		prof, err := app.AdmitWatcher(spec.name, spec.scope, spec.target)
		if err != nil {
			stopApp()
			return err
		}
		fmt.Fprintf(out, "watcher %s %s %s %s\n", prof.Name, prof.Scope, prof.Target, prof.StoreDir)
	}
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return app.Stop(ctx)
}

func versionString() string {
	s := "tinkabot " + version
	if commit != "" {
		s += " " + commit
	}
	if builtAt != "" {
		s += " " + builtAt
	}
	return s
}
