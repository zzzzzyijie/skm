package source

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseInstallInput(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantURL    string
		wantName   string
		wantSkills []string
	}{
		{
			name:     "GitHub shorthand",
			input:    "jakubkrehel/skills",
			wantURL:  "https://github.com/jakubkrehel/skills.git",
			wantName: "jakubkrehel-skills",
		},
		{
			name:     "copy pasted command",
			input:    "npx skills add jakubkrehel/skills",
			wantURL:  "https://github.com/jakubkrehel/skills.git",
			wantName: "jakubkrehel-skills",
		},
		{
			name:       "prompt package version and selected skills",
			input:      `$ npx -y skills@latest add jakubkrehel/skills --skill better-ui -s "better-colors"`,
			wantURL:    "https://github.com/jakubkrehel/skills.git",
			wantName:   "jakubkrehel-skills",
			wantSkills: []string{"better-ui", "better-colors"},
		},
		{
			name:     "all skills option",
			input:    "npx skills add jakubkrehel/skills --skill '*'",
			wantURL:  "https://github.com/jakubkrehel/skills.git",
			wantName: "jakubkrehel-skills",
		},
		{
			name:     "GitHub URL",
			input:    "https://github.com/vercel-labs/agent-skills.git",
			wantURL:  "https://github.com/vercel-labs/agent-skills.git",
			wantName: "vercel-labs-agent-skills",
		},
		{
			name:     "SSH URL",
			input:    "git@github.com:vercel-labs/agent-skills.git",
			wantURL:  "git@github.com:vercel-labs/agent-skills.git",
			wantName: "vercel-labs-agent-skills",
		},
		{
			name:     "quoted local repository",
			input:    `npx skills add "/tmp/Skill sources/team repo"`,
			wantURL:  "/tmp/Skill sources/team repo",
			wantName: "team-repo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseInstallInput(test.input)
			if err != nil {
				t.Fatal(err)
			}
			if got.URL != test.wantURL || got.SuggestedName != test.wantName || !reflect.DeepEqual(got.RequestedSkills, test.wantSkills) {
				t.Fatalf("ParseInstallInput() = %#v", got)
			}
		})
	}
}

func TestParseInstallInputRejectsUnsupportedCommands(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "source input is required"},
		{"npx cowsay add owner/repo", "expected skills"},
		{"npx skills remove owner/repo", "only npx skills add"},
		{"npx skills add owner/repo --agent codex", "option --agent"},
		{"npx skills add owner/repo; touch /tmp/example", "shell operators"},
		{`npx skills add "owner/repo`, "unterminated quote"},
	}
	for _, test := range tests {
		_, err := ParseInstallInput(test.input)
		if err == nil || !strings.Contains(err.Error(), test.want) {
			t.Fatalf("ParseInstallInput(%q) error = %v, want %q", test.input, err, test.want)
		}
	}
}
