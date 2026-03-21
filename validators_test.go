package config

import (
	"fmt"
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDurationBound_DurationString(t *testing.T) {
	n, err := parseDurationBound("1s")
	assert.NoError(t, err)
	assert.Equal(t, int64(time.Second), n)
}

func TestParseDurationBound_NanosecondInteger(t *testing.T) {
	// Backward-compat: bare integer falls through ParseDuration to ParseInt.
	n, err := parseDurationBound("1000000000")
	assert.NoError(t, err)
	assert.Equal(t, int64(time.Second), n)
}

func TestParseDurationBound_InvalidString(t *testing.T) {
	_, err := parseDurationBound("xyz")
	assert.Error(t, err)
}

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

func TestCheckRange_IntInRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(5))).Elem()
	field.SetInt(5)
	tag := reflect.StructTag(`min:"0" max:"10"`)
	assert.NoError(t, checkRange(field, tag))
}

func TestCheckRange_IntBelowMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(-1)
	tag := reflect.StructTag(`min:"0" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_IntAboveMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(11)
	tag := reflect.StructTag(`min:"0" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_FloatInRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.5)
	tag := reflect.StructTag(`min:"0.0" max:"10.0"`)
	assert.NoError(t, checkRange(field, tag))
}

func TestCheckRange_FloatOutOfRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(10.1)
	tag := reflect.StructTag(`min:"0.0" max:"10.0"`)
	assert.Error(t, checkRange(field, tag))
}

func TestCheckRange_CustomErrorMessage(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(11)
	tag := reflect.StructTag(`min:"0" max:"10" range_error:"custom error"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom error")
	assert.NotContains(t, err.Error(), "11 is greater than")
}

func TestCheckRange_UintBelowMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(0)
	tag := reflect.StructTag(`min:"10" max:"100"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_DurationInRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(5 * time.Second))
	tag := reflect.StructTag(`min:"1000000000" max:"60000000000"`)
	assert.NoError(t, checkRange(field, tag))
}

func TestCheckRange_DurationBelowMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(500 * time.Millisecond))
	tag := reflect.StructTag(`min:"1000000000"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500ms")
}

func TestCheckRange_DurationAboveMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(120 * time.Second))
	tag := reflect.StructTag(`max:"60000000000"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "2m0s")
}

func TestCheckRange_UintInRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(50)
	tag := reflect.StructTag(`min:"0" max:"100"`)
	assert.NoError(t, checkRange(field, tag))
}

func TestCheckRange_UintAboveMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(101)
	tag := reflect.StructTag(`min:"0" max:"100"`)
	assert.Error(t, checkRange(field, tag))
}

func TestCheckRange_Float32InRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float32(0))).Elem()
	field.SetFloat(float64(float32(3.14)))
	tag := reflect.StructTag(`min:"3.14" max:"3.14"`)
	assert.NoError(t, checkRange(field, tag))
}

func TestCheckRange_NoTags(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(9999)
	assert.NoError(t, checkRange(field, reflect.StructTag("")))
}

func TestCheckRange_StringReturnsError(t *testing.T) {
	// Range tags on non-numeric types return an error.
	field := reflect.New(reflect.TypeOf("")).Elem()
	field.SetString("hello")
	tag := reflect.StructTag(`min:"0" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "range validation not supported")
}

func TestCheckRange_BoolReturnsError(t *testing.T) {
	field := reflect.New(reflect.TypeOf(false)).Elem()
	field.SetBool(true)
	tag := reflect.StructTag(`min:"0" max:"1"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "range validation not supported")
}

func TestCheckRange_InvalidMinValue(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(5)
	tag := reflect.StructTag(`min:"invalid"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
}

func TestCheckRange_InvalidMaxValue(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(5)
	tag := reflect.StructTag(`max:"invalid"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
}

func TestCheckRange_InvalidMaxValue_Float(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.0)
	tag := reflect.StructTag(`max:"bad"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
}

func TestCheckRange_InvalidMinValue_Uint(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(5)
	tag := reflect.StructTag(`min:"invalid"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
}

func TestCheckRange_InvalidMinValue_Float(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.0)
	tag := reflect.StructTag(`min:"bad"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
}

func TestCheckRange_InvalidMaxValue_Uint(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(5)
	tag := reflect.StructTag(`max:"bad"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
}

func TestCheckRange_IntAtMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(0)
	assert.NoError(t, checkRange(field, reflect.StructTag(`min:"0" max:"10"`)))
}

func TestCheckRange_IntAtMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(10)
	assert.NoError(t, checkRange(field, reflect.StructTag(`min:"0" max:"10"`)))
}

