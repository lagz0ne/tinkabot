package wrapper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
)

// recorded returns the real claude CLI stream-json lines captured locally on
// 2026-06-12 (claude 2.1.173, --print --verbose --input-format stream-json
// --output-format stream-json --include-partial-messages).
func recorded(t *testing.T) [][]byte {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "recorded.jsonl"))
	if err != nil {
		t.Fatalf("recorded fixtures missing: %v", err)
	}
	defer f.Close()
	var lines [][]byte
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<20)
	for sc.Scan() {
		lines = append(lines, append([]byte(nil), sc.Bytes()...))
	}
	if len(lines) == 0 {
		t.Fatal("recorded fixtures empty")
	}
	return lines
}

func recordedOfType(t *testing.T, lines [][]byte, typ, contains string) []byte {
	t.Helper()
	for _, l := range lines {
		var ev struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(l, &ev) != nil || ev.Type != typ {
			continue
		}
		if contains == "" || strings.Contains(string(l), contains) {
			return l
		}
	}
	t.Fatalf("no recorded line of type %q containing %q", typ, contains)
	return nil
}

// TestAgentWrapperStreamJsonDecode owns the StreamJsonParseFailure family:
// recorded real-CLI lines decode to typed frames; malformed input returns a
// typed *ParseFailure.
func TestAgentWrapperStreamJsonDecode(t *testing.T) {
	t.Parallel()
	lines := recorded(t)

	for _, tc := range []struct{ name, typ, contains string }{
		{"stream_event", "stream_event", "text_delta"},
		{"assistant", "assistant", ""},
		{"result", "result", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			line := recordedOfType(t, lines, tc.typ, tc.contains)
			f, err := ParseStreamJsonFrame(line)
			if err != nil {
				t.Fatalf("recorded %s line must decode: %v", tc.typ, err)
			}
			if f.Type != tc.typ {
				t.Fatalf("Type = %q, want %q", f.Type, tc.typ)
			}
			if len(f.Raw) == 0 {
				t.Fatal("Raw must retain the original line")
			}
		})
	}

	t.Run("malformed_not_json", func(t *testing.T) {
		t.Parallel()
		_, err := ParseStreamJsonFrame([]byte("{this is not json"))
		var pf *ParseFailure
		if !errors.As(err, &pf) {
			t.Fatalf("want typed *ParseFailure (StreamJsonParseFailure family), got %v", err)
		}
	})

	t.Run("malformed_missing_type", func(t *testing.T) {
		t.Parallel()
		_, err := ParseStreamJsonFrame([]byte(`{"event":{"type":"message_stop"}}`))
		var pf *ParseFailure
		if !errors.As(err, &pf) {
			t.Fatalf("want typed *ParseFailure (StreamJsonParseFailure family), got %v", err)
		}
	})
}

// TestAgentWrapperSessionFrame proves the wrapper emits canonical
// session.frame envelopes that validate against the slice-1 contract: token
// frames for text deltas, chunk frames (body = raw event) for everything
// else, and never a status frame.
func TestAgentWrapperSessionFrame(t *testing.T) {
	t.Parallel()
	lines := recorded(t)
	reg, err := contract.Open(filepath.Join("..", "..", "..", "..", "schemas", "base", "v1"))
	if err != nil {
		t.Fatalf("contract registry: %v", err)
	}
	const sid = "wrap-unit-001"

	t.Run("token_from_text_delta", func(t *testing.T) {
		t.Parallel()
		f, err := ParseStreamJsonFrame(recordedOfType(t, lines, "stream_event", "text_delta"))
		if err != nil {
			t.Fatal(err)
		}
		out, err := SessionFrame(sid, f)
		if err != nil {
			t.Fatal(err)
		}
		if err := reg.Validate(contract.ContractSchemaID, out); err != nil {
			t.Fatalf("token frame must validate against the canonical contract: %v", err)
		}
		var frame struct {
			Kind      string `json:"kind"`
			Frame     string `json:"frame"`
			Origin    string `json:"origin"`
			SessionID string `json:"sessionId"`
			Text      string `json:"text"`
		}
		if err := json.Unmarshal(out, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Kind != "session.frame" || frame.Frame != "token" || frame.Origin != "wrapper" {
			t.Fatalf("not a canonical wrapper token frame: %s", out)
		}
		if frame.SessionID != sid {
			t.Fatalf("sessionId = %q, want %q", frame.SessionID, sid)
		}
		if frame.Text != "pong" {
			t.Fatalf("text = %q, want the recorded delta text %q", frame.Text, "pong")
		}
	})

	t.Run("chunk_from_result", func(t *testing.T) {
		t.Parallel()
		f, err := ParseStreamJsonFrame(recordedOfType(t, lines, "result", ""))
		if err != nil {
			t.Fatal(err)
		}
		out, err := SessionFrame(sid, f)
		if err != nil {
			t.Fatal(err)
		}
		if err := reg.Validate(contract.ContractSchemaID, out); err != nil {
			t.Fatalf("chunk frame must validate against the canonical contract: %v", err)
		}
		var frame struct {
			Frame string `json:"frame"`
			Body  string `json:"body"`
		}
		if err := json.Unmarshal(out, &frame); err != nil {
			t.Fatal(err)
		}
		if frame.Frame != "chunk" {
			t.Fatalf("frame = %q, want chunk", frame.Frame)
		}
		var body struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(frame.Body), &body); err != nil || body.Type != "result" {
			t.Fatalf("chunk body must carry the raw event line verbatim as a string, got %q", frame.Body)
		}
	})

	t.Run("never_status", func(t *testing.T) {
		t.Parallel()
		for _, line := range lines {
			f, err := ParseStreamJsonFrame(line)
			if err != nil {
				t.Fatal(err)
			}
			out, err := SessionFrame(sid, f)
			if err != nil {
				t.Fatal(err)
			}
			var frame struct {
				Frame string `json:"frame"`
			}
			if err := json.Unmarshal(out, &frame); err != nil {
				t.Fatal(err)
			}
			if frame.Frame != "token" && frame.Frame != "chunk" {
				t.Fatalf("wrapper may only emit token/chunk frames, got %q", frame.Frame)
			}
		}
	})
}

