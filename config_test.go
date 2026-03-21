package config

import (
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// unsetEnv makes an env var truly absent while ensuring cleanup restores the
// original state. t.Setenv registers the restore; os.Unsetenv removes the var
// immediately. Not parallel-safe — env vars are process-global.
func unsetEnv(t *testing.T, key string) {
	t.Helper()
	t.Setenv(key, "")   // registers restore-on-cleanup
	os.Unsetenv(key)     // make truly absent now
}

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
	t.Setenv("STRING_FIELD", "test_string")
	t.Setenv("INT_FIELD", "12345")
	t.Setenv("SLICE_FIELD", "12345,67890")
	t.Setenv("STRING_SLICE_FIELD", "one,two,three")
	t.Setenv("DURATION_FIELD", "60s")
	t.Setenv("OPTIONAL_FIELD", "optional_value")
	t.Setenv("BOOL_FIELD", "true")
	t.Setenv("FLOAT_FIELD", "3.14")
	t.Setenv("RANGED_INT", "50")
	unsetEnv(t, "DEFAULT_STR")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)

	assert.Equal(t, "test_string", cfg.StringField)
	assert.Equal(t, int64(12345), cfg.IntField)
	assert.Equal(t, []int64{12345, 67890}, cfg.SliceField)
	assert.ElementsMatch(t, []string{"one", "two", "three"}, cfg.StringSliceField)
	assert.Equal(t, time.Duration(60)*time.Second, cfg.DurationField)
	assert.Equal(t, "optional_value", cfg.OptionalField)
	assert.True(t, cfg.BoolField)
	assert.Equal(t, 3.14, cfg.FloatField)
	assert.Equal(t, 50, cfg.RangedInt)
	assert.Equal(t, "default-value", cfg.DefaultStr)
}

