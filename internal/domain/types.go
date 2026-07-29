package domain

import "time"

const SchemaVersion = 1

type Scope string

const (
	ScopeGlobal   Scope = "global"
	ScopePersonal Scope = "personal"
	ScopeProject  Scope = "project"
)

func (s Scope) Valid() bool {
	return s == ScopeGlobal || s == ScopePersonal || s == ScopeProject
}

func (s Scope) Priority() int {
	switch s {
	case ScopeProject:
		return 3
	case ScopePersonal:
		return 2
	case ScopeGlobal:
		return 1
	default:
		return 0
	}
}

type Agent string

const (
	AgentClaude Agent = "claude"
	AgentCodex  Agent = "codex"
)

func (a Agent) Valid() bool { return a == AgentClaude || a == AgentCodex }

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
	Version  int      `yaml:"version" json:"version"`
	Defaults Defaults `yaml:"defaults" json:"defaults"`
}

type Defaults struct {
	Tags     []string `yaml:"tags" json:"tags"`
	Agents   []Agent  `yaml:"agents" json:"agents"`
	LinkMode LinkMode `yaml:"linkMode" json:"linkMode"`
}

type Skill struct {
	ID          string         `yaml:"id" json:"id"`
	Name        string         `yaml:"name" json:"name"`
	Description string         `yaml:"description" json:"description"`
	Tags        []string       `yaml:"tags" json:"tags"`
	Source      string         `yaml:"source" json:"source"`
	Scope       Scope          `yaml:"scope" json:"scope"`
	Revision    string         `yaml:"revision,omitempty" json:"revision,omitempty"`
	Hash        string         `yaml:"hash" json:"hash"`
	Path        string         `yaml:"path" json:"path"`
	ProjectRoot string         `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	Metadata    map[string]any `yaml:"metadata,omitempty" json:"metadata,omitempty"`
	AddedAt     time.Time      `yaml:"addedAt" json:"addedAt"`
}

type Catalog struct {
	Version      int                 `yaml:"version" json:"version"`
	Skills       []Skill             `yaml:"skills,omitempty" json:"skills"`
	Dependencies []ProjectDependency `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
}

type ProjectDependency struct {
	ID     string   `yaml:"id" json:"id"`
	Tags   []string `yaml:"tags" json:"tags"`
	Agents []Agent  `yaml:"agents" json:"agents"`
	Mode   LinkMode `yaml:"mode" json:"mode"`
}

type LockedSkill struct {
	ID       string   `yaml:"id" json:"id"`
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
	Name      string    `yaml:"name" json:"name"`
	URL       string    `yaml:"url" json:"url"`
	Ref       string    `yaml:"ref,omitempty" json:"ref,omitempty"`
	Paths     []string  `yaml:"paths,omitempty" json:"paths,omitempty"`
	Tags      []string  `yaml:"tags" json:"tags"`
	Scope     Scope     `yaml:"scope" json:"scope"`
	Revision  string    `yaml:"revision,omitempty" json:"revision,omitempty"`
	UpdatedAt time.Time `yaml:"updatedAt,omitempty" json:"updatedAt,omitempty"`
}

type Sources struct {
	Version int      `yaml:"version" json:"version"`
	Sources []Source `yaml:"sources" json:"sources"`
}

type Installation struct {
	SkillID     string    `yaml:"skillId" json:"skillId"`
	Name        string    `yaml:"name" json:"name"`
	Scope       Scope     `yaml:"scope" json:"scope"`
	ProjectRoot string    `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	Agents      []Agent   `yaml:"agents" json:"agents"`
	Mode        LinkMode  `yaml:"mode" json:"mode"`
	UpdatedAt   time.Time `yaml:"updatedAt" json:"updatedAt"`
}

type Deployment struct {
	SkillID     string    `yaml:"skillId" json:"skillId"`
	Name        string    `yaml:"name" json:"name"`
	Agent       Agent     `yaml:"agent" json:"agent"`
	Scope       Scope     `yaml:"scope" json:"scope"`
	ProjectRoot string    `yaml:"projectRoot,omitempty" json:"projectRoot,omitempty"`
	Target      string    `yaml:"target" json:"target"`
	SourcePath  string    `yaml:"sourcePath" json:"sourcePath"`
	Mode        LinkMode  `yaml:"mode" json:"mode"`
	Hash        string    `yaml:"hash" json:"hash"`
	UpdatedAt   time.Time `yaml:"updatedAt" json:"updatedAt"`
}

type State struct {
	Version       int            `yaml:"version" json:"version"`
	Installations []Installation `yaml:"installations" json:"installations"`
	Deployments   []Deployment   `yaml:"deployments" json:"deployments"`
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
	Scope       Scope           `json:"scope"`
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
		Defaults: Defaults{
			Tags:     []string{"general"},
			Agents:   []Agent{AgentClaude, AgentCodex},
			LinkMode: ModeAuto,
		},
	}
}
