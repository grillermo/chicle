# chicle

A Go library for filterable, navigable terminal pickers. Give it columns and
rows and it draws a list on the terminal, lets the user search, move, tick and
act on rows, then returns what they chose.

The UI draws on `/dev/tty`, not stdout, so a picker built with chicle works
inside command substitution: `sel=$(mypicker)`. Built on
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Features

- **Plain single-select lists** in a few lines. With no actions configured,
  Enter returns the cursor row's `Key`.
- **Filtering**: just type. Terms are case-insensitive and ANDed, and each can
  match any cell, so `web prod` narrows twice. The query line supports
  `←` `→`, `home`, `end`, `backspace`, `ctrl+w` and `ctrl+u`.
- **Pre-filled filter**: open the list already narrowed to a guess, with the
  query editable so the user can correct it.
- **Multi-select** with checkboxes, select all and select none, plus **locked**
  rows that stay ticked.
- **Section headings** that group adjacent rows. The cursor skips headings.
- **Action buttons** under the list, each with an optional hotkey, an optional
  confirmation prompt and an optional `Show` condition that hides it.
- **Actions on the filter text** (`OnEmpty` + `Selection.Filter`) for
  "none of these, use what I typed".
- **Reload** after an action that keeps the picker open. The filter is kept.
- **Async row updates** through a channel, so slow columns fill in after the
  first draw without moving the cursor.
- **Scrolling window** with "N more above/below" hints, and cursor cycling
  that you can turn off.
- Wide characters are measured correctly when cells are clamped to column
  width, and colors are detected against the terminal even when stdout is a
  pipe.
- Handles SIGINT/SIGTERM without leaving the terminal in raw mode.

## Installation

```sh
go get github.com/grillermo/chicle
```

## Usage

### Pick one thing

```go
package main

import (
	"fmt"
	"os"

	"github.com/grillermo/chicle"
)

func main() {
	choice, err := chicle.Run(chicle.Config{
		Title:   "Pick a server",
		Columns: []chicle.Column{{Title: "Name", Width: 20}, {Title: "Region"}},
		Rows: []chicle.Row{
			{Key: "web-1", Cols: []string{"web-1", "us-east"}},
			{Key: "web-2", Cols: []string{"web-2", "eu-west"}},
			{Key: "db-1", Cols: []string{"db-1", "us-east"}},
		},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if choice == "" {
		os.Exit(1) // the user quit without choosing
	}
	fmt.Println(choice)
}
```

`Run` blocks until the user finishes and returns the chosen string, or `""` if
they quit. It writes nothing to stdout, so print the result yourself if a
shell is waiting for it:

```sh
server=$(mypicker) && ssh "$server"
```

A `Column` with `Width: 0` takes the remaining space, so use it only on the
last column. `Row.Key` goes back to the caller and doesn't have to appear in
`Cols`.

### Actions

Actions add a button row under the list. Each `Run` gets a `Selection` (the
cursor row, the ticked rows and the current filter) and returns an `Outcome`.

```go
chicle.Config{
	Columns: cols,
	Rows:    rows,
	Actions: []chicle.Action{
		{
			Label: "Open",
			Run: func(s chicle.Selection) chicle.Outcome {
				return chicle.Outcome{Result: s.Cursor.Key, Done: true}
			},
		},
		{
			Label: "Delete",
			Key:   "f2",
			Confirm: func(s chicle.Selection) string {
				return "Delete " + s.Cursor.Key + "?"
			},
			Run: func(s chicle.Selection) chicle.Outcome {
				if err := deleteThing(s.Cursor.Key); err != nil {
					return chicle.Outcome{Status: err.Error()}
				}
				return chicle.Outcome{Status: "deleted " + s.Cursor.Key}
			},
		},
		{Label: "Cancel"}, // nil Run: ends the picker with an empty result
	},
	// Refetch rows after an action that leaves the picker open.
	Reload: loadRows,
}
```

- `Done: true` ends the picker and `Run` returns `Result`.
- `Done: false` keeps the list open, shows `Status` under the buttons, and
  calls `Config.Reload` if it's set. If the list ends up empty, the picker
  closes.
