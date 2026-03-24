package config

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var durationType = reflect.TypeOf(time.Duration(0))

// parseValue sets field to the parsed form of value.
//
// An empty value is a no-op for all built-in types (field retains its zero value).
// The required check is handled separately in loadField.
func parseValue(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}

	// Type-specific cases first (types that share a Kind).
	switch field.Type() {
	case durationType:
		d, err := time.ParseDuration(value)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(d))
		return nil
	}

	// Kind-based cases.
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return err
		}
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("invalid float value: must be a finite number")
		}
		field.SetFloat(v)
	case reflect.Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(v)
	case reflect.Slice:
		return parseSlice(value, field)
	default:
		return fmt.Errorf("unsupported type: %s", field.Type())
	}
	return nil
}

// parseSlice splits value on commas and calls parseValue recursively for each element.
func parseSlice(value string, field reflect.Value) error {
	parts := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		elem := reflect.New(field.Type().Elem()).Elem()
		if err := parseValue(p, elem); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}
	field.Set(slice)
	return nil
}
