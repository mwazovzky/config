# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go test -v ./...          # run all tests
go test -v -run TestName  # run a single test
./coverage.sh             # run tests with race detection and generate coverage report
```

## Architecture

This is a Go package (`github.com/mwazovzky/config`) that loads environment variables into Go structs using struct tags. The core flow:

1. `LoadConfig(cfg)` in `config.go` — entry point; validates input is a pointer to a struct
2. `loadStruct()` — iterates struct fields, recursing into nested structs (but not `time.Time`)
3. `loadField()` — reads the `env` tag, applies the prefix, rejects `time.Time` and `[]time.Time`, fetches the env var or default, then calls `checkRequired`, either `dec.Decode` (for Decoder fields) or `parseValue`, and finally `checkRange`

**Key interface** (`config.go`):
- `Decoder` — implemented by a field's type to control its own parsing; takes precedence over the built-in type switch

**Struct tags**: `env`, `required`, `default`, `min`, `max`, `range_error` — constants defined in `constants.go`.

**Extensibility**: `EnvLoader` has only one option: `WithPrefix`. Custom parsing is done by implementing `Decoder` on the field's type. A package-level `LoadConfig` convenience function wraps a default loader instance.

**`required` semantics**: `required:"true"` means the env var must be set (even to an empty string) or a `default` tag must exist. A zero value (e.g. `PORT=0`) satisfies required. Only a completely absent var with no default triggers failure. Non-standard values like "True" or "TRUE" return an error; only "true" and "false" (lowercase) are accepted.

**Parsers** (`parsers.go`): a single `parseValue(value string, field reflect.Value) error` function. It checks for `Decoder` (reached only for slice elements; direct fields are decoded in `loadField`), then falls through type-specific cases (e.g. `time.Duration`), then kind-based cases covering all int/uint/float widths, bool, string, and slice. `parseSlice` recurses via `parseValue` for each element, so `[]time.Duration` and custom `Decoder` types work in slices automatically.

**Validators** (`validators.go`): `checkRequired(tag, provided bool)` and `checkRange(field, tag)` are plain package-level functions. `checkRange` uses `field.Kind()` and `field.Int()`/`field.Float()` — no type switch on concrete types. Duration range errors use `time.Duration.String()` for readable messages (e.g. "500ms" not "500000000").