func TestLoadConfigMissingRequired(t *testing.T) {
	// Truly absent vars (not just empty) trigger required failures.
	keys := []string{
		"STRING_FIELD", "INT_FIELD", "SLICE_FIELD",
		"STRING_SLICE_FIELD", "DURATION_FIELD", "BOOL_FIELD", "FLOAT_FIELD",
	}
	for _, key := range keys {
		unsetEnv(t, key)
	}

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestLoadConfigInvalidValues(t *testing.T) {
	t.Setenv("STRING_FIELD", "test_string")
	t.Setenv("INT_FIELD", "invalid")
	t.Setenv("SLICE_FIELD", "12345")
	t.Setenv("STRING_SLICE_FIELD", "a")
	t.Setenv("DURATION_FIELD", "60s")
	t.Setenv("BOOL_FIELD", "true")
	t.Setenv("FLOAT_FIELD", "1.0")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestLoadConfigEmptyValues(t *testing.T) {
	// Setting a var to "" counts as provided; required passes and fields take zero values.
	t.Setenv("STRING_FIELD", "")
	t.Setenv("INT_FIELD", "")
	t.Setenv("SLICE_FIELD", "")
	t.Setenv("STRING_SLICE_FIELD", "")
	t.Setenv("DURATION_FIELD", "")
	t.Setenv("BOOL_FIELD", "")
	t.Setenv("FLOAT_FIELD", "")
	t.Setenv("RANGED_INT", "")
	unsetEnv(t, "DEFAULT_STR")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.StringField)
	assert.Equal(t, int64(0), cfg.IntField)
	assert.Nil(t, cfg.SliceField)
	assert.Nil(t, cfg.StringSliceField)
	assert.Equal(t, time.Duration(0), cfg.DurationField)
	assert.False(t, cfg.BoolField)
	assert.Equal(t, 0.0, cfg.FloatField)
	assert.Equal(t, "default-value", cfg.DefaultStr)
	assert.Equal(t, 0, cfg.RangedInt)
}

func TestLoadConfigPartialValues(t *testing.T) {
	t.Setenv("STRING_FIELD", "test_string")
	t.Setenv("INT_FIELD", "12345")
	// Leave the remaining required fields absent to trigger required failures.
	for _, key := range []string{"SLICE_FIELD", "STRING_SLICE_FIELD", "DURATION_FIELD", "BOOL_FIELD", "FLOAT_FIELD"} {
		unsetEnv(t, key)
	}

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestRequiredPassesForZeroInt(t *testing.T) {
	// PORT=0 with required:"true" must succeed — required checks absence, not zero value.
	type Cfg struct {
		Port int `env:"ZERO_PORT" required:"true"`
	}
	t.Setenv("ZERO_PORT", "0")
	err := LoadConfig(&Cfg{})
	assert.NoError(t, err)
}

func TestRequiredFailsWhenAbsent(t *testing.T) {
	type Cfg struct {
		Port int `env:"ABSENT_PORT" required:"true"`
	}
	unsetEnv(t, "ABSENT_PORT")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestRequiredPassesWithDefault(t *testing.T) {
	type Cfg struct {
		Port int `env:"DEFAULT_PORT" required:"true" default:"8080"`
	}
	unsetEnv(t, "DEFAULT_PORT")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Port)
}

func TestUnsupportedType(t *testing.T) {
	type UnsupportedConfig struct {
		Complex complex64 `env:"COMPLEX"`
	}
	t.Setenv("COMPLEX", "1+2i")

	cfg := &UnsupportedConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestIsTimeType(t *testing.T) {
	assert.True(t, isTimeType(reflect.TypeOf(time.Time{})))
	assert.False(t, isTimeType(reflect.TypeOf("")))
	assert.False(t, isTimeType(reflect.TypeOf(0)))
	assert.False(t, isTimeType(reflect.TypeOf(struct{}{})))
}

func TestNestedStructs(t *testing.T) {
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

	t.Setenv("DB_HOST", "test-db")
	t.Setenv("DB_PORT", "1234")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("DEBUG", "true")

	cfg := &AppConfig{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "test-db", cfg.Database.Host)
	assert.Equal(t, 1234, cfg.Database.Port)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.True(t, cfg.Debug)

	type RequiredConfig struct {
		Database struct {
			Host string `env:"REQUIRED_HOST" required:"true"`
		}
	}
	unsetEnv(t, "REQUIRED_HOST")
	reqCfg := &RequiredConfig{}
	err = LoadConfig(reqCfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "REQUIRED_HOST:")
	assert.Contains(t, err.Error(), "Database") // outer field name from loadStruct wrapping
}

func TestWithPrefix(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("APP_"))

	t.Setenv("APP_TEST", "value")

	type TestConfig struct {
		Test string `env:"TEST"`
	}
	cfg := &TestConfig{}
	err := loader.LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "value", cfg.Test)
}

func TestLoadConfig_NonStructPointer(t *testing.T) {
	err := LoadConfig(new(int))
	assert.True(t, errors.Is(err, ErrNotStructPointer))
}

func TestLoadConfig_RangeValidation(t *testing.T) {
	t.Setenv("STRING_FIELD", "test")
	t.Setenv("INT_FIELD", "1")
	t.Setenv("SLICE_FIELD", "1")
	t.Setenv("STRING_SLICE_FIELD", "a")
	t.Setenv("DURATION_FIELD", "1s")
	t.Setenv("BOOL_FIELD", "true")
	t.Setenv("FLOAT_FIELD", "1.0")
	t.Setenv("RANGED_INT", "-1")

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestLoadConfig_UntaggedFieldSkipped(t *testing.T) {
	type ConfigWithUntagged struct {
		Tagged   string `env:"TAGGED_FIELD"`
		Untagged string
	}
	t.Setenv("TAGGED_FIELD", "hello")

	cfg := &ConfigWithUntagged{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "hello", cfg.Tagged)
	assert.Equal(t, "", cfg.Untagged)
}

func TestLoadConfig_TimeFieldWithoutEnvTagSkipped(t *testing.T) {
	type Cfg struct {
		T time.Time // no env tag — must be skipped silently
		S string    `env:"TTWET_S" default:"ok"`
	}
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "ok", cfg.S)
	assert.True(t, cfg.T.IsZero())
}

func TestLoadConfig_TimeFieldReturnsError(t *testing.T) {
	type ConfigWithTime struct {
		T time.Time `env:"SOME_TIME"`
	}
	t.Setenv("SOME_TIME", "2024-01-01")
	err := LoadConfig(&ConfigWithTime{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SOME_TIME:")
	assert.Contains(t, err.Error(), "time.Time is not supported")
}

func TestLoadConfig_TimeFieldAbsentReturnsError(t *testing.T) {
	type Cfg struct {
		T time.Time `env:"ABSENT_TIME"`
	}
	unsetEnv(t, "ABSENT_TIME")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ABSENT_TIME:")
	assert.Contains(t, err.Error(), "time.Time is not supported")
}

func TestLoadConfig_TimeFieldRequiredReturnsNotSupported(t *testing.T) {
	type Cfg struct {
		T time.Time `env:"REQ_TIME" required:"true"`
	}
	unsetEnv(t, "REQ_TIME")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time.Time is not supported")
	assert.NotContains(t, err.Error(), "required")
}

func TestLoadConfig_TimeFieldDefaultReturnsNotSupported(t *testing.T) {
	type Cfg struct {
		T time.Time `env:"DEF_TIME" default:"2024-01-01"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "time.Time is not supported")
	assert.NotContains(t, err.Error(), "2024-01-01")
}

func TestLoadConfig_DurationRangeValidation(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DRV_TIMEOUT" min:"1000000000" max:"60000000000"` // 1s–60s
	}
	t.Setenv("DRV_TIMEOUT", "5s")
	err := LoadConfig(&Cfg{})
	assert.NoError(t, err)

	t.Setenv("DRV_TIMEOUT", "120s")
	err = LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestLoadConfig_DurationSlice(t *testing.T) {
	type Cfg struct {
		Timeouts []time.Duration `env:"TIMEOUTS"`
	}
	t.Setenv("TIMEOUTS", "1s,2m,500ms")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Minute, 500 * time.Millisecond}, cfg.Timeouts)
}

func TestDecoder(t *testing.T) {
	type Cfg struct {
		Addr customAddr `env:"LISTEN_ADDR"`
	}
	t.Setenv("LISTEN_ADDR", "localhost:8080")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, customAddr("localhost:8080"), cfg.Addr)
}

// customAddr implements Decoder for testing.
type customAddr string

func (a *customAddr) Decode(value string) error {
	*a = customAddr(value)
	return nil
}

func TestDecoder_SliceElement(t *testing.T) {
	// Decoder should also work for slice elements.
	type Cfg struct {
		Addrs []customAddr `env:"ADDRS"`
	}
	t.Setenv("ADDRS", "host1:80,host2:443")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, []customAddr{"host1:80", "host2:443"}, cfg.Addrs)
}

