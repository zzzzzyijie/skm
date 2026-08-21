package domain

import (
	"regexp"
	"time"
)

const SchemaVersion = 2

const PromptSchemaVersion = 1

const WorkspaceSchemaVersion = 1

type SkillLocation string

const (
	LocationLibrary SkillLocation = "library"
	LocationProject SkillLocation = "project"
)

func (l SkillLocation) Valid() bool {
	return l == LocationLibrary || l == LocationProject
}

type Placement string

const (
	PlacementUser    Placement = "user"
	PlacementProject Placement = "project"
)

func (p Placement) Valid() bool {
	return p == PlacementUser || p == PlacementProject
}

type Agent string

const (
	AgentClaude   Agent = "claude"
	AgentCodex    Agent = "codex"
	AgentCursor   Agent = "cursor"
	AgentCopilot  Agent = "copilot"
	AgentGemini   Agent = "gemini"
	AgentWindsurf Agent = "windsurf"
	AgentKiro     Agent = "kiro"
	AgentCline    Agent = "cline"
	AgentOpenCode Agent = "opencode"
	AgentTrae     Agent = "trae"
	AgentHermes   Agent = "hermes"
	AgentOpenClaw Agent = "openclaw"
)

func (a Agent) Valid() bool {
	switch a {
	case AgentClaude, AgentCodex, AgentCursor, AgentCopilot, AgentGemini,
		AgentWindsurf, AgentKiro, AgentCline, AgentOpenCode, AgentTrae,
		AgentHermes, AgentOpenClaw:
		return true
	default:
		return false
	}
}

var projectAgentIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ProjectValid permits a scanned project Agent directory name without
// permitting path separators or relative path components.
func (a Agent) ProjectValid() bool {
	return projectAgentIDPattern.MatchString(string(a))
}

type LinkMode string

const (
	ModeAuto    LinkMode = "auto"
	ModeSymlink LinkMode = "symlink"
	ModeCopy    LinkMode = "copy"
)

func (m LinkMode) Valid() bool {
	return m == ModeAuto || m == ModeSymlink || m == ModeCopy
}

func (m LinkMode) Effective() LinkMode {
	if m == ModeAuto || m == "" {
		return ModeSymlink
	}
	return m
}

type Config struct {
	Version  int               `yaml:"version" json:"version"`
	Defaults Defaults          `yaml:"defaults" json:"defaults"`
	Tags     []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	Agents   []AgentDefinition `yaml:"agents,omitempty" json:"agents,omitempty"`
}

type AgentDefinition struct {
	ID         Agent  `yaml:"id" json:"id"`
	Name       string `yaml:"name" json:"name"`
	SkillsPath string `yaml:"skillsPath" json:"skillsPath"`
	Icon       string `yaml:"icon,omitempty" json:"icon,omitempty"`
}

type Defaults struct {
	Tags     []string `yaml:"tags" json:"tags"`
	Agents   []Agent  `yaml:"agents" json:"agents"`
	LinkMode LinkMode `yaml:"linkMode" json:"linkMode"`
}

