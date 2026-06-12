package embednats

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"syscall"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
)

type FilterLoop struct {
	rec    core.ScriptRecord
	rtm    *core.ScriptRuntime
	mat    *core.Materializer
	status core.StatusSink
}

type filterProc struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	closed chan struct{}
}

func NewFilterLoop(rec core.ScriptRecord, rtm *core.ScriptRuntime, mat *core.Materializer, status core.StatusSink) *FilterLoop {
	return &FilterLoop{rec: rec, rtm: rtm, mat: mat, status: status}
}

func (l *FilterLoop) Watch(in <-chan RouterResult) (<-chan ScriptRunResult, func()) {
	out := make(chan ScriptRunResult, 16)
	stop := make(chan struct{})
	var once sync.Once
	var stopped atomic.Bool
	var wg sync.WaitGroup
	var proc *filterProc
	var accMu sync.Mutex
	var current core.AcceptedActivation

	setAcc := func(acc core.AcceptedActivation) {
		accMu.Lock()
		current = acc
		accMu.Unlock()
	}
	acc := func() core.AcceptedActivation {
		accMu.Lock()
		defer accMu.Unlock()
		return current
	}
	emit := func(run ScriptRunResult) bool {
		run = l.save(run)
		select {
		case out <- run:
			return true
		case <-stop:
			return false
		}
	}
	stopProc := func(p *filterProc) {
		if p == nil {
			return
		}
		_ = p.stdin.Close()
		killGroup(p.cmd)
	}
	start := func() (*filterProc, error) {
		if proc != nil {
			select {
			case <-proc.closed:
				proc = nil
			default:
			}
		}
		if proc != nil {
			return proc, nil
		}
		p, err := l.start(acc, emit, &wg, &stopped)
		if err != nil {
			return nil, err
		}
		proc = p
		return proc, nil
	}

	go func() {
		defer func() {
			stopped.Store(true)
			stopProc(proc)
			wg.Wait()
			close(out)
		}()
		for {
			select {
			case <-stop:
				return
			case res, ok := <-in:
				if !ok {
					return
				}
				run := ScriptRunResult{Activation: res.Activation, Record: res.Record, Err: res.Err}
				if res.Err != nil {
					emit(run)
					continue
				}
				if res.Record.Status != core.Accepted {
					continue
				}
				setAcc(core.AcceptedActivation{Activation: res.Activation, Record: res.Record})
				p, err := start()
				if err != nil {
					run.Err = err
					emit(run)
					continue
				}
				if len(res.Payload) == 0 {
					continue
				}
				line := append(append([]byte(nil), res.Payload...), '\n')
				n, err := p.stdin.Write(line)
				if err == nil && n != len(line) {
					err = io.ErrShortWrite
				}
				if err != nil {
					run.Err = core.ScriptRecordErr(core.ScriptProcessFailed, "Run", "filter stdin write failed", map[string]string{"scriptKey": l.rec.Key}, err)
					stopProc(p)
					proc = nil
					emit(run)
				}
			}
		}
	}()
	return out, func() {
		once.Do(func() {
			stopped.Store(true)
			close(stop)
		})
	}
}

func (l *FilterLoop) start(acc func() core.AcceptedActivation, emit func(ScriptRunResult) bool, wg *sync.WaitGroup, stopped *atomic.Bool) (*filterProc, error) {
	if _, err := core.CheckProcess(l.rec.Process); err != nil {
		return nil, err
	}
	cmd := exec.Command(l.rec.Process.Command, l.rec.Process.Args...)
	cmd.Dir = l.rec.Process.Cwd
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, core.ScriptRecordErr(core.ScriptProcessFailed, "Run", "filter stdin pipe failed", map[string]string{"scriptKey": l.rec.Key}, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, core.ScriptRecordErr(core.ScriptProcessFailed, "Run", "filter stdout pipe failed", map[string]string{"scriptKey": l.rec.Key}, err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, core.ScriptRecordErr(core.ScriptProcessFailed, "Run", "filter process could not start", map[string]string{"scriptKey": l.rec.Key}, err)
	}
	p := &filterProc{cmd: cmd, stdin: stdin, closed: make(chan struct{})}
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(p.closed)
		readErr := l.read(stdout, acc, emit, stopped)
		if readErr != nil && !stopped.Load() {
			killGroup(cmd)
		}
		waitErr := cmd.Wait()
		if stopped.Load() || readErr != nil {
			return
		}
		emit(ScriptRunResult{Activation: acc().Activation, Record: acc().Record, Err: filterExitErr(l.rec, waitErr)})
	}()
	return p, nil
}

