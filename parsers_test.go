package config

import (
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStringParser_Parse(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"normal string", "test", "test"},
		{"empty string", "", ""},
		{"with spaces", "test value", "test value"},
		{"with special chars", "test@123!", "test@123!"},
	}

	parser := &StringParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf("")).Elem()
			err := parser.Parse(tt.value, field)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, field.String())
		})
	}
}

func TestInt64Parser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{"valid number", "123", 123, false},
		{"empty string", "", 0, false},
		{"invalid number", "abc", 0, true},
		{"negative number", "-123", -123, false},
	}

	parser := &Int64Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf(int64(0))).Elem()
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Int())
			}
		})
	}
}

func TestSliceParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		typ     reflect.Type
		want    interface{}
		wantErr bool
	}{
		{
			name:  "string slice",
			value: "a,b,c",
			typ:   reflect.TypeOf([]string{}),
			want:  []string{"a", "b", "c"},
		},
		{
			name:  "int64 slice",
			value: "1,2,3",
			typ:   reflect.TypeOf([]int64{}),
			want:  []int64{1, 2, 3},
		},
		{
			name:    "invalid int slice",
			value:   "1,a,3",
			typ:     reflect.TypeOf([]int64{}),
			wantErr: true,
		},
		{
			name:  "empty slice",
			value: "",
			typ:   reflect.TypeOf([]string{}),
			want:  []string(nil),
		},
	}

	parser := &SliceParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(tt.typ).Elem()
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Interface())
			}
		})
	}
}

func TestSliceParserEdgeCases(t *testing.T) {
	parser := &SliceParser{}

	// Test with unsupported element type
	t.Run("unsupported element type", func(t *testing.T) {
		// Create a slice of an unsupported type (e.g., complex64)
		field := reflect.New(reflect.TypeOf([]complex64{})).Elem()
		err := parser.Parse("1,2,3", field)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported slice element type")
	})

	// Test empty value
	t.Run("empty value", func(t *testing.T) {
		field := reflect.New(reflect.TypeOf([]string{})).Elem()
		err := parser.Parse("", field)
		assert.NoError(t, err)
		assert.Equal(t, 0, field.Len())
	})
}

type BoolWithDefault struct {
	Value bool `default:"true"`
}

type FloatWithDefault struct {
	Value float64 `default:"1.23"`
}

func TestBoolParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    bool
		wantErr bool
	}{
		{"true value", "true", true, false},
		{"false value", "false", false, false},
		{"invalid value", "invalid", false, true},
	}

	parser := &BoolParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf(bool(false))).Elem()
			field.SetBool(false)
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Bool())
			}
		})
	}

	// Test with the EnvLoader for default values
	t.Run("empty with default", func(t *testing.T) {
		type TestConfig struct {
			Value bool `env:"TEST_BOOL" default:"true"`
		}

		// Make sure env var is not set
		os.Unsetenv("TEST_BOOL")

		cfg := &TestConfig{}
		loader := NewEnvLoader()
		err := loader.LoadConfig(cfg)

		assert.NoError(t, err)
		assert.Equal(t, true, cfg.Value)
	})
}

func TestFloat64Parser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    float64
		wantErr bool
	}{
		{"valid float", "3.14", 3.14, false},
		{"integer float", "42", 42.0, false},
		{"invalid float", "not-a-float", 0, true},
	}

	parser := &Float64Parser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf(float64(0))).Elem()
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Float())
			}
		})
	}

	// Test with the EnvLoader for default values
	t.Run("empty with default", func(t *testing.T) {
		type TestConfig struct {
			Value float64 `env:"TEST_FLOAT" default:"1.23"`
		}

		// Make sure env var is not set
		os.Unsetenv("TEST_FLOAT")

		cfg := &TestConfig{}
		loader := NewEnvLoader()
		err := loader.LoadConfig(cfg)

		assert.NoError(t, err)
		assert.Equal(t, 1.23, cfg.Value)
	})
}

func TestIntParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64 // Changed from int to int64
		wantErr bool
	}{
		{"valid number", "123", 123, false},
		{"empty string", "", 0, false},
		{"invalid number", "abc", 0, true},
		{"negative number", "-123", -123, false},
		{"zero", "0", 0, false},
		{"large number", "2147483647", 2147483647, false},
	}

	parser := &IntParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf(0)).Elem()
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Int()) // Compare with field.Int() which returns int64
			}
		})
	}
}

func TestDurationParser_Parse(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"valid duration with unit", "5m", 5 * time.Minute, false},
		{"valid duration without unit (seconds)", "30", 30 * time.Second, false},
		{"empty string", "", 0, false},
		{"invalid duration", "invalid", 0, true},
		{"negative duration", "-10s", -10 * time.Second, false},
	}

	parser := &DurationParser{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
			err := parser.Parse(tt.value, field)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, field.Interface())
			}
		})
	}
}

func TestDefaultValues(t *testing.T) {
	type DefaultStruct struct {
		String string  `env:"TEST_STRING" default:"default-string"`
		Int    int     `env:"TEST_INT" default:"42"`
		Float  float64 `env:"TEST_FLOAT" default:"3.14"`
		Bool   bool    `env:"TEST_BOOL" default:"true"`
	}

	// Make sure env vars are not set
	os.Unsetenv("TEST_STRING")
	os.Unsetenv("TEST_INT")
	os.Unsetenv("TEST_FLOAT")
	os.Unsetenv("TEST_BOOL")

	cfg := &DefaultStruct{}
	err := LoadConfig(cfg)

	assert.NoError(t, err)
	assert.Equal(t, "default-string", cfg.String)
	assert.Equal(t, 42, cfg.Int)
	assert.Equal(t, 3.14, cfg.Float)
	assert.Equal(t, true, cfg.Bool)
}
