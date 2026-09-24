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
- **Filtering**: press `/` and type. Terms are case-insensitive and ANDed, and
  each can match any cell, so `web prod` narrows twice. Query editing supports
  readline keys (`ctrl+a`, `ctrl+e`, `ctrl+w`, `ctrl+u`).
- **Pre-filled filter**: open the list already narrowed to a guess, with the
  keyboard in the query so the user can correct it.
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
			Key:   "d",
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
- `Key` fires the action right away, even while filtering. Don't use keys
  chicle already binds (see [Keys](#keys)): an action's `Key` takes over that
  key instead of raising an error. If two actions share a `Key`, the last one
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
can't be unticked. Without `MultiSelect`, `Ticked` holds only the cursor row,
so the same `Run` works for both kinds of list.

### Sections

Set `Row.Section` to group rows under headings. Rows in the same section must
be next to each other in the slice, because chicle doesn't reorder them. No
headings are drawn when every visible row is in the same section.

### Opening on a filter

```go
chicle.Config{Filter: "prod", /* ... */}
```

The list opens as if the user had pressed `/` and typed `prod`.

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

| Key                   | List                         | While filtering           |
| --------------------- | ---------------------------- | ------------------------- |
| `↑` `↓` / `k` `j`     | Move cursor                  | `↑` `↓` move cursor       |
| `←` `→`               | Move between actions         | Move within the query     |
| `enter`               | Run focused action / pick    | Run focused action / pick |
| `/`                   | Start filtering              | Types `/`                 |
| `tab`                 |                              | Back to the list, keeping the filter |
| `esc`                 | Clear filter, else quit      | Clear filter              |
| `q`, `ctrl+c`         | Quit                         | `ctrl+c` quits            |
| `space`, `x`          | Toggle tick (multi-select)   | `space` toggles tick (multi-select) |
| `a` / `n`             | Tick all / untick all        |                           |
| `ctrl+a` / `ctrl+e`   |                              | Start / end of query      |
| `ctrl+w` / `ctrl+u`   |                              | Delete word / whole query |

In a confirmation prompt, `←`/`y` means yes and `→`/`n` means no. `enter`
answers and `esc` cancels. The prompt starts on **No**.

## Testing

```sh
go test ./...
```

`chicle.New(cfg)` returns a `Model` that implements `tea.Model` and holds no
terminal state. You can drive it in tests by calling `Update` directly and
reading `Result()`, `Selection()` and `CursorRow()`.
