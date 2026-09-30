# Code style

Rules that gofmt and eslint do not enforce. Applied across the whole tree; new code must follow them.

## Go

### Logical blocks

A logical block is a short run of statements that belong together, the canonical case being a
call and its error check:

```go
data, err := os.ReadFile(path)
if err != nil {
    return nil, err
}

var s snapshot
if err := json.Unmarshal(data, &s); err != nil {
    return nil, err
}

return &s, nil
```

* The assignment that produces `err` and the `if err != nil` check are never separated by a
  blank line: they are one block.
* Every block that ends with a closing brace (`if`, `for`, `switch`, `select`, `range`) is
  followed by a blank line when more code follows in the same scope.
* Plain statements are grouped by intent; a blank line separates groups, not every line.

### Comments

Code is the documentation, and it carries no comments. The only `//` lines that belong in the
tree are machine directives (`//go:build`, `//go:embed`, `//go:generate`, `//nolint`) and the
swag annotations (`// @...`) that generate the OpenAPI spec. Shell scripts keep only their
shebang. Naming and structure must make the intent clear on their own; if a fragment cannot be
understood without prose, restructure it until it can.

### No inline functions and structs

Anonymous functions and anonymous struct types in the middle of logic hurt readability.
Prefer:

* a named top-level function over a closure passed inline, unless the closure captures
  locals and is only a few lines;
* a named type over `struct{ ... }` literals declared at the use site, including for
  JSON request/response shapes;
* named types for map/slice element structs.

Small `func()` literals for `sort.Slice`, `defer`, or goroutine bodies of one or two lines
are fine.

## Frontend (web/)

* Pages under `src/pages/` stay thin: layout, data fetching, and wiring. Reusable or
  sizeable UI belongs in `src/components/` (feature components) or `src/components/ui/`
  (primitives).
* One component per file when the component exceeds a screenful; group related small
  components in a single file only when they are private to a feature.
* Follow Adwaita/GNOME patterns with the existing primitives.
* Configuration forms use Cheval UI controls. Use `PreferencesGroup` with
  `EntryRow`, `ComboRow`, and `SwitchRow` so labels, descriptions, and controls
  align consistently. Do not recreate field grids or use native inputs and
  selects when a shared control exists.
* Preference rows use the shared 20rem control column on desktop so every
  input, select, switch, and custom control starts at the same position. The
  control column becomes full-width on mobile.
* Hand-built field stacks use at least `space-y-1.5` between a label and its
  control. Keep denser spacing for tabular data and non-form status lists only.
* No em dashes anywhere (code, UI strings, docs); use a comma, colon, or parentheses.

## Commits

Lowercase imperative subject, optionally prefixed by the touched area (`bgp: ...`,
`friends: ...`). No conventional-commit prefixes, no trailers.

## Linting

Run `task lint` before pushing: `golangci-lint` (config in `.golangci.yml`) for
Go and ESLint (`web/eslint.config.js`) for the frontend. CI runs both on every
push and blocks packaging on failures. Intentionally ignored errors are written
as explicit `_ =` assignments, never left bare.

## AI usage

This project is open to AI-assisted contributions. I build it with tools like
Claude Code while reviewing the code they produce, and you are free to use
Claude, Codex, or just your hands.

Whatever you use, you own the result: know what the code does, test it, and be
able to explain it. A model writing it changes nothing. I review backend and
networking code closely, and the frontend (`web/`) more loosely for now, since
I care more about the UX than the frontend code itself. That may change later.
