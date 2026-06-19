package eventbus

import (
	"sync"
)

// EventSequencer provides ordering guarantees for events within the same partition key
type EventSequencer struct {
	sequences map[string]uint64 // partitionKey -> current sequence number
	mu        sync.RWMutex
}

// NewEventSequencer creates a new event sequencer
func NewEventSequencer() *EventSequencer {
	return &EventSequencer{
		sequences: make(map[string]uint64),
	}
}

// Next returns the next sequence number for a given partition key
// Thread-safe and atomic per partition key
func (s *EventSequencer) Next(partitionKey string) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.sequences[partitionKey] + 1
	s.sequences[partitionKey] = next
	return next
}

// GetCurrent returns the current sequence number for a partition key without incrementing
func (s *EventSequencer) GetCurrent(partitionKey string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sequences[partitionKey]
}

// Reset clears all sequence numbers (use with caution)
func (s *EventSequencer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sequences = make(map[string]uint64)
}

// ResetKey clears the sequence number for a specific partition key
func (s *EventSequencer) ResetKey(partitionKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sequences, partitionKey)
}

// IsOrdered checks if a sequence of events is properly ordered for a partition key
func (s *EventSequencer) IsOrdered(partitionKey string, sequenceNumbers []uint64) bool {
	if len(sequenceNumbers) < 2 {
		return true
	}

	expected := s.GetCurrent(partitionKey) - uint64(len(sequenceNumbers)-1)

	for _, seq := range sequenceNumbers {
		if seq != expected {
			return false
		}
		expected++
	}

	return true
}
