package domain

import "testing"

func TestScopePriorityAndValidation(t *testing.T) {
	if !ScopeGlobal.Valid() || !ScopePersonal.Valid() || !ScopeProject.Valid() || Scope("bad").Valid() {
		t.Fatal("scope validation is incorrect")
	}
	if !(ScopeProject.Priority() > ScopePersonal.Priority() && ScopePersonal.Priority() > ScopeGlobal.Priority()) {
		t.Fatal("scope priority is incorrect")
	}
}

func TestAgentAndModeValidation(t *testing.T) {
	if !AgentClaude.Valid() || !AgentCodex.Valid() || Agent("cursor").Valid() {
		t.Fatal("agent validation is incorrect")
	}
	if !ModeAuto.Valid() || !ModeSymlink.Valid() || !ModeCopy.Valid() || LinkMode("bad").Valid() {
		t.Fatal("mode validation is incorrect")
	}
	if ModeAuto.Effective() != ModeSymlink {
		t.Fatal("auto mode should currently resolve to symlink")
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.Version != SchemaVersion || len(config.Defaults.Tags) != 1 || len(config.Defaults.Agents) != 2 {
		t.Fatalf("default config = %#v", config)
	}
}