type Skill struct {
	ID           string         `yaml:"id" json:"id"`
	Name         string         `yaml:"name" json:"name"`
	Description  string         `yaml:"description" json:"description"`
	Tags         []string       `yaml:"tags" json:"tags"`
	Source       string         `yaml:"source" json:"source"`
	Location     SkillLocation  `yaml:"location" json:"location"`
	Revision     string         `yaml:"revision,omitempty" json:"revision,omitempty"`
	Hash         string         `yaml:"hash" json:"hash"`
	Path         string         `yaml:"path" json:"path"`
	SourcePath   string         `yaml:"sourcePath,omitempty" json:"sourcePath,omitempty"`
	SnapshotPath string         `yaml:"snapshotPath,omitempty" json:"snapshotPath,omitempty"`
	ProjectRoot  string         `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	ProjectPath  string         `yaml:"projectPath,omitempty" json:"projectPath,omitempty"`
	ProjectAgent Agent          `yaml:"projectAgent,omitempty" json:"projectAgent,omitempty"`
	ForkedFrom   string         `yaml:"forkedFrom,omitempty" json:"forkedFrom,omitempty"`
	ForkedAt     string         `yaml:"forkedRevision,omitempty" json:"forkedRevision,omitempty"`
	Agents       []Agent        `yaml:"agents,omitempty" json:"agents,omitempty"`
	Mode         LinkMode       `yaml:"mode,omitempty" json:"mode,omitempty"`
	Metadata     map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AddedAt      time.Time      `yaml:"addedAt" json:"addedAt"`
	LegacyScope  string         `yaml:"scope,omitempty" json:"-"`
}

type Catalog struct {
	Version      int                 `yaml:"version" json:"version"`
	Skills       []Skill             `yaml:"skills,omitempty" json:"skills"`
	Dependencies []ProjectDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

type ProjectDependency struct {
	ID         string   `yaml:"id" json:"id"`
	Name       string   `yaml:"name" json:"name"`
	Source     string   `yaml:"source" json:"source"`
	URL        string   `yaml:"url" json:"url"`
	Ref        string   `yaml:"ref,omitempty" json:"ref,omitempty"`
	SourcePath string   `yaml:"sourcePath,omitempty" json:"sourcePath,omitempty"`
	Revision   string   `yaml:"revision" json:"revision"`
	Hash       string   `yaml:"hash" json:"hash"`
	Tags       []string `yaml:"tags" json:"tags"`
	Agents     []Agent  `yaml:"agents" json:"agents"`
	Mode       LinkMode `yaml:"mode" json:"mode"`
}

type LockedSkill struct {
	ID       string   `yaml:"id" json:"id"`
	Name     string   `yaml:"name" json:"name"`
	Source   string   `yaml:"source" json:"source"`
	Revision string   `yaml:"revision,omitempty" json:"revision,omitempty"`
	Hash     string   `yaml:"hash" json:"hash"`
	Tags     []string `yaml:"tags" json:"tags"`
}

type LockFile struct {
	Version int           `yaml:"version" json:"version"`
	Skills  []LockedSkill `yaml:"skills" json:"skills"`
}

type Source struct {
	Name        string    `yaml:"name" json:"name"`
	URL         string    `yaml:"url" json:"url"`
	Ref         string    `yaml:"ref,omitempty" json:"ref,omitempty"`
	Paths       []string  `yaml:"paths,omitempty" json:"paths,omitempty"`
	Tags        []string  `yaml:"tags" json:"tags"`
	Revision    string    `yaml:"revision,omitempty" json:"revision,omitempty"`
	UpdatedAt   time.Time `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
	LegacyScope string    `yaml:"scope,omitempty" json:"-"`
}

type Sources struct {
	Version int      `yaml:"version" json:"version"`
	Sources []Source `yaml:"sources" json:"sources"`
}

// WorkspaceConfig identifies the one personal Git repository used to carry
// portable Skill and Prompt content between devices. Credentials deliberately
// remain in the device's Git/SSH configuration rather than this file.
type WorkspaceConfig struct {
	Version   int       `yaml:"version" json:"version"`
	URL       string    `yaml:"url" json:"url"`
	Ref       string    `yaml:"ref" json:"ref"`
	Root      string    `yaml:"root,omitempty" json:"root,omitempty"`
	UpdatedAt time.Time `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type WorkspaceState struct {
	Version      int               `yaml:"version" json:"version"`
	Revision     string            `yaml:"revision,omitempty" json:"revision,omitempty"`
	SkillBases   map[string]string `yaml:"skillBases,omitempty" json:"skillBases,omitempty"`
	PromptBases  map[string]string `yaml:"promptBases,omitempty" json:"promptBases,omitempty"`
	SourceBases  map[string]string `yaml:"sourceBases,omitempty" json:"sourceBases,omitempty"`
	LastSyncedAt time.Time         `yaml:"lastSyncedAt,omitempty" json:"lastSyncedAt,omitempty"`
}

type WorkspaceEntry struct {
	ID   string   `yaml:"id" json:"id"`
	Path string   `yaml:"path" json:"path"`
	Hash string   `yaml:"hash" json:"hash"`
	Tags []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// WorkspaceSource is the portable form of a Git source binding carried in the
// personal workspace. Revision and UpdatedAt stay device-local because they
// only describe the local checkout cache.
type WorkspaceSource struct {
	Name  string   `yaml:"name" json:"name"`
	URL   string   `yaml:"url" json:"url"`
	Ref   string   `yaml:"ref,omitempty" json:"ref,omitempty"`
	Paths []string `yaml:"paths,omitempty" json:"paths,omitempty"`
	Tags  []string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

type WorkspaceManifest struct {
	Version int               `yaml:"version" json:"version"`
	Skills  []WorkspaceEntry  `yaml:"skills,omitempty" json:"skills"`
	Prompts []WorkspaceEntry  `yaml:"prompts,omitempty" json:"prompts"`
	Sources []WorkspaceSource `yaml:"sources,omitempty" json:"sources,omitempty"`
}

type PromptVariable struct {
	Name        string   `yaml:"name" json:"name"`
	Label       string   `yaml:"label,omitempty" json:"label,omitempty"`
	Type        string   `yaml:"type,omitempty" json:"type,omitempty"`
	Required    bool     `yaml:"required,omitempty" json:"required,omitempty"`
	Default     string   `yaml:"default,omitempty" json:"default,omitempty"`
	Options     []string `yaml:"options,omitempty" json:"options,omitempty"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
}

