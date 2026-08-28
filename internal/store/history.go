package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/zzzzzyijie/skm/internal/domain"
	"github.com/zzzzzyijie/skm/internal/fsx"
)

const historyRetention = 50

func (s *Store) SaveHistory(kind domain.HistoryKind, itemID, hash, reason, content string) (domain.HistoryEntry, error) {
	if !kind.Valid() {
		return domain.HistoryEntry{}, fmt.Errorf("invalid history kind %q", kind)
	}
	itemID = strings.TrimSpace(itemID)
	if itemID == "" || hash == "" {
		return domain.HistoryEntry{}, fmt.Errorf("history item and hash are required")
	}
	if err := s.Ensure(); err != nil {
		return domain.HistoryEntry{}, err
	}
	existing, err := s.ListHistory(kind, itemID)
	if err != nil {
		return domain.HistoryEntry{}, err
	}
	for _, entry := range existing {
		if entry.Hash == hash {
			entry.Content = content
			return entry, nil
		}
	}
	now := time.Now().UTC()
	entry := domain.HistoryEntry{
		ID: fmt.Sprintf("%020d-%s", now.UnixNano(), shortHistoryHash(hash)), ItemID: itemID,
		Kind: kind, Hash: hash, CreatedAt: now, Reason: strings.TrimSpace(reason), Content: content,
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return domain.HistoryEntry{}, err
	}
	directory := s.historyDirectory(kind, itemID)
	if err := fsx.AtomicWriteFile(filepath.Join(directory, entry.ID+".json"), append(data, '\n'), 0o600); err != nil {
		return domain.HistoryEntry{}, err
	}
	if err := pruneHistoryDirectory(directory); err != nil {
		return domain.HistoryEntry{}, err
	}
	return entry, nil
}

func (s *Store) ListHistory(kind domain.HistoryKind, itemID string) ([]domain.HistoryEntry, error) {
	if !kind.Valid() {
		return nil, fmt.Errorf("invalid history kind %q", kind)
	}
	directory := s.historyDirectory(kind, itemID)
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return []domain.HistoryEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	result := make([]domain.HistoryEntry, 0, len(entries))
	for _, file := range entries {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(directory, file.Name()))
		if err != nil {
			return nil, err
		}
		var entry domain.HistoryEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			return nil, fmt.Errorf("read history %s: %w", file.Name(), err)
		}
		if entry.Kind != kind || entry.ItemID != itemID {
			return nil, fmt.Errorf("history entry %s does not match its directory", file.Name())
		}
		entry.Content = ""
		result = append(result, entry)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt.After(result[j].CreatedAt) })
	return result, nil
}

func (s *Store) ReadHistory(kind domain.HistoryKind, itemID, entryID string) (domain.HistoryEntry, error) {
	if !kind.Valid() {
		return domain.HistoryEntry{}, fmt.Errorf("invalid history kind %q", kind)
	}
	if entryID == "" || filepath.Base(entryID) != entryID || strings.Contains(entryID, "..") {
		return domain.HistoryEntry{}, fmt.Errorf("invalid history entry %q", entryID)
	}
	data, err := os.ReadFile(filepath.Join(s.historyDirectory(kind, itemID), entryID+".json"))
	if os.IsNotExist(err) {
		return domain.HistoryEntry{}, fmt.Errorf("history entry %q not found", entryID)
	}
	if err != nil {
		return domain.HistoryEntry{}, err
	}
	var entry domain.HistoryEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return domain.HistoryEntry{}, err
	}
	if entry.Kind != kind || entry.ItemID != itemID || entry.ID != entryID {
		return domain.HistoryEntry{}, fmt.Errorf("history entry %q is invalid", entryID)
	}
	return entry, nil
}

func (s *Store) historyDirectory(kind domain.HistoryKind, itemID string) string {
	digest := sha256.Sum256([]byte(itemID))
	return filepath.Join(s.Paths.Home, "history", string(kind), hex.EncodeToString(digest[:]))
}

func shortHistoryHash(hash string) string {
	if len(hash) > 12 {
		return hash[:12]
	}
	return hash
}

func pruneHistoryDirectory(directory string) error {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			names = append(names, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(names)))
	if len(names) <= historyRetention {
		return nil
	}
	for _, name := range names[historyRetention:] {
		if err := os.Remove(filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	return nil
}
