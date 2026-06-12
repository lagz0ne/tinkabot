package tinkabot

// Demo observation session: a continuously ticking stand-in subprocess run
// through the real session subsystem (ingest -> mediator -> durable stream)
// so the shell observe panel has a live session to watch and replay.

import (
	"context"
	"os"
	"os/exec"

	"github.com/lagz0ne/tinkabot/substrate/go/embednats"
)

const demoTicker = `i=0; while :; do
  i=$((i+1))
  printf '{"kind":"session.frame","frame":"token","origin":"wrapper","sessionId":"%s","text":"tick %s at %s\\n"}\n' "$TB_SID" "$i" "$(date +%H:%M:%S)"
  sleep 1
done`

func (a *App) startDemoSession(sid string) error {
	mediator, err := embednats.StartFrameMediator(context.Background(), a.rt, embednats.FrameMediatorConfig{
		SessionID:     sid,
		QuotaMaxBytes: 8 << 20,
	})
	if err != nil {
		return err
	}
	a.closers = append(a.closers, mediator.Stop)

	cmd := exec.Command("/bin/sh", "-c", demoTicker)
	cmd.Env = append(os.Environ(), "TB_SID="+sid)
	if _, err := embednats.StartSessionRuntime(context.Background(), a.rt, embednats.SessionRuntimeConfig{
		SessionID: sid,
		Cmd:       cmd,
	}); err != nil {
		return err
	}
	a.closers = append(a.closers, func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return nil
}
