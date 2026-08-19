package server

import (
	"errors"
	"net/http"

	"github.com/zzzzzyijie/skm/internal/domain"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
)

type promptDetails struct {
	domain.Prompt
	Content string `json:"content"`
	Body    string `json:"body"`
}

type promptWriteRequest struct {
	Content     string                  `json:"content"`
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Tags        []string                `json:"tags"`
	Body        string                  `json:"body"`
	Variables   []domain.PromptVariable `json:"variables"`
	Source      string                  `json:"source"`
	BaseHash    string                  `json:"baseHash"`
}

func (body promptWriteRequest) promptContent() (string, error) {
	if body.Content != "" {
		return body.Content, nil
	}
	data, err := promptpkg.Build(body.Name, body.Description, body.Body, body.Tags, body.Variables)
	return string(data), err
}

func (s *Server) handleListPrompts(w http.ResponseWriter, r *http.Request) {
	values, err := promptpkg.New(s.store).List(r.URL.Query()["tag"])
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, values)
}

func (s *Server) handleShowPrompt(w http.ResponseWriter, r *http.Request) {
	value, document, err := promptpkg.New(s.store).Read(splitPromptID(r))
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, promptDetails{Prompt: value, Content: document.Content, Body: document.Body})
}

func (s *Server) handleCreatePrompt(w http.ResponseWriter, r *http.Request) {
	var body promptWriteRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	content, err := body.promptContent()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.Prompt
	err = s.withLock(func() error {
		var createErr error
		value, createErr = promptpkg.New(s.store).Create(content, body.Source, body.Tags)
		return createErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) handleUpdatePrompt(w http.ResponseWriter, r *http.Request) {
	var body promptWriteRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	content, err := body.promptContent()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.Prompt
	err = s.withLock(func() error {
		var updateErr error
		value, updateErr = promptpkg.New(s.store).Update(splitPromptID(r), content, body.BaseHash, body.Tags)
		return updateErr
	})
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, promptpkg.ErrEditConflict) {
			status = http.StatusConflict
		}
		writeError(w, status, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleRemovePrompt(w http.ResponseWriter, r *http.Request) {
	var value domain.Prompt
	err := s.withLock(func() error {
		var removeErr error
		value, removeErr = promptpkg.New(s.store).Remove(splitPromptID(r))
		return removeErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleValidatePrompt(w http.ResponseWriter, r *http.Request) {
	var body promptWriteRequest
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	content, err := body.promptContent()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	document, err := promptpkg.Parse([]byte(content))
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	document.Content = ""
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) handleRenderPrompt(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Prompt    string            `json:"prompt"`
		Content   string            `json:"content"`
		Variables map[string]string `json:"variables"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var document promptpkg.Document
	var err error
	if body.Content != "" {
		document, err = promptpkg.Parse([]byte(body.Content))
	} else {
		_, document, err = promptpkg.New(s.store).Read(body.Prompt)
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	result, err := promptpkg.Render(document, body.Variables)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func splitPromptID(r *http.Request) string {
	return splitSkillID(r)
}
