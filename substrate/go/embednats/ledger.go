package embednats

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

type KVLedgerStore struct {
	nc *nats.Conn
	kv nats.KeyValue
}

func NewKVLedgerStore(ctx context.Context, rt *Runtime, bucket string) (*KVLedgerStore, error) {
	nc, err := rt.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return OpenKVLedgerStore(nc, bucket)
}

// OpenKVLedgerStore opens the ledger bucket over a caller-supplied connection
// (operator-mode assemblies connect with minted creds, where the static
// rt.Connect path does not exist). The store owns nc from here on.
func OpenKVLedgerStore(nc *nats.Conn, bucket string) (*KVLedgerStore, error) {
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, ledgerErr(core.JetStreamUnavailable, "OpenLedgerStore", "JetStream context is unavailable", err)
	}
	kv, err := js.KeyValue(bucket)
	if errors.Is(err, nats.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket, Storage: nats.FileStorage})
	}
	if err != nil {
		nc.Close()
		return nil, ledgerErr(core.BucketMissing, "OpenLedgerStore", "ledger KV bucket is unavailable", err)
	}
	return &KVLedgerStore{nc: nc, kv: kv}, nil
}

func (s *KVLedgerStore) Close() {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
}

func (s *KVLedgerStore) Dedupe(key string) (core.LedgerRecord, bool, error) {
	rec, ok, err := s.get("a." + keyEnc(key))
	if err != nil {
		return core.LedgerRecord{}, false, ledgerErr(core.ReplayCursorFailed, "Dedupe", "dedupe lookup failed", err)
	}
	return rec, ok, nil
}

func (s *KVLedgerStore) Source(id string) (core.LedgerRecord, bool, error) {
	recs, err := s.records("a.")
	if err != nil {
		return core.LedgerRecord{}, false, err
	}
	var out core.LedgerRecord
	for _, rec := range recs {
		if rec.SourceID == id {
			out = rec
		}
	}
	return out, out.ActivationID != "", nil
}

func (s *KVLedgerStore) SaveAccepted(rec core.LedgerRecord) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return ledgerErr(core.WriteConflict, "Accept", "ledger record could not be encoded", err)
	}
	if _, err := s.kv.Create("a."+keyEnc(rec.DedupeKey), body); err != nil {
		return ledgerErr(core.WriteConflict, "Accept", "ledger record could not be written", err)
	}
	return nil
}

func (s *KVLedgerStore) SaveSuppressed(rec core.LedgerRecord) error {
	body, err := json.Marshal(rec)
	if err != nil {
		return ledgerErr(core.WriteConflict, "Suppress", "suppressed record could not be encoded", err)
	}
	if _, err := s.kv.Put("x."+keyEnc(rec.ReplayCursor), body); err != nil {
		return ledgerErr(core.WriteConflict, "Suppress", "suppressed record could not be written", err)
	}
	return nil
}

func (s *KVLedgerStore) Replay(after string, limit int) ([]core.LedgerRecord, error) {
	if limit <= 0 {
		return nil, ledgerErr(core.ReplayCursorFailed, "Replay", "replay limit is invalid", nil)
	}
	recs, err := s.records("a.")
	if err != nil {
		return nil, err
	}
	start := after == ""
	out := []core.LedgerRecord{}
	for _, rec := range recs {
		if !start {
			start = rec.ReplayCursor == after
			continue
		}
		out = append(out, rec)
		if len(out) == limit {
			return out, nil
		}
	}
	if after != "" && !start {
		return nil, ledgerErr(core.ReplayCursorFailed, "Replay", "replay cursor is unknown", nil)
	}
	return out, nil
}

type kvRec struct {
	rev uint64
	rec core.LedgerRecord
}

func (s *KVLedgerStore) records(prefix string) ([]core.LedgerRecord, error) {
	keys, err := s.kv.Keys()
	if errors.Is(err, nats.ErrNoKeysFound) {
		return nil, nil
	}
	if err != nil {
		return nil, ledgerErr(core.ReplayCursorFailed, "Replay", "ledger keys could not be listed", err)
	}
	items := []kvRec{}
	for _, key := range keys {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		entry, err := s.kv.Get(key)
		if err != nil {
			return nil, ledgerErr(core.ReplayCursorFailed, "Replay", "ledger record could not be read", err)
		}
		var rec core.LedgerRecord
		if err := json.Unmarshal(entry.Value(), &rec); err != nil {
			return nil, ledgerErr(core.ReplayCursorFailed, "Replay", "ledger record could not be decoded", err)
		}
		items = append(items, kvRec{rev: entry.Revision(), rec: rec})
	}
	slices.SortFunc(items, func(a, b kvRec) int {
		return cmp(a.rev, b.rev)
	})
	out := make([]core.LedgerRecord, len(items))
	for i, item := range items {
		out[i] = item.rec
	}
	return out, nil
}

func (s *KVLedgerStore) get(key string) (core.LedgerRecord, bool, error) {
	entry, err := s.kv.Get(key)
	if errors.Is(err, nats.ErrKeyNotFound) {
		return core.LedgerRecord{}, false, nil
	}
	if err != nil {
		return core.LedgerRecord{}, false, err
	}
	var rec core.LedgerRecord
	if err := json.Unmarshal(entry.Value(), &rec); err != nil {
		return core.LedgerRecord{}, false, err
	}
	return rec, true, nil
}

func keyEnc(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func ledgerErr(kind core.Kind, op, msg string, cause error) *core.Error {
	details := map[string]string{}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return &core.Error{Kind: kind, Layer: "ActivationLedger", Operation: op, Message: msg, Details: details}
}

func cmp[T ~uint64](a, b T) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

var _ core.LedgerStore = (*KVLedgerStore)(nil)
