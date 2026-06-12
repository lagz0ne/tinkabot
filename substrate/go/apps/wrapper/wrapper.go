// Package wrapper drives a real agent CLI (claude) over structured
// stream-json stdio as the trusted-tier session principal: it connects with a
// MintTrustedWrapper credential, publishes canonical session frames to the
// session ingest subject, and forwards mediated steer intents to the agent's
// stdin.
package wrapper

import (
	"bufio"
	"context"
	"io"
	"os/exec"

	"github.com/nats-io/nats.go"
)

// WrapperConfig configures one wrapper run. CredsFile is the path to the
// MintTrustedWrapper credential file; Cmd is the fully configured agent
// command, owned by the caller.
type WrapperConfig struct {
	NATSUrl   string
	CredsFile string
	SessionID string
	Cmd       *exec.Cmd
}

// WrapperHandle reports wrapper termination.
type WrapperHandle struct {
	done chan struct{}
	err  error
}

// Wait blocks until the agent subprocess exits or ctx is done.
func (h *WrapperHandle) Wait(ctx context.Context) error {
	select {
	case <-h.done:
		return h.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// StartWrapper connects with the trusted-wrapper credential (publish allow on
// tb.session.<id>.ingest, subscribe allow on tb.session.<id>.steer), starts
// the agent subprocess, translates steer intents onto its stdin, and
// republishes its stream-json output as canonical session frames on the
// ingest subject. Malformed output lines and non-steer payloads are skipped.
// Cancelling ctx kills the subprocess.
func StartWrapper(ctx context.Context, cfg WrapperConfig) (*WrapperHandle, error) {
	nc, err := nats.Connect(cfg.NATSUrl, nats.UserCredentials(cfg.CredsFile))
	if err != nil {
		return nil, err
	}

	ingest := "tb.session." + cfg.SessionID + ".ingest"
	steer := "tb.session." + cfg.SessionID + ".steer"

	stdout, err := cfg.Cmd.StdoutPipe()
	if err != nil {
		nc.Close()
		return nil, err
	}
	stdin, err := cfg.Cmd.StdinPipe()
	if err != nil {
		nc.Close()
		return nil, err
	}
	if err := cfg.Cmd.Start(); err != nil {
		nc.Close()
		return nil, err
	}

	if _, err := nc.Subscribe(steer, func(msg *nats.Msg) {
		line, ok, err := SteerToStdin(msg.Data)
		if err != nil || !ok {
			return
		}
		_, _ = stdin.Write(append(line, '\n'))
	}); err != nil {
		_ = cfg.Cmd.Process.Kill()
		nc.Close()
		return nil, err
	}

	go func() {
		<-ctx.Done()
		_ = cfg.Cmd.Process.Kill()
	}()

	h := &WrapperHandle{done: make(chan struct{})}
	go func() {
		defer func() {
			if err := nc.Drain(); err != nil {
				nc.Close()
			}
			close(h.done)
		}()
		perr := pump(cfg.SessionID, stdout, func(frame []byte) error {
			return nc.Publish(ingest, frame)
		})
		if perr != nil {
			_ = cfg.Cmd.Process.Kill()
		}
		_ = stdin.Close()
		werr := cfg.Cmd.Wait()
		if perr != nil {
			h.err = perr
		} else {
			h.err = werr
		}
	}()
	return h, nil
}

// pump republishes each agent stdout line as a canonical session frame,
// skipping lines that fail to parse or render. A scanner error (an agent line
// beyond the buffer cap) ends the session rather than freezing observation
// silently.
func pump(sid string, r io.Reader, pub func([]byte) error) error {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 1<<20), 8<<20)
	for sc.Scan() {
		f, err := ParseStreamJsonFrame(sc.Bytes())
		if err != nil {
			continue
		}
		frame, err := SessionFrame(sid, f)
		if err != nil {
			continue
		}
		_ = pub(frame)
	}
	return sc.Err()
}
