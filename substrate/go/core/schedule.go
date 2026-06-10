package core

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

type ScheduleTick struct {
	TickID       string
	DueAt        string
	AcquiredAt   string
	ExpiresAt    string
	ClockID      string
	Clock        string
	LeaderEpoch  int64
	FencingToken string
}

type ScheduleState struct {
	ScheduleID    string
	TickID        string
	ClockID       string
	Clock         string
	ClockPosition int64
	LeaderEpoch   int64
	FencingToken  string
	SourceID      string
	LeaseID       string
}

type ScheduleStore interface {
	LoadSchedule(string) (ScheduleState, bool, error)
	SaveSchedule(ScheduleState) error
}

type ScheduleEngine struct {
	auth   Auth
	ledger *DurableLedger
	store  ScheduleStore
}

// MemoryScheduleStore is a test fake (allowlisted in substrate/go/fakes-allowlist.json):
// it exists to force the RecoverFailed and WriteFailed branches impossible to force
// on a live embedded JetStream KV store. The real path is proven by
// TestEmbeddedScheduleStorePersistsRestartCatchUp over the embednats KVScheduleStore.
type MemoryScheduleStore struct {
	mu            sync.Mutex
	state         map[string]ScheduleState
	RecoverFailed bool
	WriteFailed   bool
}

func NewScheduleEngine(auth Auth, ledger *DurableLedger, store ScheduleStore) (*ScheduleEngine, error) {
	if ledger == nil || store == nil {
		return nil, fail(ScheduleConfigInvalid, "ScheduleEngine", "Configure", "schedule ledger and store are required", nil)
	}
	if strings.TrimSpace(auth.User) == "" {
		return nil, fail(ScheduleConfigInvalid, "ScheduleEngine", "Configure", "schedule auth user is required", nil)
	}
	return &ScheduleEngine{auth: auth, ledger: ledger, store: store}, nil
}

func NewMemoryScheduleStore() *MemoryScheduleStore {
	return &MemoryScheduleStore{state: map[string]ScheduleState{}}
}

func (e *ScheduleEngine) Accept(tpl Activation, tick ScheduleTick) (LedgerRecord, error) {
	act, pos, err := e.activation(tpl, tick)
	if err != nil {
		return LedgerRecord{}, err
	}
	if err := e.check(act, pos); err != nil {
		return LedgerRecord{}, err
	}
	if _, err := AuthorizeSource(e.auth, act); err != nil {
		return LedgerRecord{}, err
	}
	rec, err := e.ledger.Accept(act, Lease{
		ID:          act.SourceLease.LeaseID,
		Status:      act.SourceLease.LeaseStatus,
		PrincipalID: act.SourcePrincipal.PrincipalID,
	})
	if err != nil {
		if isKind(err, LoopSuppressed) {
			_ = e.store.SaveSchedule(scheduleState(act, pos))
		}
		return LedgerRecord{}, err
	}
	if rec.Status == Accepted {
		if err := e.store.SaveSchedule(scheduleState(act, pos)); err != nil {
			return LedgerRecord{}, err
		}
	}
	return rec, nil
}

func isKind(err error, kind Kind) bool {
	var got *Error
	return errors.As(err, &got) && got.Kind == kind
}

func (e *ScheduleEngine) CatchUp(tpl Activation, ticks []ScheduleTick) ([]LedgerRecord, error) {
	if e == nil {
		return nil, fail(ScheduleConfigInvalid, "ScheduleEngine", "CatchUp", "schedule engine is nil", nil)
	}
	state, ok, err := e.store.LoadSchedule(tpl.Source.ScheduleID)
	if err != nil {
		return nil, err
	}
	out := []LedgerRecord{}
	for _, tick := range ticks {
		pos, err := clockPos(tick.ClockID, tick.Clock)
		if err != nil {
			return nil, fail(ClockInvalid, "ScheduleEngine", "CatchUp", "schedule clock is invalid", scheduleDetails(tpl, tick))
		}
		if ok && pos <= state.ClockPosition {
			continue
		}
		rec, err := e.Accept(tpl, tick)
		if err != nil {
			return nil, err
		}
		if rec.Status == Accepted {
			out = append(out, rec)
			act, _, _ := e.activation(tpl, tick)
			state = scheduleState(act, pos)
			ok = true
		}
	}
	return out, nil
}

