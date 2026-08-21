package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/tags"
)

type librarySkillDetails struct {
	librarySkillView
	Body string `json:"body"`
}

type librarySkillView struct {
	domain.Skill
	Health        string `json:"health"`
	HealthDetail  string `json:"healthDetail,omitempty"`
	UsingFallback bool   `json:"usingFallback,omitempty"`
	EffectivePath string `json:"effectivePath"`
}

func (s *Server) handleListSkills(w http.ResponseWriter, r *http.Request) {
	tagValues := r.URL.Query()["tag"]
	skills, err := catalog.New(s.store).List(domain.LocationLibrary, tagValues)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	result := make([]librarySkillView, len(skills))
	for index := range skills {
		result[index], _ = inspectLibrarySkill(skills[index])
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleShowSkill(w http.ResponseWriter, r *http.Request) {
	id := splitSkillID(r)
	value, err := catalog.New(s.store).Resolve(id)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	view, document := inspectLibrarySkill(value)
	body := ""
	if document != nil {
		body = document.Body
	}
	writeJSON(w, http.StatusOK, librarySkillDetails{librarySkillView: view, Body: body})
}

func inspectLibrarySkill(value domain.Skill) (librarySkillView, *skill.Document) {
	view := librarySkillView{Skill: value, Health: "available", EffectivePath: value.Path}
	document, err := skill.Validate(value.Path)
	if err == nil {
		if value.Mode == domain.ModeSymlink && value.ProjectRoot != "" && document.Hash != value.Hash {
			view.Health = "changed"
		}
		view.Hash = document.Hash
		view.Description = document.Description
		view.Metadata = document.Metadata
		return view, &document
	}
	if value.Mode != domain.ModeSymlink || value.ProjectRoot == "" {
		view.Health = "invalid"
		view.HealthDetail = err.Error()
		return view, nil
	}
	if _, rootErr := os.Stat(value.ProjectRoot); rootErr != nil {
		view.Health = "unreachable"
	} else if _, sourceErr := os.Stat(value.Path); os.IsNotExist(sourceErr) {
		view.Health = "missing"
	} else {
		view.Health = "invalid"
	}
	view.HealthDetail = err.Error()
	if value.SnapshotPath != "" {
		fallback, fallbackErr := skill.Validate(value.SnapshotPath)
		if fallbackErr == nil {
			view.EffectivePath = value.SnapshotPath
			view.Hash = fallback.Hash
			view.Description = fallback.Description
			view.Metadata = fallback.Metadata
			view.UsingFallback = true
			return view, &fallback
		}
		view.HealthDetail += "; fallback snapshot: " + fallbackErr.Error()
	}
	return view, nil
}

func (s *Server) handleDetachSkill(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skill string `json:"skill"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var detached domain.Skill
	var plan domain.Plan
	err := s.withLock(func() error {
		var err error
		detached, err = catalog.New(s.store).DetachProjectLink(body.Skill)
		if err != nil {
			return err
		}
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		skills, err := s.store.LoadAllSkills()
		if err != nil {
			return err
		}
		engine := planner.New(s.store)
		plan, err = engine.Build(skills, state)
		if err != nil {
			return err
		}
		return engine.Apply(plan, &state)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"skill": detached, "plan": plan})
}

func (s *Server) handleAddSkill(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path   string   `json:"path"`
		Tags   []string `json:"tags"`
		Source string   `json:"source"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if body.Path == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("path is required"))
		return
	}
	path := body.Path
	cleanup := func() {}
	info, statErr := os.Stat(path)
	if statErr != nil {
		writeError(w, http.StatusBadRequest, statErr)
		return
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(path), ".zip") {
			writeError(w, http.StatusBadRequest, fmt.Errorf("local import must be a Skill directory or .zip archive"))
			return
		}
		temporary, err := os.MkdirTemp("", "skm-skill-import-*")
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		cleanup = func() { _ = os.RemoveAll(temporary) }
		path, err = skill.ExtractZIP(path, temporary)
		if err != nil {
			cleanup()
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	defer cleanup()
	var value domain.Skill
	err := s.withLock(func() error {
		var addErr error
		value, addErr = catalog.New(s.store).AddLocal(path, body.Source, body.Tags)
		return addErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) handleRemoveSkill(w http.ResponseWriter, r *http.Request) {
	id := splitSkillID(r)
	var removed domain.Skill
	err := s.withLock(func() error {
		manager := catalog.New(s.store)
		value, err := manager.ResolveLibrary(id)
		if err != nil {
			return err
		}
		state, err := s.store.LoadState()
		if err != nil {
			return err
		}
		for _, activation := range state.Activations {
			if activation.SkillID != value.ID {
				continue
			}
			if activation.Placement == domain.PlacementProject {
				return fmt.Errorf("skill %s is enabled by project %s", value.ID, activation.ProjectRoot)
			}
			return fmt.Errorf("skill %s is enabled; disable it first", value.ID)
		}
		removed, err = manager.Remove(value.ID)
		if err != nil {
			return err
		}
		_, err = s.store.DeleteObjectIfUnreferenced(removed.Hash, removed.Name)
		return err
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, removed)
}

type tagCount struct {
	Name        string `json:"name"`
	Count       int    `json:"count"`
	SkillCount  int    `json:"skillCount"`
	PromptCount int    `json:"promptCount"`
	Default     bool   `json:"default"`
}

func (s *Server) handleListTags(w http.ResponseWriter, r *http.Request) {
	config, err := s.store.LoadConfig()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	library, err := s.store.LoadCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	prompts, err := s.store.LoadPromptCatalog()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	counts := make(map[string]*tagCount)
	ensure := func(name string) *tagCount {
		if counts[name] == nil {
			counts[name] = &tagCount{Name: name}
		}
		return counts[name]
	}
	for _, name := range append(append([]string(nil), config.Tags...), config.Defaults.Tags...) {
		ensure(name)
	}
	for _, name := range config.Defaults.Tags {
		ensure(name).Default = true
	}
	for _, value := range library.Skills {
		for _, tag := range value.Tags {
			counts[tag] = ensure(tag)
			counts[tag].SkillCount++
			counts[tag].Count++
		}
	}
	for _, value := range prompts.Prompts {
		for _, tag := range value.Tags {
			counts[tag] = ensure(tag)
			counts[tag].PromptCount++
			counts[tag].Count++
		}
	}
	result := make([]tagCount, 0, len(counts))
	for _, count := range counts {
		result = append(result, *count)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCreateTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	values, err := tags.Normalize([]string{body.Name}, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := values[0]
	err = s.withLock(func() error {
		config, loadErr := s.store.LoadConfig()
		if loadErr != nil {
			return loadErr
		}
		config.Tags, loadErr = tags.Normalize(append(config.Tags, name), config.Defaults.Tags)
		if loadErr != nil {
			return loadErr
		}
		return s.store.SaveConfig(config)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, tagCount{Name: name})
}

func (s *Server) handleDeleteTag(w http.ResponseWriter, r *http.Request) {
	name := strings.ToLower(strings.TrimSpace(r.PathValue("name")))
	var deleted bool
	err := s.withLock(func() error {
		config, err := s.store.LoadConfig()
		if err != nil {
			return err
		}
		for _, value := range config.Defaults.Tags {
			if value == name {
				return fmt.Errorf("default tag %q cannot be deleted", name)
			}
		}
		library, err := s.store.LoadCatalog()
		if err != nil {
			return err
		}
		for _, value := range library.Skills {
			if containsTag(value.Tags, name) {
				return fmt.Errorf("tag %q is still used by a Skill", name)
			}
		}
		prompts, err := s.store.LoadPromptCatalog()
		if err != nil {
			return err
		}
		for _, value := range prompts.Prompts {
			if containsTag(value.Tags, name) {
				return fmt.Errorf("tag %q is still used by a Prompt", name)
			}
		}
		result := config.Tags[:0]
		for _, value := range config.Tags {
			if value == name {
				deleted = true
				continue
			}
			result = append(result, value)
		}
		if !deleted {
			return fmt.Errorf("tag %q not found", name)
		}
		config.Tags = result
		return s.store.SaveConfig(config)
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"name": name, "deleted": deleted})
}

func (s *Server) handleReplaceSkillTags(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skill string   `json:"skill"`
		Tags  []string `json:"tags"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.Skill
	err := s.withLock(func() error {
		var tagErr error
		value, tagErr = catalog.New(s.store).UpdateTags(body.Skill, func([]string) []string {
			return append([]string(nil), body.Tags...)
		})
		return tagErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleAddTags(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skill string   `json:"skill"`
		Tags  []string `json:"tags"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.Skill
	err := s.withLock(func() error {
		var tagErr error
		value, tagErr = catalog.New(s.store).UpdateTags(body.Skill, func(current []string) []string {
			return append(current, body.Tags...)
		})
		return tagErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleRemoveTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Skill string `json:"skill"`
		Tag   string `json:"tag"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var value domain.Skill
	err := s.withLock(func() error {
		var tagErr error
		value, tagErr = catalog.New(s.store).UpdateTags(body.Skill, func(current []string) []string {
			result := current[:0]
			for _, t := range current {
				if t != body.Tag {
					result = append(result, t)
				}
			}
			return result
		})
		return tagErr
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) handleRenameTag(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Old string `json:"old"`
		New string `json:"new"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	validated, err := tags.Normalize([]string{body.New}, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	oldName := strings.ToLower(body.Old)
	newName := validated[0]
	skillsChanged := 0
	promptsChanged := 0
	registryChanged := false
	err = s.withLock(func() error {
		config, err := s.store.LoadConfig()
		if err != nil {
			return err
		}
		library, err := s.store.LoadCatalog()
		if err != nil {
			return err
		}
		for _, value := range library.Skills {
			found := false
			for i, tag := range value.Tags {
				if tag == oldName {
					value.Tags[i] = newName
					found = true
				}
			}
			if !found {
				continue
			}
			value.Tags, err = tags.Normalize(value.Tags, nil)
			if err != nil {
				return err
			}
			if err := s.store.UpsertSkill(value); err != nil {
				return err
			}
			skillsChanged++
		}
		promptCatalog, err := s.store.LoadPromptCatalog()
		if err != nil {
			return err
		}
		manager := promptpkg.New(s.store)
		for _, value := range promptCatalog.Prompts {
			if !containsTag(value.Tags, oldName) {
				continue
			}
			_, document, readErr := manager.Read(value.ID)
			if readErr != nil {
				return readErr
			}
			nextTags := replaceTag(value.Tags, oldName, newName)
			content, buildErr := promptpkg.Build(document.Name, document.Description, document.Body, nextTags, document.Variables)
			if buildErr != nil {
				return buildErr
			}
			if _, updateErr := manager.Update(value.ID, string(content), value.Hash, nextTags); updateErr != nil {
				return updateErr
			}
			promptsChanged++
		}
		config.Tags, registryChanged = replaceConfiguredTag(config.Tags, oldName, newName)
		var defaultsChanged bool
		config.Defaults.Tags, defaultsChanged = replaceConfiguredTag(config.Defaults.Tags, oldName, newName)
		registryChanged = registryChanged || defaultsChanged
		if skillsChanged > 0 || promptsChanged > 0 {
			config.Tags, err = tags.Normalize(append(config.Tags, newName), config.Defaults.Tags)
			if err != nil {
				return err
			}
			registryChanged = true
		}
		if registryChanged {
			if err := s.store.SaveConfig(config); err != nil {
				return err
			}
		}
		if skillsChanged == 0 && promptsChanged == 0 && !registryChanged {
			return fmt.Errorf("tag %q not found", oldName)
		}
		return nil
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"old": oldName, "new": newName,
		"skillsChanged": skillsChanged, "promptsChanged": promptsChanged,
	})
}

func containsTag(values []string, name string) bool {
	for _, value := range values {
		if value == name {
			return true
		}
	}
	return false
}

func replaceTag(values []string, oldName, newName string) []string {
	result := append([]string(nil), values...)
	for index, value := range result {
		if value == oldName {
			result[index] = newName
		}
	}
	return result
}

func replaceConfiguredTag(values []string, oldName, newName string) ([]string, bool) {
	changed := containsTag(values, oldName)
	if !changed {
		return values, false
	}
	result, err := tags.Normalize(replaceTag(values, oldName, newName), nil)
	if err != nil {
		return values, false
	}
	return result, true
}