- `Key` fires the action right away, even with a query typed. It must be a
  named key such as `f2`, `ctrl+n` or `alt+x`: printable keys type into the
  query, so a `Key` like `"d"` never fires. Don't use keys chicle already
  binds (see [Keys](#keys)): an action's `Key` takes over that key instead of
  raising an error. If two actions share a `Key`, the last one
  wins.
- `Show` hides the action, including its hotkey, whenever it returns false.
- `OnEmpty` lets the action run when the filter matches no rows. Use it with
  `Selection.Filter`:

```go
{
	Label:   "Create",
	Key:     "ctrl+n",
	OnEmpty: true,
	Show:    func(s chicle.Selection) bool { return s.Filter != "" },
	Run: func(s chicle.Selection) chicle.Outcome {
		return chicle.Outcome{Result: "new:" + s.Filter, Done: true}
	},
}
```

### Multi-select

```go
chicle.Config{
	MultiSelect: true,
	Rows: []chicle.Row{
		{Key: "a", Cols: []string{"already installed"}, Locked: true},
		{Key: "b", Cols: []string{"suggested"}, Ticked: true},
		{Key: "c", Cols: []string{"optional"}},
	},
	// ...
}
```

`Selection.Ticked` holds every ticked row. `Locked` rows are always ticked and
can't be unticked. `space` toggles the row under the cursor, `ctrl+a` ticks
every row and `ctrl+x` unticks every unlocked row. Because `space` toggles, a
space can't be typed into the query under `MultiSelect`. Without `MultiSelect`, `Ticked` holds only the cursor row,
so the same `Run` works for both kinds of list.

### Sections

Set `Row.Section` to group rows under headings. Rows in the same section must
be next to each other in the slice, because chicle doesn't reorder them. No
headings are drawn when every visible row is in the same section.

### Opening on a filter

```go
chicle.Config{Filter: "prod", /* ... */}
```

The list opens with `prod` already in the query, narrowed to matching rows.
The user can edit it like any typed text.

### Slow columns

Send replacement row sets on `Updates` and the list redraws in place:

```go
updates := make(chan []chicle.Row)
go func() {
	defer close(updates)
	updates <- withSizes(rows) // each send replaces all rows
}()

chicle.Run(chicle.Config{Rows: rows, Updates: updates /* ... */})
```

The cursor stays on the row it was on (matched by `Key`), and ticks are kept.

### Other options

| Field     | Effect                                                         |
| --------- | -------------------------------------------------------------- |
| `Title`   | Line drawn above the list.                                     |
| `NoCycle` | Stop the cursor at the ends of the list instead of wrapping.   |

## Keys

Printable keys type straight into the query; there is no separate filter mode.

| Key                        | Does                                              |
| -------------------------- | ------------------------------------------------- |
| any printable character    | Insert into the query at the cursor               |
| `space`                    | Toggle tick under `MultiSelect`, else insert a space |
| `↑` `↓`                    | Move the row cursor                               |
| `ctrl+e`, `ctrl+f`, `pgdn` | Page down                                         |
| `ctrl+b`, `pgup`           | Page up                                           |
| `←` `→`                    | Move the query cursor                             |
| `home` / `end`             | Query cursor to start / end                       |
| `backspace`                | Delete a character                                |
| `ctrl+w` / `ctrl+u`        | Delete a word / the whole query                   |
| `tab` / `shift+tab`        | Next / previous action button                     |
| `enter`                    | Run focused action / pick                         |
| `esc`                      | Clear the query if there is one, else quit        |
| `ctrl+c`                   | Quit                                              |
| `ctrl+a` / `ctrl+x`        | `MultiSelect`: tick all / untick all              |

In a confirmation prompt, `←`/`y` means yes and `→`/`n` means no. `enter`
answers and `esc` cancels. The prompt starts on **No**.

## Upgrading to v0.4.0

Breaking change: the `/` step is gone and typing filters immediately. `/`, `j`,
`k`, `q`, `a`, `n` and `x` are now ordinary characters that go into the query.
Quit with `esc` (when the query is empty) or `ctrl+c`; tick all and none are
`ctrl+a` and `ctrl+x`; move between buttons with `tab` and `shift+tab`. An
`Action.Key` that is a printable character never fires, so switch it to a
named key such as `f2` or `ctrl+n`.

## Testing

```sh
go test ./...
```

`chicle.New(cfg)` returns a `Model` that implements `tea.Model` and holds no
terminal state. You can drive it in tests by calling `Update` directly and
reading `Result()`, `Selection()` and `CursorRow()`.
