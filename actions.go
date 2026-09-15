package chicle

// shownActions lists the actions currently on offer, as indices into
// cfg.Actions. Indices rather than Actions because m.button names an action by
// its position in the configured slice, which must not shift underneath it
// when a Show starts or stops returning true.
func (m Model) shownActions() []int {
	out := make([]int, 0, len(m.cfg.Actions))
	var sel Selection
	built := false
	for i, a := range m.cfg.Actions {
		if a.Show == nil {
			out = append(out, i)
			continue
		}
		if !built {
			sel, built = m.Selection(), true // re-filters the rows; only pay if asked
		}
		if a.Show(sel) {
			out = append(out, i)
		}
	}
	return out
}

// clampButton pulls focus onto an action that is actually drawn. The focused
// one can vanish -- clearing the filter takes its action with it -- and focus
// left on an undrawn button is focus the user cannot see.
func (m Model) clampButton() Model {
	shown := m.shownActions()
	if len(shown) == 0 {
		m.button = 0
		return m
	}
	nearest := shown[0]
	for _, i := range shown {
		if i == m.button {
			return m
		}
		if i < m.button {
			nearest = i // the last one before the gap; shown is ascending
		}
	}
	m.button = nearest
	return m
}

// moveButton walks the action row, skipping hidden actions. Unlike the row
// cursor this clamps: with a handful of buttons, holding a direction should
// stop at the ends rather than snap back to the other side.
func (m Model) moveButton(delta int) Model {
	shown := m.shownActions()
	if len(shown) == 0 {
		return m
	}
	m = m.clampButton()

	at := 0
	for j, i := range shown {
		if i == m.button {
			at = j
			break
		}
	}
	at += delta
	if at < 0 {
		at = 0
	}
	if at >= len(shown) {
		at = len(shown) - 1
	}
	m.button = shown[at]
	return m
}

// keyedAction reports the index of the action whose Key matches s, if any.
// When more than one Action shares a Key, the last one in the slice wins.
// A hidden action keeps its Key to itself: it is not on offer.
func (m Model) keyedAction(s string) (int, bool) {
	found := -1
	for _, i := range m.shownActions() {
		if a := m.cfg.Actions[i]; a.Key != "" && a.Key == s {
			found = i
		}
	}
	return found, found >= 0
}

// activate fires the focused action, or — on a list with no actions at all —
// returns the cursor row's Key, which is the whole API for a plain picker.
func (m Model) activate() (Model, bool) {
	if len(m.cfg.Actions) == 0 {
		cur := m.CursorRow()
		if cur.Key == "" {
			return m, false
		}
		m.result = cur.Key
		m.quit = true
		return m, true
	}

	// Focus can be stale: the filter may have hidden the action under it since
	// the last keystroke that moved the button row.
	m = m.clampButton()
	if len(m.shownActions()) == 0 {
		return m, false
	}

	action := m.cfg.Actions[m.button]
	if action.Run == nil {
		m.quit = true // a bare cancel button
		return m, true
	}
	if len(m.visibleRows()) == 0 && !action.OnEmpty {
		return m, false // nothing matches the filter to act on
	}
	if action.Confirm != nil {
		if q := action.Confirm(m.Selection()); q != "" {
			m.confirming = true
			m.confirmYes = false // start on No: the prompt exists because the
			m.question = q       // action is worth a second thought
			m.status = ""
			return m, false
		}
	}
	return m.runAction(action)
}

// updateConfirm answers the pending prompt. The prompt reads "[ Yes ]  [ No ]",
// so left means Yes and right means No — positional rather than a toggle, so
// holding a direction never flips you back onto the destructive answer.
func (m Model) updateConfirm(k string) (Model, bool) {
	switch k {
	case "left", "h":
		m.confirmYes = true
	case "right", "l":
		m.confirmYes = false
	case "esc", "ctrl+c":
		m.confirming = false
	case "y", "Y":
		m.confirming = false
		return m.runAction(m.cfg.Actions[m.button])
	case "n", "N", "q":
		m.confirming = false
	case "enter":
		m.confirming = false
		if m.confirmYes {
			return m.runAction(m.cfg.Actions[m.button])
		}
	}
	return m, false
}

// runAction performs an action and folds in whatever it changed.
func (m Model) runAction(action Action) (Model, bool) {
	out := action.Run(m.Selection())
	m.status = out.Status
	if out.Done {
		m.result = out.Result
		m.quit = true
		return m, true
	}

	if m.cfg.Reload != nil {
		rows, err := m.cfg.Reload()
		if err != nil {
			// Append rather than replace: the action's own message still
			// matters even though the refresh failed.
			if m.status != "" {
				m.status += "; "
			}
			m.status += "reloading failed: " + err.Error()
			return m, false
		}
		m.rows = rows
		m = m.normaliseLocks().clampCursor().scrollToCursor()
	}

	// Acting on the last row leaves nothing to act on.
	if len(m.rows) == 0 {
		m.quit = true
		return m, true
	}
	return m, false
}
