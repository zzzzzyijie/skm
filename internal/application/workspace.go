package application

import (
	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/planner"
	gitSource "github.com/zzzzzyijie/skm/internal/source"
	"github.com/zzzzzyijie/skm/internal/workspace"
)

type WorkspaceSyncInput struct {
	Resolutions map[string]string `json:"resolutions"`
}

func (s *Service) ConfigureWorkspace(input domain.WorkspaceConfig) (WorkspaceView, error) {
	var config domain.WorkspaceConfig
	err := s.withLock(func() error {
		var err error
		config, err = workspace.New(s.Store).Configure(input)
		return err
	})
	if err != nil {
		return WorkspaceView{}, err
	}
	return WorkspaceView{Configured: true, Config: &config}, nil
}

func (s *Service) newWorkspaceManager() *workspace.Manager {
	return workspace.New(s.Store).WithSources(gitSource.NewGitManager(s.Store, catalog.New(s.Store)))
}

func (s *Service) PreviewWorkspace() (workspace.Preview, error) {
	var result workspace.Preview
	err := s.withLock(func() error {
		var err error
		result, err = s.newWorkspaceManager().Preview()
		return err
	})
	if result.Changes == nil {
		result.Changes = []workspace.Change{}
	}
	return result, err
}

func (s *Service) SyncWorkspace(resolutions map[string]string) (workspace.Result, error) {
	var result workspace.Result
	err := s.withLock(func() error {
		var err error
		result, err = s.newWorkspaceManager().ApplyResolved(resolutions)
		if err != nil {
			return err
		}
		state, loadErr := s.Store.LoadState()
		if loadErr != nil {
			result.DeploymentError = loadErr.Error()
			return nil
		}
		skills, loadErr := s.Store.LoadAllSkills()
		if loadErr != nil {
			result.DeploymentError = loadErr.Error()
			return nil
		}
		engine := planner.New(s.Store)
		plan, buildErr := engine.Build(skills, state)
		if buildErr != nil {
			result.DeploymentError = buildErr.Error()
			return nil
		}
		result.Plan = &plan
		if applyErr := engine.Apply(plan, &state); applyErr != nil {
			result.DeploymentError = applyErr.Error()
		}
		return nil
	})
	return result, err
}
