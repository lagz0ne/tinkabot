package embednats

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

const (
	HeaderRequestID = "Tinkabot-Request-Id"
	HeaderMessageID = "Tinkabot-Message-Id"
)

type SourceRouter struct {
	auth   core.Auth
	ledger *core.DurableLedger
}

type RouterResult struct {
	Activation core.Activation
	Record     core.LedgerRecord
	Err        error
}

type Route struct {
	once sync.Once
	stop func() error
}

func NewSourceRouter(auth core.Auth, ledger *core.DurableLedger) (*SourceRouter, error) {
	if strings.TrimSpace(auth.User) == "" {
		return nil, routeErr(RouterConfigInvalid, "Configure", "source auth user is required", nil, nil)
	}
	if ledger == nil {
		return nil, routeErr(RouterConfigInvalid, "Configure", "durable ledger is required", nil, nil)
	}
	return &SourceRouter{auth: auth, ledger: ledger}, nil
}

func (r *SourceRouter) RequestReply(nc *nats.Conn, act core.Activation) (*Route, <-chan RouterResult, error) {
	if nc == nil {
		return nil, nil, routeErr(RequestReplyListenFailed, "RequestReply", "NATS connection is required", nil, nil)
	}
	out := make(chan RouterResult, 16)
	sub, err := nc.Subscribe(act.Source.Subject, func(msg *nats.Msg) {
		rec, err := r.AcceptRequest(act, msg)
		send(out, RouterResult{Activation: normRequest(act, msg), Record: rec, Err: err})
		_ = msg.Respond([]byte(resultKind(rec, err)))
	})
	if err != nil {
		return nil, nil, routeErr(RequestReplyListenFailed, "RequestReply", "request/reply subscribe failed", nil, err)
	}
	if err := nc.FlushTimeout(time.Second); err != nil {
		_ = sub.Unsubscribe()
		return nil, nil, routeErr(RequestReplyListenFailed, "RequestReply", "request/reply subscribe was not acknowledged", nil, err)
	}
	return &Route{stop: sub.Unsubscribe}, out, nil
}

// Subject routes messages through an ephemeral JetStream stream so the server
// assigns a monotone sequence to each arrival. The stream is deleted when the
// route stops.
func (r *SourceRouter) Subject(nc *nats.Conn, act core.Activation) (*Route, <-chan RouterResult, error) {
	if nc == nil {
		return nil, nil, routeErr(SubjectSubscribeFailed, "Subject", "NATS connection is required", nil, nil)
	}
	js, err := nc.JetStream()
	if err != nil {
		return nil, nil, routeErr(SubjectSubscribeFailed, "Subject", "JetStream context unavailable", nil, err)
	}
	h := sha256.Sum256([]byte(act.Source.Pattern + ":" + act.SourcePrincipal.SourceID))
	streamName := "TB_SUBJ_" + hex.EncodeToString(h[:8])
	if _, err := js.AddStream(&nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{act.Source.Pattern},
		Storage:  nats.MemoryStorage,
	}); err != nil {
		return nil, nil, routeErr(SubjectSubscribeFailed, "Subject", "subject stream could not be created", nil, err)
	}
	out := make(chan RouterResult, 16)
	sub, err := js.Subscribe(act.Source.Pattern,
		func(msg *nats.Msg) {
			next := normSubject(act, msg)
			rec, acceptErr := r.accept(next)
			send(out, RouterResult{Activation: next, Record: rec, Err: acceptErr})
			_ = msg.Ack()
		},
		nats.BindStream(streamName),
		nats.OrderedConsumer(),
		nats.DeliverNew(),
	)
	if err != nil {
		_ = js.DeleteStream(streamName)
		return nil, nil, routeErr(SubjectSubscribeFailed, "Subject", "subject subscribe failed", nil, err)
	}
	return &Route{stop: func() error {
		err := sub.Unsubscribe()
		_ = js.DeleteStream(streamName)
		return err
	}}, out, nil
}

