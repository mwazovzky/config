package config

import (
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseValue_String(t *testing.T) {
	field := reflect.New(reflect.TypeOf("")).Elem()
	assert.NoError(t, parseValue("hello", field))
	assert.Equal(t, "hello", field.String())
}

func TestParseValue_EmptyStringIsNoop(t *testing.T) {
	field := reflect.New(reflect.TypeOf(int64(0))).Elem()
	field.SetInt(42)
	assert.NoError(t, parseValue("", field))
	assert.Equal(t, int64(42), field.Int()) // unchanged
}

func TestParseValue_EmptyStringIsNoop_Slice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	field.Set(reflect.ValueOf([]int64{99}))
	assert.NoError(t, parseValue("", field))
	assert.Equal(t, []int64{99}, field.Interface())
}

func TestParseValue_Int64(t *testing.T) {
	tests := []struct {
		value   string
		want    int64
		wantErr bool
	}{
		{"123", 123, false},
		{"-123", -123, false},
		{"abc", 0, true},
	}
	for _, tt := range tests {
		field := reflect.New(reflect.TypeOf(int64(0))).Elem()
		err := parseValue(tt.value, field)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.want, field.Int())
		}
	}
}

func TestParseValue_Duration(t *testing.T) {
	tests := []struct {
		value   string
		want    time.Duration
		wantErr bool
	}{
		{"5m", 5 * time.Minute, false},
		{"30s", 30 * time.Second, false},
		{"30", 0, true},
		{"invalid", 0, true},
		{"-10s", -10 * time.Second, false},
	}
	for _, tt := range tests {
		field := reflect.New(reflect.TypeOf(time.Duration(0))).Elem()
		err := parseValue(tt.value, field)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.want, field.Interface())
		}
	}
}

func TestParseValue_Bool(t *testing.T) {
	field := reflect.New(reflect.TypeOf(false)).Elem()
	assert.NoError(t, parseValue("true", field))
	assert.True(t, field.Bool())

	assert.Error(t, parseValue("invalid", field))
}

func TestParseValue_Float(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	assert.NoError(t, parseValue("3.14", field))
	assert.Equal(t, 3.14, field.Float())

	assert.Error(t, parseValue("not-a-float", field))
}

func TestParseValue_FloatNaN(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	err := parseValue("NaN", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid float value")
}

func TestParseValue_FloatInf(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	err := parseValue("+Inf", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid float value")
}

func TestParseValue_FloatNegInf(t *testing.T) {
	field := reflect.New(reflect.TypeOf(float64(0))).Elem()
	err := parseValue("-Inf", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid float value")
}

func TestParseValue_UnsupportedType(t *testing.T) {
	field := reflect.New(reflect.TypeOf(complex64(0))).Elem()
	err := parseValue("1+2i", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestParseSlice_StringSlice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]string{})).Elem()
	assert.NoError(t, parseValue("a,b,c", field))
	assert.Equal(t, []string{"a", "b", "c"}, field.Interface())
}

func TestParseSlice_Int64Slice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	assert.NoError(t, parseValue("1,2,3", field))
	assert.Equal(t, []int64{1, 2, 3}, field.Interface())
}

func TestParseSlice_InvalidElement(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	err := parseValue("1,abc,3", field)
	assert.Error(t, err)
}

func TestParseSlice_StructSlice(t *testing.T) {
	type Inner struct{ Port int }
	field := reflect.New(reflect.SliceOf(reflect.TypeOf(Inner{}))).Elem()
	err := parseValue("foo,bar", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestParseSlice_DurationSlice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]time.Duration{})).Elem()
	assert.NoError(t, parseValue("1s,2m,500ms", field))
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Minute, 500 * time.Millisecond}, field.Interface())
}

func TestDefaultValues_Slice(t *testing.T) {
	type Cfg struct {
		Tags []string `env:"CFG_TAGS" default:"a,b,c"`
	}
	unsetEnv(t, "CFG_TAGS")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, []string{"a", "b", "c"}, cfg.Tags)
}

func TestParseSlice_EmptyElement(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	assert.NoError(t, parseValue("1,,3", field))
	assert.Equal(t, []int64{1, 0, 3}, field.Interface())
}

func TestParseSlice_TrailingComma(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	assert.NoError(t, parseValue("1,2,", field))
	assert.Equal(t, []int64{1, 2, 0}, field.Interface())
}

func TestParseSlice_TrimsWhitespace(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]string{})).Elem()
	assert.NoError(t, parseValue("a, b, c", field))
	assert.Equal(t, []string{"a", "b", "c"}, field.Interface())
}

func TestParseSlice_TrimsWhitespace_Int(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	assert.NoError(t, parseValue("1, 2, 3", field))
	assert.Equal(t, []int64{1, 2, 3}, field.Interface())
}

func TestParseSlice_WhitespaceOnlyString(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]string{})).Elem()
	assert.NoError(t, parseValue(" ", field))
	assert.Equal(t, []string{""}, field.Interface())
}

func TestParseSlice_WhitespaceOnlyInt(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]int64{})).Elem()
	assert.NoError(t, parseValue(" ", field))
	assert.Equal(t, []int64{0}, field.Interface())
}

func TestParseSlice_Float64Slice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]float64{})).Elem()
	assert.NoError(t, parseValue("1.1,2.2,3.3", field))
	assert.Equal(t, []float64{1.1, 2.2, 3.3}, field.Interface())
}

func TestParseSlice_BoolSlice(t *testing.T) {
	field := reflect.New(reflect.TypeOf([]bool{})).Elem()
	assert.NoError(t, parseValue("true,false,true", field))
	assert.Equal(t, []bool{true, false, true}, field.Interface())
}

func TestParseSlice_PointerNonStructElement(t *testing.T) {
	field := reflect.New(reflect.SliceOf(reflect.TypeOf((*string)(nil)))).Elem()
	err := parseValue("foo,bar", field)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestDefaultValues_Slice_InvalidElement(t *testing.T) {
	type Cfg struct {
		Nums []int64 `env:"DSIE_NUMS" default:"1,bad,3"`
	}
	unsetEnv(t, "DSIE_NUMS")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DSIE_NUMS:")
}

func TestDefaultValues(t *testing.T) {
	type DefaultStruct struct {
		String string  `env:"CFG_DEFAULT_STR" default:"default-string"`
		Int    int     `env:"CFG_DEFAULT_INT" default:"42"`
		Float  float64 `env:"CFG_DEFAULT_FLOAT" default:"3.14"`
		Bool   bool    `env:"CFG_DEFAULT_BOOL" default:"true"`
	}

	for _, k := range []string{"CFG_DEFAULT_STR", "CFG_DEFAULT_INT", "CFG_DEFAULT_FLOAT", "CFG_DEFAULT_BOOL"} {
		unsetEnv(t, k)
	}

	cfg := &DefaultStruct{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "default-string", cfg.String)
	assert.Equal(t, 42, cfg.Int)
	assert.Equal(t, 3.14, cfg.Float)
	assert.Equal(t, true, cfg.Bool)
}
