package config

import (
	"fmt"
	"reflect"
	"strconv"
)

// RequiredValidator ensures a field isn't empty or zero
type RequiredValidator struct{}

// Validate checks if the field satisfies the required constraint
func (v *RequiredValidator) Validate(field reflect.Value, tags reflect.StructTag) error {
	if tags.Get(RequiredTag) != TagTrue {
		return nil
	}

	if isZeroValue(field) {
		return fmt.Errorf(ErrRequiredField)
	}

	return nil
}

func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	default:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	}
}

// RangeValidator checks if a field's value falls within a specified range
type RangeValidator struct{}

// Validate checks if the field satisfies the range constraints
func (v *RangeValidator) Validate(field reflect.Value, tags reflect.StructTag) error {
	min := tags.Get(MinTag)
	max := tags.Get(MaxTag)
	if min == "" && max == "" {
		return nil
	}

	var err error
	value := field.Interface()
	errMsg := tags.Get(RangeErrTag)
	if errMsg == "" {
		errMsg = ErrOutOfRange
	}

	switch v := value.(type) {
	case int, int64:
		err = validateIntRange(v, min, max)
	case float64:
		err = validateFloatRange(v, min, max)
	}

	if err != nil {
		return fmt.Errorf("%s: %w", errMsg, err)
	}
	return nil
}

// validateIntRange checks if an integer value falls within the specified range
func validateIntRange(value interface{}, minStr, maxStr string) error {
	var val int64
	switch v := value.(type) {
	case int:
		val = int64(v)
	case int64:
		val = v
	default:
		return fmt.Errorf("unsupported integer type: %T", value)
	}

	if minStr != "" {
		min, err := strconv.ParseInt(minStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %w", err)
		}
		if val < min {
			return fmt.Errorf("value %d is less than minimum %d", val, min)
		}
	}

	if maxStr != "" {
		max, err := strconv.ParseInt(maxStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %w", err)
		}
		if val > max {
			return fmt.Errorf("value %d is greater than maximum %d", val, max)
		}
	}

	return nil
}

// validateFloatRange checks if a float value falls within the specified range
func validateFloatRange(value float64, minStr, maxStr string) error {
	if minStr != "" {
		min, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return fmt.Errorf("invalid min value: %w", err)
		}
		if value < min {
			return fmt.Errorf("value %f is less than minimum %f", value, min)
		}
	}

	if maxStr != "" {
		max, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return fmt.Errorf("invalid max value: %w", err)
		}
		if value > max {
			return fmt.Errorf("value %f is greater than maximum %f", value, max)
		}
	}

	return nil
}
