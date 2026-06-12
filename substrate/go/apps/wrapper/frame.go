package wrapper

import "encoding/json"

// SessionFrame renders one decoded agent event as a canonical session.frame
// envelope: a token frame for streaming text deltas, otherwise a chunk frame
// whose body is the verbatim event line as a string value — agent event keys
// (usage.input_tokens et al.) collide with the reserved-vocab facade, which
// scans property names, never values. The wrapper never emits status frames;
// those are runner-originated by contract.
func SessionFrame(sessionID string, f StreamJsonFrame) ([]byte, error) {
	if text, ok := deltaText(f); ok {
		return json.Marshal(map[string]any{
			"kind":      "session.frame",
			"frame":     "token",
			"origin":    "wrapper",
			"sessionId": sessionID,
			"text":      text,
		})
	}
	return json.Marshal(map[string]any{
		"kind":      "session.frame",
		"frame":     "chunk",
		"origin":    "wrapper",
		"sessionId": sessionID,
		"body":      string(f.Raw),
	})
}

func deltaText(f StreamJsonFrame) (string, bool) {
	if f.Type != "stream_event" {
		return "", false
	}
	var ev struct {
		Event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		} `json:"event"`
	}
	if json.Unmarshal(f.Raw, &ev) != nil ||
		ev.Event.Type != "content_block_delta" || ev.Event.Delta.Type != "text_delta" || ev.Event.Delta.Text == "" {
		return "", false
	}
	return ev.Event.Delta.Text, true
}

// SteerToStdin translates a canonical session.steer_intent into the claude
// stream-json stdin user message. ok is false for well-formed payloads of
// another kind, which the wrapper skips.
func SteerToStdin(data []byte) (line []byte, ok bool, err error) {
	var in struct {
		Kind string `json:"kind"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &in); err != nil {
		return nil, false, &ParseFailure{Msg: "malformed steer payload", Err: err}
	}
	if in.Kind != "session.steer_intent" {
		return nil, false, nil
	}
	line, err = json.Marshal(map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []map[string]string{{"type": "text", "text": in.Text}},
		},
	})
	if err != nil {
		return nil, false, err
	}
	return line, true, nil
}
