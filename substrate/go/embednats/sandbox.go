package embednats

// SandboxConfig jails a bundle process inside bubblewrap. BundleDir is the
// bundle root (kept read-only inside the jail); Bwrap is the resolved bwrap
// binary path the host preflighted.
type SandboxConfig struct {
	BundleDir string
	Bwrap     string
}

// sandboxCommand wraps command+args into a bwrap argv. The whole filesystem is
// bound read-only (--ro-bind / /) so the bundle dir cannot be written; the
// per-run outDir is the only writable host path (path-artifacts land there),
// alongside a private /tmp. --unshare-all denies the network — deps are
// installed at load, before the jail. --chdir keeps the script's working dir,
// so the outer cmd must not set cmd.Dir.
//
// --tmpfs /tmp overlays an empty tmpfs, which would mask any path living under
// /tmp — and a bundle dir (and its outDir) routinely do (t.TempDir, MkdirTemp).
// So the bundle dir is re-bound read-only and the outDir writable AFTER the
// tmpfs, restoring exactly the script and its one writable spot.
func sandboxCommand(cfg SandboxConfig, command string, args []string, cwd, outDir string) (string, []string) {
	argv := []string{
		"--ro-bind", "/", "/",
		"--dev", "/dev",
		"--proc", "/proc",
		"--tmpfs", "/tmp",
		"--ro-bind", cfg.BundleDir, cfg.BundleDir,
		"--bind", outDir, outDir,
		"--chdir", cwd,
		"--unshare-all",
		"--die-with-parent",
		"--",
		command,
	}
	argv = append(argv, args...)
	return cfg.Bwrap, argv
}
