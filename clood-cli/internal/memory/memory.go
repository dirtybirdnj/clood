// Package memory provides persistent knowledge storage for clood.
// Memories are stored locally in ~/.clood/memory.json
package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Memory represents a single stored fact or note
type Memory struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	Tags      []string  `json:"tags,omitempty"`
	Context   string    `json:"context,omitempty"` // project/file context when stored
	CreatedAt time.Time `json:"created_at"`
}

// Store manages the memory storage
type Store struct {
	path     string
	memories []Memory
}

// NewStore creates a new memory store, loading existing memories if present
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cloodDir := filepath.Join(home, ".clood")
	if err := os.MkdirAll(cloodDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .clood directory: %w", err)
	}

	s := &Store{
		path:     filepath.Join(cloodDir, "memory.json"),
		memories: []Memory{},
	}

	// Load existing memories if file exists
	if data, err := os.ReadFile(s.path); err == nil {
		if err := json.Unmarshal(data, &s.memories); err != nil {
			return nil, fmt.Errorf("failed to parse memory file: %w", err)
		}
	}

	return s, nil
}

// save persists memories to disk
func (s *Store) save() error {
	data, err := json.MarshalIndent(s.memories, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memories: %w", err)
	}

	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write memory file: %w", err)
	}

	return nil
}

// Store adds a new memory
func (s *Store) Store(content string, tags []string, context string) (*Memory, error) {
	mem := Memory{
		ID:        uuid.New().String()[:8], // Short ID for easy reference
		Content:   content,
		Tags:      tags,
		Context:   context,
		CreatedAt: time.Now(),
	}

	s.memories = append(s.memories, mem)

	if err := s.save(); err != nil {
		return nil, err
	}

	return &mem, nil
}

// Recall searches memories by keyword or tag
func (s *Store) Recall(query string, tag string, limit int) []Memory {
	var results []Memory

	queryLower := strings.ToLower(query)

	for _, mem := range s.memories {
		// Filter by tag if specified
		if tag != "" {
			hasTag := false
			for _, t := range mem.Tags {
				if strings.EqualFold(t, tag) {
					hasTag = true
					break
				}
			}
			if !hasTag {
				continue
			}
		}

		// Filter by query if specified
		if query != "" {
			contentLower := strings.ToLower(mem.Content)
			contextLower := strings.ToLower(mem.Context)
			if !strings.Contains(contentLower, queryLower) && !strings.Contains(contextLower, queryLower) {
				continue
			}
		}

		results = append(results, mem)
	}

	// Sort by most recent first
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})

	// Apply limit
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// List returns memories, optionally filtered by tag
func (s *Store) List(tag string, limit int) []Memory {
	return s.Recall("", tag, limit)
}

// Forget removes a memory by ID
func (s *Store) Forget(id string) error {
	for i, mem := range s.memories {
		if mem.ID == id {
			s.memories = append(s.memories[:i], s.memories[i+1:]...)
			return s.save()
		}
	}
	return fmt.Errorf("memory not found: %s", id)
}

// Count returns the total number of memories
func (s *Store) Count() int {
	return len(s.memories)
}

// Tags returns all unique tags used
func (s *Store) Tags() []string {
	tagSet := make(map[string]bool)
	for _, mem := range s.memories {
		for _, tag := range mem.Tags {
			tagSet[tag] = true
		}
	}

	var tags []string
	for tag := range tagSet {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags
}
