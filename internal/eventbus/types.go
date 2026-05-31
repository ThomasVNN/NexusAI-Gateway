package eventbus

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventPriority defines the priority level for event processing
type EventPriority int

const (
	PriorityLow    EventPriority = iota // Background processing, can be delayed
	PriorityMedium                      // Standard processing, default priority
	PriorityHigh                        // Urgent processing, expedited delivery
)

// EventType defines the type of agent event
type EventType string

const (
	EventTypeIntent    EventType = "agent.intent"     // Agent intends to take action
	EventTypeDecision  EventType = "agent.decision"   // Agent made a decision
	EventTypeApproval  EventType = "agent.approval"   // Human approved action
	EventTypeRejection EventType = "agent.rejection"  // Human rejected action
	EventTypeError     EventType = "agent.error"      // Agent encountered error
)

// Event represents a structured event in the agent communication system
type Event struct {
	// ID uniquely identifies this event (ULID for sortable ordering)
	ID string `json:"id"`
	// Type categorizes the event for routing and handling
	Type EventType `json:"type"`
	// Priority determines processing order (high events are processed first)
	Priority EventPriority `json:"priority"`
	// SourceAgent identifies the agent that generated this event
	SourceAgent string `json:"source_agent"`
	// TargetAgent optionally specifies the intended recipient (empty = broadcast)
	TargetAgent string `json:"target_agent,omitempty"`
	// CorrelationID links related events for tracing
	CorrelationID string `json:"correlation_id,omitempty"`
	// Payload contains the event-specific data
	Payload json.RawMessage `json:"payload"`
	// Metadata provides additional context
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	// Timestamp records when the event was created
	Timestamp time.Time `json:"timestamp"`
	// SequenceNumber ensures ordering within a partition
	SequenceNumber uint64 `json:"sequence_number,omitempty"`
	// TTL defines how long this event is valid (0 = no expiration)
	TTL time.Duration `json:"ttl,omitempty"`
}

// NewEvent creates a new event with the current timestamp and a generated ID
func NewEvent(eventType EventType, sourceAgent string, payload interface{}, priority EventPriority) (*Event, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	return &Event{
		ID:           generateEventID(),
		Type:         eventType,
		Priority:     priority,
		SourceAgent:  sourceAgent,
		Payload:      payloadBytes,
		Metadata:     make(map[string]interface{}),
		Timestamp:    time.Now().UTC(),
	}, nil
}

// WithTarget sets the target agent for direct delivery
func (e *Event) WithTarget(targetAgent string) *Event {
	e.TargetAgent = targetAgent
	return e
}

// WithCorrelationID sets the correlation ID for event chaining
func (e *Event) WithCorrelationID(correlationID string) *Event {
	e.CorrelationID = correlationID
	return e
}

// WithMetadata adds key-value pairs to the event metadata
func (e *Event) WithMetadata(key string, value interface{}) *Event {
	if e.Metadata == nil {
		e.Metadata = make(map[string]interface{})
	}
	e.Metadata[key] = value
	return e
}

// WithTTL sets the time-to-live for the event
func (e *Event) WithTTL(ttl time.Duration) *Event {
	e.TTL = ttl
	return e
}

// WithSequenceNumber sets the sequence number for ordering
func (e *Event) WithSequenceNumber(seq uint64) *Event {
	e.SequenceNumber = seq
	return e
}

// ToJSON serializes the event to JSON bytes
func (e *Event) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ParseEvent deserializes a JSON byte slice into an Event
func ParseEvent(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}
	return &event, nil
}

// GetPriority returns the processing priority for routing
func (e *Event) GetPriority() EventPriority {
	return e.Priority
}

// IsExpired checks if the event has passed its TTL
func (e *Event) IsExpired() bool {
	if e.TTL == 0 {
		return false
	}
	return time.Since(e.Timestamp) > e.TTL
}

// String returns a human-readable representation of the event
func (e *Event) String() string {
	return fmt.Sprintf("Event{id=%s, type=%s, priority=%d, source=%s, target=%s}",
		e.ID, e.Type, e.Priority, e.SourceAgent, e.TargetAgent)
}

