package wrapper

import (
	"encoding/json"
	"fmt"
)

// StreamJsonFrame is one decoded stream-json line from the claude CLI
// (--output-format stream-json --include-partial-messages). Type is the
// top-level event type; Raw retains the verbatim line.
type StreamJsonFrame struct {
	Type string
	Raw  json.RawMessage
}

// ParseFailure is the StreamJsonParseFailure family: a line that is not valid
// stream-json (not JSON, or missing the type discriminator).
type ParseFailure struct {
	Msg string
	Err error
}

func (e *ParseFailure) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("StreamJsonParseFailure: %s: %v", e.Msg, e.Err)
	}
	return "StreamJsonParseFailure: " + e.Msg
}

func (e *ParseFailure) Unwrap() error { return e.Err }

func ParseStreamJsonFrame(data []byte) (StreamJsonFrame, error) {
	var ev struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return StreamJsonFrame{}, &ParseFailure{Msg: "not valid JSON", Err: err}
	}
	if ev.Type == "" {
		return StreamJsonFrame{}, &ParseFailure{Msg: "missing type field"}
	}
	return StreamJsonFrame{Type: ev.Type, Raw: append(json.RawMessage(nil), data...)}, nil
}
