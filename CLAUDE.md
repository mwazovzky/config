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
2. `loadStruct()` — iterates struct fields, recursing into nested structs
3. `loadField()` — reads the `env` tag, fetches the env var or default, then calls `checkRequired` and `parseValue`

**Struct tags**: `env`, `required`, `default` — constants defined in `constants.go`.

**`required` semantics**: `required:"true"` means the env var must be set (even to an empty string) or a `default` tag must exist. A zero value (e.g. `PORT=0`) satisfies required. Only a completely absent var with no default triggers failure. Non-standard values like "True" or "TRUE" return an error; only "true" and "false" (lowercase) are accepted.

**Parsers** (`parsers.go`): a single `parseValue(value string, field reflect.Value) error` function. It handles type-specific cases (e.g. `time.Duration`), then kind-based cases covering all int/uint/float widths, bool, string, and slice. `parseSlice` recurses via `parseValue` for each element, so `[]time.Duration` works in slices automatically.

**Validators** (`validators.go`): `checkRequired(tag, provided bool)` is a plain package-level function.
