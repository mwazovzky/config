/*
Package config loads environment variables into Go structs using struct tags.

# Basic usage

Annotate a struct with `env` tags and call [LoadConfig]:

	type Config struct {
		Port        int           `env:"PORT" required:"true" default:"8080"`
		Host        string        `env:"HOST" required:"true" default:"localhost"`
		Timeout     time.Duration `env:"TIMEOUT" default:"30s"`
		Debug       bool          `env:"DEBUG" default:"false"`
		AllowedIPs  []string      `env:"ALLOWED_IPS" default:"127.0.0.1,::1"`
	}

	cfg := &Config{}
	if err := config.LoadConfig(cfg); err != nil {
		log.Fatal(err)
	}

# Supported types

string, int/int8/int16/int32/int64, uint/uint8/uint16/uint32/uint64,
float32/float64, bool, time.Duration, and slices of any supported type
(including []time.Duration).
Slice values are parsed from comma-separated strings (e.g. "1,2,3"). Leading and trailing whitespace is trimmed from each element.
Empty tokens (from consecutive or trailing commas, e.g. "1,,3" or "1,2,") produce zero values for built-in types.
Types implementing Decoder receive Decode("") for each empty token. Leading and trailing whitespace is trimmed before Decode is called, the same as for built-in types.
time.Duration values must include an explicit unit (e.g. "30s", "5m").
time.Time, []time.Time, and []*time.Time are not supported; use time.Duration, a string field,
or a type implementing Decoder instead.

# Nested structs

Nested structs are supported and processed recursively. Only value (non-pointer) struct fields are recursed into; exported *NestedConfig pointer fields and []*NestedConfig slice-of-pointer-to-struct fields are not supported and will cause LoadConfig to return an error — use a value field or a slice of types implementing Decoder instead. Anonymous (embedded) struct fields are supported and their env tags are resolved at the same level with no additional prefix. Unlike named nested struct fields, errors from embedded fields propagate without a field-name wrapper.

	type DatabaseConfig struct {
		Host string `env:"DB_HOST" default:"localhost"`
		Port int    `env:"DB_PORT" default:"5432"`
	}

	type AppConfig struct {
		Database DatabaseConfig
		Debug    bool `env:"DEBUG" default:"false"`
	}

# Validation

Fields tagged with `required:"true"` must be provided: the env var must be set
(even to an empty string) or a `default` tag must be present. A zero value (e.g.
PORT=0) satisfies required.
The tag value must be exactly "true" or "false" (lowercase). Any other value
(e.g. "True", "TRUE", "yes") returns an error to catch misconfiguration early.

Numeric fields support range validation via `min` and `max` tags.
A custom error message can be set with `range_error`:

	type Config struct {
		Workers int `env:"WORKERS" required:"true" min:"1" max:"64" range_error:"WORKERS must be between 1 and 64"`
	}

For time.Duration fields, min and max accept either a duration string or a
nanosecond integer:

	Timeout time.Duration `env:"TIMEOUT" min:"1s" max:"60s"`
	Timeout time.Duration `env:"TIMEOUT" min:"1000000000" max:"60000000000"` // equivalent

Range tags may also be applied to Decoder fields. If the field's underlying
kind is numeric, the parsed value is range-checked after Decode returns.
For non-numeric kinds (string, struct, etc.) an error is returned.

Range validation always runs, even when the env var is absent. A non-required
field that is absent (no env var, no default) retains its zero value, which is
still checked against min/max — add a default tag or remove the range tags if a
zero value should be allowed. Parsing only runs when the env var is set or a
default tag is present.

# Prefix

Use [NewEnvLoader] with [WithPrefix] to prepend a string to all env var names:

	loader := config.NewEnvLoader(config.WithPrefix("MYAPP_"))
	// looks up MYAPP_PORT instead of PORT

# Custom types via Decoder

Any type can control its own parsing by implementing the [Decoder] interface:

	type NetAddr string

	func (a *NetAddr) Decode(value string) error {
		*a = NetAddr(value)
		return nil
	}

	type Config struct {
		ListenAddr NetAddr `env:"LISTEN_ADDR" default:"localhost:8080"`
	}

Decoder works for both direct fields and slice elements (e.g. []NetAddr).

Decode is called only when the value is provided — the env var is set (even to "")
or a default tag exists. When a default tag is present and the env var is absent,
Decode receives the default string, not "". Absent non-required fields retain their
zero value without calling Decode, consistent with built-in type behavior.

# Sentinel errors

The package exports three sentinel errors that callers can match with [errors.Is]:

  - [ErrNilConfig] — cfg argument is nil
  - [ErrNotStructPointer] — cfg is not a pointer to a struct
  - [ErrNilPointer] — cfg is a typed nil pointer (e.g. `var p *Config; LoadConfig(p)`)
*/
package config
