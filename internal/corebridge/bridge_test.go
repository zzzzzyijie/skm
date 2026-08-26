package corebridge

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zzzzzyijie/skm/internal/store"
)

func TestHandshake(t *testing.T) {
	bridge := newTestBridge(t)
	responses := runRequests(t, bridge, `{"jsonrpc":"2.0","id":"1","method":"system.handshake","params":{"client":"test"}}`)
	var result struct {
		ProtocolVersion int      `json:"protocolVersion"`
		SchemaVersion   int      `json:"schemaVersion"`
		Capabilities    []string `json:"capabilities"`
	}
	decodeResult(t, responses[0], &result)
	if result.ProtocolVersion != 1 || result.SchemaVersion != 2 || len(result.Capabilities) == 0 {
		t.Fatalf("unexpected handshake: %+v", result)
	}
}

func TestAddAndListSkill(t *testing.T) {
	bridge := newTestBridge(t)
	directory := filepath.Join(t.TempDir(), "sample")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "SKILL.md"), []byte("---\nname: sample\ndescription: Sample Skill\n---\n\nUse it well.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	params, _ := json.Marshal(map[string]any{"path": directory, "tags": []string{"desktop"}})
	requests := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"1","method":"skills.add","params":` + string(params) + `}`,
		`{"jsonrpc":"2.0","id":"2","method":"skills.list","params":{}}`,
	}, "\n")
	responses := runRequests(t, bridge, requests)
	var skills []struct {
		Name string `json:"name"`
	}
	decodeResult(t, responses[1], &skills)
	if len(skills) != 1 || skills[0].Name != "sample" {
		t.Fatalf("unexpected skills: %+v", skills)
	}
}

func TestUnknownMethodReturnsStructuredError(t *testing.T) {
	bridge := newTestBridge(t)
	responses := runRequests(t, bridge, `{"jsonrpc":"2.0","id":"1","method":"missing.method","params":{}}`)
	var value struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(responses[0], &value); err != nil {
		t.Fatal(err)
	}
	if value.Error == nil || value.Error.Data.Kind != "method_not_found" {
		t.Fatalf("unexpected error: %+v", value.Error)
	}
}

func TestOversizedMessageDoesNotStopBridge(t *testing.T) {
	bridge := newTestBridge(t)
	oversized := strings.Repeat("x", maxMessageSize+2)
	requests := oversized + "\n" + `{"jsonrpc":"2.0","id":"2","method":"system.handshake","params":{}}`
	responses := runRequests(t, bridge, requests)
	if len(responses) != 2 {
		t.Fatalf("expected two responses, got %d", len(responses))
	}
	var first struct {
		Error *rpcError `json:"error"`
	}
	if err := json.Unmarshal(responses[0], &first); err != nil {
		t.Fatal(err)
	}
	if first.Error == nil || first.Error.Data.Kind != "message_too_large" {
		t.Fatalf("unexpected oversized error: %+v", first.Error)
	}
	var handshake struct {
		ProtocolVersion int `json:"protocolVersion"`
	}
	decodeResult(t, responses[1], &handshake)
	if handshake.ProtocolVersion != 1 {
		t.Fatalf("bridge did not recover after oversized message: %+v", handshake)
	}
}

func newTestBridge(t *testing.T) *Bridge {
	t.Helper()
	root := t.TempDir()
	storage, err := store.New(store.Paths{
		Home: filepath.Join(root, "skm"), UserHome: filepath.Join(root, "user"), ProjectRoot: filepath.Join(root, "project"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Ensure(); err != nil {
		t.Fatal(err)
	}
	return New(storage)
}

func runRequests(t *testing.T, bridge *Bridge, requests string) []json.RawMessage {
	t.Helper()
	var output bytes.Buffer
	if err := bridge.Run(context.Background(), strings.NewReader(requests+"\n"), &output); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	result := make([]json.RawMessage, len(lines))
	for index, line := range lines {
		result[index] = json.RawMessage(line)
	}
	return result
}

func decodeResult(t *testing.T, response json.RawMessage, target any) {
	t.Helper()
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  *rpcError       `json:"error"`
	}
	if err := json.Unmarshal(response, &envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error != nil {
		t.Fatalf("unexpected RPC error: %+v", envelope.Error)
	}
	if err := json.Unmarshal(envelope.Result, target); err != nil {
		t.Fatal(err)
	}
}
