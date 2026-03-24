# Config

A simple, type-safe environment variable configuration loader for Go applications.

## Features

- Load configuration from environment variables
- Support for various data types:
  - Strings
  - Integers (int/int8/int16/int32/int64, uint/uint8/uint16/uint32/uint64)
  - Floats (float32/float64)
  - Booleans
  - Slices of supported types (comma-separated values), including `[]time.Duration`
  - Durations (`time.Duration`; values must include an explicit unit, e.g. `"30s"`, `"5m"`)
- Nested struct support
- Required field validation
- Default values

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

### Slices

Slice fields split the env var value on commas. Leading and trailing whitespace is
trimmed from each element. Setting a slice field to an empty string (`TAGS=""`) is
a no-op — the field retains its zero value (`nil`).

## License

MIT