func (l *FilterLoop) read(stdout io.Reader, acc func() core.AcceptedActivation, emit func(ScriptRunResult) bool, stopped *atomic.Bool) error {
	br := bufio.NewReader(stdout)
	for {
		body, err := streamFrame(br)
		if err != nil {
			if errors.Is(err, io.EOF) || stopped.Load() {
				return nil
			}
			emit(ScriptRunResult{Activation: acc().Activation, Record: acc().Record, Err: err})
			return err
		}
		eff, err := decodeEffect(body)
		if err != nil {
			emit(ScriptRunResult{Activation: acc().Activation, Record: acc().Record, Err: err})
			continue
		}
		cur := acc()
		run := core.ScriptRun{ActivationID: cur.Activation.ActivationID, Status: "applied", Effects: []core.ScriptEffect{eff}}
		if err := l.rtm.Allow(eff); err != nil {
			emit(ScriptRunResult{Activation: cur.Activation, Record: cur.Record, Run: run, Err: err})
			continue
		}
		// Filter output is a live KV-fed stream; ScriptLoop's durable ClaimRun does not apply here.
		err = l.mat.Apply(core.MaterialContext{Accepted: cur, Record: l.rec}, run.Effects)
		emit(ScriptRunResult{Activation: cur.Activation, Record: cur.Record, Run: run, Err: err})
	}
}

func (l *FilterLoop) save(run ScriptRunResult) ScriptRunResult {
	if l.status == nil || (run.Record.Status != core.Accepted && run.Err == nil) {
		return run
	}
	if err := l.status.SaveEvent(eventFor(run)); err != nil && run.Err == nil {
		run.Err = err
	}
	return run
}

func streamFrame(r *bufio.Reader) ([]byte, error) {
	var header []byte
	for {
		line, err := r.ReadBytes('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && len(header) == 0 && len(line) == 0 {
				return nil, io.EOF
			}
			return nil, core.ProtocolErr("ReadFrame", "script frame header is incomplete", err)
		}
		if len(header) == 0 && len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if bytes.Equal(line, []byte("\n")) || bytes.Equal(line, []byte("\r\n")) {
			break
		}
		header = append(header, line...)
		if len(header) > maxFrameBody {
			return nil, core.ProtocolErr("ReadFrame", "script frame header exceeded limit", nil)
		}
	}
	n, err := contentLength(header)
	if err != nil {
		return nil, err
	}
	if n > maxFrameBody {
		return nil, core.ProtocolErr("ReadFrame", "script frame body exceeded limit", nil)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return nil, core.ProtocolErr("ReadFrame", "script frame body is incomplete", err)
	}
	return body, nil
}

func decodeEffect(body []byte) (core.ScriptEffect, error) {
	if err := checkFrameShape(body); err != nil {
		return core.ScriptEffect{}, err
	}
	var f frame
	if err := decodeStrict(body, &f); err != nil {
		return core.ScriptEffect{}, core.ProtocolErr("ReadFrame", "script frame is malformed", err)
	}
	if f.Kind != "script.effect" {
		return core.ScriptEffect{}, core.ProtocolErr("ReadFrame", "script frame kind is invalid", nil)
	}
	return core.ScriptEffect{
		Type:             f.Type,
		ProjectionID:     f.ProjectionID,
		SnapshotRevision: f.SnapshotRevision,
		ArtifactRevision: f.ArtifactRevision,
		Sequence:         f.Sequence,
		Value:            f.Value,
		ArtifactName:     f.ArtifactName,
		MediaType:        f.MediaType,
		Body:             []byte(f.Body),
		Subject:          f.Subject,
	}, nil
}

func filterExitErr(rec core.ScriptRecord, cause error) error {
	return core.ScriptRecordErr(core.ScriptProcessFailed, "Run", fmt.Sprintf("filter process exited: %s", rec.Key), map[string]string{"scriptKey": rec.Key}, cause)
}
