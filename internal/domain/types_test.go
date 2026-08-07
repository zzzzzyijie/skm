package domain

import "testing"

func TestLocationAndPlacementValidation(t *testing.T) {
	if !LocationLibrary.Valid() || !LocationProject.Valid() || SkillLocation("bad").Valid() {
		t.Fatal("location validation is incorrect")
	}
	if !PlacementUser.Valid() || !PlacementProject.Valid() || Placement("bad").Valid() {
		t.Fatal("placement validation is incorrect")
	}
}

func TestAgentAndModeValidation(t *testing.T) {
	agents := []Agent{
		AgentClaude, AgentCodex, AgentCursor, AgentCopilot, AgentGemini,
		AgentWindsurf, AgentKiro, AgentCline, AgentOpenCode, AgentTrae,
		AgentHermes, AgentOpenClaw,
	}
	for _, agent := range agents {
		if !agent.Valid() {
			t.Fatalf("agent %q should be valid", agent)
		}
	}
	if Agent("bad").Valid() {
		t.Fatal("unknown agent should be invalid")
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