func TestDecoder_SliceElement_Error(t *testing.T) {
	type Cfg struct {
		Addrs []errorAddr `env:"ERR_ADDRS"`
	}
	t.Setenv("ERR_ADDRS", "good,bad")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode error")
	assert.Contains(t, err.Error(), "ERR_ADDRS:")
}

func TestIntWidths(t *testing.T) {
	type Cfg struct {
		I8  int8   `env:"IW_I8"`
		I16 int16  `env:"IW_I16"`
		I32 int32  `env:"IW_I32"`
		U8  uint8  `env:"IW_U8"`
		U16 uint16 `env:"IW_U16"`
		U32 uint32 `env:"IW_U32"`
		U64 uint64 `env:"IW_U64"`
		F32 float32 `env:"IW_F32"`
	}
	t.Setenv("IW_I8", "-5")
	t.Setenv("IW_I16", "1000")
	t.Setenv("IW_I32", "-100000")
	t.Setenv("IW_U8", "255")
	t.Setenv("IW_U16", "65535")
	t.Setenv("IW_U32", "4294967295")
	t.Setenv("IW_U64", "18446744073709551615")
	t.Setenv("IW_F32", "3.14")

	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, int8(-5), cfg.I8)
	assert.Equal(t, int16(1000), cfg.I16)
	assert.Equal(t, int32(-100000), cfg.I32)
	assert.Equal(t, uint8(255), cfg.U8)
	assert.Equal(t, uint16(65535), cfg.U16)
	assert.Equal(t, uint32(4294967295), cfg.U32)
	assert.Equal(t, uint64(18446744073709551615), cfg.U64)
	assert.InDelta(t, float32(3.14), cfg.F32, 0.001)
}

func TestDurationRangeError_HumanReadable(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DR_TIMEOUT" min:"1000000000"` // 1s minimum
	}
	t.Setenv("DR_TIMEOUT", "500ms")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500ms")
	assert.NotContains(t, err.Error(), "500000000")
}

func TestDecoder_Error(t *testing.T) {
	type Cfg struct {
		Addr errorAddr `env:"ERR_ADDR"`
	}
	t.Setenv("ERR_ADDR", "bad")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode error")
	assert.Contains(t, err.Error(), "ERR_ADDR:")
}

type errorAddr string

func (a *errorAddr) Decode(value string) error {
	return fmt.Errorf("decode error: %s", value)
}

func TestIntWidths_Overflow(t *testing.T) {
	type Cfg8 struct {
		V int8 `env:"OVF_I8"`
	}
	t.Setenv("OVF_I8", "128")
	assert.Error(t, LoadConfig(&Cfg8{}))

	type CfgU8 struct {
		V uint8 `env:"OVF_U8"`
	}
	t.Setenv("OVF_U8", "256")
	assert.Error(t, LoadConfig(&CfgU8{}))

	type Cfg16 struct {
		V int16 `env:"OVF_I16"`
	}
	t.Setenv("OVF_I16", "32768")
	assert.Error(t, LoadConfig(&Cfg16{}))
}

func TestDecoder_ExplicitEmptyEnvVar_CallsDecode(t *testing.T) {
	// When the env var is explicitly set to "" (provided=true), Decode IS called,
	// overwriting any pre-existing value. Distinct from absent fields (Decode skipped).
	type Cfg struct {
		Addr customAddr `env:"EMPTY_ADDR"`
	}
	t.Setenv("EMPTY_ADDR", "")
	cfg := &Cfg{Addr: customAddr("initial")} // non-zero sentinel
	assert.NoError(t, LoadConfig(cfg))
	// Decode("") was called → field is now customAddr(""), not "initial".
	assert.Equal(t, customAddr(""), cfg.Addr)
}

func TestDecoder_StructType(t *testing.T) {
	type Cfg struct {
		Addr structAddr `env:"STRUCT_ADDR"`
	}
	t.Setenv("STRUCT_ADDR", "127.0.0.1:9000")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9000", cfg.Addr.Raw)
}

type structAddr struct {
	Raw string
}

func (a *structAddr) Decode(value string) error {
	a.Raw = value
	return nil
}

func TestRequiredPassesWithEmptyDefault(t *testing.T) {
	type Cfg struct {
		S string `env:"EMPTY_DEFAULT_FIELD" required:"true" default:""`
	}
	unsetEnv(t, "EMPTY_DEFAULT_FIELD")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, "", cfg.S)
}

func TestLoadConfig_NilPointer(t *testing.T) {
	type Cfg struct{ X string }
	err := LoadConfig((*Cfg)(nil))
	assert.True(t, errors.Is(err, ErrNilPointer))
}

func TestLoadConfig_UntypedNil(t *testing.T) {
	err := LoadConfig(nil)
	assert.True(t, errors.Is(err, ErrNilConfig))
}