func (e *ScheduleEngine) activation(tpl Activation, tick ScheduleTick) (Activation, int64, error) {
	if e == nil {
		return Activation{}, 0, fail(ScheduleConfigInvalid, "ScheduleEngine", "Accept", "schedule engine is nil", nil)
	}
	if tpl.Source.Kind != "schedule" || tpl.Source.ScheduleID == "" {
		return Activation{}, 0, fail(ScheduleConfigInvalid, "ScheduleEngine", "Accept", "schedule source is required", nil)
	}
	if tick.TickID == "" || tick.DueAt == "" || tick.AcquiredAt == "" || tick.ExpiresAt == "" {
		return Activation{}, 0, fail(ScheduleConfigInvalid, "ScheduleEngine", "Accept", "schedule tick is incomplete", scheduleDetails(tpl, tick))
	}
	if tpl.Source.OwnerPrincipalID != "" && tpl.Source.OwnerPrincipalID != tpl.SourcePrincipal.PrincipalID {
		return Activation{}, 0, fail(ScheduleConfigInvalid, "ScheduleEngine", "Accept", "schedule owner does not match source principal", scheduleDetails(tpl, tick))
	}
	if tpl.SourceLease.LeaseID == "" {
		return Activation{}, 0, fail(ScheduleLeaseMissing, "ScheduleEngine", "Accept", "schedule source lease is missing", sourceCtx(tpl))
	}
	if tick.LeaderEpoch <= 0 || tick.FencingToken == "" {
		return Activation{}, 0, fail(ScheduleLeaseLost, "ScheduleEngine", "Accept", "schedule leadership fence is missing", scheduleDetails(tpl, tick))
	}
	pos, err := clockPos(tick.ClockID, tick.Clock)
	if err != nil {
		return Activation{}, 0, fail(ClockInvalid, "ScheduleEngine", "Accept", "schedule clock is invalid", scheduleDetails(tpl, tick))
	}
	act := tpl
	act.Source.TickID = tick.TickID
	act.Source.DueAt = tick.DueAt
	act.Source.AcquiredAt = tick.AcquiredAt
	act.Source.ExpiresAt = tick.ExpiresAt
	act.Source.ClockID = tick.ClockID
	act.Source.Clock = tick.Clock
	act.Source.LeaderEpoch = tick.LeaderEpoch
	act.Source.FencingToken = tick.FencingToken
	act.DedupeKey = fmt.Sprintf("schedule:%s:%s:%s", act.ScriptKey, act.Source.ScheduleID, tick.TickID)
	act.ActivationID = fmt.Sprintf("act:%s:schedule:%s", act.ScriptKey, tick.TickID)
	return act, pos, nil
}

func (e *ScheduleEngine) check(act Activation, pos int64) error {
	state, ok, err := e.store.LoadSchedule(act.Source.ScheduleID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if state.TickID == act.Source.TickID {
		return fail(ScheduleTickDuplicate, "ScheduleEngine", "Accept", "schedule tick is duplicate", scheduleDetails(act, tickFromSource(act.Source)))
	}
	if act.Source.LeaderEpoch < state.LeaderEpoch {
		return fail(ScheduleLeaseLost, "ScheduleEngine", "Accept", "schedule leader epoch is stale", scheduleDetails(act, tickFromSource(act.Source)))
	}
	if act.Source.LeaderEpoch == state.LeaderEpoch && state.FencingToken != "" && state.FencingToken != act.Source.FencingToken {
		return fail(ScheduleLeaseLost, "ScheduleEngine", "Accept", "schedule fencing token is stale", scheduleDetails(act, tickFromSource(act.Source)))
	}
	if pos <= state.ClockPosition {
		return fail(ClockInvalid, "ScheduleEngine", "Accept", "schedule clock did not advance", scheduleDetails(act, tickFromSource(act.Source)))
	}
	return nil
}

func (s *MemoryScheduleStore) LoadSchedule(id string) (ScheduleState, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.RecoverFailed {
		return ScheduleState{}, false, fail(RestartRecoveryFailed, "ScheduleEngine", "Recover", "schedule state could not be recovered", map[string]string{"scheduleId": id})
	}
	state, ok := s.state[id]
	return state, ok, nil
}

func (s *MemoryScheduleStore) SaveSchedule(state ScheduleState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.WriteFailed {
		return fail(CatchUpFailed, "ScheduleEngine", "Save", "schedule state could not be written", map[string]string{"scheduleId": state.ScheduleID})
	}
	s.state[state.ScheduleID] = state
	return nil
}

func scheduleState(act Activation, pos int64) ScheduleState {
	return ScheduleState{
		ScheduleID:    act.Source.ScheduleID,
		TickID:        act.Source.TickID,
		ClockID:       act.Source.ClockID,
		Clock:         act.Source.Clock,
		ClockPosition: pos,
		LeaderEpoch:   act.Source.LeaderEpoch,
		FencingToken:  act.Source.FencingToken,
		SourceID:      act.SourcePrincipal.SourceID,
		LeaseID:       act.SourceLease.LeaseID,
	}
}

func tickFromSource(src Source) ScheduleTick {
	return ScheduleTick{
		TickID:       src.TickID,
		DueAt:        src.DueAt,
		AcquiredAt:   src.AcquiredAt,
		ExpiresAt:    src.ExpiresAt,
		ClockID:      src.ClockID,
		Clock:        src.Clock,
		LeaderEpoch:  src.LeaderEpoch,
		FencingToken: src.FencingToken,
	}
}

func clockPos(id, clock string) (int64, error) {
	if id == "" || clock == "" {
		return 0, fmt.Errorf("clock is required")
	}
	prefix, raw, ok := strings.Cut(clock, ":")
	if !ok || prefix != id || raw == "" {
		return 0, fmt.Errorf("clock id mismatch")
	}
	pos, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || pos <= 0 {
		return 0, fmt.Errorf("clock position is invalid")
	}
	return pos, nil
}

func scheduleDetails(act Activation, tick ScheduleTick) map[string]string {
	ctx := sourceCtx(act)
	ctx["scheduleId"] = act.Source.ScheduleID
	ctx["tickId"] = tick.TickID
	ctx["clock"] = tick.Clock
	ctx["leaderEpoch"] = fmt.Sprint(tick.LeaderEpoch)
	ctx["fencingToken"] = tick.FencingToken
	return ctx
}

var _ ScheduleStore = (*MemoryScheduleStore)(nil)
