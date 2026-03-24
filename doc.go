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
Empty tokens (from consecutive or trailing commas, e.g. "1,,3" or "1,2,") produce zero values.
time.Duration values must include an explicit unit (e.g. "30s", "5m").

# Nested structs

Nested structs are supported and processed recursively. Anonymous (embedded) struct fields are supported and their env tags are resolved at the same level with no additional prefix. Unlike named nested struct fields, errors from embedded fields propagate without a field-name wrapper.

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

# Sentinel errors

The package exports three sentinel errors that callers can match with [errors.Is]:

  - [ErrNilConfig] — cfg argument is nil
  - [ErrNotStructPointer] — cfg is not a pointer to a struct
  - [ErrNilPointer] — cfg is a typed nil pointer (e.g. `var p *Config; LoadConfig(p)`)
*/
package config
