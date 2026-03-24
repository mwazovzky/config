package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
)

var (
	// ErrNilConfig is returned when a nil interface is passed to LoadConfig.
	ErrNilConfig = errors.New("config must not be nil")
	// ErrNotStructPointer is returned when the argument is not a pointer to a struct.
	ErrNotStructPointer = errors.New("config must be a pointer to a struct")
	// ErrNilPointer is returned when a typed nil pointer is passed to LoadConfig.
	ErrNilPointer = errors.New("config must be a non-nil pointer to a struct")
)

// LoadConfig loads environment variables into cfg, which must be a pointer to a struct.
func LoadConfig(cfg interface{}) error {
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
	return loadStruct(v.Elem())
}

// loadStruct processes a struct, loading environment variables into its fields.
func loadStruct(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !fieldType.IsExported() {
			continue
		}

		// Only recurse into struct fields that have no env tag.
		// A struct field with an env tag (e.g. time.Time) should fall through
		// to loadField where parseValue will return an "unsupported type" error.
		if field.Kind() == reflect.Struct && fieldType.Tag.Get(EnvTag) == "" {
			if err := loadStruct(field); err != nil {
				if fieldType.Anonymous {
					return err
				}
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
			continue
		}

		if err := loadField(field, fieldType); err != nil {
			return err
		}
	}
	return nil
}

// loadField reads the env tag, fetches the value, then parses and validates it.
func loadField(field reflect.Value, fieldType reflect.StructField) error {
	envKey := fieldType.Tag.Get(EnvTag)
	if envKey == "" {
		return nil
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
		if err := parseValue(rawValue, field); err != nil {
			return fmt.Errorf("%s: %w", envKey, err)
		}
	}

	return nil
}
