package config

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

// parseDurationBound parses a min/max tag value for a time.Duration field.
// It tries time.ParseDuration first (e.g. "1s", "500ms"), then falls back to
// strconv.ParseInt for bare nanosecond integers (e.g. "1000000000").
func parseDurationBound(s string) (int64, error) {
	if d, err := time.ParseDuration(s); err == nil {
		return int64(d), nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// applyBounds checks that val is within [min, max], builds human-readable errors
// using format, and applies customErr if set. It is called after bounds are already
// parsed from tag strings.
func applyBounds[T int64 | uint64 | float64](
	val, min, max T, hasMin, hasMax bool,
	format func(T) string, customErr string,
) error {
	if hasMin && hasMax && min > max {
		return fmt.Errorf("invalid range: min %s is greater than max %s",
			format(min), format(max))
	}
	var rangeErr error
	if hasMin && val < min {
		rangeErr = fmt.Errorf("%s is less than minimum %s", format(val), format(min))
	}
	if rangeErr == nil && hasMax && val > max {
		rangeErr = fmt.Errorf("%s is greater than maximum %s", format(val), format(max))
	}
	if rangeErr != nil {
		if customErr != "" {
			return errors.New(customErr)
		}
		return fmt.Errorf("value out of range: %w", rangeErr)
	}
	return nil
}

// checkRequired returns an error when the field is tagged required:"true" but
// was not provided (no env var set and no default tag).
// It also rejects non-standard tag values (e.g. "True", "TRUE") to catch
// misconfiguration early.
func checkRequired(tag reflect.StructTag, provided bool) error {
	v := tag.Get(RequiredTag)
	if v == "" || v == tagTrue || v == "false" {
		if v == tagTrue && !provided {
			return fmt.Errorf("required field is missing")
		}
		return nil
	}
	return fmt.Errorf("invalid required tag value %q (must be %q or %q)", v, "true", "false")
}

// checkRange validates numeric fields against min/max struct tags.
// It handles all int/uint/float widths and time.Duration.
// Range tags on non-numeric types return an error.
// Duration error messages use time.Duration.String() for readability.
// The range_error tag is not applied to tag misconfiguration errors
// (invalid min/max format or min > max); those always return a plain error.
func checkRange(field reflect.Value, tag reflect.StructTag) error {
	minStr := tag.Get(MinTag)
	maxStr := tag.Get(MaxTag)
	if minStr == "" && maxStr == "" {
		return nil
	}

	customErr := tag.Get(RangeErrTag)

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		isDuration := field.Type() == durationType
		val := field.Int()
		var min, max int64
		if minStr != "" {
			var err error
			if isDuration {
				min, err = parseDurationBound(minStr)
				if err != nil {
					return fmt.Errorf("invalid min value (must be a duration string (e.g. \"1s\") or nanosecond integer): %w", err)
				}
			} else {
				min, err = strconv.ParseInt(minStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid min value: %w", err)
				}
			}
		}
		if maxStr != "" {
			var err error
			if isDuration {
				max, err = parseDurationBound(maxStr)
				if err != nil {
					return fmt.Errorf("invalid max value (must be a duration string (e.g. \"1s\") or nanosecond integer): %w", err)
				}
			} else {
				max, err = strconv.ParseInt(maxStr, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid max value: %w", err)
				}
			}
		}
		var fmtFn func(int64) string
		if isDuration {
			fmtFn = func(n int64) string { return time.Duration(n).String() }
		} else {
			fmtFn = func(n int64) string { return strconv.FormatInt(n, 10) }
		}
		return applyBounds(val, min, max, minStr != "", maxStr != "", fmtFn, customErr)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val := field.Uint()
		var min, max uint64
		if minStr != "" {
			var err error
			min, err = strconv.ParseUint(minStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid min value: %w", err)
			}
		}
		if maxStr != "" {
			var err error
			max, err = strconv.ParseUint(maxStr, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid max value: %w", err)
			}
		}
		return applyBounds(val, min, max, minStr != "", maxStr != "",
			func(n uint64) string { return strconv.FormatUint(n, 10) }, customErr)

	case reflect.Float32, reflect.Float64:
		val := field.Float()
		// Guard for Decoder implementations that may produce non-finite values;
		// parseValue already blocks NaN/Inf for built-in float parsing.
		if math.IsNaN(val) || math.IsInf(val, 0) {
			if customErr != "" {
				return errors.New(customErr)
			}
			return fmt.Errorf("value out of range: field value is not a finite number")
		}
		var min, max float64
		if minStr != "" {
			var err error
			min, err = strconv.ParseFloat(minStr, field.Type().Bits())
			if err != nil {
				return fmt.Errorf("invalid min value: %w", err)
			}
			if math.IsNaN(min) || math.IsInf(min, 0) {
				return fmt.Errorf("invalid min value: must be a finite number")
			}
		}
		if maxStr != "" {
			var err error
			max, err = strconv.ParseFloat(maxStr, field.Type().Bits())
			if err != nil {
				return fmt.Errorf("invalid max value: %w", err)
			}
			if math.IsNaN(max) || math.IsInf(max, 0) {
				return fmt.Errorf("invalid max value: must be a finite number")
			}
		}
		bits := field.Type().Bits()
		return applyBounds(val, min, max, minStr != "", maxStr != "",
			func(n float64) string { return strconv.FormatFloat(n, 'g', -1, bits) }, customErr)

	case reflect.Slice:
		return fmt.Errorf("min/max range tags are not supported on slice fields")
	default:
		return fmt.Errorf("range validation not supported for type %s", field.Type())
	}
}