func (r *SourceRouter) KV(kv nats.KeyValue, act core.Activation) (*Route, <-chan RouterResult, error) {
	if kv == nil {
		return nil, nil, routeErr(KVWatchFailed, "KV", "KV bucket is required", nil, nil)
	}
	out := make(chan RouterResult, 16)
	w, err := kv.Watch(act.Source.Key, nats.UpdatesOnly())
	if err != nil {
		return nil, nil, routeErr(KVWatchFailed, "KV", "KV watch failed", nil, err)
	}
	stop := make(chan struct{})
	go func() {
		defer close(out)
		errs := w.Error()
		updates := w.Updates()
		for {
			select {
			case <-stop:
				return
			case err, ok := <-errs:
				if !ok {
					errs = nil
					if updates == nil {
						return
					}
					continue
				}
				if err != nil {
					send(out, RouterResult{Activation: act, Err: routeErr(KVWatchFailed, "KV", "KV watcher failed", nil, err)})
				}
			case entry, ok := <-updates:
				if !ok {
					updates = nil
					if errs == nil {
						return
					}
					continue
				}
				if entry == nil {
					continue
				}
				next := normKV(act, entry)
				rec, err := r.AcceptKV(act, entry)
				send(out, RouterResult{Activation: next, Record: rec, Err: err})
			}
		}
	}()
	return &Route{stop: func() error {
		close(stop)
		return w.Stop()
	}}, out, nil
}

func (r *SourceRouter) Object(js nats.JetStreamContext, act core.Activation) (*Route, <-chan RouterResult, error) {
	if js == nil {
		return nil, nil, routeErr(ObjectWatchFailed, "Object", "JetStream context is required", nil, nil)
	}
	out := make(chan RouterResult, 16)
	sub, err := js.Subscribe(
		fmt.Sprintf("$O.%s.M.>", act.Source.Bucket),
		func(msg *nats.Msg) {
			next, rec, err := r.acceptObjectMsg(act, msg)
			send(out, RouterResult{Activation: next, Record: rec, Err: err})
		},
		nats.OrderedConsumer(),
		nats.BindStream("OBJ_"+act.Source.Bucket),
		nats.DeliverNew(),
	)
	if err != nil {
		return nil, nil, routeErr(ObjectWatchFailed, "Object", "object metadata watch failed", nil, err)
	}
	return &Route{stop: sub.Unsubscribe}, out, nil
}

func (r *SourceRouter) Stream(js nats.JetStreamContext, act core.Activation) (*Route, <-chan RouterResult, error) {
	if js == nil {
		return nil, nil, routeErr(StreamConsumeFailed, "Stream", "JetStream context is required", nil, nil)
	}
	out := make(chan RouterResult, 16)
	sub, err := js.PullSubscribe(act.Source.Subject, act.Source.Consumer, nats.BindStream(act.Source.Stream), nats.ManualAck())
	if err != nil {
		return nil, nil, routeErr(StreamConsumeFailed, "Stream", "stream pull subscribe failed", nil, err)
	}
	stop := make(chan struct{})
	go func() {
		defer close(out)
		for {
			select {
			case <-stop:
				return
			default:
			}
			msgs, err := sub.Fetch(1, nats.MaxWait(100*time.Millisecond))
			if errors.Is(err, nats.ErrTimeout) {
				continue
			}
			if err != nil {
				send(out, RouterResult{Activation: act, Err: routeErr(StreamConsumeFailed, "Stream", "stream fetch failed", nil, err)})
				continue
			}
			for _, msg := range msgs {
				next := normStream(act, msg)
				rec, err := r.AcceptStream(act, msg)
				send(out, RouterResult{Activation: next, Record: rec, Err: err})
				ack(msg, err)
			}
		}
	}()
	return &Route{stop: func() error {
		close(stop)
		return sub.Unsubscribe()
	}}, out, nil
}

func (r *SourceRouter) AcceptRequest(act core.Activation, msg *nats.Msg) (core.LedgerRecord, error) {
	return r.accept(normRequest(act, msg))
}

func (r *SourceRouter) AcceptSubject(act core.Activation, msg *nats.Msg) (core.LedgerRecord, error) {
	return r.accept(normSubject(act, msg))
}

func (r *SourceRouter) AcceptKV(act core.Activation, entry nats.KeyValueEntry) (core.LedgerRecord, error) {
	if entry == nil {
		return core.LedgerRecord{}, routeErr(SourceMalformed, "AcceptKV", "KV entry is required", nil, nil)
	}
	return r.accept(normKV(act, entry))
}

