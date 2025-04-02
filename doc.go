/*
Package config provides a flexible environment variable configuration loader for Go applications.

Basic usage:

	type Config struct {
		Port        int           `env:"PORT" required:"true"`
		Host        string        `env:"HOST" required:"true"`
		Timeout     time.Duration `env:"TIMEOUT" required:"true"`
		DatabaseURL string        `env:"DATABASE_URL" required:"true"`
	}

	func main() {
		cfg := &Config{}
		if err := config.LoadConfig(cfg); err != nil {
			log.Fatal(err)
		}
	}

Custom parsers:

	type BoolParser struct{}

	func (p *BoolParser) Parse(value string, field reflect.Value) error {
		if value == "" {
			return nil
		}
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(v)
		return nil
	}

	loader := config.NewEnvLoader(
		config.WithParser(reflect.Bool, &BoolParser{}),
	)
*/
package config
