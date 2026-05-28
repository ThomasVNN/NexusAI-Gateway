package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/domain/model"
)

type Store struct {
	mu        sync.RWMutex
	keys      map[string]*model.RegisteredKey
	usage     []*model.UsageRecord
	providers map[string]*model.ProviderConnection
}

func NewStore() *Store {
	s := &Store{
		keys:      make(map[string]*model.RegisteredKey),
		usage:     make([]*model.UsageRecord, 0),
		providers: make(map[string]*model.ProviderConnection),
	}
	s.seedDefaultData()
	return s
}

func (s *Store) seedDefaultData() {
	// Seed some default provider configurations
	s.providers["openai"] = &model.ProviderConnection{
		ID:        "openai",
		Provider:  "openai",
		Name:      "OpenAI Connection",
		Endpoint:  "https://api.openai.com/v1/chat/completions",
		IsActive:  true,
		Priority:  1,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	s.providers["anthropic"] = &model.ProviderConnection{
		ID:        "anthropic",
		Provider:  "anthropic",
		Name:      "Anthropic Connection",
		Endpoint:  "https://api.anthropic.com/v1/messages",
		IsActive:  false,
		Priority:  2,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// Registered Keys Repository
func (s *Store) GetByID(ctx context.Context, id string) (*model.RegisteredKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	k, exists := s.keys[id]
	if !exists {
		return nil, errors.New("key not found")
	}
	return k, nil
}

func (s *Store) GetByHash(ctx context.Context, hash string) (*model.RegisteredKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, k := range s.keys {
		if k.KeyHash == hash {
			return k, nil
		}
	}
	return nil, errors.New("key not found by hash")
}

func (s *Store) Save(ctx context.Context, key *model.RegisteredKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[key.ID] = key
	return nil
}

func (s *Store) ListAll(ctx context.Context) ([]*model.RegisteredKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.RegisteredKey, 0, len(s.keys))
	for _, k := range s.keys {
		list = append(list, k)
	}
	return list, nil
}

// Usage Accounting Repository
func (s *Store) LogUsage(ctx context.Context, record *model.UsageRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	record.ID = len(s.usage) + 1
	record.CreatedAt = time.Now()
	s.usage = append(s.usage, record)
	return nil
}

func (s *Store) GetHourlyUsage(ctx context.Context, keyID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	for _, r := range s.usage {
		if r.KeyID == keyID && r.CreatedAt.After(oneHourAgo) {
			count++
		}
	}
	return count, nil
}

func (s *Store) GetDailyUsage(ctx context.Context, keyID string) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	count := 0
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	for _, r := range s.usage {
		if r.KeyID == keyID && r.CreatedAt.After(oneDayAgo) {
			count++
		}
	}
	return count, nil
}

func (s *Store) GetAggregateUsage(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	totalCalls := len(s.usage)
	totalPrompt := 0
	totalCompletion := 0
	totalLatency := 0

	for _, r := range s.usage {
		totalPrompt += r.PromptTokens
		totalCompletion += r.CompletionTokens
		totalLatency += r.LatencyMS
	}

	avgLatency := 0.0
	if totalCalls > 0 {
		avgLatency = float64(totalLatency) / float64(totalCalls)
	}

	return map[string]interface{}{
		"total_calls":             totalCalls,
		"total_prompt_tokens":     totalPrompt,
		"total_completion_tokens": totalCompletion,
		"average_latency_ms":      avgLatency,
	}, nil
}

func (s *Store) ListLogs(ctx context.Context) ([]*model.UsageRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.usage, nil
}

// Providers Repository
func (s *Store) ListAllProviders(ctx context.Context) ([]*model.ProviderConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]*model.ProviderConnection, 0, len(s.providers))
	for _, p := range s.providers {
		list = append(list, p)
	}
	return list, nil
}

func (s *Store) SaveProvider(ctx context.Context, conn *model.ProviderConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.providers[conn.ID] = conn
	return nil
}

func (s *Store) GetProviderByID(ctx context.Context, id string) (*model.ProviderConnection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, exists := s.providers[id]
	if !exists {
		return nil, errors.New("provider not found")
	}
	return p, nil
}