// IntentPayload represents the payload for agent.intent events
type IntentPayload struct {
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	Reason       string                 `json:"reason,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	EstimatedImpact string               `json:"estimated_impact,omitempty"`
}

// DecisionPayload represents the payload for agent.decision events
type DecisionPayload struct {
	Decision    string                 `json:"decision"`
	Options     []string               `json:"options,omitempty"`
	ChosenOption string                `json:"chosen_option,omitempty"`
	Confidence  float64                `json:"confidence,omitempty"`
	Reasoning   string                 `json:"reasoning,omitempty"`
}

// ApprovalPayload represents the payload for agent.approval events
type ApprovalPayload struct {
	IntentID    string `json:"intent_id"`
	Approver    string `json:"approver"`
	Comments    string `json:"comments,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// RejectionPayload represents the payload for agent.rejection events
type RejectionPayload struct {
	IntentID    string `json:"intent_id"`
	Rejector    string `json:"rejector"`
	Reason      string `json:"reason"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// ErrorPayload represents the payload for agent.error events
type ErrorPayload struct {
	ErrorCode   string                 `json:"error_code"`
	Message     string                 `json:"message"`
	Recoverable bool                   `json:"recoverable"`
	Context     map[string]interface{} `json:"context,omitempty"`
	StackTrace  string                 `json:"stack_trace,omitempty"`
}

// Subscription represents a client subscription to event types
type Subscription struct {
	ID          string      `json:"id"`
	ClientID    string      `json:"client_id"`
	EventTypes  []EventType `json:"event_types"`
	TargetAgent string      `json:"target_agent,omitempty"` // Empty = all agents
	Filter      *Filter     `json:"filter,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
	Active      bool        `json:"active"`
}

// Filter defines optional filtering criteria for subscriptions
type Filter struct {
	// SourceAgents limits events to specific source agents
	SourceAgents []string `json:"source_agents,omitempty"`
	// PriorityThreshold only receives events at or above this priority
	PriorityThreshold EventPriority `json:"priority_threshold,omitempty"`
	// MetadataFilters key-value pairs that must match
	MetadataFilters map[string]interface{} `json:"metadata_filters,omitempty"`
}

// DLQEntry represents a failed event in the dead letter queue
type DLQEntry struct {
	ID            string    `json:"id"`
	Event         *Event    `json:"event"`
	Error         string    `json:"error"`
	RetryCount    int       `json:"retry_count"`
	MaxRetries    int       `json:"max_retries"`
	FirstFailedAt time.Time `json:"first_failed_at"`
	LastFailedAt  time.Time `json:"last_failed_at"`
	NextRetryAt   time.Time `json:"next_retry_at,omitempty"`
	Status        DLQStatus `json:"status"`
}

// DLQStatus represents the status of a DLQ entry
type DLQStatus string

const (
	DLQStatusPending    DLQStatus = "pending"    // Awaiting retry
	DLQStatusRetrying   DLQStatus = "retrying"  // Currently retrying
	DLQStatusDead       DLQStatus = "dead"      // Exceeded max retries
	DLQStatusProcessed  DLQStatus = "processed"  // Successfully processed after retry
)

// NewDLQEntry creates a new DLQ entry from a failed event
func NewDLQEntry(event *Event, err error, maxRetries int) *DLQEntry {
	return &DLQEntry{
		ID:            generateEventID(),
		Event:         event,
		Error:         err.Error(),
		RetryCount:    0,
		MaxRetries:    maxRetries,
		FirstFailedAt: time.Now().UTC(),
		LastFailedAt:  time.Now().UTC(),
		Status:        DLQStatusPending,
	}
}

// ShouldRetry returns true if the entry should be retried
func (d *DLQEntry) ShouldRetry() bool {
	return d.RetryCount < d.MaxRetries && d.Status != DLQStatusDead
}

// RecordRetry increments the retry count and updates timestamps
func (d *DLQEntry) RecordRetry(err error) {
	d.RetryCount++
	d.LastFailedAt = time.Now().UTC()
	d.Status = DLQStatusRetrying

	// Exponential backoff: 1s, 2s, 4s, 8s, etc.
	backoffSeconds := 1 << d.RetryCount
	if backoffSeconds > 300 { // Cap at 5 minutes
		backoffSeconds = 300
	}
	d.NextRetryAt = time.Now().UTC().Add(time.Duration(backoffSeconds) * time.Second)
}

// MarkDead marks the entry as dead after exhausting retries
func (d *DLQEntry) MarkDead() {
	d.Status = DLQStatusDead
	d.NextRetryAt = time.Time{}
}

// MarkProcessed marks the entry as successfully processed
func (d *DLQEntry) MarkProcessed() {
	d.Status = DLQStatusProcessed
	d.NextRetryAt = time.Time{}
}

// EventBusConfig holds configuration for the event bus
type EventBusConfig struct {
	// Provider specifies the underlying message broker (nats, kafka)
	Provider string
	// NATSConfig holds NATS-specific configuration
	NATSConfig *NATSConfig
	// KafkaConfig holds Kafka-specific configuration
	KafkaConfig *KafkaConfig
	// DLQMaxRetries defines how many times to retry before sending to DLQ
	DLQMaxRetries int
	// DefaultTTL sets the default TTL for events
	DefaultTTL time.Duration
	// EnableOrdering enables strict ordering guarantees within partitions
	EnableOrdering bool
	// OrderingKeyField specifies which metadata field to use for partitioning
	OrderingKeyField string
}

// NATSConfig holds NATS JetStream configuration
type NATSConfig struct {
	URLs           []string // NATS server URLs
	StreamName     string   // JetStream stream name
	ConsumerName   string   // Consumer name
	MaxBytes       int64    // Maximum message size
	AckWait        time.Duration
	MaxDeliver     int
	ReplayPolicy   string // "instant", "original", "last"
}

// KafkaConfig holds Kafka configuration
type KafkaConfig struct {
	Brokers         []string // Kafka broker addresses
	Topic           string   // Topic name
	ConsumerGroup   string   // Consumer group ID
	AutoOffsetReset string   // "earliest", "latest"
	MaxMessageBytes int      // Maximum message size
}
