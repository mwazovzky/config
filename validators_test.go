package config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckRequired_Provided(t *testing.T) {
	tag := reflect.StructTag(`required:"true"`)
	assert.NoError(t, checkRequired(tag, true))
}

func TestCheckRequired_Absent(t *testing.T) {
	tag := reflect.StructTag(`required:"true"`)
	err := checkRequired(tag, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required field is missing")
}

func TestCheckRequired_NotRequired(t *testing.T) {
	tag := reflect.StructTag(`required:"false"`)
	assert.NoError(t, checkRequired(tag, false))
}

func TestCheckRequired_NoTag(t *testing.T) {
	assert.NoError(t, checkRequired(reflect.StructTag(""), false))
}

func TestCheckRequired_InvalidValueReturnsError(t *testing.T) {
	for _, val := range []string{"True", "TRUE", "yes", "1"} {
		tag := reflect.StructTag(fmt.Sprintf(`required:"%s"`, val))
		err := checkRequired(tag, true)
		assert.Error(t, err, "value %q should be rejected", val)
		assert.Contains(t, err.Error(), "invalid required tag value")
	}
}
