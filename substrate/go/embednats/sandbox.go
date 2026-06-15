package embednats

import (
	"os"
	"path/filepath"
	"strings"
)

// SandboxConfig jails a bundle process inside bubblewrap. BundleDir is the
// bundle root (kept read-only inside the jail); Bwrap is the resolved bwrap
// binary path the host preflighted. StoreDir, when set, is the substrate's
// store dir (operator key + role creds) — masked inside the jail so a jailed
// bundle process cannot read the crown jewels through the --ro-bind / /.
type SandboxConfig struct {
	BundleDir string
	Bwrap     string
	StoreDir  string
}

// sandboxCommand wraps command+args into a bwrap argv. The whole filesystem is
// bound read-only (--ro-bind / /); --unshare-all denies the network (deps are
// installed at load, before the jail). --chdir keeps the script's working dir,
// so the outer cmd must not set cmd.Dir.
//
// Masking comes BEFORE re-binding, because bwrap applies ops in order and a
// later bind under a tmpfs would be clobbered:
//   - --tmpfs $HOME hides the operator's home secrets (~/.ssh, ~/.aws, tokens)
//     that --ro-bind / / would otherwise expose — and which a jailed script
//     could surface through a projection value or artifact body.
//   - --tmpfs StoreDir hides the substrate store (operator.nk, role creds).
//   - --tmpfs /tmp gives a private scratch.
//
// Then the toolchain and the bundle are re-exposed: in dev/devbox layouts the
// $PATH binaries (bun, coreutils, sh) AND the bundle dir live UNDER $HOME, so
// after masking $HOME we re-bind the $PATH dirs that fall under it (secrets
// like ~/.ssh are not on $PATH, so they stay masked) plus the bundle dir (ro)
// and the run's output dir (rw). A bundle/outDir under /tmp is likewise
// re-bound after --tmpfs /tmp.
func sandboxCommand(cfg SandboxConfig, command string, args []string, cwd, outDir string) (string, []string) {
	home := os.Getenv("HOME")
	argv := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
	}
	if home != "" {
		argv = append(argv, "--tmpfs", home)
	}
	if cfg.StoreDir != "" {
		argv = append(argv, "--tmpfs", cfg.StoreDir)
	}
	argv = append(argv, "--tmpfs", "/tmp")
	if home != "" {
		for _, dir := range pathDirsUnder(home) {
			argv = append(argv, "--ro-bind", dir, dir)
		}
	}
	argv = append(argv,
		"--ro-bind", cfg.BundleDir, cfg.BundleDir,
		"--bind", outDir, outDir,
		"--chdir", cwd,
		"--unshare-all",
		"--die-with-parent",
		"--",
		command,
	)
	argv = append(argv, args...)
	return cfg.Bwrap, argv
}

// pathDirsUnder returns the existing $PATH entries that are strict subdirs of
// root, de-duplicated — the toolchain dirs to re-expose after masking root.
// Root itself is excluded so re-binding can never undo the mask.
func pathDirsUnder(root string) []string {
	seen := map[string]bool{}
	var dirs []string
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil || seen[abs] {
			continue
		}
		if strings.HasPrefix(abs, root+string(os.PathSeparator)) {
			if _, err := os.Stat(abs); err == nil {
				seen[abs] = true
				dirs = append(dirs, abs)
			}
		}
	}
	return dirs
}
