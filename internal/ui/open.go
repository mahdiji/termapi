package ui

// openSelectedRequest mengisi panel request dari item yang dipilih di sidebar.
func (m *AppModel) openSelectedRequest() {
	if m == nil || m.Sidebar == nil || m.ReqState == nil {
		return
	}
	entry, collectionName, collectionPath, ok := m.Sidebar.SelectedRequestEntry()
	if !ok || entry.Request == nil {
		return
	}
	request := entry.Request
	request.Normalize()
	m.ReqState.LoadRequest(request)
	m.ReqState.CollectionName = collectionName
	m.ReqState.CollectionPath = collectionPath
	m.ReqState.RequestFilePath = entry.RelPath
	m.ReqState.RequestName = request.Name
	m.ReqState.SetFocused(true)
	m.Sidebar.IsFocused = false
	if m.State != nil {
		m.State.TabActive = 0
	}
}

func (m *AppModel) openHistory() {
	if m == nil || m.HistoryScreen == nil {
		return
	}
	m.reloadHistory()
	m.HistoryMode = true
	if m.State != nil {
		m.HistoryScreen.SetSize(m.State.Widht, m.State.Height)
	}
}

func (m *AppModel) openScopeManager() {
	if m == nil || m.ScopeScreen == nil {
		return
	}
	m.ScopeScreen.Load()
	m.ScopeMode = true
	if m.State != nil {
		m.ScopeScreen.SetSize(m.State.Widht, m.State.Height)
	}
}

func (m *AppModel) openEnvManager() {
	if m == nil || m.EnvScreen == nil {
		return
	}
	m.EnvScreen.Load()
	m.EnvMode = true
	if m.State != nil {
		m.EnvScreen.SetSize(m.State.Widht, m.State.Height)
	}
}

func (m *AppModel) openHelp() {
	if m == nil || m.HelpScreen == nil {
		return
	}
	m.HelpMode = true
	m.HelpScreen.ScrollTop()
	if m.State != nil {
		m.HelpScreen.SetSize(m.State.Widht, m.State.Height)
	}
}

func (m *AppModel) reloadHistory() {
	if m == nil || m.HistoryScreen == nil || m.HistoryStore == nil {
		return
	}
	runs, err := m.HistoryStore.List(500)
	if err != nil {
		m.HistoryScreen.Err = err.Error()
		return
	}
	m.HistoryScreen.Err = ""
	m.HistoryScreen.SetRuns(runs)
}
