package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"
)

var (
	// ErrNilConfig is returned when a nil interface is passed to LoadConfig.
	ErrNilConfig = errors.New("config must not be nil")
	// ErrNotStructPointer is returned when the argument is not a pointer to a struct.
	ErrNotStructPointer = errors.New("config must be a pointer to a struct")
	// ErrNilPointer is returned when a typed nil pointer is passed to LoadConfig.
	ErrNilPointer = errors.New("config must be a non-nil pointer to a struct")
)

// Decoder can be implemented by any type to control its own parsing.
// If a field's type (addressed as a pointer) implements Decoder, Decode is called
// instead of the built-in type switch.
// Decode is called only when the value is provided — the env var is set (even to "")
// or a default tag exists. Absent non-required fields retain their zero value without
// calling Decode, consistent with built-in type behavior.
type Decoder interface {
	Decode(value string) error
}

// EnvLoader loads values from environment variables.
type EnvLoader struct {
	prefix string
}

// Option represents a configuration option for EnvLoader.
type Option func(*EnvLoader)

// WithPrefix adds a prefix to all environment variable names.
// The prefix is prepended verbatim; include a trailing separator if needed
// (e.g. WithPrefix("MYAPP_") looks up MYAPP_PORT instead of PORT).
func WithPrefix(prefix string) Option {
	return func(l *EnvLoader) {
		l.prefix = prefix
	}
}

var (
	defaultLoader = NewEnvLoader()
	timeType      = reflect.TypeOf(time.Time{})
)

// LoadConfig loads configuration from environment variables using the default loader.
func LoadConfig(cfg interface{}) error {
	return defaultLoader.LoadConfig(cfg)
}

// NewEnvLoader creates a new EnvLoader with the given options.
func NewEnvLoader(opts ...Option) *EnvLoader {
	l := &EnvLoader{}
	for _, opt := range opts {
		opt(l)
	}
	return l
}

// LoadConfig loads environment variables into cfg, which must be a pointer to a struct.
func (l *EnvLoader) LoadConfig(cfg interface{}) error {
	if cfg == nil {
		return ErrNilConfig
	}
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr {
		return ErrNotStructPointer
	}
	if v.IsNil() {
		return ErrNilPointer
	}
	if v.Elem().Kind() != reflect.Struct {
		return ErrNotStructPointer
	}
	return l.loadStruct(v.Elem())
}

// loadStruct processes a struct, loading environment variables into its fields.
func (l *EnvLoader) loadStruct(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Skip unexported fields early — they can't be set and shouldn't
		// trigger pointer-to-struct rejection either.
		if !fieldType.IsExported() {
			continue
		}

		dec, isDec := asDecoder(field)

		// Reject pointer-to-struct fields even without an env tag.
		// A nil pointer causes a panic on dereference; catching it here
		// gives a clear error regardless of whether an env tag is present.
		// Decoder is checked first so a type with Decode on *T can still opt in.
		if !isDec &&
			field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
			return fmt.Errorf("field %s: pointer to struct is not supported for type %s; use a value field instead", fieldType.Name, field.Type())
		}

		if !isDec &&
			field.Kind() == reflect.Slice &&
			field.Type().Elem() == reflect.PtrTo(timeType) {
			return fmt.Errorf("field %s: []*time.Time is not supported; use []time.Duration or a slice of a Decoder type instead", fieldType.Name)
		}

		if !isDec &&
			field.Kind() == reflect.Slice &&
			field.Type().Elem().Kind() == reflect.Ptr &&
			field.Type().Elem().Elem().Kind() == reflect.Struct {
			return fmt.Errorf("field %s: slice of pointer-to-struct is not supported for type %s; use a slice of types implementing Decoder instead", fieldType.Name, field.Type())
		}

		// A struct field with Decoder is a scalar from the loader's point of
		// view and must not be recursed into.
		if !isDec && isNestedStruct(field) {
			if err := l.loadStruct(field); err != nil {
				if fieldType.Anonymous {
					return err
				}
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
			continue
		}

		if err := l.loadField(field, fieldType, dec); err != nil {
			return err
		}
	}
	return nil
}

func isNestedStruct(field reflect.Value) bool {
	return field.Kind() == reflect.Struct && !isTimeType(field.Type())
}

func asDecoder(field reflect.Value) (Decoder, bool) {
	d, ok := field.Addr().Interface().(Decoder)
	return d, ok
}

// isTimeType reports whether t is time.Time.
func isTimeType(t reflect.Type) bool {
	return t == timeType
}

// loadField reads the env tag, fetches the value, then parses and validates it.
func (l *EnvLoader) loadField(field reflect.Value, fieldType reflect.StructField, dec Decoder) error {
	envKey := fieldType.Tag.Get(EnvTag)
	if envKey == "" {
		return nil
	}

	envKey = l.prefix + envKey

	if field.Type() == timeType {
		return fmt.Errorf("%s: time.Time is not supported; use time.Duration or a string field instead", envKey)
	}

	if field.Kind() == reflect.Slice && field.Type().Elem() == timeType {
		return fmt.Errorf("%s: []time.Time is not supported; use []time.Duration or a slice of a Decoder type instead", envKey)
	}

	rawValue, ok := os.LookupEnv(envKey)
	provided := ok
	if !ok {
		if def, ok := fieldType.Tag.Lookup(DefaultTag); ok {
			rawValue = def
			provided = true
		}
	}

	if err := checkRequired(fieldType.Tag, provided); err != nil {
		return fmt.Errorf("%s: %w", envKey, err)
	}

	if provided {
		var err error
		if dec != nil {
			err = dec.Decode(rawValue)
		} else {
			err = parseValue(rawValue, field)
		}
		if err != nil {
			return fmt.Errorf("%s: %w", envKey, err)
		}
	}

	if err := checkRange(field, fieldType.Tag); err != nil {
		return fmt.Errorf("%s: %w", envKey, err)
	}
	return nil
}
