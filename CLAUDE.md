# chicle — notes for Claude

A Go library for filterable terminal pickers, built on Bubble Tea + Lip Gloss.
See `README.md` for the public API. This file is about **how to test a change**.

## The key fact: the UI is testable without a terminal

`Model` (model.go) holds the whole UI and is terminal-free. `Run` (run.go) only
adds the `/dev/tty` wiring on top of it; `runOn` is split out so even the
open-tty failure path is testable. So **never open a terminal in a test and
never call `Run`** — drive the `Model` directly:

- `New(cfg)` builds the model.
- `Update(msg)` advances it. Feed it `tea.WindowSizeMsg` and `tea.KeyMsg`.
- `View() string` is the rendered frame. Assert on substrings of it.
- `Result()`, `Selection()`, `CursorRow()`, `visibleRows()` read state out.

Two things that trip people up:

- **Resize before asserting on rows.** The windowing code needs a size to know
  what fits; a model that never got a `WindowSizeMsg` has height 0 and draws no
  rows. Send one first.
- **`Reload` runs synchronously inside `Update`.** After a keystroke that fires
  an action returning `Done:false`, the returned `Model` already has the
  reloaded rows — no command to pump, no async. Assert on it directly.

## Running the tests

```bash
go test ./...
go vet ./...
gofmt -l .   # should print nothing
```

## Writing tests inside the package (`package chicle`)

Use the shared helpers in `keys_test.go` — don't reinvent them:

- `press(t, m, "down", "enter")` — feed keys in order; named keys (`"enter"`,
  `"tab"`, `"ctrl+a"`, `"f2"`, …) map to the right `tea.KeyType`, and any other
  multi-char string is typed one rune at a time (`"foo"` → f, o, o).
- `resize(t, m, 80, 24)` — deliver a terminal size.
- `view(t, m)` — the rendered frame.
- `fixture("a", "b", "c")` / `rows(...)` — a quick single-select list.
- `keys(m)` — the visible rows' keys; `eq(t, got, want)` compares slices.

Canonical shape:

```go
func TestDownThenEnterPicksSecond(t *testing.T) {
	m := press(t, fixture("a", "b", "c"), "down", "enter")
	if got := m.Result(); got != "b" {
		t.Fatalf("result %q, want b", got)
	}
}
```

For rendering assertions, resize first: `view(t, resize(t, m, 80, 24))`.

## Testing from a package that imports chicle

The internal helpers above aren't exported, so an external consumer drives the
exported API itself: `New(cfg)`, send a `tea.WindowSizeMsg`, then `tea.KeyMsg`s
to `Update`, and read `View()` / `Result()`. The claude-menu repo next door
(`../claude-menu/ux_test.go`) is a worked example — it drives chicle's picker
through a full disable → reload → enable flow with no terminal.

## When changing behaviour

- Add or update a test in the matching `*_test.go` (actions, filter, render,
  window, select, section, …) and keep `go test ./...` green.
- If a change is user-visible (keys, layout, an API field), update `README.md`
  too — its Keys table and Upgrading notes are the contract.