type Prompt struct {
	ID          string           `yaml:"id" json:"id"`
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Tags        []string         `yaml:"tags" json:"tags"`
	Source      string           `yaml:"source" json:"source"`
	Hash        string           `yaml:"hash" json:"hash"`
	Path        string           `yaml:"path" json:"path"`
	Variables   []PromptVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
	AddedAt     time.Time        `yaml:"addedAt" json:"addedAt"`
	UpdatedAt   time.Time        `yaml:"updatedAt" json:"updatedAt"`
}

type PromptCatalog struct {
	Version int      `yaml:"version" json:"version"`
	Prompts []Prompt `yaml:"prompts,omitempty" json:"prompts"`
}

type Project struct {
	ID        string    `yaml:"id" json:"id"`
	Path      string    `yaml:"path" json:"path"`
	AddedAt   time.Time `yaml:"addedAt" json:"addedAt"`
	UpdatedAt time.Time `yaml:"updatedAt" json:"updatedAt"`
}

type Projects struct {
	Version  int       `yaml:"version" json:"version"`
	Projects []Project `yaml:"projects" json:"projects"`
}

type Activation struct {
	SkillID     string    `yaml:"skillId" json:"skillId"`
	Name        string    `yaml:"name" json:"name"`
	Placement   Placement `yaml:"placement" json:"placement"`
	ProjectRoot string    `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	Agents      []Agent   `yaml:"agents" json:"agents"`
	Mode        LinkMode  `yaml:"mode" json:"mode"`
	PinnedHash  string    `yaml:"pinnedHash,omitempty" json:"pinnedHash,omitempty"`
	PinnedPath  string    `yaml:"pinnedPath,omitempty" json:"pinnedPath,omitempty"`
	UpdatedAt   time.Time `yaml:"updatedAt" json:"updatedAt"`
}

type LegacyInstallation struct {
	SkillID     string    `yaml:"skillId"`
	Name        string    `yaml:"name"`
	Scope       string    `yaml:"scope"`
	ProjectRoot string    `yaml:"projectRoot,omitempty"`
	Agents      []Agent   `yaml:"agents"`
	Mode        LinkMode  `yaml:"mode"`
	UpdatedAt   time.Time `yaml:"updatedAt"`
}

type Deployment struct {
	SkillID     string    `yaml:"skillId" json:"skillId"`
	Name        string    `yaml:"name" json:"name"`
	Agent       Agent     `yaml:"agent" json:"agent"`
	Placement   Placement `yaml:"placement" json:"placement"`
	ProjectRoot string    `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	Target      string    `yaml:"target" json:"target"`
	SourcePath  string    `yaml:"sourcePath" json:"sourcePath"`
	Mode        LinkMode  `yaml:"mode" json:"mode"`
	Hash        string    `yaml:"hash" json:"hash"`
	UpdatedAt   time.Time `yaml:"updatedAt" json:"updatedAt"`
	LegacyScope string    `yaml:"scope,omitempty" json:"-"`
}

type State struct {
	Version             int                  `yaml:"version" json:"version"`
	Activations         []Activation         `yaml:"activations,omitempty" json:"activations"`
	Deployments         []Deployment         `yaml:"deployments,omitempty" json:"deployments"`
	LegacyInstallations []LegacyInstallation `yaml:"installations,omitempty" json:"-"`
}

type OperationStatus string

const (
	StatusCreate            OperationStatus = "create"
	StatusUnchanged         OperationStatus = "unchanged"
	StatusReplaceManaged    OperationStatus = "replace-managed"
	StatusConflictUnmanaged OperationStatus = "conflict-unmanaged"
	StatusBroken            OperationStatus = "broken"
)

type Operation struct {
	Status      OperationStatus `json:"status"`
	SkillID     string          `json:"skillId"`
	Name        string          `json:"name"`
	Agent       Agent           `json:"agent"`
	Placement   Placement       `json:"placement"`
	ProjectRoot string          `json:"projectRoot,omitempty"`
	Target      string          `json:"target"`
	SourcePath  string          `json:"sourcePath"`
	Mode        LinkMode        `json:"mode"`
	Hash        string          `json:"hash"`
	Message     string          `json:"message,omitempty"`
}

type Plan struct {
	Digest     string      `json:"digest"`
	Operations []Operation `json:"operations"`
}

func DefaultConfig() Config {
	return Config{
		Version: SchemaVersion,
		Tags:    []string{"general"},
		Defaults: Defaults{
			Tags:     []string{"general"},
			Agents:   []Agent{AgentClaude, AgentCodex},
			LinkMode: ModeAuto,
		},
	}
}
