---
target: rule-bundle-sandbox-default-fail-closed
scope: whole
type: rule
parent: c3-0
title: Bundle Sandbox Default Fail Closed
---
## Goal

Enforce that bundle processes do not run unjailed by accident when sandboxing is unavailable.

## Rule

Bundle execution must fail closed unless the operator explicitly selects the trusted unsandboxed tier.

## Golden Example

Literal code from `substrate/go/embednats/sandbox.go`:

```go
// Preflight proves bwrap actually jails before any bundle entry is wired.
// Fail-closed: an empty path or a failed smoke run is an error, so the bundle
// refuses to start rather than running unjailed.
func (s BwrapSandbox) Preflight() error {
	if s.Bwrap == "" { // REQUIRED: missing bwrap is an error.
		return os.ErrNotExist
	}
	return exec.Command(s.Bwrap, "--ro-bind", "/", "/", "--unshare-all", "true").Run() // REQUIRED: smoke the jail.
}
```

## Not This

| Anti-Pattern | Correct | Why Wrong Here |
| --- | --- | --- |
| Silently fall back to host execution when bwrap is missing. | Return an error before any bundle entry is wired. | It would turn a sandbox failure into full host access for generated code. |
| Auto-select `TrustedSandbox`. | Require explicit `--bundle-sandbox none`. | The trust decision belongs to the operator, not the runtime. |

## Scope

Applies to bundle runtime script execution and future transform/install steps that execute bundle-provided commands. It does not apply to tests that explicitly inject fake sandbox behavior.

## Override

Only explicit operator configuration may choose trusted unsandboxed execution, and docs/tests must call that out as an unsafe local trust posture.