// TestAgentWrapperPump covers the publish loop: recorded lines become
// canonical frames in order, and a malformed line is skipped without
// stopping the loop.
func TestAgentWrapperPump(t *testing.T) {
	t.Parallel()
	lines := recorded(t)
	var in strings.Builder
	in.WriteString("{malformed first\n")
	for _, l := range lines {
		in.Write(l)
		in.WriteByte('\n')
	}
	var frames [][]byte
	pump("pump-001", strings.NewReader(in.String()), func(b []byte) error {
		frames = append(frames, append([]byte(nil), b...))
		return nil
	})
	if len(frames) != len(lines) {
		t.Fatalf("want %d frames (malformed line skipped), got %d", len(lines), len(frames))
	}
	for _, b := range frames {
		var f struct {
			Kind      string `json:"kind"`
			Frame     string `json:"frame"`
			SessionID string `json:"sessionId"`
		}
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(err)
		}
		if f.Kind != "session.frame" || f.SessionID != "pump-001" || (f.Frame != "token" && f.Frame != "chunk") {
			t.Fatalf("non-canonical frame from pump: %s", b)
		}
	}
}

// TestAgentWrapperPumpReaderError proves a failing agent stdout reader ends
// the pump with the underlying error instead of freezing silently.
func TestAgentWrapperPumpReaderError(t *testing.T) {
	t.Parallel()
	want := errors.New("stdout torn down")
	err := pump("pump-err-001", iotest.ErrReader(want), func([]byte) error { return nil })
	if !errors.Is(err, want) {
		t.Fatalf("pump must surface the reader error, got %v", err)
	}
}

// TestAgentWrapperHandleWait covers both Wait exits: handle completion and
// caller context cancellation.
func TestAgentWrapperHandleWait(t *testing.T) {
	t.Parallel()

	t.Run("done", func(t *testing.T) {
		t.Parallel()
		want := errors.New("agent exited")
		h := &WrapperHandle{done: make(chan struct{}), err: want}
		close(h.done)
		if err := h.Wait(context.Background()); !errors.Is(err, want) {
			t.Fatalf("want handle error, got %v", err)
		}
	})

	t.Run("ctx_canceled", func(t *testing.T) {
		t.Parallel()
		h := &WrapperHandle{done: make(chan struct{})}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if err := h.Wait(ctx); !errors.Is(err, context.Canceled) {
			t.Fatalf("want context.Canceled, got %v", err)
		}
	})
}

// TestAgentWrapperSteerTranslate proves the wrapper translates canonical
// session.steer_intent payloads into the claude stream-json stdin user
// message, skips non-steer kinds, and rejects malformed payloads.
func TestAgentWrapperSteerTranslate(t *testing.T) {
	t.Parallel()

	t.Run("steer_intent", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{"kind":"session.steer_intent","intent":"steer","sessionId":"s1","text":"focus on parity"}`)
		line, ok, err := SteerToStdin(in)
		if err != nil || !ok {
			t.Fatalf("valid steer intent must translate, ok=%v err=%v", ok, err)
		}
		var msg struct {
			Type    string `json:"type"`
			Message struct {
				Role    string `json:"role"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"message"`
		}
		if err := json.Unmarshal(line, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != "user" || msg.Message.Role != "user" {
			t.Fatalf("not a claude user message: %s", line)
		}
		if len(msg.Message.Content) != 1 || msg.Message.Content[0].Text != "focus on parity" {
			t.Fatalf("steer text must ride into the user message verbatim: %s", line)
		}
	})

	t.Run("non_steer_kind_skipped", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{"kind":"session.stop_intent","intent":"stop","sessionId":"s1"}`)
		_, ok, err := SteerToStdin(in)
		if err != nil {
			t.Fatalf("non-steer kinds are skipped, not errors: %v", err)
		}
		if ok {
			t.Fatal("non-steer kinds must not reach the agent stdin")
		}
	})

	t.Run("malformed_rejected", func(t *testing.T) {
		t.Parallel()
		_, ok, err := SteerToStdin([]byte("{not json"))
		if err == nil || ok {
			t.Fatalf("malformed steer payloads must be rejected, ok=%v err=%v", ok, err)
		}
	})
}
