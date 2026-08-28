package application

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/catalog"
	"github.com/zzzzzyijie/skm/internal/domain"
	promptpkg "github.com/zzzzzyijie/skm/internal/prompt"
)

type HistoryInput struct {
	Kind   domain.HistoryKind `json:"kind"`
	ItemID string             `json:"itemId"`
}

type HistoryEntryInput struct {
	Kind    domain.HistoryKind `json:"kind"`
	ItemID  string             `json:"itemId"`
	EntryID string             `json:"entryId"`
}

type HistoryDiffInput struct {
	Kind   domain.HistoryKind `json:"kind"`
	ItemID string             `json:"itemId"`
	From   string             `json:"from"`
	To     string             `json:"to"`
}

type HistoryDiff struct {
	From string `json:"from"`
	To   string `json:"to"`
	Diff string `json:"diff"`
}

type HistoryRollbackResult struct {
	Entry  domain.HistoryEntry `json:"entry"`
	Skill  *SkillUpdateResult  `json:"skill,omitempty"`
	Prompt *domain.Prompt      `json:"prompt,omitempty"`
}

func (s *Service) ListHistory(input HistoryInput) ([]domain.HistoryEntry, error) {
	current, err := s.currentHistory(input.Kind, input.ItemID)
	if err != nil {
		return nil, err
	}
	entries, err := s.Store.ListHistory(input.Kind, input.ItemID)
	if err != nil {
		return nil, err
	}
	result := []domain.HistoryEntry{withoutHistoryContent(current)}
	for _, entry := range entries {
		if entry.Hash != current.Hash {
			result = append(result, entry)
		}
	}
	return result, nil
}

func (s *Service) GetHistory(input HistoryEntryInput) (domain.HistoryEntry, error) {
	if input.EntryID == "current" {
		return s.currentHistory(input.Kind, input.ItemID)
	}
	return s.Store.ReadHistory(input.Kind, input.ItemID, input.EntryID)
}

func (s *Service) DiffHistory(input HistoryDiffInput) (HistoryDiff, error) {
	if input.From == "" {
		return HistoryDiff{}, fmt.Errorf("from history entry is required")
	}
	if input.To == "" {
		input.To = "current"
	}
	from, err := s.GetHistory(HistoryEntryInput{Kind: input.Kind, ItemID: input.ItemID, EntryID: input.From})
	if err != nil {
		return HistoryDiff{}, err
	}
	to, err := s.GetHistory(HistoryEntryInput{Kind: input.Kind, ItemID: input.ItemID, EntryID: input.To})
	if err != nil {
		return HistoryDiff{}, err
	}
	return HistoryDiff{From: from.ID, To: to.ID, Diff: simpleLineDiff(from.Content, to.Content)}, nil
}

func (s *Service) RollbackHistory(input HistoryEntryInput) (HistoryRollbackResult, error) {
	if input.EntryID == "" || input.EntryID == "current" {
		return HistoryRollbackResult{}, fmt.Errorf("select a previous history entry to restore")
	}
	entry, err := s.Store.ReadHistory(input.Kind, input.ItemID, input.EntryID)
	if err != nil {
		return HistoryRollbackResult{}, err
	}
	switch input.Kind {
	case domain.HistorySkill:
		current, err := catalog.New(s.Store).ResolveLibrary(input.ItemID)
		if err != nil {
			return HistoryRollbackResult{}, err
		}
		result, err := s.updateSkill(UpdateSkillInput{ID: input.ItemID, Content: entry.Content, BaseHash: current.Hash, Tags: current.Tags}, "rollback")
		if err != nil {
			return HistoryRollbackResult{}, err
		}
		return HistoryRollbackResult{Entry: entry, Skill: &result}, nil
	case domain.HistoryPrompt:
		current, _, err := promptpkg.New(s.Store).Read(input.ItemID)
		if err != nil {
			return HistoryRollbackResult{}, err
		}
		result, err := s.updatePrompt(PromptWriteInput{ID: input.ItemID, Content: entry.Content, BaseHash: current.Hash}, "rollback")
		if err != nil {
			return HistoryRollbackResult{}, err
		}
		return HistoryRollbackResult{Entry: entry, Prompt: &result}, nil
	default:
		return HistoryRollbackResult{}, fmt.Errorf("invalid history kind %q", input.Kind)
	}
}

func (s *Service) currentHistory(kind domain.HistoryKind, itemID string) (domain.HistoryEntry, error) {
	if !kind.Valid() {
		return domain.HistoryEntry{}, fmt.Errorf("invalid history kind %q", kind)
	}
	switch kind {
	case domain.HistorySkill:
		details, err := s.GetSkill(itemID)
		if err != nil {
			return domain.HistoryEntry{}, err
		}
		createdAt := time.Now().UTC()
		if info, statErr := os.Stat(filepath.Join(details.EffectivePath, "SKILL.md")); statErr == nil {
			createdAt = info.ModTime().UTC()
		}
		return domain.HistoryEntry{ID: "current", ItemID: details.ID, Kind: kind, Hash: details.Hash, CreatedAt: createdAt, Reason: "current", Current: true, Content: details.Content}, nil
	case domain.HistoryPrompt:
		details, err := s.GetPrompt(itemID)
		if err != nil {
			return domain.HistoryEntry{}, err
		}
		return domain.HistoryEntry{ID: "current", ItemID: details.ID, Kind: kind, Hash: details.Hash, CreatedAt: details.UpdatedAt, Reason: "current", Current: true, Content: details.Content}, nil
	default:
		return domain.HistoryEntry{}, fmt.Errorf("invalid history kind %q", kind)
	}
}

func withoutHistoryContent(entry domain.HistoryEntry) domain.HistoryEntry {
	entry.Content = ""
	return entry
}

func simpleLineDiff(before, after string) string {
	if before == after {
		return "No changes.\n"
	}
	oldLines := strings.Split(strings.ReplaceAll(before, "\r\n", "\n"), "\n")
	newLines := strings.Split(strings.ReplaceAll(after, "\r\n", "\n"), "\n")
	prefix := 0
	for prefix < len(oldLines) && prefix < len(newLines) && oldLines[prefix] == newLines[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(oldLines)-prefix && suffix < len(newLines)-prefix && oldLines[len(oldLines)-1-suffix] == newLines[len(newLines)-1-suffix] {
		suffix++
	}
	var builder strings.Builder
	builder.WriteString("--- previous\n+++ selected\n")
	start := prefix
	oldEnd := len(oldLines) - suffix
	newEnd := len(newLines) - suffix
	fmt.Fprintf(&builder, "@@ -%d,%d +%d,%d @@\n", start+1, oldEnd-start, start+1, newEnd-start)
	for _, line := range oldLines[start:oldEnd] {
		builder.WriteString("-")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	for _, line := range newLines[start:newEnd] {
		builder.WriteString("+")
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	return builder.String()
}
