package config

import (
	"fmt"
	"os"
	"reflect"
	"time"
)

// ValueParser is responsible for parsing string values into specific types
type ValueParser interface {
	Parse(value string, field reflect.Value) error
}

// Validator is responsible for validating field values
type Validator interface {
	Validate(field reflect.Value, tags reflect.StructTag) error
}

// EnvLoader loads values from environment variables
type EnvLoader struct {
	parsers    map[reflect.Kind]ValueParser
	validators []Validator
	prefix     string
}

// Option represents a configuration option for EnvLoader
type Option func(*EnvLoader)

// WithParser adds a custom parser for a specific type
func WithParser(kind reflect.Kind, parser ValueParser) Option {
	return func(l *EnvLoader) {
		l.parsers[kind] = parser
	}
}

// WithValidator adds a custom validator
func WithValidator(validator Validator) Option {
	return func(l *EnvLoader) {
		l.validators = append(l.validators, validator)
	}
}

// WithPrefix adds a prefix to all environment variable names
func WithPrefix(prefix string) Option {
	return func(l *EnvLoader) {
		l.prefix = prefix
	}
}

var defaultLoader = NewEnvLoader()

// LoadConfig maintains backward compatibility using the default loader
func LoadConfig(cfg interface{}) error {
	return defaultLoader.LoadConfig(cfg)
}

// NewEnvLoader creates a new EnvLoader with default parsers and validators
func NewEnvLoader(opts ...Option) *EnvLoader {
	l := &EnvLoader{
		parsers: map[reflect.Kind]ValueParser{
			reflect.String:  &StringParser{},
			reflect.Int64:   &Int64Parser{},
			reflect.Int:     &IntParser{},
			reflect.Slice:   &SliceParser{},
			reflect.Bool:    &BoolParser{},
			reflect.Float64: &Float64Parser{},
		},
		validators: []Validator{
			&RequiredValidator{},
			&RangeValidator{},
		},
	}

	// Apply custom options
	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LoadConfig loads configuration from environment variables
func (l *EnvLoader) LoadConfig(cfg interface{}) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("config must be a pointer")
	}

	return l.loadStruct(v.Elem())
}

// loadStruct processes a struct, loading environment variables into its fields
func (l *EnvLoader) loadStruct(v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// Handle nested structs
		if l.isNestedStruct(field) {
			if err := l.loadStruct(field); err != nil {
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
			continue
		}

		if err := l.loadField(field, fieldType); err != nil {
			return fmt.Errorf("field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// Helper to identify special types like time.Time
func isTimeType(t reflect.Type) bool {
	return t == reflect.TypeOf(time.Time{})
}

// Helper to check if a field is a nested struct
func (l *EnvLoader) isNestedStruct(field reflect.Value) bool {
	return field.Kind() == reflect.Struct && !isTimeType(field.Type())
}

// getParserForType returns a parser for the specified kind
func (l *EnvLoader) getParserForType(kind reflect.Kind) (ValueParser, bool) {
	parser, ok := l.parsers[kind]
	return parser, ok
}

// loadField processes a single field, loading from environment variable
func (l *EnvLoader) loadField(field reflect.Value, fieldType reflect.StructField) error {
	envKey := fieldType.Tag.Get("env")
	if envKey == "" {
		return nil
	}

	envValue := l.getEnvValueWithDefault(envKey, fieldType)

	return l.parseAndValidateField(envValue, field, fieldType)
}

// getEnvValueWithDefault retrieves the environment value or uses default if provided
func (l *EnvLoader) getEnvValueWithDefault(envKey string, fieldType reflect.StructField) string {
	// Apply prefix if set
	if l.prefix != "" {
		envKey = l.prefix + envKey
	}

	// Get value from environment or use default
	envValue := os.Getenv(envKey)
	if envValue == "" {
		defaultValue := fieldType.Tag.Get("default")
		if defaultValue != "" {
			envValue = defaultValue
		}
	}

	return envValue
}

// parseAndValidateField handles parsing and validation for a single field
func (l *EnvLoader) parseAndValidateField(envValue string, field reflect.Value, fieldType reflect.StructField) error {
	// Special handling for time.Duration
	if fieldType.Type == reflect.TypeOf(time.Duration(0)) {
		return l.parseAndValidateDuration(envValue, field, fieldType)
	}

	// Special handling for slices
	if field.Kind() == reflect.Slice {
		return l.parseAndValidateSlice(envValue, field, fieldType)
	}

	// Parse other types
	parser, ok := l.parsers[field.Kind()]
	if !ok {
		return fmt.Errorf("unsupported type: %v", field.Kind())
	}

	if err := parser.Parse(envValue, field); err != nil {
		return err
	}

	return l.validateField(field, fieldType)
}

// parseAndValidateDuration parses and validates a time.Duration field
func (l *EnvLoader) parseAndValidateDuration(envValue string, field reflect.Value, fieldType reflect.StructField) error {
	parser := &DurationParser{}
	if err := parser.Parse(envValue, field); err != nil {
		return err
	}
	return l.validateField(field, fieldType)
}

// parseAndValidateSlice parses and validates a slice field
func (l *EnvLoader) parseAndValidateSlice(envValue string, field reflect.Value, fieldType reflect.StructField) error {
	sliceParser := &SliceParser{}
	// Use ParseWithContext to inject the parser provider function
	if err := sliceParser.ParseWithContext(envValue, field, l.getParserForType); err != nil {
		return err
	}
	return l.validateField(field, fieldType)
}

// validateField validates a field using all registered validators
func (l *EnvLoader) validateField(field reflect.Value, fieldType reflect.StructField) error {
	for _, validator := range l.validators {
		if err := validator.Validate(field, fieldType.Tag); err != nil {
			return err
		}
	}
	return nil
}
