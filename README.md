# Config

A flexible, type-safe environment variable configuration loader for Go applications.

## Features

- Load configuration from environment variables
- Support for various data types:
  - Strings
  - Integers (int/int8/int16/int32/int64, uint/uint8/uint16/uint32/uint64)
  - Floats (float32/float64)
  - Booleans
  - Slices of supported types (comma-separated values), including `[]time.Duration`
  - Durations (`time.Duration`; values must include an explicit unit, e.g. `"30s"`, `"5m"`)
- `time.Time`, `[]time.Time`, and `[]*time.Time` are **not** supported — use `time.Duration`, a string field, or a custom `Decoder` type
- Nested struct support
- Required field validation
- Default values
- Range validation (min/max)
- Custom error messages
- Prefix support for environment variables
- Extensible via the `Decoder` interface for custom types

## Installation

```bash
go get github.com/mwazovzky/config
```

## Basic Usage

```go
package main

import (
	"log"
	"time"

	"github.com/mwazovzky/config"
)

type Config struct {
	Port        int           `env:"PORT" required:"true" default:"8080"`
	Host        string        `env:"HOST" required:"true" default:"localhost"`
	Timeout     time.Duration `env:"TIMEOUT" required:"true" default:"30s"`
	Debug       bool          `env:"DEBUG" default:"false"`
	AllowedIPs  []string      `env:"ALLOWED_IPS" default:"127.0.0.1,::1"`
}

func main() {
	cfg := &Config{}
	if err := config.LoadConfig(cfg); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting server at %s:%d with timeout %v", cfg.Host, cfg.Port, cfg.Timeout)
}
```

## Nested Structs

Anonymous (embedded) struct fields are also supported and resolved at the same level with no additional prefix.

```go
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" default:"localhost"`
	Port     int    `env:"DB_PORT" default:"5432"`
	User     string `env:"DB_USER" required:"true"`
	Password string `env:"DB_PASSWORD" required:"true"`
}

type AppConfig struct {
	Database DatabaseConfig
	Debug    bool `env:"DEBUG" default:"false"`
}

// Usage
cfg := &AppConfig{}
if err := config.LoadConfig(cfg); err != nil {
	log.Fatal(err)
}
```

## Validation

### Required Fields

```go
type Config struct {
	APIKey string `env:"API_KEY" required:"true"` // Must be set
}
```

> **Note:** `required` checks that the env var is set or a `default` tag exists — zero values
> are valid. `PORT=0` with `required:"true"` succeeds; only a completely absent var with no
> default triggers an error. The tag value must be exactly `"true"` or `"false"` (lowercase);
> other values like `"True"` or `"TRUE"` return an error.

### Range Validation

```go
type Config struct {
	Port int `env:"PORT" min:"1024" max:"65535"`
	Age  int `env:"AGE" min:"0" max:"120" range_error:"Age must be between 0 and 120"`
}
```

For `time.Duration` fields, `min` and `max` accept either a duration string or a
nanosecond integer:

```go
type Config struct {
	Timeout time.Duration `env:"TIMEOUT" min:"1s" max:"60s"`
}
```

> **Note:** Range validation runs after parsing regardless of whether the env var was
> provided. A non-required field that is absent (no env var, no default) retains its zero
> value, which is still checked against `min`/`max` — add a `default` tag or remove the
> range tags if a zero value should be allowed.

### Slices

Slice fields split the env var value on commas. Leading and trailing whitespace is
trimmed from each element. Setting a slice field to an empty string (`TAGS=""`) is
a no-op — the field retains its zero value (`nil`).

## Custom Environment Variable Prefix

```go
loader := config.NewEnvLoader(
	config.WithPrefix("MYAPP_"),
)

// Will look for MYAPP_PORT instead of PORT
type Config struct {
	Port int `env:"PORT" default:"8080"`
}

cfg := &Config{}
if err := loader.LoadConfig(cfg); err != nil {
	log.Fatal(err)
}
```

## Custom Types via Decoder

Implement the `Decoder` interface on any type to control its own parsing. `Decode` is called
instead of the built-in type switch when the field's pointer type satisfies the interface.
This works for scalar fields, slice elements, and value (non-pointer) struct fields.

When the env var is absent and the field is not required, `Decode` is not called — the field
retains its zero value, the same as built-in types.

```go
type HostPort struct {
	Host string
	Port string
}

func (hp *HostPort) Decode(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("expected host:port, got %q", value)
	}
	hp.Host, hp.Port = parts[0], parts[1]
	return nil
}

type Config struct {
	Addr HostPort `env:"LISTEN_ADDR"`
}
```

## License

MIT
