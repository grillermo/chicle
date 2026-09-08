package chicle

// moveButton walks the action row. Unlike the row cursor this clamps: with a
// handful of buttons, holding a direction should stop at the ends rather than
// snap back to the other side.
func (m Model) moveButton(delta int) Model {
	if len(m.cfg.Actions) == 0 {
		return m
	}
	m.button += delta
	if m.button < 0 {
		m.button = 0
	}
	if m.button >= len(m.cfg.Actions) {
		m.button = len(m.cfg.Actions) - 1
	}
	return m
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

	action := m.cfg.Actions[m.button]
	if action.Run == nil {
		m.quit = true // a bare cancel button
		return m, true
	}
	if len(m.visibleRows()) == 0 {
		return m, false // nothing matches the filter to act on
	}
	return m.runAction(action)
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
	return m, false
}
