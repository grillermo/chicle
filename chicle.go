// Package chicle renders a filterable, navigable list in the terminal and
// returns what the user picked.
//
// The zero Config is not useful: at minimum set Columns and Rows. Everything
// else is opt-in, so a plain single-select list stays a three-line setup.
package chicle

// Column is one field of the table. A zero Width means "whatever is left",
// which only makes sense on the last column.
type Column struct {
	Title string
	Width int
}

// Row is one selectable line: the cells to render, plus the Key handed back to
// the caller. Key is opaque to chicle — it is whatever the caller needs to act
// on the row, and does not have to appear in Cols.
type Row struct {
	Key  string
	Cols []string

	// Section groups rows under a heading. Rows sharing a Section must be
	// adjacent in the slice; chicle does not reorder them. An empty Section on
	// every row means no headings are drawn.
	Section string

	// Ticked is the initial checkbox state, and is only read at New time.
	// Meaningless unless Config.MultiSelect is set.
	Ticked bool

	// Locked rows are always ticked and can never be unticked — not by the
	// spacebar, and not by "select none". Use it for choices that are already
	// committed elsewhere and must be undone deliberately.
	Locked bool
}

// Selection is what an Action receives: the row under the cursor, plus every
// ticked row. With MultiSelect off, Ticked holds exactly Cursor.
type Selection struct {
	Cursor Row
	Ticked []Row

	// Filter is the query the list is narrowed by, empty when it is not
	// narrowed at all. An action can act on it rather than on a row, which is
	// how a list offers "none of these, use what I typed".
	Filter string
}

// Outcome is what an Action reports back.
type Outcome struct {
	// Result is returned from Run when Done is set. It is the picker's answer.
	Result string
	// Done ends the picker. When false the list stays up, Status is shown
	// underneath, and Config.Reload runs if it is set.
	Done bool
	// Status is a one-line message under the action row. Cleared by the next
	// action.
	Status string
}

// Action is one button in the row under the list.
type Action struct {
	Label string
	// Key, when set, fires this action immediately on that key (bubbletea's
	// tea.KeyMsg.String() form, e.g. "f2"), without requiring the action to
	// be focused first. Leave empty for actions only reachable via
	// left/right and Enter. If more than one Action shares a Key, the last
	// one in the slice wins — chicle does not validate this at New or Run
	// time, consistent with the rest of Config. Avoid keys chicle already
	// binds (up/down/left/right/enter/esc/q/ctrl+c/j/k, and space/a/n under
	// MultiSelect) — Key shadows them for this action rather than erroring.
	// Pressing Key also moves button focus to this action so Confirm/Run
	// behave exactly as Enter would; if a Confirm prompt is then cancelled,
	// focus stays on this action rather than reverting to what was focused
	// before the keypress.
	Key string
	// OnEmpty lets the action run when the filter matches nothing. Off by
	// default, because an action normally acts on the cursor row and there is
	// no cursor row to act on. Set it for an action that acts on
	// Selection.Filter instead: "nothing matched" is precisely when the user
	// wants to do something with what they typed.
	OnEmpty bool
	// Show decides whether the action is offered at all. Nil means always.
	// A false Show hides the button rather than greying it out, and takes its
	// Key with it: an action that cannot be reached by Enter should not be
	// reachable by hotkey either. Focus never rests on a hidden action -- it
	// slides to the nearest shown one -- so an action can appear and disappear
	// as the filter changes without stranding the keyboard.
	Show func(Selection) bool
	// Confirm, when set, is asked before Run. Returning "" skips the prompt.
	Confirm func(Selection) string
	// Run acts on the selection. A nil Run makes a bare "cancel" button that
	// ends the picker with an empty result.
	Run func(Selection) Outcome
}

// Config describes the list. It is copied into the Model by New; mutating it
// afterwards has no effect.
type Config struct {
	// Title is drawn above the list. Empty draws no title line.
	Title   string
	Columns []Column
	Rows    []Row

	// Filter pre-fills the query, as though the user had pressed "/" and typed
	// it: the list opens narrowed, with the keyboard in the query so it can be
	// refined, and Enter fires the focused action on whatever is left. Use it
	// to open on a guess -- the caller had a name in mind and wants it
	// confirmed rather than assumed.
	Filter string

	// Actions is the button row under the list. When empty, chicle draws no
	// button row and Enter returns the cursor row's Key directly — which is
	// the whole API for a plain "pick one thing" list.
	Actions []Action

	// MultiSelect turns on checkboxes: space toggles, "a" ticks everything,
	// "n" unticks everything unlocked.
	MultiSelect bool

	// NoCycle stops the cursor at the ends of the list instead of wrapping
	// around. Cycling is the default because it is what a short list wants;
	// clamping is what a long one wants.
	NoCycle bool

	// Reload refetches rows after an action that left the picker open. Nil
	// when nothing an action does can make the list stale.
	Reload func() ([]Row, error)

	// Updates carries rows filled in after the list is already on screen, so a
	// slow column does not hold up the first draw. Each send replaces all
	// rows; closing the channel says no more are coming.
	Updates <-chan []Row
}
