package config

import (
	"fmt"
	"reflect"
)

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
