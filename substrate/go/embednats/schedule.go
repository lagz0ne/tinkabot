package embednats

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/lagz0ne/tinkabot/substrate/go/core"
	"github.com/nats-io/nats.go"
)

type KVScheduleStore struct {
	nc *nats.Conn
	kv nats.KeyValue
}

func NewKVScheduleStore(ctx context.Context, rt *Runtime, bucket string) (*KVScheduleStore, error) {
	nc, err := rt.Connect(ctx)
	if err != nil {
		return nil, err
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, scheduleErr(core.JetStreamUnavailable, "OpenScheduleStore", "JetStream context is unavailable", err)
	}
	kv, err := js.KeyValue(bucket)
	if errors.Is(err, nats.ErrBucketNotFound) {
		kv, err = js.CreateKeyValue(&nats.KeyValueConfig{Bucket: bucket, Storage: nats.FileStorage})
	}
	if err != nil {
		nc.Close()
		return nil, scheduleErr(core.BucketMissing, "OpenScheduleStore", "schedule KV bucket is unavailable", err)
	}
	return &KVScheduleStore{nc: nc, kv: kv}, nil
}

func (s *KVScheduleStore) Close() {
	if s != nil && s.nc != nil {
		s.nc.Close()
	}
}

func (s *KVScheduleStore) LoadSchedule(id string) (core.ScheduleState, bool, error) {
	entry, err := s.kv.Get("s." + keyEnc(id))
	if errors.Is(err, nats.ErrKeyNotFound) {
		return core.ScheduleState{}, false, nil
	}
	if err != nil {
		return core.ScheduleState{}, false, scheduleErr(core.RestartRecoveryFailed, "Recover", "schedule state could not be read", err)
	}
	var state core.ScheduleState
	if err := json.Unmarshal(entry.Value(), &state); err != nil {
		return core.ScheduleState{}, false, scheduleErr(core.RestartRecoveryFailed, "Recover", "schedule state could not be decoded", err)
	}
	return state, true, nil
}

func (s *KVScheduleStore) SaveSchedule(state core.ScheduleState) error {
	body, err := json.Marshal(state)
	if err != nil {
		return scheduleErr(core.CatchUpFailed, "Save", "schedule state could not be encoded", err)
	}
	if _, err := s.kv.Put("s."+keyEnc(state.ScheduleID), body); err != nil {
		return scheduleErr(core.CatchUpFailed, "Save", "schedule state could not be written", err)
	}
	return nil
}

func scheduleErr(kind core.Kind, op, msg string, cause error) *core.Error {
	details := map[string]string{}
	if cause != nil {
		details["cause"] = cause.Error()
	}
	return &core.Error{Kind: kind, Layer: "ScheduleEngine", Operation: op, Message: msg, Details: details}
}

var _ core.ScheduleStore = (*KVScheduleStore)(nil)
