package embednats

import "testing"

// start is the one harness factory seam (gate:parallel): every embednats test
// obtains its embedded server here, never by constructing Start directly.
// Each call builds a fresh server on a random port with its own store dir
// (valid(t) uses Port -1 and t.TempDir()), so parallel tests stay isolated.
// When a runtime starts, its shutdown is owned by the test via t.Cleanup.
func start(t *testing.T, cfg Config) (*Runtime, error) {
	t.Helper()
	rt, err := Start(cfg)
	if rt != nil {
		t.Cleanup(func() { stop(t, rt) })
	}
	return rt, err
}
