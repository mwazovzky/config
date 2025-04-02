package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// StringParser parses string values into the target field type
type StringParser struct{}

// Parse converts a string value to the target field type
func (p *StringParser) Parse(value string, field reflect.Value) error {
	field.SetString(value)
	return nil
}

// Int64Parser parses int64 values into the target field type
type Int64Parser struct{}

// Parse converts a string value to an int64 and sets it to the target field
func (p *Int64Parser) Parse(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}
	v, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return err
	}
	field.SetInt(v)
	return nil
}

// IntParser parses int values into the target field type
type IntParser struct{}

// Parse converts a string value to an int and sets it to the target field
func (p *IntParser) Parse(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	field.SetInt(int64(v))
	return nil
}

// SliceParser parses slice values into the target field type
type SliceParser struct{}

// Parse converts a comma-separated string into a slice and sets it to the target field
func (p *SliceParser) Parse(value string, field reflect.Value) error {
	return p.ParseWithContext(value, field)
}

// ParseWithContext provides the full functionality with parser provider
func (p *SliceParser) ParseWithContext(value string, field reflect.Value, parserProvider ...func(reflect.Kind) (ValueParser, bool)) error {
	if value == "" {
		return nil
	}

	values := strings.Split(value, ",")
	slice := reflect.MakeSlice(field.Type(), 0, len(values))

	// Get the element parser either from the provided function or defaultParsers
	var getParser func(reflect.Kind) (ValueParser, bool)
	if len(parserProvider) > 0 && parserProvider[0] != nil {
		getParser = parserProvider[0]
	} else {
		getParser = func(k reflect.Kind) (ValueParser, bool) {
			p, ok := defaultParsers[k]
			return p, ok
		}
	}

	elemParser, ok := getParser(field.Type().Elem().Kind())
	if !ok {
		return fmt.Errorf("unsupported slice element type: %v", field.Type().Elem().Kind())
	}

	for _, v := range values {
		elem := reflect.New(field.Type().Elem()).Elem()
		if err := elemParser.Parse(v, elem); err != nil {
			return err
		}
		slice = reflect.Append(slice, elem)
	}

	field.Set(slice)
	return nil
}

// DurationParser parses duration values into the target field type
type DurationParser struct{}

// Parse converts a string value to a time.Duration and sets it to the target field
func (p *DurationParser) Parse(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}

	// If no time unit is specified, assume seconds
	if _, err := strconv.Atoi(value); err == nil {
		value += "s"
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	field.Set(reflect.ValueOf(d))
	return nil
}

// BoolParser parses boolean values into the target field type
type BoolParser struct{}

// Parse converts a string value to a bool and sets it to the target field
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

// Float64Parser parses float64 values into the target field type
type Float64Parser struct{}

// Parse converts a string value to a float64 and sets it to the target field
func (p *Float64Parser) Parse(value string, field reflect.Value) error {
	if value == "" {
		return nil
	}
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	field.SetFloat(v)
	return nil
}

// defaultParsers maps reflect.Kind to their respective ValueParser implementations
var defaultParsers = map[reflect.Kind]ValueParser{
	reflect.String:  &StringParser{},
	reflect.Int64:   &Int64Parser{},
	reflect.Int:     &IntParser{},
	reflect.Slice:   &SliceParser{},
	reflect.Bool:    &BoolParser{},
	reflect.Float64: &Float64Parser{},
}