func (r *SourceRouter) AcceptStream(act core.Activation, msg *nats.Msg) (core.LedgerRecord, error) {
	return r.accept(normStream(act, msg))
}

func (r *SourceRouter) acceptObjectMsg(act core.Activation, msg *nats.Msg) (core.Activation, core.LedgerRecord, error) {
	var info nats.ObjectInfo
	if msg == nil {
		return act, core.LedgerRecord{}, routeErr(SourceMalformed, "AcceptObject", "object metadata message is required", nil, nil)
	}
	if err := json.Unmarshal(msg.Data, &info); err != nil {
		return act, core.LedgerRecord{}, routeErr(SourceMalformed, "AcceptObject", "object metadata is malformed", nil, err)
	}
	meta, err := msg.Metadata()
	if err != nil {
		return act, core.LedgerRecord{}, routeErr(SourceMalformed, "AcceptObject", "object metadata sequence is missing", nil, err)
	}
	next := normObject(act, info, meta)
	rec, err := r.accept(next)
	return next, rec, err
}

func (r *SourceRouter) accept(act core.Activation) (core.LedgerRecord, error) {
	if r == nil {
		return core.LedgerRecord{}, routeErr(RouterCritical, "Accept", "router is nil", nil, nil)
	}
	if act.Source.Kind == "" {
		return core.LedgerRecord{}, routeErr(SourceMalformed, "Accept", "source kind is required", nil, nil)
	}
	if malformed(act) {
		return core.LedgerRecord{}, routeErr(SourceMalformed, "Accept", "source frame is missing identity or cursor", sourceDetails(act), nil)
	}
	if _, err := core.AuthorizeSource(r.auth, act); err != nil {
		return core.LedgerRecord{}, err
	}
	return r.ledger.Accept(act, core.Lease{
		ID:          act.SourceLease.LeaseID,
		Status:      act.SourceLease.LeaseStatus,
		PrincipalID: act.SourcePrincipal.PrincipalID,
	})
}

func (r *Route) Stop() (err error) {
	if r == nil || r.stop == nil {
		return nil
	}
	r.once.Do(func() { err = r.stop() })
	return err
}

func normRequest(act core.Activation, msg *nats.Msg) core.Activation {
	next := act
	if msg != nil {
		next.Source.Subject = msg.Subject
		next.Source.RequestID = msg.Header.Get(HeaderRequestID)
	}
	stamp(&next, next.Source.RequestID)
	return next
}

func normSubject(act core.Activation, msg *nats.Msg) core.Activation {
	next := act
	if msg != nil {
		next.Source.ObservedSubject = msg.Subject
		next.Source.MessageID = msg.Header.Get(HeaderMessageID)
		if meta, err := msg.Metadata(); err == nil {
			next.Source.StreamSequence = int64(meta.Sequence.Stream)
			next.Source.Stream = meta.Stream
			next.Source.Consumer = meta.Consumer
			next.Source.ConsumerSequence = int64(meta.Sequence.Consumer)
		}
	}
	if next.Source.StreamSequence > 0 {
		stamp(&next, fmt.Sprint(next.Source.StreamSequence))
	} else {
		stamp(&next, next.Source.MessageID)
	}
	return next
}

func normKV(act core.Activation, entry nats.KeyValueEntry) core.Activation {
	next := act
	next.Source.Bucket = entry.Bucket()
	next.Source.Key = entry.Key()
	next.Source.Operation = kvOp(entry.Operation())
	next.Source.Revision = int64(entry.Revision())
	next.Source.Resume = fmt.Sprintf("kv:%s:%s:%d", entry.Bucket(), entry.Key(), entry.Revision())
	stamp(&next, next.Source.Bucket, next.Source.Key, fmt.Sprint(next.Source.Revision))
	return next
}

func normObject(act core.Activation, info nats.ObjectInfo, meta *nats.MsgMetadata) core.Activation {
	next := act
	next.Source.Bucket = info.Bucket
	next.Source.Name = info.Name
	next.Source.Digest = info.Digest
	next.Source.Revision = int64(info.Size)
	next.Source.ObjectMetaSequence = int64(meta.Sequence.Stream)
	next.Source.WatchPosition = fmt.Sprintf("obj:%s:%s:%d", info.Bucket, info.Name, meta.Sequence.Stream)
	stamp(&next, next.Source.Bucket, next.Source.Name, fmt.Sprint(next.Source.ObjectMetaSequence))
	return next
}

