package application

import (
	"fmt"

	"github.com/zzzzzyijie/skm/internal/domain"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
)

type PromptDetails struct {
	domain.Prompt
	Content string `json:"content"`
	Body    string `json:"body"`
}

type PromptWriteInput struct {
	ID          string                  `json:"id"`
	Content     string                  `json:"content"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Tags        []string                `json:"tags"`
	Body        string                  `json:"body"`
	Variables   []domain.PromptVariable `json:"variables"`
	Source      string                  `json:"source"`
	BaseHash    string                  `json:"baseHash"`
}

type PromptRenderInput struct {
	ID     string            `json:"id"`
	Values map[string]string `json:"values"`
}

func (input PromptWriteInput) promptContent() (string, error) {
	if input.Content != "" {
		return input.Content, nil
	}
	data, err := promptpkg.Build(input.Name, input.Description, input.Body, input.Tags, input.Variables)
	return string(data), err
}

func (s *Service) ListPrompts(tags []string) ([]domain.Prompt, error) {
	return promptpkg.New(s.Store).List(tags)
}

func (s *Service) GetPrompt(id string) (PromptDetails, error) {
	value, document, err := promptpkg.New(s.Store).Read(id)
	if err != nil {
		return PromptDetails{}, err
	}
	return PromptDetails{Prompt: value, Content: document.Content, Body: document.Body}, nil
}

func (s *Service) CreatePrompt(input PromptWriteInput) (domain.Prompt, error) {
	content, err := input.promptContent()
	if err != nil {
		return domain.Prompt{}, err
	}
	var value domain.Prompt
	err = s.withLock(func() error {
		var createErr error
		value, createErr = promptpkg.New(s.Store).Create(content, input.Source, input.Tags)
		return createErr
	})
	return value, err
}

func (s *Service) UpdatePrompt(input PromptWriteInput) (domain.Prompt, error) {
	return s.updatePrompt(input, "edit")
}

func (s *Service) updatePrompt(input PromptWriteInput, historyReason string) (domain.Prompt, error) {
	if input.ID == "" {
		return domain.Prompt{}, fmt.Errorf("id is required")
	}
	content, err := input.promptContent()
	if err != nil {
		return domain.Prompt{}, err
	}
	var value domain.Prompt
	err = s.withLock(func() error {
		manager := promptpkg.New(s.Store)
		before, current, readErr := manager.Read(input.ID)
		if readErr != nil {
			return readErr
		}
		if input.BaseHash != "" && input.BaseHash != before.Hash {
			return fmt.Errorf("%w: %s changed since editing started", promptpkg.ErrEditConflict, before.ID)
		}
		proposed, parseErr := promptpkg.Parse([]byte(content))
		if parseErr != nil {
			return parseErr
		}
		if proposed.Name != before.Name {
			return fmt.Errorf("Prompt name cannot change from %q to %q; duplicate it instead", before.Name, proposed.Name)
		}
		if proposed.Hash != before.Hash {
			if _, historyErr := s.Store.SaveHistory(domain.HistoryPrompt, before.ID, before.Hash, historyReason, current.Content); historyErr != nil {
				return fmt.Errorf("save Prompt history: %w", historyErr)
			}
		}
		var updateErr error
		value, updateErr = manager.Update(input.ID, content, input.BaseHash, input.Tags)
		return updateErr
	})
	return value, err
}

func (s *Service) RemovePrompt(id string) (domain.Prompt, error) {
	var value domain.Prompt
	err := s.withLock(func() error {
		var removeErr error
		value, removeErr = promptpkg.New(s.Store).Remove(id)
		return removeErr
	})
	return value, err
}

func (s *Service) ValidatePrompt(input PromptWriteInput) (promptpkg.Document, error) {
	content, err := input.promptContent()
	if err != nil {
		return promptpkg.Document{}, err
	}
	document, err := promptpkg.Parse([]byte(content))
	if err != nil {
		return promptpkg.Document{}, err
	}
	document.Content = ""
	return document, nil
}

func (s *Service) RenderPrompt(input PromptRenderInput) (promptpkg.RenderResult, error) {
	if input.ID == "" {
		return promptpkg.RenderResult{}, fmt.Errorf("id is required")
	}
	_, document, err := promptpkg.New(s.Store).Read(input.ID)
	if err != nil {
		return promptpkg.RenderResult{}, err
	}
	if input.Values == nil {
		input.Values = map[string]string{}
	}
	return promptpkg.Render(document, input.Values)
}
