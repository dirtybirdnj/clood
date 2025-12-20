// Package system provides hardware detection and benchmark storage.
package system

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// ModelBenchmark stores performance data for a model
type ModelBenchmark struct {
	Model       string    `json:"model"`
	Host        string    `json:"host"`
	TokPerSec   float64   `json:"tok_per_sec"`
	PromptTokPS float64   `json:"prompt_tok_per_sec,omitempty"`
	TotalTimeS  float64   `json:"total_time_sec,omitempty"`
	Tokens      int       `json:"tokens,omitempty"`
	Source      string    `json:"source"` // "catfight", "bench", "ask"
	RecordedAt  time.Time `json:"recorded_at"`
}

// BenchmarkStore manages model performance data
type BenchmarkStore struct {
	path       string
	Benchmarks []ModelBenchmark `json:"benchmarks"`
}

// NewBenchmarkStore creates or loads the benchmark store
func NewBenchmarkStore() (*BenchmarkStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cloodDir := filepath.Join(home, ".clood")
	if err := os.MkdirAll(cloodDir, 0755); err != nil {
		return nil, err
	}

	store := &BenchmarkStore{
		path:       filepath.Join(cloodDir, "benchmarks.json"),
		Benchmarks: []ModelBenchmark{},
	}

	// Load existing benchmarks
	if data, err := os.ReadFile(store.path); err == nil {
		json.Unmarshal(data, &store.Benchmarks)
	}

	return store, nil
}

// Save persists benchmarks to disk
func (s *BenchmarkStore) Save() error {
	data, err := json.MarshalIndent(s.Benchmarks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// Record adds a new benchmark result
func (s *BenchmarkStore) Record(model, host, source string, tokPerSec float64) error {
	// Update existing or add new
	found := false
	for i, b := range s.Benchmarks {
		if b.Model == model && b.Host == host {
			// Keep the best result
			if tokPerSec > b.TokPerSec {
				s.Benchmarks[i].TokPerSec = tokPerSec
				s.Benchmarks[i].Source = source
				s.Benchmarks[i].RecordedAt = time.Now()
			}
			found = true
			break
		}
	}

	if !found {
		s.Benchmarks = append(s.Benchmarks, ModelBenchmark{
			Model:      model,
			Host:       host,
			TokPerSec:  tokPerSec,
			Source:     source,
			RecordedAt: time.Now(),
		})
	}

	return s.Save()
}

// RecordFull adds a detailed benchmark result
func (s *BenchmarkStore) RecordFull(bm ModelBenchmark) error {
	bm.RecordedAt = time.Now()

	// Update existing or add new
	found := false
	for i, b := range s.Benchmarks {
		if b.Model == bm.Model && b.Host == bm.Host {
			// Keep the best result
			if bm.TokPerSec > b.TokPerSec {
				s.Benchmarks[i] = bm
			}
			found = true
			break
		}
	}

	if !found {
		s.Benchmarks = append(s.Benchmarks, bm)
	}

	return s.Save()
}

// GetBenchmark returns the benchmark for a model on any host
func (s *BenchmarkStore) GetBenchmark(model string) *ModelBenchmark {
	var best *ModelBenchmark
	for i, b := range s.Benchmarks {
		if b.Model == model {
			if best == nil || b.TokPerSec > best.TokPerSec {
				best = &s.Benchmarks[i]
			}
		}
	}
	return best
}

// GetBenchmarkForHost returns the benchmark for a specific model+host
func (s *BenchmarkStore) GetBenchmarkForHost(model, host string) *ModelBenchmark {
	for i, b := range s.Benchmarks {
		if b.Model == model && b.Host == host {
			return &s.Benchmarks[i]
		}
	}
	return nil
}

// GetAllForModel returns all benchmarks for a model across hosts
func (s *BenchmarkStore) GetAllForModel(model string) []ModelBenchmark {
	var results []ModelBenchmark
	for _, b := range s.Benchmarks {
		if b.Model == model {
			results = append(results, b)
		}
	}
	return results
}

// GetTopModels returns the fastest models by tok/s
func (s *BenchmarkStore) GetTopModels(limit int) []ModelBenchmark {
	// Get best result per model
	bestByModel := make(map[string]ModelBenchmark)
	for _, b := range s.Benchmarks {
		if existing, ok := bestByModel[b.Model]; !ok || b.TokPerSec > existing.TokPerSec {
			bestByModel[b.Model] = b
		}
	}

	var results []ModelBenchmark
	for _, b := range bestByModel {
		results = append(results, b)
	}

	// Sort by tok/s descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].TokPerSec > results[j].TokPerSec
	})

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}

	return results
}

// Count returns total benchmark entries
func (s *BenchmarkStore) Count() int {
	return len(s.Benchmarks)
}