func TestLoadConfig_PointerNestedStructErrors(t *testing.T) {
	// The error fires in loadStruct before any env var is read, so env var
	// state (set, absent, empty) is irrelevant — all three cases are identical.
	type Inner struct {
		Port int `env:"PNS_PORT" default:"9090"`
	}
	type Cfg struct {
		DB *Inner `env:"PNS_DB"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct is not supported")
}

func TestLoadConfig_PointerDecoderStructRejected(t *testing.T) {
	// *structAddr implements Decoder, but pointer-to-struct fields are still
	// rejected — the Decoder check operates on **T, not *T.
	type Cfg struct {
		Addr *structAddr `env:"PDS_ADDR"`
	}
	t.Setenv("PDS_ADDR", "host:1234")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct is not supported")
}

func TestLoadConfig_PointerNestedStructNoEnvTag(t *testing.T) {
	type Inner struct {
		Port int `env:"PNSNET_PORT" default:"9090"`
	}
	type Cfg struct {
		DB     *Inner // no env tag — pointer-to-struct must still be rejected
		Normal string `env:"PNSNET_NORMAL" default:"ok"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct is not supported")
}

func TestLoadConfig_PointerStructSliceErrors(t *testing.T) {
	// []*Inner is not supported; the error must surface at LoadConfig level.
	// The error fires in loadStruct before any env var is read, so env var
	// state (set, absent, empty) is irrelevant.
	type Inner struct {
		Port int `env:"PSSE_PORT"`
	}
	type Cfg struct {
		Nodes []*Inner `env:"PSSE_NODES"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slice of pointer-to-struct is not supported")
}

func TestLoadConfig_PointerStructSliceAbsentEnvVar(t *testing.T) {
	type Inner struct{ Port int `env:"PSSA_PORT"` }
	type Cfg struct {
		Nodes []*Inner `env:"PSSA_NODES"`
	}
	unsetEnv(t, "PSSA_NODES")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slice of pointer-to-struct is not supported")
}

func TestLoadConfig_PointerTimeFieldError(t *testing.T) {
	// *time.Time hits the ptr-to-struct guard (different error from plain time.Time).
	type Cfg struct {
		T *time.Time `env:"PTF_T"`
	}
	t.Setenv("PTF_T", "2024-01-01")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pointer to struct is not supported")
}

func TestLoadConfig_DurationInvalidMinTag(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DIT_TIMEOUT" min:"xyz"` // unparseable
	}
	t.Setenv("DIT_TIMEOUT", "5s")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid min value")
	assert.NotContains(t, err.Error(), "nanoseconds")
}

func TestLoadConfig_DurationInvalidMaxTag(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DIMAXT_TIMEOUT" max:"xyz"` // unparseable
	}
	t.Setenv("DIMAXT_TIMEOUT", "5s")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid max value")
	assert.NotContains(t, err.Error(), "nanoseconds")
}

func TestLoadConfig_DurationRangeTag_MixedFormats(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DRFMIX_TIMEOUT" min:"1s" max:"60000000000"`
	}
	// In-range: 30s is within [1s, 60s].
	t.Setenv("DRFMIX_TIMEOUT", "30s")
	assert.NoError(t, LoadConfig(&Cfg{}))
	// Out-of-range: 120s exceeds max 60s.
	t.Setenv("DRFMIX_TIMEOUT", "120s")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestLoadConfig_DurationRangeTag_DurationStringFormat(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DRTSF_TIMEOUT" min:"1s" max:"60s"`
	}
	// In-range: 5s is within [1s, 60s].
	t.Setenv("DRTSF_TIMEOUT", "5s")
	assert.NoError(t, LoadConfig(&Cfg{}))
	// Out-of-range: 120s exceeds max 60s.
	t.Setenv("DRTSF_TIMEOUT", "120s")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
	assert.Contains(t, err.Error(), "2m0s")
}

func TestLoadConfig_SliceRangeTagReturnsError(t *testing.T) {
	type Cfg struct {
		Tags []string `env:"SRT_TAGS" min:"1"`
	}
	t.Setenv("SRT_TAGS", "a,b")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not supported on slice fields")
	assert.Contains(t, err.Error(), "SRT_TAGS:")
}

func TestRangeFailsOnZeroValueFromEmptyEnvVar(t *testing.T) {
	type Cfg struct {
		N int `env:"ZERO_RANGE_N" min:"1"`
	}
	t.Setenv("ZERO_RANGE_N", "")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestRangeFailsOnAbsentNonRequiredField(t *testing.T) {
	// Truly absent var: no t.Setenv, no default — field keeps zero value,
	// which still must pass range validation.
	type Cfg struct {
		N int `env:"ABSENT_RANGE_N" min:"1"`
	}
	unsetEnv(t, "ABSENT_RANGE_N")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestLoadConfig_CustomRangeError(t *testing.T) {
	type Cfg struct {
		Workers int `env:"CRE_WORKERS" min:"1" max:"64" range_error:"WORKERS must be between 1 and 64"`
	}
	t.Setenv("CRE_WORKERS", "0")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WORKERS must be between 1 and 64")
	assert.NotContains(t, err.Error(), "0 is less than")
}

func TestLoadConfig_DurationCustomRangeError(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DCRE_TIMEOUT" min:"1000000000" range_error:"timeout must be at least 1s"`
	}
	t.Setenv("DCRE_TIMEOUT", "500ms")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "timeout must be at least 1s")
	assert.NotContains(t, err.Error(), "500ms")
}

func TestWithPrefix_DefaultFallback(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("PFX_"))
	type Cfg struct {
		Port int `env:"PORT" default:"9090"`
	}
	unsetEnv(t, "PFX_PORT")
	cfg := &Cfg{}
	err := loader.LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 9090, cfg.Port)
}

