# Config

A flexible, type-safe environment variable configuration loader for Go applications.

## Features

- Load configuration from environment variables
- Support for various data types:
  - Strings
  - Integers (int, int64)
  - Floats (float64)
  - Booleans
  - Slices (of supported types)
  - Durations
- Nested struct support
- Required field validation
- Default values
- Range validation (min/max)
- Custom error messages
- Prefix support for environment variables
- Extensible with custom parsers and validators

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

```go
type DatabaseConfig struct {
	Host     string `env:"DB_HOST" default:"localhost"`
	Port     int    `env:"DB_PORT" default:"5432"`
	User     string `env:"DB_USER" required:"true"`
	Password string `env:"DB_PASSWORD" required:"true"`
}

type AppConfig struct {
	Server   ServerConfig   `env:"SERVER"`
	Database DatabaseConfig // Nested struct
	Debug    bool           `env:"DEBUG" default:"false"`
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

### Range Validation

```go
type Config struct {
	Port int `env:"PORT" min:"1024" max:"65535"`
	Age  int `env:"AGE" min:"0" max:"120" range_error:"Age must be between 0 and 120"`
}
```

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

## Custom Parsers

```go
type IPParser struct{}

func (p *IPParser) Parse(value string, field reflect.Value) error {
	ip := net.ParseIP(value)
	if ip == nil {
		return fmt.Errorf("invalid IP address: %s", value)
	}
	field.Set(reflect.ValueOf(ip))
	return nil
}

loader := config.NewEnvLoader(
	config.WithParser(reflect.TypeOf(net.IP{}).Kind(), &IPParser{}),
)
```

## Custom Validators

```go
type EmailValidator struct{}

func (v *EmailValidator) Validate(field reflect.Value, tags reflect.StructTag) error {
	if tags.Get("validate_email") != "true" {
		return nil
	}

	email := field.String()
	if !strings.Contains(email, "@") {
		return fmt.Errorf("invalid email address: %s", email)
	}
	return nil
}

loader := config.NewEnvLoader(
	config.WithValidator(&EmailValidator{}),
)
```

## License

MIT
