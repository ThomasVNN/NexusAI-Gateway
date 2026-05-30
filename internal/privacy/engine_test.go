package privacy

import (
	"testing"
)

func TestEnhancedEngine_DetectPII(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Detect email", func(t *testing.T) {
		text := "Contact me at john.doe@example.com for details"
		detections := engine.DetectPII(text)

		if len(detections) != 1 {
			t.Fatalf("Expected 1 detection, got %d", len(detections))
		}
		if detections[0].Type != PIIEmail {
			t.Errorf("Expected email type, got %s", detections[0].Type)
		}
	})

	t.Run("Detect multiple PII types", func(t *testing.T) {
		text := "Email: test@test.com, Phone: +1-234-567-8900, SSN: 123-45-6789"
		detections := engine.DetectPII(text)

		if len(detections) != 3 {
			t.Fatalf("Expected 3 detections, got %d", len(detections))
		}
	})

	t.Run("Detect credit card", func(t *testing.T) {
		text := "Card number: 4532-1234-5678-9012"
		detections := engine.DetectPII(text)

		if len(detections) != 1 {
			t.Fatalf("Expected 1 detection, got %d", len(detections))
		}
		if detections[0].Type != PIICreditCard {
			t.Errorf("Expected credit card type, got %s", detections[0].Type)
		}
	})

	t.Run("Detect password pattern", func(t *testing.T) {
		text := "password=supersecret123"
		detections := engine.DetectPII(text)

		if len(detections) != 1 {
			t.Fatalf("Expected 1 detection, got %d", len(detections))
		}
		if detections[0].Type != PIIPassword {
			t.Errorf("Expected password type, got %s", detections[0].Type)
		}
	})

	t.Run("Detect API token", func(t *testing.T) {
		text := "api_key=sk-1234567890abcdefghijklmnop"
		detections := engine.DetectPII(text)

		if len(detections) != 1 {
			t.Fatalf("Expected 1 detection, got %d", len(detections))
		}
	})

	t.Run("No detections", func(t *testing.T) {
		text := "This is just a normal text with no sensitive data"
		detections := engine.DetectPII(text)

		if len(detections) != 0 {
			t.Errorf("Expected 0 detections, got %d", len(detections))
		}
	})
}

func TestEnhancedEngine_Redact(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Redact email", func(t *testing.T) {
		text := "Contact me at john.doe@example.com"
		result := engine.Redact(text)

		if result == text {
			t.Error("Email should be redacted")
		}
		if result == "[REDACTED_EMAIL]@" {
			t.Error("Email should be partially masked, not fully redacted")
		}
	})

	t.Run("Redact phone", func(t *testing.T) {
		text := "Call me at +1-234-567-8900"
		result := engine.Redact(text)

		if result == text {
			t.Error("Phone should be redacted")
		}
	})

	t.Run("Redact credit card", func(t *testing.T) {
		text := "Card: 4532-1234-5678-9012"
		result := engine.Redact(text)

		if result == text {
			t.Error("Credit card should be redacted")
		}
		if result == text {
			t.Error("Credit card should be masked")
		}
	})

	t.Run("Redact SSN", func(t *testing.T) {
		text := "SSN: 123-45-6789"
		result := engine.Redact(text)

		if result == text {
			t.Error("SSN should be redacted")
		}
	})

	t.Run("Redact password", func(t *testing.T) {
		text := "password=supersecret"
		result := engine.Redact(text)

		if result == text {
			t.Error("Password should be redacted")
		}
	})

	t.Run("Redact multiple", func(t *testing.T) {
		text := "Email: test@test.com, Phone: +1-234-567-8900, SSN: 123-45-6789"
		result := engine.Redact(text)

		if result == text {
			t.Error("Multiple PII should be redacted")
		}
	})

	t.Run("No PII unchanged", func(t *testing.T) {
		text := "This is a normal text"
		result := engine.Redact(text)

		if result != text {
			t.Error("Normal text should not be modified")
		}
	})
}

