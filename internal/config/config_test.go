package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config == nil {
		t.Error("Expected non-nil config")
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("LOG_LEVEL")
	}()

	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.Server.Port != 9090 {
		t.Errorf("Expected port 9090, got %d", config.Server.Port)
	}
}

func TestValidateConfig(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	err = config.Validate()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestValidateConfigInvalidPort(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:         -1,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			MaxConns: 10,
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		API: APIConfig{
			Timeout: 60,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid port")
	}
}

func TestValidateConfigInvalidLogLevel(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			MaxConns: 10,
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		API: APIConfig{
			Timeout: 60,
		},
		Logging: LoggingConfig{
			Level:  "invalid",
			Format: "json",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid log level")
	}
}

func TestValidateConfigInvalidLogFormat(t *testing.T) {
	config := &Config{
		Server: ServerConfig{
			Port:         8080,
			ReadTimeout:  30,
			WriteTimeout: 30,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			MaxConns: 10,
		},
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
		API: APIConfig{
			Timeout: 60,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "invalid",
		},
	}

	err := config.Validate()
	if err == nil {
		t.Error("Expected error for invalid log format")
	}
}

func TestGetEnvString(t *testing.T) {
	os.Setenv("TEST_KEY", "test_value")
	defer os.Unsetenv("TEST_KEY")

	value := getEnv("TEST_KEY", "default")
	if value != "test_value" {
		t.Errorf("Expected 'test_value', got '%s'", value)
	}
}

func TestGetEnvStringDefault(t *testing.T) {
	os.Unsetenv("NONEXISTENT_KEY")
	value := getEnv("NONEXISTENT_KEY", "default_value")
	if value != "default_value" {
		t.Errorf("Expected 'default_value', got '%s'", value)
	}
}

func TestGetEnvInt(t *testing.T) {
	os.Setenv("TEST_INT", "42")
	defer os.Unsetenv("TEST_INT")

	value := getEnvInt("TEST_INT", 10)
	if value != 42 {
		t.Errorf("Expected 42, got %d", value)
	}
}

func TestGetEnvIntDefault(t *testing.T) {
	os.Unsetenv("NONEXISTENT_INT")
	value := getEnvInt("NONEXISTENT_INT", 99)
	if value != 99 {
		t.Errorf("Expected 99, got %d", value)
	}
}

func TestGetEnvDuration(t *testing.T) {
	os.Setenv("TEST_DURATION", "5s")
	defer os.Unsetenv("TEST_DURATION")

	value := getEnvDuration("TEST_DURATION", 0)
	if value.Seconds() != 5 {
		t.Errorf("Expected 5s, got %v", value)
	}
}

func TestGetEnvSlice(t *testing.T) {
	os.Setenv("TEST_SLICE", "a,b,c")
	defer os.Unsetenv("TEST_SLICE")

	value := getEnvSlice("TEST_SLICE", ",")
	if len(value) != 3 {
		t.Errorf("Expected 3 items, got %d", len(value))
	}
}

func TestGetEnvSliceDefault(t *testing.T) {
	os.Unsetenv("NONEXISTENT_SLICE")
	value := getEnvSlice("NONEXISTENT_SLICE", ",")
	if len(value) != 0 {
		t.Errorf("Expected 0 items, got %d", len(value))
	}
}

func TestConfigServer(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.Server.Port == 0 {
		t.Error("Expected non-zero port")
	}
	if config.Server.Host == "" {
		t.Error("Expected non-empty host")
	}
}

func TestConfigDatabase(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.Database.Host == "" {
		t.Error("Expected non-empty database host")
	}
}

func TestConfigRedis(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.Redis.Host == "" {
		t.Error("Expected non-empty redis host")
	}
}

func TestConfigAPI(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.API.Timeout == 0 {
		t.Error("Expected non-zero API timeout")
	}
}

func TestConfigLogging(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if config.Logging.Level == "" {
		t.Error("Expected non-empty log level")
	}
	if config.Logging.Format == "" {
		t.Error("Expected non-empty log format")
	}
}

func TestConfigString(t *testing.T) {
	config, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	str := config.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}
}
