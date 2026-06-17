// Command tinkalet is the profile-aware edge client for Tinkabot.
package main

import (
	"io"
	"os"

	"github.com/lagz0ne/tinkabot/substrate/go/tinkalet"
)

var (
	version = "dev"
	commit  = ""
	builtAt = ""
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errOut io.Writer) int {
	return tinkalet.Run(tinkalet.Config{
		Args:    args,
		Stdout:  out,
		Stderr:  errOut,
		Env:     os.Environ(),
		Version: versionString(),
	})
}

func versionString() string {
	s := "tinkalet " + version
	if commit != "" {
		s += " " + commit
	}
	if builtAt != "" {
		s += " " + builtAt
	}
	return s
}