func TestLoadConfig_UnexportedPointerStructSliceSkipped(t *testing.T) {
	type Inner struct {
		Port int `env:"UPSSS_PORT" default:"9090"`
	}
	type Cfg struct {
		nodes  []*Inner // unexported — must be silently skipped
		Normal string   `env:"UPSSS_NORMAL" default:"ok"`
	}
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "ok", cfg.Normal)
	assert.Nil(t, cfg.nodes)
}

func TestLoadConfig_PointerStructSliceNoEnvTag(t *testing.T) {
	type Inner struct {
		Port int `env:"PSSNET_PORT" default:"9090"`
	}
	type Cfg struct {
		Nodes  []*Inner // no env tag — pointer-to-struct slice must still be rejected
		Normal string   `env:"PSSNET_NORMAL" default:"ok"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slice of pointer-to-struct is not supported")
}

func TestLoadConfig_PointerTimeSliceFieldError(t *testing.T) {
	type Cfg struct {
		T []*time.Time `env:"PTSF_T"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "[]*time.Time is not supported")
}

func TestLoadConfig_PointerTimeSliceNoEnvTag(t *testing.T) {
	type Cfg struct {
		T      []*time.Time // no env tag — must still be rejected
		Normal string       `env:"PTSNET_NORMAL" default:"ok"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "[]*time.Time is not supported")
}

func TestLoadConfig_UnexportedPointerStructSkipped(t *testing.T) {
	type Inner struct {
		Port int `env:"UPSS_PORT" default:"9090"`
	}
	type Cfg struct {
		db     *Inner // unexported pointer-to-struct — must be silently skipped
		Normal string `env:"UPSS_NORMAL" default:"ok"`
	}
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "ok", cfg.Normal)
	assert.Nil(t, cfg.db) // unexported, never touched
}

func TestLoadConfig_UnexportedNestedStructSkipped(t *testing.T) {
	type Inner struct {
		Port int `env:"UNEXPORTED_INNER_PORT" default:"9999"`
	}
	type Cfg struct {
		inner  Inner
		Normal string `env:"UNEXPORTED_INNER_NORMAL" default:"ok"`
	}
	cfg := &Cfg{}
	assert.NotPanics(t, func() {
		err := LoadConfig(cfg)
		assert.NoError(t, err)
	})
	assert.Equal(t, "ok", cfg.Normal)
	assert.Equal(t, 0, cfg.inner.Port) // unexported, never set
}

func TestLoadConfig_UnexportedFieldSkipped(t *testing.T) {
	type Cfg struct {
		unexported string `env:"UNEXPORTED_FIELD"`
		Exported   string `env:"EXPORTED_FIELD" default:"ok"`
	}
	t.Setenv("UNEXPORTED_FIELD", "value")
	cfg := &Cfg{}
	assert.NotPanics(t, func() {
		err := LoadConfig(cfg)
		assert.NoError(t, err)
	})
	assert.Equal(t, "ok", cfg.Exported)
	assert.Equal(t, "", cfg.unexported)
}

func TestErrorIncludesEnvKey(t *testing.T) {
	type Cfg struct {
		Port int `env:"MY_PORT" required:"true"`
	}
	unsetEnv(t, "MY_PORT")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "MY_PORT:")
}

func TestDecoder_RangeTagsOnNonNumericDecoderReturnsError(t *testing.T) {
	// customAddr is a string kind; range tags on it must error, not silently pass.
	type Cfg struct {
		Addr customAddr `env:"DR_ADDR" min:"1"`
	}
	t.Setenv("DR_ADDR", "localhost:8080")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "range validation not supported")
	assert.Contains(t, err.Error(), "DR_ADDR:")
}

type myPort int

func (p *myPort) Decode(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	*p = myPort(v)
	return nil
}

func TestDecoder_NumericTypeRangeValidation(t *testing.T) {
	type Cfg struct {
		Port myPort `env:"NR_PORT" min:"1" max:"65535"`
	}
	t.Setenv("NR_PORT", "8080")
	assert.NoError(t, LoadConfig(&Cfg{}))

	t.Setenv("NR_PORT", "0")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
	assert.Contains(t, err.Error(), "NR_PORT:")
}

func TestDecoder_NumericType_AbsentNonRequired_RangeCheck(t *testing.T) {
	// Decode is skipped; myPort stays 0; range check fires → "out of range" error.
	type Cfg struct {
		Port myPort `env:"ABSENT_NR_PORT" min:"1"`
	}
	unsetEnv(t, "ABSENT_NR_PORT")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range") // range error, not parse error
}

type nanFloatDecoder float64

func (f *nanFloatDecoder) Decode(_ string) error {
	*f = nanFloatDecoder(math.NaN())
	return nil
}

func TestDecoder_NaNValue_TriggersCheckRangeGuard(t *testing.T) {
	type Cfg struct {
		V nanFloatDecoder `env:"NAN_DECODER_V" min:"0.0"`
	}
	t.Setenv("NAN_DECODER_V", "anything")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "finite")
	assert.Contains(t, err.Error(), "NAN_DECODER_V:")
}

func TestDecoder_AbsentNonRequired_KeepsZeroValue(t *testing.T) {
	// Decode is NOT called for absent fields; the pre-set sentinel must survive.
	type Cfg struct {
		Addr customAddr `env:"ABSENT_DECODE_ADDR"`
	}
	unsetEnv(t, "ABSENT_DECODE_ADDR")
	cfg := &Cfg{Addr: customAddr("sentinel")} // non-zero sentinel
	assert.NoError(t, LoadConfig(cfg))
	// Decode was skipped → field stays at sentinel, not overwritten with "".
	assert.Equal(t, customAddr("sentinel"), cfg.Addr)
}

func TestDecoder_NumericType_AbsentNonRequired_NoRangeTag(t *testing.T) {
	// After D1 fix: Decode is skipped; field keeps zero value myPort(0); no error.
	type Cfg struct {
		Port myPort `env:"ABSENT_NR_PORT_NORANGE"`
	}
	unsetEnv(t, "ABSENT_NR_PORT_NORANGE")
	assert.NoError(t, LoadConfig(&Cfg{}))
}

func TestWithPrefix_DecoderErrorIncludesPrefixedKey(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("PFX_"))
	type Cfg struct {
		Addr errorAddr `env:"ERR_ADDR"`
	}
	t.Setenv("PFX_ERR_ADDR", "bad")
	err := loader.LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode error")
	assert.Contains(t, err.Error(), "PFX_ERR_ADDR:")
}

func TestDecoder_InsideNestedStruct(t *testing.T) {
	type Inner struct {
		Addr customAddr `env:"NESTED_DEC_ADDR"`
	}
	type Cfg struct {
		Inner Inner
	}
	t.Setenv("NESTED_DEC_ADDR", "host:1234")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, customAddr("host:1234"), cfg.Inner.Addr)
}

func TestLoadConfig_ValueStructSliceErrors(t *testing.T) {
	type Item struct{ Port int }
	type Cfg struct {
		Items []Item `env:"VSS_ITEMS"`
	}
	t.Setenv("VSS_ITEMS", "foo,bar")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestDecoder_DefaultTagFallback(t *testing.T) {
	type Cfg struct {
		Addr customAddr `env:"DEC_DEF_ADDR" default:"host:9090"`
	}
	unsetEnv(t, "DEC_DEF_ADDR")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, customAddr("host:9090"), cfg.Addr)
}

func TestDecoder_SliceDefaultTagFallback(t *testing.T) {
	type Cfg struct {
		Addrs []customAddr `env:"DEC_SLICE_DEF" default:"a:1,b:2"`
	}
	unsetEnv(t, "DEC_SLICE_DEF")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, []customAddr{"a:1", "b:2"}, cfg.Addrs)
}

func TestDecoder_EnvOverridesDefault(t *testing.T) {
	type Cfg struct {
		Addr customAddr `env:"DEC_OVERRIDE_ADDR" default:"host:9090"`
	}
	t.Setenv("DEC_OVERRIDE_ADDR", "other:1234")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, customAddr("other:1234"), cfg.Addr)
}

func TestWithPrefix_NestedStruct(t *testing.T) {
	type DB struct {
		Host string `env:"HOST" default:"localhost"`
	}
	type Cfg struct {
		DB DB
	}
	loader := NewEnvLoader(WithPrefix("APP_"))
	t.Setenv("APP_HOST", "prod-db")
	cfg := &Cfg{}
	assert.NoError(t, loader.LoadConfig(cfg))
	assert.Equal(t, "prod-db", cfg.DB.Host)
}

func TestRequired_InvalidValueReturnsError(t *testing.T) {
	type Cfg struct {
		Port int `env:"RWCI_PORT" required:"True"`
	}
	unsetEnv(t, "RWCI_PORT")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid required tag value")

	type CfgUpper struct {
		Port int `env:"RWCI_PORT2" required:"TRUE"`
	}
	unsetEnv(t, "RWCI_PORT2")
	err = LoadConfig(&CfgUpper{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid required tag value")

	type CfgWithEnv struct {
		Port int `env:"RWCI_PORT" required:"True"`
	}
	t.Setenv("RWCI_PORT", "8080")
	err = LoadConfig(&CfgWithEnv{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid required tag value")
}

func TestWithPrefix_NestedStruct_Decoder(t *testing.T) {
	type Inner struct {
		Addr customAddr `env:"ADDR"`
	}
	type Cfg struct {
		Inner Inner
	}
	loader := NewEnvLoader(WithPrefix("PFX_"))
	t.Setenv("PFX_ADDR", "host:1234")
	cfg := &Cfg{}
	assert.NoError(t, loader.LoadConfig(cfg))
	assert.Equal(t, customAddr("host:1234"), cfg.Inner.Addr)
}

func TestWithPrefix_EmptyString(t *testing.T) {
	loader := NewEnvLoader(WithPrefix(""))
	type Cfg struct {
		Port int `env:"WPE_PORT" default:"1234"`
	}
	t.Setenv("WPE_PORT", "5678")
	cfg := &Cfg{}
	err := loader.LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 5678, cfg.Port)
}

func TestWithPrefix_LastOptionWins(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("FIRST_"), WithPrefix("SECOND_"))
	type Cfg struct {
		Port int `env:"PORT" default:"1234"`
	}
	t.Setenv("SECOND_PORT", "5678")
	cfg := &Cfg{}
	err := loader.LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 5678, cfg.Port)
}

func TestDecoder_RequiredAbsent(t *testing.T) {
	type Cfg struct {
		Addr customAddr `env:"REQ_DEC_ADDR" required:"true"`
	}
	unsetEnv(t, "REQ_DEC_ADDR")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestLoadConfig_EmbeddedStruct(t *testing.T) {
	type Base struct {
		Host string `env:"EMBED_HOST" default:"localhost"`
	}
	type Cfg struct {
		Base
		Port int `env:"EMBED_PORT" default:"8080"`
	}
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 8080, cfg.Port)

	t.Run("env var overrides default", func(t *testing.T) {
		t.Setenv("EMBED_HOST", "prod-server")
		t.Setenv("EMBED_PORT", "9090")
		cfg := &Cfg{}
		assert.NoError(t, LoadConfig(cfg))
		assert.Equal(t, "prod-server", cfg.Host)
		assert.Equal(t, 9090, cfg.Port)
	})
}

func TestWithPrefix_RequiredFieldAbsent(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("PFX_"))
	type Cfg struct {
		Port int `env:"PORT" required:"true"`
	}
	unsetEnv(t, "PFX_PORT")
	err := loader.LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "required")
}

func TestLoadConfig_EmbeddedStruct_RequiredMissing(t *testing.T) {
	type Base struct {
		Host string `env:"EMBED_REQ_HOST" required:"true"`
	}
	type Cfg struct {
		Base
		Port int `env:"EMBED_REQ_PORT" default:"8080"`
	}
	unsetEnv(t, "EMBED_REQ_HOST")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "EMBED_REQ_HOST:")
	assert.NotContains(t, err.Error(), "field Base:")
}

func TestLoadConfig_EmbeddedStruct_Decoder(t *testing.T) {
	type Base struct {
		Addr customAddr `env:"EMBED_DEC_ADDR" default:"host:9090"`
	}
	type Cfg struct {
		Base
		Port int `env:"EMBED_DEC_PORT" default:"8080"`
	}
	unsetEnv(t, "EMBED_DEC_ADDR")
	unsetEnv(t, "EMBED_DEC_PORT")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, customAddr("host:9090"), cfg.Addr)
	assert.Equal(t, 8080, cfg.Port)
}

func TestIntOverflow(t *testing.T) {
	// Value exceeding math.MaxInt64 must produce a parse error for int fields.
	type CfgInt struct {
		V int `env:"OVF_INT"`
	}
	t.Setenv("OVF_INT", "9999999999999999999999")
	assert.Error(t, LoadConfig(&CfgInt{}))

	// Value exceeding math.MaxUint64 must produce a parse error for uint fields.
	type CfgUint struct {
		V uint `env:"OVF_UINT"`
	}
	t.Setenv("OVF_UINT", "99999999999999999999999")
	assert.Error(t, LoadConfig(&CfgUint{}))
}

func TestDecoder_StructSlice(t *testing.T) {
	// []structAddr — each comma-separated token should invoke Decode.
	type Cfg struct {
		Addrs []structAddr `env:"STRUCT_ADDRS"`
	}
	t.Setenv("STRUCT_ADDRS", "host1:80,host2:443,host3:8080")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, []structAddr{
		{Raw: "host1:80"},
		{Raw: "host2:443"},
		{Raw: "host3:8080"},
	}, cfg.Addrs)
}

func TestLoadConfig_DefaultViolatesRange(t *testing.T) {
	type Cfg struct {
		N int `env:"DVR_N" default:"0" min:"1" max:"10"`
	}
	unsetEnv(t, "DVR_N")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestLoadConfig_UintDefault(t *testing.T) {
	type Cfg struct {
		N uint `env:"UD_N" default:"42"`
	}
	unsetEnv(t, "UD_N")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, uint(42), cfg.N)
}

func TestLoadConfig_IntDefaultWithRange(t *testing.T) {
	type Cfg struct {
		Workers int `env:"IDWR_WORKERS" default:"5" min:"1" max:"10"`
	}
	unsetEnv(t, "IDWR_WORKERS")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 5, cfg.Workers)
}

// Not parallel-safe: env vars are process-global, so t.Parallel() must not be used.
func TestLoadConfig_Concurrent(t *testing.T) {
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	results := make([]string, goroutines)
	panics := make([]any, goroutines)
	for i := 0; i < goroutines; i++ {
		t.Setenv(fmt.Sprintf("CONCURRENT_%d_VAL", i), strconv.Itoa(i))
	}
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					panics[i] = r
				}
				wg.Done()
			}()
			// Each goroutine uses its own struct and unique env var via a fresh loader.
			loader := NewEnvLoader(WithPrefix(fmt.Sprintf("CONCURRENT_%d_", i)))
			type SimpleCfg struct {
				V string `env:"VAL"`
			}
			cfg := &SimpleCfg{}
			errs[i] = loader.LoadConfig(cfg)
			results[i] = cfg.V
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d", i)
	}
	for i := 0; i < goroutines; i++ {
		assert.Nil(t, panics[i], "goroutine %d panicked: %v", i, panics[i])
		assert.Equal(t, strconv.Itoa(i), results[i], "goroutine %d", i)
	}
}

// Not parallel-safe: env vars are process-global, so t.Parallel() must not be used.
func TestLoadConfig_ConcurrentSharedLoader(t *testing.T) {
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	results := make([]int, goroutines)
	panics := make([]any, goroutines)

	// All goroutines read the same env var — struct tags are compile-time
	// constants so we can't vary them per goroutine. This still exercises
	// concurrent access on the shared defaultLoader.
	t.Setenv("SHARED_LOADER_VAL", "42")

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					panics[i] = r
				}
				wg.Done()
			}()
			type Cfg struct {
				Val int `env:"SHARED_LOADER_VAL"`
			}
			cfg := &Cfg{}
			errs[i] = LoadConfig(cfg)
			results[i] = cfg.Val
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d", i)
	}
	for i := 0; i < goroutines; i++ {
		assert.Nil(t, panics[i], "goroutine %d panicked: %v", i, panics[i])
		assert.Equal(t, 42, results[i], "goroutine %d", i)
	}
}

func TestRequiredFalse_AbsentEnvVar(t *testing.T) {
	type Cfg struct {
		Host string     `env:"RF_HOST" required:"false"`
		Port int        `env:"RF_PORT" required:"false"`
		Addr customAddr `env:"RF_ADDR" required:"false"`
	}
	unsetEnv(t, "RF_HOST")
	unsetEnv(t, "RF_PORT")
	unsetEnv(t, "RF_ADDR")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "", cfg.Host)
	assert.Equal(t, 0, cfg.Port)
	assert.Equal(t, customAddr(""), cfg.Addr)
}

// Not parallel-safe: env vars are process-global, so t.Parallel() must not be used.
func TestLoadConfig_ConcurrentDecoder(t *testing.T) {
	const goroutines = 10
	var wg sync.WaitGroup
	wg.Add(goroutines)
	errs := make([]error, goroutines)
	results := make([]customAddr, goroutines)
	panics := make([]any, goroutines)

	t.Setenv("SHARED_DEC_ADDR", "host:8080")

	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer func() {
				if r := recover(); r != nil {
					panics[i] = r
				}
				wg.Done()
			}()
			type Cfg struct {
				Addr customAddr `env:"SHARED_DEC_ADDR"`
			}
			cfg := &Cfg{}
			errs[i] = LoadConfig(cfg)
			results[i] = cfg.Addr
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		assert.NoError(t, err, "goroutine %d", i)
	}
	for i := 0; i < goroutines; i++ {
		assert.Nil(t, panics[i], "goroutine %d panicked: %v", i, panics[i])
		assert.Equal(t, customAddr("host:8080"), results[i], "goroutine %d", i)
	}
}

func TestLoadConfig_DurationDefault(t *testing.T) {
	type Cfg struct {
		Timeout time.Duration `env:"DD_TIMEOUT" default:"5s"`
	}
	unsetEnv(t, "DD_TIMEOUT")
	cfg := &Cfg{}
	err := LoadConfig(cfg)
	assert.NoError(t, err)
	assert.Equal(t, 5*time.Second, cfg.Timeout)

	t.Run("env overrides default", func(t *testing.T) {
		t.Setenv("DD_TIMEOUT", "10s")
		cfg := &Cfg{}
		err := LoadConfig(cfg)
		assert.NoError(t, err)
		assert.Equal(t, 10*time.Second, cfg.Timeout)
	})
}

func TestLoadConfig_TimeSliceFieldReturnsError(t *testing.T) {
	type Cfg struct {
		Times []time.Time `env:"SOME_TIMES"`
	}
	t.Setenv("SOME_TIMES", "2024-01-01")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SOME_TIMES:")
	assert.Contains(t, err.Error(), "[]time.Time is not supported")
}

func TestLoadConfig_TimeSliceFieldAbsentReturnsError(t *testing.T) {
	type Cfg struct {
		Times []time.Time `env:"SOME_TIMES"`
	}
	unsetEnv(t, "SOME_TIMES")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SOME_TIMES:")
	assert.Contains(t, err.Error(), "[]time.Time is not supported")
}

func TestLoadConfig_TimeSliceFieldWithoutEnvTagSkipped(t *testing.T) {
	type Cfg struct {
		Times []time.Time // no env tag
		S     string      `env:"TSFWET_S" default:"ok"`
	}
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "ok", cfg.S)
	assert.Nil(t, cfg.Times)
}

func TestLoadConfig_TimeSliceFieldRequiredReturnsNotSupported(t *testing.T) {
	type Cfg struct {
		T []time.Time `env:"REQ_TIMES" required:"true"`
	}
	unsetEnv(t, "REQ_TIMES")
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "[]time.Time is not supported")
	assert.NotContains(t, err.Error(), "required")
}

func TestLoadConfig_TimeSliceFieldDefaultReturnsNotSupported(t *testing.T) {
	type Cfg struct {
		T []time.Time `env:"DEF_TIMES" default:"2024-01-01"`
	}
	err := LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "[]time.Time is not supported")
	assert.NotContains(t, err.Error(), "2024-01-01")
}

func TestWithPrefix_TimeFieldError(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("PFX_"))
	type Cfg struct {
		T time.Time `env:"MY_TIME"`
	}
	err := loader.LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PFX_MY_TIME:")
	assert.Contains(t, err.Error(), "time.Time is not supported")
}

func TestWithPrefix_TimeSliceFieldError(t *testing.T) {
	loader := NewEnvLoader(WithPrefix("PFX_"))
	type Cfg struct {
		T []time.Time `env:"MY_TIMES"`
	}
	err := loader.LoadConfig(&Cfg{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PFX_MY_TIMES:")
	assert.Contains(t, err.Error(), "[]time.Time is not supported")
}

func TestDecoder_RequiredWithDefault(t *testing.T) {
	type Cfg struct {
		Addr customAddr `env:"RWD_ADDR" required:"true" default:"host:9090"`
	}
	unsetEnv(t, "RWD_ADDR")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, customAddr("host:9090"), cfg.Addr)
}
