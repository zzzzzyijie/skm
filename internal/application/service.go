// Package application contains transport-independent SKM use cases used by
// the native Core Bridge and available for incremental CLI/HTTP migration.
package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/prompt"
	"github.com/zzzzzyijie/skm/internal/store"
)

type Service struct {
	Store *store.Store
	mu    sync.Mutex
}

type Error struct {
	Kind      string
	Retryable bool
	Err       error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func New(storage *store.Store) *Service {
	return &Service{Store: storage}
}

func (s *Service) Invoke(ctx context.Context, method string, params json.RawMessage) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, wrap(err)
	}
	switch method {
	case "dashboard.get":
		return result(s.Dashboard())
	case "skills.list":
		return result(s.ListSkills(nil))
	case "skills.get":
		var input IDInput
		return decoded(params, &input, func() (any, error) { return result(s.GetSkill(input.ID)) })
	case "skills.add":
		var input AddSkillInput
		return decoded(params, &input, func() (any, error) { return result(s.AddSkill(input)) })
	case "skills.update":
		var input UpdateSkillInput
		return decoded(params, &input, func() (any, error) { return result(s.UpdateSkill(input)) })
	case "skills.remove":
		var input IDInput
		return decoded(params, &input, func() (any, error) { return result(s.RemoveSkill(input.ID)) })
	case "skills.tags.replace":
		var input SkillTagsInput
		return decoded(params, &input, func() (any, error) { return result(s.ReplaceSkillTags(input)) })
	case "agents.list":
		return result(s.ListAgents())
	case "agents.configure":
		var input ConfigureAgentsInput
		return decoded(params, &input, func() (any, error) { return result(s.ConfigureAgents(input.Agents)) })
	case "agents.custom.save":
		var input CustomAgentInput
		return decoded(params, &input, func() (any, error) { return result(s.SaveCustomAgent(input)) })
	case "agents.custom.delete":
		var input IDInput
		return decoded(params, &input, func() (any, error) { return result(s.DeleteCustomAgent(input.ID)) })
	case "activations.status":
		return result(s.ActivationStatus())
	case "activations.enable":
		var input ActivationInput
		return decoded(params, &input, func() (any, error) { return result(s.Enable(input)) })
	case "activations.disable":
		var input ActivationInput
		return decoded(params, &input, func() (any, error) { return result(s.Disable(input)) })
	case "prompts.list":
		return result(s.ListPrompts(nil))
	case "prompts.get":
		var input IDInput
		return decoded(params, &input, func() (any, error) { return result(s.GetPrompt(input.ID)) })
	case "prompts.create":
		var input PromptWriteInput
		return decoded(params, &input, func() (any, error) { return result(s.CreatePrompt(input)) })
	case "prompts.update":
		var input PromptWriteInput
		return decoded(params, &input, func() (any, error) { return result(s.UpdatePrompt(input)) })
	case "prompts.remove":
		var input IDInput
		return decoded(params, &input, func() (any, error) { return result(s.RemovePrompt(input.ID)) })
	case "prompts.validate":
		var input PromptWriteInput
		return decoded(params, &input, func() (any, error) { return result(s.ValidatePrompt(input)) })
	case "sources.list":
		return result(s.ListSources())
	case "sources.add":
		var input AddSourceInput
		return decoded(params, &input, func() (any, error) { return result(s.AddSource(input)) })
	case "projects.list":
		return result(s.ListProjects())
	case "workspace.get":
		return result(s.GetWorkspace())
	case "system.doctor":
		return result(s.Doctor())
	default:
		return nil, &Error{Kind: "method_not_found", Err: fmt.Errorf("method %q is not supported", method)}
	}
}

func (s *Service) withLock(fn func() error) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	unlock, err := s.Store.Lock()
	if err != nil {
		return err
	}
	defer func() {
		if unlockErr := unlock(); unlockErr != nil {
			log.Printf("unlock application service: %v", unlockErr)
		}
	}()
	return fn()
}

func decoded(data json.RawMessage, target any, fn func() (any, error)) (any, error) {
	if len(data) == 0 {
		data = json.RawMessage("{}")
	}
	if err := json.Unmarshal(data, target); err != nil {
		return nil, &Error{Kind: "validation", Err: fmt.Errorf("invalid params: %w", err)}
	}
	return fn()
}

func result[T any](value T, err error) (any, error) {
	if err != nil {
		return nil, wrap(err)
	}
	return value, nil
}

func wrap(err error) error {
	var appError *Error
	if errors.As(err, &appError) {
		return err
	}
	lower := strings.ToLower(err.Error())
	kind := "validation"
	retryable := false
	switch {
	case errors.Is(err, catalog.ErrEditConflict), errors.Is(err, prompt.ErrEditConflict), strings.Contains(lower, "conflict"):
		kind = "conflict"
	case errors.Is(err, catalog.ErrNotEditable), strings.Contains(lower, "not editable"):
		kind = "not_editable"
	case strings.Contains(lower, "not found"):
		kind = "not_found"
	case strings.Contains(lower, "still enabled"), strings.Contains(lower, "disable it first"), strings.Contains(lower, "is enabled"):
		kind = "still_referenced"
	case strings.Contains(lower, "permission denied"):
		kind = "permission"
	case strings.Contains(lower, "input/output error"):
		kind = "internal"
		retryable = true
	}
	return &Error{Kind: kind, Retryable: retryable, Err: err}
}

type IDInput struct {
	ID string `json:"id"`
}
