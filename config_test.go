package config

import (
	"errors"
	"os"
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
	assert.Equal(t, "default-value", cfg.DefaultStr)
}

func TestLoadConfigMissingRequired(t *testing.T) {
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
	t.Setenv("STRING_FIELD", "")
	t.Setenv("INT_FIELD", "")
	t.Setenv("SLICE_FIELD", "")
	t.Setenv("STRING_SLICE_FIELD", "")
	t.Setenv("DURATION_FIELD", "")
	t.Setenv("BOOL_FIELD", "")
	t.Setenv("FLOAT_FIELD", "")
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
}

func TestLoadConfigPartialValues(t *testing.T) {
	t.Setenv("STRING_FIELD", "test_string")
	t.Setenv("INT_FIELD", "12345")
	for _, key := range []string{"SLICE_FIELD", "STRING_SLICE_FIELD", "DURATION_FIELD", "BOOL_FIELD", "FLOAT_FIELD"} {
		unsetEnv(t, key)
	}

	cfg := &TestConfig{}
	err := LoadConfig(cfg)
	assert.Error(t, err)
}

func TestRequiredPassesForZeroInt(t *testing.T) {
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
	assert.Contains(t, err.Error(), "Database")
}

func TestLoadConfig_NonStructPointer(t *testing.T) {
	err := LoadConfig(new(int))
	assert.True(t, errors.Is(err, ErrNotStructPointer))
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

func TestLoadConfig_DurationSlice(t *testing.T) {
	type Cfg struct {
		Timeouts []time.Duration `env:"TIMEOUTS"`
	}
	t.Setenv("TIMEOUTS", "1s,2m,500ms")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, []time.Duration{time.Second, 2 * time.Minute, 500 * time.Millisecond}, cfg.Timeouts)
}

func TestIntWidths(t *testing.T) {
	type Cfg struct {
		I8  int8    `env:"IW_I8"`
		I16 int16   `env:"IW_I16"`
		I32 int32   `env:"IW_I32"`
		U8  uint8   `env:"IW_U8"`
		U16 uint16  `env:"IW_U16"`
		U32 uint32  `env:"IW_U32"`
		U64 uint64  `env:"IW_U64"`
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
	assert.Equal(t, 0, cfg.inner.Port)
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

func TestIntOverflow(t *testing.T) {
	type CfgInt struct {
		V int `env:"OVF_INT"`
	}
	t.Setenv("OVF_INT", "9999999999999999999999")
	assert.Error(t, LoadConfig(&CfgInt{}))

	type CfgUint struct {
		V uint `env:"OVF_UINT"`
	}
	t.Setenv("OVF_UINT", "99999999999999999999999")
	assert.Error(t, LoadConfig(&CfgUint{}))
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

func TestRequiredFalse_AbsentEnvVar(t *testing.T) {
	type Cfg struct {
		Host string `env:"RF_HOST" required:"false"`
		Port int    `env:"RF_PORT" required:"false"`
	}
	unsetEnv(t, "RF_HOST")
	unsetEnv(t, "RF_PORT")
	cfg := &Cfg{}
	assert.NoError(t, LoadConfig(cfg))
	assert.Equal(t, "", cfg.Host)
	assert.Equal(t, 0, cfg.Port)
}
