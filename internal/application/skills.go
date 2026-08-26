package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	"github.com/zzzzzyijie/skm/internal/skill"
	"github.com/zzzzzyijie/skm/internal/tags"
)

type SkillView struct {
	domain.Skill
	Health        string `json:"health"`
	HealthDetail  string `json:"healthDetail,omitempty"`
	UsingFallback bool   `json:"usingFallback,omitempty"`
	EffectivePath string `json:"effectivePath"`
	Editable      bool   `json:"editable"`
	EditReason    string `json:"editReason,omitempty"`
}

type SkillDetails struct {
	SkillView
	Content string `json:"content"`
	Body    string `json:"body"`
}

type AddSkillInput struct {
	Path   string   `json:"path"`
	Tags   []string `json:"tags"`
	Source string   `json:"source"`
}

type UpdateSkillInput struct {
	ID       string   `json:"id"`
	Content  string   `json:"content"`
	BaseHash string   `json:"baseHash"`
	Tags     []string `json:"tags"`
}

type SkillTagsInput struct {
	Skill string   `json:"skill"`
	Tags  []string `json:"tags"`
}

type SkillUpdateResult struct {
	Skill           domain.Skill `json:"skill"`
	Plan            domain.Plan  `json:"plan"`
	Applied         bool         `json:"applied"`
	DeploymentError string       `json:"deploymentError,omitempty"`
	Warning         string       `json:"warning,omitempty"`
}

func (s *Service) ListSkills(tags []string) ([]SkillView, error) {
	values, err := catalog.New(s.Store).List(domain.LocationLibrary, tags)
	if err != nil {
		return nil, err
	}
	manager := catalog.New(s.Store)
	result := make([]SkillView, len(values))
	for index := range values {
		result[index], _ = inspectLibrarySkill(values[index], s.Store)
		result[index].Editable, result[index].EditReason, err = manager.Editability(values[index])
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (s *Service) GetSkill(id string) (SkillDetails, error) {
	manager := catalog.New(s.Store)
	value, err := manager.ResolveLibrary(id)
	if err != nil {
		return SkillDetails{}, err
	}
	view, document := inspectLibrarySkill(value, s.Store)
	view.Editable, view.EditReason, err = manager.Editability(value)
	if err != nil {
		return SkillDetails{}, err
	}
	result := SkillDetails{SkillView: view}
	if document != nil {
		result.Body = document.Body
	}
	if data, readErr := os.ReadFile(filepath.Join(view.EffectivePath, "SKILL.md")); readErr == nil && int64(len(data)) <= skill.MaxSkillMDSize {
		result.Content = string(data)
	}
	return result, nil
}

func (s *Service) AddSkill(input AddSkillInput) (domain.Skill, error) {
	if strings.TrimSpace(input.Path) == "" {
		return domain.Skill{}, fmt.Errorf("path is required")
	}
	path := input.Path
	cleanup := func() {}
	info, err := os.Stat(path)
	if err != nil {
		return domain.Skill{}, err
	}
	if !info.IsDir() {
		if !info.Mode().IsRegular() || !strings.EqualFold(filepath.Ext(path), ".zip") {
			return domain.Skill{}, fmt.Errorf("local import must be a Skill directory or .zip archive")
		}
		temporary, temporaryErr := os.MkdirTemp("", "skm-skill-import-*")
		if temporaryErr != nil {
			return domain.Skill{}, temporaryErr
		}
		cleanup = func() { _ = os.RemoveAll(temporary) }
		path, err = skill.ExtractZIP(path, temporary)
		if err != nil {
			cleanup()
			return domain.Skill{}, err
		}
	}
	defer cleanup()
	var value domain.Skill
	err = s.withLock(func() error {
		var addErr error
		value, addErr = catalog.New(s.Store).AddLocal(path, input.Source, input.Tags)
		return addErr
	})
	return value, err
}

func (s *Service) UpdateSkill(input UpdateSkillInput) (SkillUpdateResult, error) {
	if strings.TrimSpace(input.ID) == "" {
		return SkillUpdateResult{}, fmt.Errorf("id is required")
	}
	var result SkillUpdateResult
	err := s.withLock(func() error {
		config, err := s.Store.LoadConfig()
		if err != nil {
			return err
		}
		normalizedTags, err := tags.Normalize(input.Tags, config.Defaults.Tags)
		if err != nil {
			return err
		}
		manager := catalog.New(s.Store)
		before, err := manager.ResolveLibrary(input.ID)
		if err != nil {
			return err
		}
		result.Skill, err = manager.UpdateContent(input.ID, input.Content, input.BaseHash)
		if err != nil {
			return err
		}
		result.Skill, err = manager.UpdateTags(input.ID, func([]string) []string { return normalizedTags })
		if err != nil {
			return err
		}
		state, err := s.Store.LoadState()
		if err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		allSkills, err := s.Store.LoadAllSkills()
		if err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		engine := planner.New(s.Store)
		result.Plan, err = engine.BuildScoped(allSkills, state, domain.PlacementUser, "")
		if err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		if err := engine.Apply(result.Plan, &state); err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		result.Applied = true
		if before.Hash != result.Skill.Hash {
			if _, cleanupErr := s.Store.DeleteObjectIfUnreferenced(before.Hash, before.Name); cleanupErr != nil {
				result.Warning = fmt.Sprintf("updated Skill but failed to clean old snapshot: %v", cleanupErr)
			}
		}
		return nil
	})
	if result.Plan.Operations == nil {
		result.Plan.Operations = []domain.Operation{}
	}
	return result, err
}

func (s *Service) RemoveSkill(id string) (domain.Skill, error) {
	var removed domain.Skill
	err := s.withLock(func() error {
		manager := catalog.New(s.Store)
		value, err := manager.ResolveLibrary(id)
		if err != nil {
			return err
		}
		state, err := s.Store.LoadState()
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
		_, err = s.Store.DeleteObjectIfUnreferenced(removed.Hash, removed.Name)
		return err
	})
	return removed, err
}

func (s *Service) ReplaceSkillTags(input SkillTagsInput) (domain.Skill, error) {
	var value domain.Skill
	err := s.withLock(func() error {
		var tagErr error
		value, tagErr = catalog.New(s.Store).UpdateTags(input.Skill, func([]string) []string {
			return append([]string(nil), input.Tags...)
		})
		return tagErr
	})
	return value, err
}

func inspectLibrarySkill(value domain.Skill, storage interface{ ObjectPath(string, string) string }) (SkillView, *skill.Document) {
	view := SkillView{Skill: value, Health: "available", EffectivePath: value.Path}
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
			setFallback(&view, fallback)
			return view, &fallback
		}
		view.HealthDetail += "; fallback snapshot: " + fallbackErr.Error()
	} else if storage != nil {
		if objectDocument, objectErr := skill.Validate(storage.ObjectPath(value.Hash, value.Name)); objectErr == nil && objectDocument.Hash == value.Hash {
			setFallback(&view, objectDocument)
			return view, &objectDocument
		}
	}
	return view, nil
}

func setFallback(view *SkillView, document skill.Document) {
	view.EffectivePath = document.Path
	view.Hash = document.Hash
	view.Description = document.Description
	view.Metadata = document.Metadata
	view.UsingFallback = true
}
