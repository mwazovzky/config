package config

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequiredValidator_Validate(t *testing.T) {
	tests := []struct {
		name     string
		field    interface{}
		required bool
		wantErr  bool
	}{
		{"empty string required", "", true, true},
		{"non-empty string required", "test", true, false},
		{"empty string not required", "", false, false},
		{"empty slice required", []string{}, true, true},
		{"non-empty slice required", []string{"test"}, true, false},
		{"zero int required", 0, true, true},
		{"non-zero int required", 42, true, false},
	}

	validator := &RequiredValidator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := reflect.ValueOf(tt.field)
			tag := reflect.StructTag("")
			if tt.required {
				tag = `required:"true"`
			}

			err := validator.Validate(value, tag)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_isZeroValue(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		want  bool
	}{
		{"empty string", "", true},
		{"non-empty string", "test", false},
		{"zero int", 0, true},
		{"non-zero int", 42, false},
		{"empty slice", []string{}, true},
		{"non-empty slice", []string{"test"}, false},
		{"zero float", 0.0, true},
		{"non-zero float", 3.14, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isZeroValue(reflect.ValueOf(tt.value))
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRangeValidator_Validate(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		tag     string
		wantErr bool
	}{
		{"int in range", 5, `min:"0" max:"10"`, false},
		{"int below min", -1, `min:"0" max:"10"`, true},
		{"int above max", 11, `min:"0" max:"10"`, true},
		{"float in range", 5.5, `min:"0.0" max:"10.0"`, false},
		{"float out of range", 10.1, `min:"0.0" max:"10.0"`, true},
		{"custom error message", 11, `min:"0" max:"10" range_error:"custom error"`, true},
	}

	validator := &RangeValidator{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := reflect.ValueOf(tt.value)
			tag := reflect.StructTag(tt.tag)
			err := validator.Validate(value, tag)
			if tt.wantErr {
				assert.Error(t, err)
				if strings.Contains(tt.tag, "range_error") {
					assert.Contains(t, err.Error(), "custom error")
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIntRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		value   interface{}
		min     string
		max     string
		wantErr bool
		errMsg  string
	}{
		{"invalid min value", 5, "invalid", "10", true, "invalid min value"},
		{"invalid max value", 5, "0", "invalid", true, "invalid max value"},
		{"unsupported type", "not an int", "0", "10", true, "unsupported integer type"},
		{"no min or max", 5, "", "", false, ""},
		{"only min", 5, "0", "", false, ""},
		{"only max", 5, "", "10", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIntRange(tt.value, tt.min, tt.max)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFloatRange_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		min     string
		max     string
		wantErr bool
		errMsg  string
	}{
		{"invalid min value", 5.5, "invalid", "10.0", true, "invalid min value"},
		{"invalid max value", 5.5, "0.0", "invalid", true, "invalid max value"},
		{"no min or max", 5.5, "", "", false, ""},
		{"only min", 5.5, "0.0", "", false, ""},
		{"only max", 5.5, "", "10.0", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFloatRange(tt.value, tt.min, tt.max)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
