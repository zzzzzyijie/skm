// Package corebridge exposes SKM's Go behavior to trusted native clients.
// It is a private, versioned stdio protocol and is not a public network API.
package corebridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zzzzzyijie/skm/internal/application"
	"github.com/zzzzzyijie/skm/internal/buildinfo"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/store"
)

const (
	protocolVersion = 1
	maxMessageSize  = 16 << 20
)

var errMessageTooLarge = errors.New("Core Bridge message exceeds 16 MiB")

type Bridge struct {
	service *application.Service
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Data    rpcErrorData `json:"data"`
}

type rpcErrorData struct {
	Kind      string `json:"kind"`
	Retryable bool   `json:"retryable"`
}

func New(storage *store.Store) *Bridge {
	return &Bridge{service: application.New(storage)}
}

// Run serves newline-delimited JSON-RPC until stdin closes or the context is cancelled.
func (b *Bridge) Run(ctx context.Context, input io.Reader, output io.Writer) error {
	reader := bufio.NewReaderSize(input, maxMessageSize+1)
	encoder := json.NewEncoder(output)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		message, err := readMessage(reader)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if errors.Is(err, errMessageTooLarge) {
			if encodeErr := encoder.Encode(errorResponse(nil, -32600, err.Error(), "message_too_large", false)); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("read Core Bridge request: %w", err)
		}
		var value request
		if err := json.Unmarshal(message, &value); err != nil {
			if encodeErr := encoder.Encode(errorResponse(nil, -32700, "invalid JSON", "parse_error", false)); encodeErr != nil {
				return encodeErr
			}
			continue
		}
		if err := encoder.Encode(b.handle(ctx, value)); err != nil {
			return err
		}
	}
}

func readMessage(reader *bufio.Reader) ([]byte, error) {
	message, err := reader.ReadSlice('\n')
	if err == nil {
		return append([]byte(nil), message[:len(message)-1]...), nil
	}
	if !errors.Is(err, bufio.ErrBufferFull) {
		if errors.Is(err, io.EOF) && len(message) > 0 {
			return append([]byte(nil), message...), nil
		}
		return nil, err
	}
	for errors.Is(err, bufio.ErrBufferFull) {
		_, err = reader.ReadSlice('\n')
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return nil, errMessageTooLarge
}

func (b *Bridge) handle(ctx context.Context, value request) response {
	if value.JSONRPC != "2.0" || len(value.ID) == 0 || strings.TrimSpace(value.Method) == "" {
		return errorResponse(value.ID, -32600, "invalid JSON-RPC request", "invalid_request", false)
	}
	if value.Method == "system.handshake" {
		data, _ := json.Marshal(map[string]any{
			"protocolVersion":        protocolVersion,
			"coreVersion":            buildinfo.Current(),
			"schemaVersion":          domain.SchemaVersion,
			"promptSchemaVersion":    domain.PromptSchemaVersion,
			"workspaceSchemaVersion": domain.WorkspaceSchemaVersion,
			"capabilities": []string{
				"skills.read", "skills.write", "activations.write", "agents.write",
				"prompts.write", "sources.write", "sources.preview", "projects.read", "projects.write",
				"projects.dependencies", "prompts.render", "history.read", "history.write",
				"workspace.read", "workspace.write", "diagnostics.read",
			},
		})
		return response{JSONRPC: "2.0", ID: value.ID, Result: data}
	}

	result, err := b.service.Invoke(ctx, value.Method, value.Params)
	if err != nil {
		var appError *application.Error
		if !errors.As(err, &appError) {
			return errorResponse(value.ID, -32603, err.Error(), "internal", true)
		}
		code := -32000
		if appError.Kind == "method_not_found" {
			code = -32601
		}
		return errorResponse(value.ID, code, appError.Error(), appError.Kind, appError.Retryable)
	}
	data, err := json.Marshal(result)
	if err != nil {
		return errorResponse(value.ID, -32603, "Core could not encode its response", "internal", true)
	}
	return response{JSONRPC: "2.0", ID: value.ID, Result: data}
}

func errorResponse(id json.RawMessage, code int, message, kind string, retryable bool) response {
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	return response{JSONRPC: "2.0", ID: id, Error: &rpcError{
		Code: code, Message: message, Data: rpcErrorData{Kind: kind, Retryable: retryable},
	}}
}
