package config

// Tag keys used for configuration
const (
	EnvTag      = "env"
	RequiredTag = "required"
	DefaultTag  = "default"
	MinTag      = "min"
	MaxTag      = "max"
	RangeErrTag = "range_error"
)

// Common tag values
const (
	TagTrue = "true"
)

// Error messages
const (
	ErrRequiredField   = "required field is empty"
	ErrOutOfRange      = "value out of range"
	ErrUnsupportedType = "unsupported type: %v"
	ErrConfigNotPtr    = "config must be a pointer"
)