func TestCheckRange_IntMinOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(-1)
	err := checkRange(field, reflect.StructTag(`min:"0"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_IntMaxOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(11)
	err := checkRange(field, reflect.StructTag(`max:"10"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_FloatMinOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(-1.0)
	err := checkRange(field, reflect.StructTag(`min:"0.0"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_FloatMaxOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(11.0)
	err := checkRange(field, reflect.StructTag(`max:"10.0"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_UintMinOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(0)
	err := checkRange(field, reflect.StructTag(`min:"1"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_UintMaxOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(101)
	err := checkRange(field, reflect.StructTag(`max:"100"`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_MinGreaterThanMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(5)
	tag := reflect.StructTag(`min:"100" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
}

func TestCheckRange_MinGreaterThanMax_Uint(t *testing.T) {
	field := reflect.New(reflect.TypeOf(uint(0))).Elem()
	field.SetUint(5)
	tag := reflect.StructTag(`min:"100" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
}

func TestCheckRange_MinGreaterThanMax_Float(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.0)
	tag := reflect.StructTag(`min:"10.0" max:"1.0"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
}

func TestCheckRange_MinGreaterThanMax_Duration(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(5 * time.Second))
	tag := reflect.StructTag(`min:"60000000000" max:"1000000000"`) // 1m > 1s
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
	assert.Contains(t, err.Error(), "1m0s")
	assert.NotContains(t, err.Error(), "60000000000")
}

func TestCheckRange_CustomErrorNotAppliedOnMinGreaterThanMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int(0))).Elem()
	field.SetInt(5)
	tag := reflect.StructTag(`min:"100" max:"10" range_error:"custom"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range")
	assert.NotContains(t, err.Error(), "custom")
}

func TestCheckRange_CustomErrorNotAppliedOnMinGreaterThanMax_Duration(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(5 * time.Second))
	tag := reflect.StructTag(`min:"60000000000" max:"1000000000" range_error:"custom"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "1m0s")
	assert.NotContains(t, err.Error(), "custom")
}

func TestCheckRange_FloatNaNMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.0)
	tag := reflect.StructTag(`min:"NaN"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
}

func TestCheckRange_FloatInfMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(5.0)
	tag := reflect.StructTag(`max:"+Inf"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
}

func TestCheckRange_Float32OutOfRange_ErrorMessage(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float32(0))).Elem()
	field.SetFloat(float64(float32(5.5)))
	tag := reflect.StructTag(`min:"10.0" max:"20.0"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
	// Formatted at float32 precision — "5.5", not the long float64 expansion.
	assert.Contains(t, err.Error(), "5.5")
}

func TestCheckRange_FloatNaNValue_WithRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(math.NaN())
	tag := reflect.StructTag(`min:"0.0" max:"10.0"`)
	err := checkRange(field, tag)
	assert.Error(t, err) // previously passed silently
	assert.Contains(t, err.Error(), "finite")
}

func TestCheckRange_FloatInfValue_WithRange(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(math.Inf(1))
	tag := reflect.StructTag(`min:"0.0" max:"10.0"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "finite")
}

func TestCheckRange_FloatInfValue_CustomError(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(math.Inf(1))
	tag := reflect.StructTag(`min:"0.0" max:"10.0" range_error:"custom"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom")
	assert.NotContains(t, err.Error(), "finite")
}

func TestCheckRange_FloatNaNValue_MinOnly(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(math.NaN())
	tag := reflect.StructTag(`min:"0.0"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "finite")
}

func TestCheckRange_FloatNaNValue_CustomError(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	field.SetFloat(math.NaN())
	tag := reflect.StructTag(`min:"0.0" max:"10.0" range_error:"custom"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "custom")
	assert.NotContains(t, err.Error(), "finite")
}

func TestCheckRange_SliceReturnsError(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int{})).Elem()
	tag := reflect.StructTag(`min:"1" max:"10"`)
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on slice fields")
}

func TestCheckRange_Duration_InvalidMinTag(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(5 * time.Second))
	tag := reflect.StructTag(`min:"xyz"`) // unparseable
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
}

func TestCheckRange_Duration_InvalidMaxTag(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(5 * time.Second))
	tag := reflect.StructTag(`max:"xyz"`) // unparseable
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
}

func TestCheckRange_Duration_DurationStringMin(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(500 * time.Millisecond))
	tag := reflect.StructTag(`min:"1s"`) // 500ms < 1s → out of range
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestCheckRange_Duration_DurationStringMax(t *testing.T) {
	field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
	field.Set(reflect.ValueOf(120 * time.Second))
	tag := reflect.StructTag(`max:"60s"`) // 120s > 60s → out of range
	err := checkRange(field, tag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}
