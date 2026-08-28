package application

import (
	"fmt"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
)

type UpdateSourcesInput struct {
	Names []string `json:"names"`
}

type SourceUpdateResult struct {
	Sources []domain.Source `json:"sources"`
	Skills  []domain.Skill  `json:"skills"`
}

type SourceSyncItem struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	Source     *domain.Source `json:"source,omitempty"`
	SkillCount int            `json:"skillCount"`
	Error      string         `json:"error,omitempty"`
}

type SourceSyncResult struct {
	Configured      int              `json:"configured"`
	Updated         int              `json:"updated"`
	Failed          int              `json:"failed"`
	SkillCount      int              `json:"skillCount"`
	Results         []SourceSyncItem `json:"results"`
	Plan            *domain.Plan     `json:"plan,omitempty"`
	Applied         bool             `json:"applied"`
	DeploymentError string           `json:"deploymentError,omitempty"`
	SyncedAt        time.Time        `json:"syncedAt"`
}

func (s *Service) UpdateSources(names []string) (SourceUpdateResult, error) {
	result := SourceUpdateResult{Sources: []domain.Source{}, Skills: []domain.Skill{}}
	err := s.withLock(func() error {
		var err error
		result.Sources, result.Skills, err = gitSource.NewGitManager(s.Store, catalog.New(s.Store)).Update(names)
		if result.Sources == nil {
			result.Sources = []domain.Source{}
		}
		if result.Skills == nil {
			result.Skills = []domain.Skill{}
		}
		return err
	})
	return result, err
}

func (s *Service) RemoveSource(name string) (gitSource.RemovalResult, error) {
	var result gitSource.RemovalResult
	err := s.withLock(func() error {
		var err error
		result, err = gitSource.NewGitManager(s.Store, catalog.New(s.Store)).Remove(name)
		return err
	})
	return result, err
}

func (s *Service) SyncSources() (SourceSyncResult, error) {
	result := SourceSyncResult{Results: []SourceSyncItem{}, SyncedAt: time.Now().UTC()}
	err := s.withLock(func() error {
		configured, err := s.Store.LoadSources()
		if err != nil {
			return err
		}
		result.Configured = len(configured.Sources)
		if result.Configured == 0 {
			return fmt.Errorf("no Git sources configured")
		}

		manager := gitSource.NewGitManager(s.Store, catalog.New(s.Store))
		for _, configuredSource := range configured.Sources {
			updated, skills, updateErr := manager.Update([]string{configuredSource.Name})
			item := SourceSyncItem{Name: configuredSource.Name}
			if updateErr != nil {
				item.Status = "error"
				item.Error = updateErr.Error()
				result.Failed++
				result.Results = append(result.Results, item)
				continue
			}
			item.Status = "updated"
			item.SkillCount = len(skills)
			if len(updated) > 0 {
				item.Source = &updated[0]
			}
			result.Updated++
			result.SkillCount += len(skills)
			result.Results = append(result.Results, item)
		}
		if result.Updated == 0 {
			return nil
		}

		state, err := s.Store.LoadState()
		if err != nil {
			return err
		}
		skills, err := s.Store.LoadAllSkills()
		if err != nil {
			return err
		}
		engine := planner.New(s.Store)
		plan, err := engine.Build(skills, state)
		if err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		result.Plan = &plan
		if err := engine.Apply(plan, &state); err != nil {
			result.DeploymentError = err.Error()
			return nil
		}
		result.Applied = true
		return nil
	})
	return result, err
}
