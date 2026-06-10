package embednats

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/lagz0ne/tinkabot/substrate/go/contract"
	"github.com/lagz0ne/tinkabot/substrate/go/edge"
	"github.com/nats-io/nats.go"
)

func TestBrowserGatewayCommandAcceptanceOverRealNATS(t *testing.T) {
	cfg := valid(t)
	cfg.Auth.Permissions.Publish.Allow = []string{"tb.app.browser.command", "_INBOX.>"}
	cfg.Auth.Permissions.Subscribe.Allow = []string{"tb.app.browser.command", "_INBOX.>"}
	rt, err := Start(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stop(t, rt) })

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	gw, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer gw.Close()

	reg := registry(t)
	pol := edge.GatewayMutationPolicy{
		SessionID:               "session-001",
		LeaseID:                 "lease-001",
		LeaseStatus:             "active",
		CSRFToken:               "csrf-001",
		AllowedOrigin:           "https://app.localhost",
		CurrentArtifactRevision: "artifact.rev.7",
	}

	sub, err := gw.Subscribe("tb.app.browser.command", func(msg *nats.Msg) {
		var req gatewayCommand
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			_ = msg.Respond(denied("cmd-invalid", "BrowserEdgeInvalid", "BrowserEdge", "decodeGatewayCommand"))
			return
		}
		if err := edge.CheckGatewayMutation(pol, req.Gateway); err != nil {
			var edgeErr *edge.EdgeError
			if errors.As(err, &edgeErr) {
				_ = msg.Respond(denied(req.CommandID, string(edgeErr.Kind), edgeErr.Layer, edgeErr.Operation))
				return
			}
			_ = msg.Respond(denied(req.CommandID, "BrowserEdgeCritical", "BrowserEdge", "CheckGatewayMutation"))
			return
		}
		_ = msg.Respond(accepted(req.CommandID))
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()
	if err := gw.FlushTimeout(time.Second); err != nil {
		t.Fatal(err)
	}

	client, err := rt.Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	res, err := client.Request("tb.app.browser.command", command("cmd-nats-001", validGatewayRequest()), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Validate(contract.ContractSchemaID, res.Data); err != nil {
		t.Fatalf("accepted status is not canonical command.acceptance: %v\n%s", err, res.Data)
	}
	var ok map[string]any
	if err := json.Unmarshal(res.Data, &ok); err != nil {
		t.Fatal(err)
	}
	if ok["status"] != "accepted" || ok["commandId"] != "cmd-nats-001" {
		t.Fatalf("accepted status drift: %#v", ok)
	}

	bad := validGatewayRequest()
	bad.CSRFToken = "csrf-bad"
	res, err = client.Request("tb.app.browser.command", command("cmd-nats-denied", bad), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := reg.Validate(contract.ContractSchemaID, res.Data); err != nil {
		t.Fatalf("denied status is not canonical command.acceptance: %v\n%s", err, res.Data)
	}
	var no map[string]any
	if err := json.Unmarshal(res.Data, &no); err != nil {
		t.Fatal(err)
	}
	errDoc := no["error"].(map[string]any)
	origin := errDoc["origin"].(map[string]any)
	if no["status"] != "rejected" || errDoc["kind"] != "CSRFDenied" || origin["layer"] != "BrowserEdge" {
		t.Fatalf("denied status drift: %#v", no)
	}
}

type gatewayCommand struct {
	CommandID string                      `json:"commandId"`
	Gateway   edge.GatewayMutationRequest `json:"gateway"`
}

func command(id string, req edge.GatewayMutationRequest) []byte {
	out, err := json.Marshal(gatewayCommand{CommandID: id, Gateway: req})
	if err != nil {
		panic(err)
	}
	return out
}

func validGatewayRequest() edge.GatewayMutationRequest {
	return edge.GatewayMutationRequest{
		SessionID:        "session-001",
		LeaseID:          "lease-001",
		CSRFToken:        "csrf-001",
		Origin:           "https://app.localhost",
		FetchSite:        "same-origin",
		FetchMode:        "cors",
		FetchDest:        "empty",
		ArtifactRevision: "artifact.rev.7",
	}
}

func accepted(commandID string) []byte {
	return status(commandID, "accepted", "", "", "")
}

func denied(commandID, kind, layer, op string) []byte {
	return status(commandID, "rejected", kind, layer, op)
}

func status(commandID, state, kind, layer, op string) []byte {
	doc := map[string]any{
		"kind":       "command.acceptance",
		"type":       "command.acceptance",
		"commandId":  commandID,
		"status":     state,
		"sequence":   1,
		"observedAt": "2026-06-09T00:00:00.000Z",
		"provenance": map[string]any{
			"schemaId":      contract.ContractSchemaID,
			"schemaVersion": "v1",
			"appRevision":   "app.rev.1",
			"createdAt":     "2026-06-09T00:00:00.000Z",
			"producer":      "browser-gateway",
		},
		"capability": map[string]any{
			"principalId":   "principal.browser.001",
			"sessionId":     "session-001",
			"capabilityId":  "cap-001",
			"leaseId":       "lease-001",
			"leaseStatus":   "active",
			"appRevision":   "app.rev.1",
			"schemaVersion": "v1",
		},
		"chain": map[string]any{
			"chainId": "chain-001",
			"rootId":  "root-001",
			"hop":     1,
			"maxHops": 5,
		},
	}
	if kind != "" {
		doc["error"] = map[string]any{
			"kind":    kind,
			"message": kind,
			"origin": map[string]any{
				"layer":     layer,
				"operation": op,
			},
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return out
}

func registry(t *testing.T) *contract.Registry {
	t.Helper()
	reg, err := contract.Open(filepath.Join("..", "..", "..", "schemas", "base", "v1"))
	if err != nil {
		t.Fatal(err)
	}
	return reg
}