func TestEnhancedEngine_RedactStructured(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Redact nested map", func(t *testing.T) {
		data := map[string]interface{}{
			"name": "John Doe",
			"contact": map[string]interface{}{
				"email":   "john@example.com",
				"phone":   "+1-234-567-8900",
				"address": "123 Main St",
			},
			"age": 30,
		}

		result := engine.RedactStructured(data)

		contact := result["contact"].(map[string]interface{})
		email := contact["email"].(string)
		if email == "john@example.com" {
			t.Error("Email should be redacted")
		}
	})

	t.Run("Redact array", func(t *testing.T) {
		data := map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"email": "user1@example.com"},
				map[string]interface{}{"email": "user2@example.com"},
			},
		}

		result := engine.RedactStructured(data)
		users := result["users"].([]interface{})

		for i, u := range users {
			user := u.(map[string]interface{})
			if user["email"] == "user1@example.com" || user["email"] == "user2@example.com" {
				t.Errorf("User %d email should be redacted", i)
			}
		}
	})
}

func TestEnhancedEngine_FilterResponse(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Filter code injection", func(t *testing.T) {
		response := "To delete files, use os.system('rm -rf /')"
		result := engine.FilterResponse(response)

		if result == response {
			t.Error("Code injection should be filtered")
		}
	})

	t.Run("Filter exposed passwords", func(t *testing.T) {
		response := "The password is password=secret123"
		result := engine.FilterResponse(response)

		if result == response {
			t.Error("Password should be filtered")
		}
	})
}

func TestLuhnCheck(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Valid visa card", func(t *testing.T) {
		// 4532-0151-2612-4469 is a valid test card number
		valid := engine.luhnCheck("4532015112830366")
		if !valid {
			t.Error("Valid credit card should pass Luhn check")
		}
	})

	t.Run("Invalid card", func(t *testing.T) {
		valid := engine.luhnCheck("1234567890123456")
		if valid {
			t.Error("Invalid credit card should fail Luhn check")
		}
	})
}

func TestIsValidIP(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())

	t.Run("Valid IPv4", func(t *testing.T) {
		valid := engine.isValidIP("192.168.1.1")
		if !valid {
			t.Error("Valid IP should pass check")
		}
	})

	t.Run("Invalid IP", func(t *testing.T) {
		valid := engine.isValidIP("999.999.999.999")
		if valid {
			t.Error("Invalid IP should fail check")
		}
	})
}

func TestGenerateReport(t *testing.T) {
	detections := []PIIDetection{
		{Type: PIIEmail, Value: "test@test.com", Start: 0, End: 14},
		{Type: PIIEmail, Value: "test2@test.com", Start: 15, End: 29},
		{Type: PIIPhone, Value: "+1234567890", Start: 30, End: 40},
	}

	report := GenerateReport(detections, 100, 80)

	if report.TotalDetections != 3 {
		t.Errorf("Total detections: got %d, want 3", report.TotalDetections)
	}
	if report.ByType[PIIEmail] != 2 {
		t.Errorf("Email detections: got %d, want 2", report.ByType[PIIEmail])
	}
	if report.Redactions != 20 {
		t.Errorf("Redactions: got %d, want 20", report.Redactions)
	}
}

func TestPrivacyConfig(t *testing.T) {
	t.Run("Default config has all types enabled", func(t *testing.T) {
		config := DefaultPrivacyConfig()
		
		if len(config.EnabledTypes) == 0 {
			t.Error("Default config should have enabled types")
		}
		
		if !config.EnablePIIRedaction {
			t.Error("Default config should enable PII redaction")
		}
	})
}

func TestAddCustomPattern(t *testing.T) {
	engine := NewEnhancedEngine(DefaultPrivacyConfig())
	
	// Add a custom pattern for "customer_id"
	engine.AddCustomPattern("customer_id", regexp.MustCompile(`(?i)customer_\d+`))
	
	text := "Customer ID: CUSTOMER_12345"
	detections := engine.DetectPII(text)
	
	if len(detections) == 0 {
		t.Error("Custom pattern should detect customer ID")
	}
	
	redacted := engine.Redact(text)
	if redacted == text {
		t.Error("Custom pattern should redact")
	}
}