func normStream(act core.Activation, msg *nats.Msg) core.Activation {
	next := act
	if msg != nil {
		next.Source.Subject = msg.Subject
		if meta, err := msg.Metadata(); err == nil {
			next.Source.Stream = meta.Stream
			next.Source.Consumer = meta.Consumer
			next.Source.StreamSequence = int64(meta.Sequence.Stream)
			next.Source.ConsumerSequence = int64(meta.Sequence.Consumer)
			next.Source.DeliveryAttempt = int64(meta.NumDelivered)
		}
	}
	stamp(&next, next.Source.Stream, next.Source.Consumer, fmt.Sprint(next.Source.StreamSequence), fmt.Sprint(next.Source.ConsumerSequence))
	return next
}

func stamp(act *core.Activation, parts ...string) {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			clean = append(clean, part)
		}
	}
	id := strings.Join(clean, ":")
	act.DedupeKey = strings.Join([]string{act.Source.Kind, act.ScriptKey, act.Source.ActivationName, id}, ":")
	act.ActivationID = strings.Join([]string{"act", act.ScriptKey, act.Source.ActivationName, id}, ":")
}

func malformed(act core.Activation) bool {
	src := act.Source
	switch src.Kind {
	case "request_reply":
		return src.Subject == "" || src.RequestID == ""
	case "subject":
		if src.StreamSequence > 0 {
			return src.Pattern == "" || src.ObservedSubject == ""
		}
		return src.Pattern == "" || src.ObservedSubject == "" || src.MessageID == ""
	case "kv":
		return src.Bucket == "" || src.Key == "" || src.Revision <= 0 || src.Resume == ""
	case "object":
		return src.Bucket == "" || src.Name == "" || src.ObjectMetaSequence <= 0 || src.WatchPosition == ""
	case "stream":
		return src.Subject == "" || src.Stream == "" || src.Consumer == "" || src.StreamSequence <= 0 || src.ConsumerSequence <= 0
	default:
		return true
	}
}

func kvOp(op nats.KeyValueOp) string {
	switch op {
	case nats.KeyValuePut:
		return "put"
	case nats.KeyValueDelete:
		return "delete"
	case nats.KeyValuePurge:
		return "purge"
	default:
		return "unknown"
	}
}

func resultKind(rec core.LedgerRecord, err error) string {
	if err == nil {
		return string(rec.Status)
	}
	var router *Error
	if errors.As(err, &router) {
		return string(router.Kind)
	}
	var owned *core.Error
	if errors.As(err, &owned) {
		return string(owned.Kind)
	}
	return string(RouterCritical)
}

// send blocks until the consumer reads, preserving per-subscription FIFO and
// applying backpressure instead of dropping. out may close during teardown.
func send(out chan<- RouterResult, res RouterResult) {
	defer func() { recover() }() //nolint:errcheck
	out <- res
}

func ack(msg *nats.Msg, err error) {
	if msg == nil {
		return
	}
	if err == nil {
		_ = msg.Ack()
		return
	}
	var owned *core.Error
	if errors.As(err, &owned) && (owned.Kind == core.WriteConflict || owned.Kind == core.ReplayCursorFailed) {
		_ = msg.Nak()
		return
	}
	_ = msg.Term()
}

func sourceDetails(act core.Activation) map[string]string {
	return map[string]string{
		"sourceId":     act.SourcePrincipal.SourceID,
		"sourceKind":   act.Source.Kind,
		"principalId":  act.SourcePrincipal.PrincipalID,
		"authorityRef": act.SourcePrincipal.AuthorityRef,
	}
}

func routeErr(kind Kind, op, msg string, details map[string]string, cause error) *Error {
	if details == nil {
		details = map[string]string{}
	}
	return &Error{
		Kind:      kind,
		Layer:     "LiveSourceRouter",
		Operation: op,
		Message:   msg,
		Details:   details,
		Cause:     cause,
	}
}
