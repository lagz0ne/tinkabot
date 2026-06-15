package embednats

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Sandbox is the contract the substrate depends on instead of bwrap directly.
// Preflight proves the sandbox can actually jail before any bundle entry is
// wired (fail-closed for the default impl); Wrap turns a SandboxSpec into the
// command+args to exec; Name identifies the tier for logs.
type Sandbox interface {
	Preflight() error
	Wrap(spec SandboxSpec) (command string, args []string)
	Name() string
}

// SandboxSpec describes one jailed run. The same spec shape drives both the
// runtime jail (BundleWritable=false, OutDir set, Network=false) and the
// install jail (BundleWritable=true, OutDir="", Network=true) — the only
// differences are these flags, which is why Wrap can fold both recipes.
type SandboxSpec struct {
	Command   string
	Args      []string
	Cwd       string
	OutDir    string
	BundleDir string
	StoreDir  string
	// BundleWritable binds BundleDir read-write (install: node_modules lands
	// there) instead of read-only (runtime: the script cannot write back).
	BundleWritable bool
	// Network shares the net namespace (install: deps download) instead of
	// unsharing everything (runtime: no net).
	Network bool
}

// SandboxConfig is the bundle-fixed half of a runtime SandboxSpec: the bundle
// root, the resolved bwrap binary, and the substrate store dir to mask. Kept
// as the seam LocalScriptRunner/FilterLoop and bundle.go pass around.
type SandboxConfig struct {
	BundleDir string
	Bwrap     string
	StoreDir  string
}

// BwrapSandbox is the default tier: every bundle process runs jailed in
// bubblewrap, fail-closed when bwrap is unavailable.
type BwrapSandbox struct {
	Bwrap string
}

func (s BwrapSandbox) Name() string { return "bwrap" }

// Preflight proves bwrap actually jails before any bundle entry is wired.
// Fail-closed: an empty path or a failed smoke run is an error, so the bundle
// refuses to start rather than running unjailed.
func (s BwrapSandbox) Preflight() error {
	if s.Bwrap == "" {
		return os.ErrNotExist
	}
	return exec.Command(s.Bwrap, "--ro-bind", "/", "/", "--unshare-all", "true").Run()
}

// Wrap folds the runtime jail and the install jail into ONE recipe so there is
// a single, audited argv builder — they differ only by spec.BundleWritable and
// spec.Network (and OutDir presence). Masking comes BEFORE re-binding, because
// bwrap applies ops in order and a later bind under a tmpfs would be clobbered:
//   - --tmpfs $HOME hides the operator's home secrets (~/.ssh, ~/.aws, tokens)
//     that --ro-bind / / would otherwise expose — and gives install a fresh
//     writable HOME for bun's cache.
//   - --tmpfs StoreDir hides the substrate store (operator.nk, role creds).
//   - --tmpfs /tmp gives a private scratch.
//
// Then the toolchain and the bundle are re-exposed: in dev/devbox layouts the
// $PATH binaries (bun, coreutils, sh) AND the bundle dir live UNDER $HOME, so
// after masking $HOME we re-bind the $PATH dirs that fall under it (secrets
// like ~/.ssh are not on $PATH, so they stay masked) plus the bundle dir and
// the run's output dir. Network: install shares net (only pid/uts unshared);
// runtime unshares all (no net). --chdir keeps the script's working dir, so
// the outer cmd must not set cmd.Dir.
func (s BwrapSandbox) Wrap(spec SandboxSpec) (string, []string) {
	home := os.Getenv("HOME")
	argv := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
	}
	if home != "" {
		argv = append(argv, "--tmpfs", home)
	}
	if spec.StoreDir != "" {
		argv = append(argv, "--tmpfs", spec.StoreDir)
	}
	argv = append(argv, "--tmpfs", "/tmp")
	if home != "" {
		for _, dir := range pathDirsUnder(home) {
			argv = append(argv, "--ro-bind", dir, dir)
		}
	}
	if spec.BundleWritable {
		argv = append(argv, "--bind", spec.BundleDir, spec.BundleDir)
	} else {
		argv = append(argv, "--ro-bind", spec.BundleDir, spec.BundleDir)
	}
	if spec.OutDir != "" {
		argv = append(argv, "--bind", spec.OutDir, spec.OutDir)
	}
	argv = append(argv, "--chdir", spec.Cwd)
	if spec.Network {
		argv = append(argv, "--unshare-pid", "--unshare-uts")
	} else {
		argv = append(argv, "--unshare-all")
	}
	argv = append(argv, "--die-with-parent", "--", spec.Command)
	argv = append(argv, spec.Args...)
	return s.Bwrap, argv
}

// TrustedSandbox is the opt-in tier for hosts without user namespaces: it runs
// the command BARE, with no jail. It is never auto-selected — only chosen by
// explicit Config.BundleSandbox="none" — and the assembly logs a loud warning
// when it is, because an unsandboxed bundle process has full host access.
type TrustedSandbox struct{}

func (TrustedSandbox) Name() string                             { return "trusted" }
func (TrustedSandbox) Preflight() error                         { return nil }
func (TrustedSandbox) Wrap(spec SandboxSpec) (string, []string) { return spec.Command, spec.Args }

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

var _ Sandbox = BwrapSandbox{}
var _ Sandbox = TrustedSandbox{}
