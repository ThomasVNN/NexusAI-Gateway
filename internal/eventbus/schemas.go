package eventbus

import (
	"encoding/json"
	"fmt"
	"time"
)

// EventSchema defines the JSON schema for a given event type
type EventSchema struct {
	Type        EventType             `json:"type"`
	Version     string                `json:"version"`
	Description string                `json:"description"`
	Payload     map[string]SchemaField `json:"payload"`
	Required    []string              `json:"required"`
}

// SchemaField defines a field in an event schema
type SchemaField struct {
	Type        string   `json:"type"`
	Description string   `json:"description,omitempty"`
	Format      string   `json:"format,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Minimum     *float64 `json:"minimum,omitempty"`
	Maximum     *float64 `json:"maximum,omitempty"`
	MinLength   *int     `json:"minLength,omitempty"`
	MaxLength   *int     `json:"maxLength,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
}

// Well-known event schemas
var EventSchemas = map[EventType]*EventSchema{
	EventTypeIntent: {
		Type:        EventTypeIntent,
		Version:     "1.0",
		Description: "Agent intends to take action",
		Payload: map[string]SchemaField{
			"action": {
				Type:        "string",
				Description: "The action the agent intends to perform",
				MinLength:   intPtr(1),
				MaxLength:   intPtr(256),
			},
			"resource": {
				Type:        "string",
				Description: "The resource the action targets",
				Pattern:     "^[a-zA-Z0-9._-]+$",
			},
			"reason": {
				Type:        "string",
				Description: "The rationale for the intended action",
				MaxLength:   intPtr(1024),
			},
			"parameters": {
				Type:        "object",
				Description: "Additional parameters for the action",
			},
			"estimated_impact": {
				Type:        "string",
				Description: "Estimated impact of the action",
				Enum:        []string{"low", "medium", "high", "critical"},
			},
		},
		Required: []string{"action", "resource"},
	},
	EventTypeDecision: {
		Type:        EventTypeDecision,
		Version:     "1.0",
		Description: "Agent made a decision",
		Payload: map[string]SchemaField{
			"decision": {
				Type:        "string",
				Description: "The decision made",
				MinLength:   intPtr(1),
				MaxLength:   intPtr(512),
			},
			"options": {
				Type:        "array",
				Description: "Options that were considered",
				Items:       &SchemaField{Type: "string"},
			},
			"chosen_option": {
				Type:        "string",
				Description: "The option that was chosen",
			},
			"confidence": {
				Type:        "number",
				Description: "Confidence score (0-1)",
				Minimum:     float64Ptr(0),
				Maximum:     float64Ptr(1),
			},
			"reasoning": {
				Type:        "string",
				Description: "The reasoning behind the decision",
				MaxLength:   intPtr(2048),
			},
		},
		Required: []string{"decision", "chosen_option"},
	},
	EventTypeApproval: {
		Type:        EventTypeApproval,
		Version:     "1.0",
		Description: "Human approved action",
		Payload: map[string]SchemaField{
			"intent_id": {
				Type:        "string",
				Description: "The intent ID that was approved",
				Pattern:     "^int-[a-zA-Z0-9-]+$",
			},
			"approver": {
				Type:        "string",
				Description: "The user who approved",
				MinLength:   intPtr(1),
			},
			"comments": {
				Type:        "string",
				Description: "Optional comments",
				MaxLength:   intPtr(1024),
			},
			"expires_at": {
				Type:        "string",
				Description: "When the approval expires (ISO 8601)",
				Format:      "date-time",
			},
		},
		Required: []string{"intent_id", "approver"},
	},
	EventTypeRejection: {
		Type:        EventTypeRejection,
		Version:     "1.0",
		Description: "Human rejected action",
		Payload: map[string]SchemaField{
			"intent_id": {
				Type:        "string",
				Description: "The intent ID that was rejected",
				Pattern:     "^int-[a-zA-Z0-9-]+$",
			},
			"rejector": {
				Type:        "string",
				Description: "The user who rejected",
				MinLength:   intPtr(1),
			},
			"reason": {
				Type:        "string",
				Description: "The reason for rejection",
				MinLength:   intPtr(1),
				MaxLength:   intPtr(512),
			},
			"alternatives": {
				Type:        "array",
				Description: "Alternative approaches suggested",
				Items:       &SchemaField{Type: "string"},
			},
		},
		Required: []string{"intent_id", "rejector", "reason"},
	},
	EventTypeError: {
		Type:        EventTypeError,
		Version:     "1.0",
		Description: "Agent encountered error",
		Payload: map[string]SchemaField{
			"error_code": {
				Type:        "string",
				Description: "Machine-readable error code",
				Pattern:     "^[A-Z0-9_]+$",
			},
			"message": {
				Type:        "string",
				Description: "Human-readable error message",
				MinLength:   intPtr(1),
				MaxLength:   intPtr(512),
			},
			"recoverable": {
				Type:        "boolean",
				Description: "Whether the error can be recovered from",
			},
			"context": {
				Type:        "object",
				Description: "Additional error context",
			},
			"stack_trace": {
				Type:        "string",
				Description: "Stack trace for debugging",
				MaxLength:   intPtr(4096),
			},
		},
		Required: []string{"error_code", "message"},
	},
}

