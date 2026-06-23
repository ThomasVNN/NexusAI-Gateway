package mcp

// JSON schemas for all tools

var emptySchema = JSONSchema{
	Type:       "object",
	Properties: map[string]interface{}{},
	Required:   []string{},
}

// Memory tool schemas
var memorySearchSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"query":      map[string]interface{}{"type": "string", "description": "Search query"},
		"memory_type": map[string]interface{}{"type": "string", "description": "Type of memory (factual, episodic, procedural, semantic)"},
		"limit":      map[string]interface{}{"type": "integer", "description": "Maximum results to return", "default": 10},
	},
	Required: []string{"query"},
}

var memoryAddSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"content":     map[string]interface{}{"type": "string", "description": "Content to add to memory"},
		"memory_type": map[string]interface{}{"type": "string", "description": "Type of memory"},
		"metadata":   map[string]interface{}{"type": "object", "description": "Optional metadata"},
	},
	Required: []string{"content", "memory_type"},
}

var memoryClearSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"memory_type": map[string]interface{}{"type": "string", "description": "Type of memory to clear (optional, clears all if not specified)"},
	},
	Required: []string{},
}

var memoryDeleteSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"id":         map[string]interface{}{"type": "string", "description": "Memory entry ID to delete"},
		"memory_type": map[string]interface{}{"type": "string", "description": "Type of memory"},
	},
	Required: []string{"id"},
}

// Routing tool schemas
var routeRequestSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"request":       map[string]interface{}{"type": "object", "description": "The AI request to route"},
		"provider_type": map[string]interface{}{"type": "string", "description": "Preferred provider type"},
	},
	Required: []string{"request"},
}

var explainRouteSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"request_id": map[string]interface{}{"type": "string", "description": "Request ID to explain"},
	},
	Required: []string{"request_id"},
}

var getHealthSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"provider_id": map[string]interface{}{"type": "string", "description": "Provider ID (optional, returns all if not specified)"},
	},
	Required: []string{},
}

var setFallbackSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"provider_id":  map[string]interface{}{"type": "string", "description": "Fallback provider ID"},
		"provider_type": map[string]interface{}{"type": "string", "description": "Provider type"},
	},
	Required: []string{"provider_id"},
}

var analyzeCostSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"model":      map[string]interface{}{"type": "string", "description": "Model name"},
		"input_tokens": map[string]interface{}{"type": "integer", "description": "Input token count"},
		"output_tokens": map[string]interface{}{"type": "integer", "description": "Output token count"},
	},
	Required: []string{"model", "input_tokens", "output_tokens"},
}

var validateConfigSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"config": map[string]interface{}{"type": "object", "description": "Configuration to validate"},
	},
	Required: []string{"config"},
}

// Budget tool schemas
var setBudgetGuardSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"limit":     map[string]interface{}{"type": "number", "description": "Budget limit"},
		"period":    map[string]interface{}{"type": "string", "description": "Budget period (daily, weekly, monthly)"},
		"alert_at":  map[string]interface{}{"type": "number", "description": "Alert threshold percentage"},
	},
	Required: []string{"limit", "period"},
}

var getUsageSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"start_date": map[string]interface{}{"type": "string", "description": "Start date (ISO 8601)"},
		"end_date":   map[string]interface{}{"type": "string", "description": "End date (ISO 8601)"},
		"dimension":  map[string]interface{}{"type": "string", "description": "Dimension to filter by"},
	},
	Required: []string{},
}

var forecastCostSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"period": map[string]interface{}{"type": "string", "description": "Forecast period (week, month)"},
	},
	Required: []string{"period"},
}

var setSpendLimitSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"amount":  map[string]interface{}{"type": "number", "description": "Spend limit amount"},
		"period":  map[string]interface{}{"type": "string", "description": "Period (daily, weekly, monthly)"},
		"enabled": map[string]interface{}{"type": "boolean", "description": "Enable/disable the limit"},
	},
	Required: []string{"amount", "period"},
}

var getCostBreakdownSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"start_date": map[string]interface{}{"type": "string", "description": "Start date"},
		"end_date":   map[string]interface{}{"type": "string", "description": "End date"},
		"group_by":   map[string]interface{}{"type": "string", "description": "Group by (provider, model, tenant)"},
	},
	Required: []string{},
}

var exportUsageSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"format":    map[string]interface{}{"type": "string", "description": "Export format (csv, json)"},
		"start_date": map[string]interface{}{"type": "string", "description": "Start date"},
		"end_date":   map[string]interface{}{"type": "string", "description": "End date"},
	},
	Required: []string{"format"},
}

var setAlertSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"threshold":  map[string]interface{}{"type": "number", "description": "Alert threshold"},
		"metric":     map[string]interface{}{"type": "string", "description": "Metric to alert on"},
		"webhook_url": map[string]interface{}{"type": "string", "description": "Webhook URL for notifications"},
	},
	Required: []string{"threshold", "metric"},
}

// Skill tool schemas
var executeSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Name of the skill to execute"},
		"parameters": map[string]interface{}{"type": "object", "description": "Skill parameters"},
	},
	Required: []string{"skill_name"},
}

var registerSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"name":        map[string]interface{}{"type": "string", "description": "Skill name"},
		"description": map[string]interface{}{"type": "string", "description": "Skill description"},
		"version":    map[string]interface{}{"type": "string", "description": "Skill version"},
		"code":       map[string]interface{}{"type": "string", "description": "Skill code"},
		"config":     map[string]interface{}{"type": "object", "description": "Skill configuration"},
	},
	Required: []string{"name", "description", "code"},
}

var getSkillInfoSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name"},
	},
	Required: []string{"skill_name"},
}

var validateSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name to validate"},
	},
	Required: []string{"skill_name"},
}

var deleteSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name to delete"},
	},
	Required: []string{"skill_name"},
}

var searchSkillsSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"query":      map[string]interface{}{"type": "string", "description": "Search query"},
		"category":   map[string]interface{}{"type": "string", "description": "Skill category"},
		"limit":      map[string]interface{}{"type": "integer", "description": "Maximum results"},
	},
	Required: []string{"query"},
}

var installSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"marketplace_id": map[string]interface{}{"type": "string", "description": "Marketplace skill ID"},
		"version":        map[string]interface{}{"type": "string", "description": "Version to install"},
	},
	Required: []string{"marketplace_id"},
}

var updateSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name to update"},
		"version":    map[string]interface{}{"type": "string", "description": "New version"},
		"code":       map[string]interface{}{"type": "string", "description": "New code"},
	},
	Required: []string{"skill_name"},
}

var getSkillLogsSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name"},
		"limit":      map[string]interface{}{"type": "integer", "description": "Maximum log entries"},
		"offset":     map[string]interface{}{"type": "integer", "description": "Offset for pagination"},
	},
	Required: []string{"skill_name"},
}

var testSkillSchema = JSONSchema{
	Type: "object",
	Properties: map[string]interface{}{
		"skill_name": map[string]interface{}{"type": "string", "description": "Skill name to test"},
		"parameters": map[string]interface{}{"type": "object", "description": "Test parameters"},
	},
	Required: []string{"skill_name"},
}
