package embednats

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// FrameMediatorConfig configures the validating republisher for one session.
type FrameMediatorConfig struct {
	SessionID     string
	QuotaMaxBytes int
}

// FrameMediator is the sole consumer of a session ingest subject and the sole
// writer of the session output subject and its durable JetStream stream.
// Stop tears down the mediator. OutputJS returns a JetStream handle bound to
// the mediator's NATS connection for reading the output stream.
type FrameMediator struct {
	nc  *nats.Conn
	js  jetstream.JetStream
	sub *nats.Subscription
}

// Stop unsubscribes the ingest consumer and closes the mediator connection.
func (m *FrameMediator) Stop() {
	if m.sub != nil {
		_ = m.sub.Unsubscribe()
	}
	if m.nc != nil {
		m.nc.Close()
	}
}

// OutputJS returns a JetStream context on the mediator connection, used by
// callers to open an ordered consumer on the session output stream.
func (m *FrameMediator) OutputJS() jetstream.JetStream {
	return m.js
}

// StartFrameMediator starts the validating republisher for one session:
//   - subscribes to tb.session.<sessionID>.ingest as the sole consumer
//   - validates each frame against the session frame contract
//   - enforces a per-session byte quota (stops republishing at QuotaMaxBytes)
//   - publishes valid frames to tb.session.<sessionID>.out
//   - writes to the durable JetStream stream tb-session-out-<sessionID>
//
// The mediator uses a dedicated internal credential; only that credential may
// publish to the output subject, enforcing the sole-writer invariant.
func StartFrameMediator(ctx context.Context, rt *Runtime, cfg FrameMediatorConfig) (*FrameMediator, error) {
	outSubj := "tb.session." + cfg.SessionID + ".out"
	ingestSubj := "tb.session." + cfg.SessionID + ".ingest"
	stream := "tb-session-out-" + cfg.SessionID

	mediatorPerms := core.Permissions{
		Publish: core.PermList{Allow: []string{
			"$JS.API.INFO",
			"$JS.API.STREAM.CREATE." + stream,
			"$JS.API.STREAM.UPDATE." + stream,
			"$JS.API.STREAM.INFO." + stream,
			"$JS.API.STREAM.DELETE." + stream,
			"$JS.API.CONSUMER.CREATE." + stream + ".>",
			"$JS.API.CONSUMER.MSG.NEXT." + stream + ".>",
			"$JS.API.CONSUMER.DELETE." + stream + ".>",
			"$JS.API.DIRECT.GET." + stream + ".>",
			"$JS.API.DIRECT.GET." + stream,
			"$JS.ACK." + stream + ".>",
			outSubj,
		}},
		Subscribe: core.PermList{Allow: []string{ingestSubj, "_INBOX.>"}},
	}

	var nc *nats.Conn
	var err error
	if rt.op != nil {
		nc, err = mintedConn(ctx, rt, "_tb_mediator_"+cfg.SessionID, mediatorPerms)
	} else {
		nc, err = internalConn(ctx, rt, "_tb_mediator_"+cfg.SessionID, mediatorPerms)
	}
	if err != nil {
		return nil, err
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, err
	}

	_, err = js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:     stream,
		Subjects: []string{outSubj},
		Storage:  jetstream.FileStorage,
	})
	if err != nil {
		nc.Close()
		return nil, err
	}

	quota := int64(cfg.QuotaMaxBytes)
	var used atomic.Int64

	sub, err := nc.Subscribe(ingestSubj, func(msg *nats.Msg) {
		data := msg.Data

		if !validFrame(data) {
			return
		}

		// Quota stops republishing once cumulative bytes exceed it; the frame
		// straddling the boundary is forwarded.
		if quota > 0 {
			total := used.Add(int64(len(data)))
			if total-int64(len(data)) >= quota {
				// Already over quota before this frame — drop it.
				return
			}
		}

		_ = nc.Publish(outSubj, data)
	})
	if err != nil {
		nc.Close()
		return nil, err
	}

	return &FrameMediator{nc: nc, js: js, sub: sub}, nil
}

// validFrame enforces the session frame contract.
// FakeStatusImpersonation (origin=wrapper, frame=status) is explicitly rejected.
func validFrame(data []byte) bool {
	var f map[string]json.RawMessage
	if err := json.Unmarshal(data, &f); err != nil {
		return false
	}

	var frameVal string
	if raw, ok := f["frame"]; !ok || json.Unmarshal(raw, &frameVal) != nil || frameVal == "" {
		return false
	}

	var origin string
	if raw, ok := f["origin"]; ok {
		_ = json.Unmarshal(raw, &origin)
	}

	switch frameVal {
	case "status":
		// Status frames must come from the runner; wrapper-emitted status frames
		// are the FakeStatusImpersonation contract violation.
		return origin == "runner"
	case "token", "chunk":
		// Token and chunk frames must come from the wrapper.
		return origin == "wrapper"
	default:
		// Unknown frame types are rejected to keep the contract closed.
		return false
	}
}