// ValidateEvent validates an event against its schema
func ValidateEvent(event *Event) error {
	schema, ok := EventSchemas[event.Type]
	if !ok {
		return fmt.Errorf("no schema defined for event type: %s", event.Type)
	}

	// Check required fields
	for _, field := range schema.Required {
		if _, exists := schema.Payload[field]; !exists {
			return fmt.Errorf("schema error: required field %s not defined in schema", field)
		}
	}

	// Parse and validate payload
	var payload map[string]interface{}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("invalid payload JSON: %w", err)
	}

	// Validate required fields are present
	for _, field := range schema.Required {
		if _, exists := payload[field]; !exists {
			return fmt.Errorf("missing required field: %s", field)
		}
	}

	// Validate field types
	for field, value := range payload {
		if schemaField, ok := schema.Payload[field]; ok {
			if err := validateFieldType(field, value, schemaField); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateFieldType(field string, value interface{}, schema SchemaField) error {
	switch schema.Type {
	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("field %s must be a string", field)
		}
		if schema.MinLength != nil && len(str) < *schema.MinLength {
			return fmt.Errorf("field %s length %d is less than minimum %d", field, len(str), *schema.MinLength)
		}
		if schema.MaxLength != nil && len(str) > *schema.MaxLength {
			return fmt.Errorf("field %s length %d exceeds maximum %d", field, len(str), *schema.MaxLength)
		}
		if schema.Enum != nil {
			found := false
			for _, e := range schema.Enum {
				if str == e {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("field %s value %q not in enum %v", field, str, schema.Enum)
			}
		}
	case "number":
		switch v := value.(type) {
		case float64:
			if schema.Minimum != nil && v < *schema.Minimum {
				return fmt.Errorf("field %s value %f is less than minimum %f", field, v, *schema.Minimum)
			}
			if schema.Maximum != nil && v > *schema.Maximum {
				return fmt.Errorf("field %s value %f exceeds maximum %f", field, v, *schema.Maximum)
			}
		case int:
			if schema.Minimum != nil && float64(v) < *schema.Minimum {
				return fmt.Errorf("field %s value %d is less than minimum %f", field, v, *schema.Minimum)
			}
			if schema.Maximum != nil && float64(v) > *schema.Maximum {
				return fmt.Errorf("field %s value %d exceeds maximum %f", field, v, *schema.Maximum)
			}
		}
	case "boolean":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("field %s must be a boolean", field)
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			return fmt.Errorf("field %s must be an object", field)
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			return fmt.Errorf("field %s must be an array", field)
		}
	}
	return nil
}

// GetSchema returns the schema for a given event type
func GetSchema(eventType EventType) *EventSchema {
	return EventSchemas[eventType]
}

// GetAllSchemaVersions returns all schema versions
func GetAllSchemaVersions() map[string]string {
	versions := make(map[string]string)
	for et, schema := range EventSchemas {
		versions[string(et)] = schema.Version
	}
	return versions
}

// SchemaField references the SchemaField type for Items
var Items = &SchemaField{Type: "string"}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func float64Ptr(f float64) *float64 {
	return &f
}

// Well-known event categories for grouping
const (
	EventCategoryAgent       = "agent"
	EventCategoryWorkflow   = "workflow"
	EventCategorySystem     = "system"
	EventCategoryAudit      = "audit"
	EventCategoryMonitoring = "monitoring"
)

// GetEventCategory returns the category for an event type
func GetEventCategory(eventType EventType) string {
	switch eventType {
	case EventTypeIntent, EventTypeDecision:
		return EventCategoryAgent
	case EventTypeApproval, EventTypeRejection:
		return EventCategoryWorkflow
	case EventTypeError:
		return EventCategorySystem
	default:
		return EventCategoryAgent
	}
}

// EventMetadata contains standard metadata fields
type EventMetadata struct {
	// Correlation links related events
	CorrelationID string `json:"correlation_id,omitempty"`
	// Causation links cause-effect
	CausationID string `json:"causation_id,omitempty"`
	// Tracing
	TraceID string `json:"trace_id,omitempty"`
	SpanID  string `json:"span_id,omitempty"`
	// Timestamps
	OccurredAt  time.Time `json:"occurred_at"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	// Versioning
	SchemaVersion string `json:"schema_version,omitempty"`
	// Tags for filtering
	Tags []string `json:"tags,omitempty"`
}
