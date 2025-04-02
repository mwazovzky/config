package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	StringField      string        `env:"STRING_FIELD" required:"true"`
	IntField         int64         `env:"INT_FIELD" required:"true"`
	SliceField       []int64       `env:"SLICE_FIELD" required:"true"`
	StringSliceField []string      `env:"STRING_SLICE_FIELD" required:"true"`
	DurationField    time.Duration `env:"DURATION_FIELD" required:"true"`
	OptionalField    string        `env:"OPTIONAL_FIELD" required:"false"`
	BoolField        bool          `env:"BOOL_FIELD" required:"true"`
	FloatField       float64       `env:"FLOAT_FIELD" required:"true"`
	RangedInt        int           `env:"RANGED_INT" min:"0" max:"100"`
	DefaultStr       string        `env:"DEFAULT_STR" default:"default-value"`
}

func TestLoadConfig(t *testing.T) {
	// Set environment variables for testing
	os.Setenv("STRING_FIELD", "test_string")
	os.Setenv("INT_FIELD", "12345")
	os.Setenv("SLICE_FIELD", "12345,67890")
	os.Setenv("STRING_SLICE_FIELD", "one,two,three")
	os.Setenv("DURATION_FIELD", "60s") // Changed: explicitly specify seconds
	os.Setenv("OPTIONAL_FIELD", "optional_value")
	os.Setenv("BOOL_FIELD", "true")
	os.Setenv("FLOAT_FIELD", "3.14")
	os.Setenv("RANGED_INT", "50")
	// DEFAULT_STR intentionally not set to test default value

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)

	assert.Equal(t, "test_string", cfg.StringField)
	assert.Equal(t, int64(12345), cfg.IntField)
	assert.Equal(t, []int64{12345, 67890}, cfg.SliceField)
	assert.ElementsMatch(t, []string{"one", "two", "three"}, cfg.StringSliceField) // Test string slice
	assert.Equal(t, time.Duration(60)*time.Second, cfg.DurationField)
	assert.Equal(t, "optional_value", cfg.OptionalField)
	assert.True(t, cfg.BoolField)
	assert.Equal(t, 3.14, cfg.FloatField)
	assert.Equal(t, 50, cfg.RangedInt)
	assert.Equal(t, "default-value", cfg.DefaultStr)
}

func TestLoadConfigMissingRequired(t *testing.T) {
	// Unset environment variables to test missing required fields
	os.Unsetenv("STRING_FIELD")
	os.Unsetenv("INT_FIELD")
	os.Unsetenv("SLICE_FIELD")
	os.Unsetenv("STRING_SLICE_FIELD")
	os.Unsetenv("DURATION_FIELD")
	os.Unsetenv("BOOL_FIELD")
	os.Unsetenv("FLOAT_FIELD")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestLoadConfigInvalidValues(t *testing.T) {
	// Set invalid environment variables for testing
	os.Setenv("STRING_FIELD", "test_string")
	os.Setenv("INT_FIELD", "invalid")
	os.Setenv("SLICE_FIELD", "invalid")
	os.Setenv("STRING_SLICE_FIELD", "") // Test empty string slice
	os.Setenv("DURATION_FIELD", "invalid")
	os.Setenv("BOOL_FIELD", "invalid")
	os.Setenv("FLOAT_FIELD", "invalid")
	os.Setenv("RANGED_INT", "-1")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestLoadConfigEmptyValues(t *testing.T) {
	// Set empty environment variables for testing
	os.Setenv("STRING_FIELD", "")
	os.Setenv("INT_FIELD", "")
	os.Setenv("SLICE_FIELD", "")
	os.Setenv("STRING_SLICE_FIELD", "") // Test empty string slice
	os.Setenv("DURATION_FIELD", "")
	os.Setenv("BOOL_FIELD", "")
	os.Setenv("FLOAT_FIELD", "")
	os.Setenv("RANGED_INT", "")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestLoadConfigPartialValues(t *testing.T) {
	// Set partial environment variables for testing
	os.Setenv("STRING_FIELD", "test_string")
	os.Setenv("INT_FIELD", "12345")
	os.Unsetenv("SLICE_FIELD")
	os.Unsetenv("STRING_SLICE_FIELD") // Test missing string slice
	os.Unsetenv("DURATION_FIELD")
	os.Unsetenv("BOOL_FIELD")
	os.Unsetenv("FLOAT_FIELD")
	os.Unsetenv("RANGED_INT")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestUnsupportedType(t *testing.T) {
	// Define a config with an unsupported type (complex64)
	type UnsupportedConfig struct {
		Complex complex64 `env:"COMPLEX"`
	}

	os.Setenv("COMPLEX", "1+2i")

	cfg := &UnsupportedConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestIsTimeType(t *testing.T) {
	// Test with time.Time
	assert.True(t, isTimeType(reflect.TypeOf(time.Time{})))

	// Test with other types
	assert.False(t, isTimeType(reflect.TypeOf("")))
	assert.False(t, isTimeType(reflect.TypeOf(0)))
	assert.False(t, isTimeType(reflect.TypeOf(struct{}{})))
}

func TestNestedStructs(t *testing.T) {
	// Define a nested config structure
	type DatabaseConfig struct {
		Host string `env:"DB_HOST" default:"localhost"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	type ServerConfig struct {
		Port int `env:"SERVER_PORT" default:"8080"`
	}

	type AppConfig struct {
		Database DatabaseConfig
		Server   ServerConfig
		Debug    bool `env:"DEBUG" default:"false"`
	}

	// Set environment variables
	os.Setenv("DB_HOST", "test-db")
	os.Setenv("DB_PORT", "1234")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("DEBUG", "true")

	cfg := &AppConfig{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)

	// Verify nested structs were loaded correctly
	assert.Equal(t, "test-db", cfg.Database.Host)
	assert.Equal(t, 1234, cfg.Database.Port)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.True(t, cfg.Debug)

	// Test with missing required field in nested struct
	type RequiredConfig struct {
		Database struct {
			Host string `env:"REQUIRED_HOST" required:"true"`
		}
	}

	os.Unsetenv("REQUIRED_HOST")

	reqCfg := &RequiredConfig{}
	err = LoadConfig(reqCfg)
	assert.Error(t, err)
}

func TestWithParser(t *testing.T) {
	// Create a mock parser
	mockParser := &StringParser{}

	// Create loader with the custom parser
	loader := NewEnvLoader(
		WithParser(reflect.Bool, mockParser),
	)

	// Verify that the parser was added
	parser, ok := loader.parsers[reflect.Bool]
	assert.True(t, ok)
	assert.Equal(t, mockParser, parser)
}

func TestWithValidator(t *testing.T) {
	// Create a mock validator
	mockValidator := &RequiredValidator{}

	// Create loader with the custom validator
	loader := NewEnvLoader(
		WithValidator(mockValidator),
	)

	// Verify that the validator was added
	found := false
	for _, validator := range loader.validators {
		if validator == mockValidator {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestWithPrefix(t *testing.T) {
	// Create loader with prefix
	prefix := "APP_"
	loader := NewEnvLoader(
		WithPrefix(prefix),
	)

	// Verify the prefix was set
	assert.Equal(t, prefix, loader.prefix)

	// Test that environment variables are properly prefixed
	os.Setenv("APP_TEST", "value")

	type TestConfig struct {
		Test string `env:"TEST"`
	}

	cfg := &TestConfig{}
	err := loader.LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "value", cfg.Test)
}
