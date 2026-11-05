package ui

import (
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	"api/internal/collectionstore"
	"api/internal/historydb"
	requestinput "api/internal/ui/requestInput"
	tea "charm.land/bubbletea/v2"
)

type batchResponseMsg struct {
	index    int
	response requestinput.ResponseMsg
}

func (m *AppModel) openBatchChooser() {
	if m.Sidebar == nil || m.Sidebar.CollectionStore == nil {
		return
	}
	folder := m.Sidebar.SelectedFolderPath()
	if folder == "" {
		m.BatchError = "Pilih folder koleksi terlebih dahulu"
		return
	}
	entries, err := m.Sidebar.CollectionStore.Load()
	if err != nil {
		m.BatchError = err.Error()
		m.BatchChoosing = true
		return
	}
	var selected *collectionstore.Entry
	var find func([]collectionstore.Entry)
	find = func(items []collectionstore.Entry) {
		for i := range items {
			if items[i].RelPath == folder {
				selected = &items[i]
				return
			}
			find(items[i].Children)
			if selected != nil {
				return
			}
		}
	}
	find(entries)
	if selected == nil {
		m.BatchError = "Folder koleksi tidak ditemukan"
		m.BatchChoosing = true
		return
	}
	m.BatchFolder = folder
	m.BatchMaxDepth = subtreeDepth(*selected)
	m.BatchDepth = m.BatchMaxDepth // default includes every nested folder
	m.BatchChain = false
	m.BatchRetries = 0
	m.BatchError = ""
	m.BatchChoosing = true
}

func subtreeDepth(e collectionstore.Entry) int {
	maxDepth := 0
	for _, child := range e.Children {
		if child.IsFolder {
			maxDepth = max(maxDepth, 1+subtreeDepth(child))
		}
	}
	return maxDepth
}

func (m *AppModel) beginBatch() tea.Cmd {
	entries, err := m.Sidebar.CollectionStore.Load()
	if err != nil {
		m.BatchChoosing = true
		m.BatchError = err.Error()
		return nil
	}
	var selected *collectionstore.Entry
	var find func([]collectionstore.Entry)
	find = func(items []collectionstore.Entry) {
		for i := range items {
			if items[i].RelPath == m.BatchFolder {
				selected = &items[i]
				return
			}
			find(items[i].Children)
			if selected != nil {
				return
			}
		}
	}
	find(entries)
	if selected == nil {
		m.BatchChoosing = true
		m.BatchError = "Folder koleksi tidak ditemukan"
		return nil
	}
	var requests []collectionstore.Entry
	collectRequests(*selected, 0, m.BatchDepth, &requests)
	if len(requests) == 0 {
		m.BatchChoosing = true
		m.BatchError = "Tidak ada request pada kedalaman yang dipilih"
		return nil
	}
	sort.Slice(requests, func(i, j int) bool { return requests[i].RelPath < requests[j].RelPath })
	m.BatchChoosing, m.BatchMode = false, true
	m.BatchExpected, m.BatchSelected = len(requests), 0
	m.BatchResults = make([]requestinput.ResponseMsg, len(requests))
	m.BatchNames = make([]string, len(requests))
	for i, entry := range requests {
		m.BatchNames[i] = entry.Request.Name
	}
	if m.Response != nil {
		m.Response.SetLoading()
	}
	m.BatchError = ""
	m.BatchSpecs = make([]requestinput.BatchSpec, 0, len(requests))
	for _, entry := range requests {
		req := *entry.Request
		req.Normalize()
		collectionName := strings.Split(entry.RelPath, "/")[0]
		m.BatchSpecs = append(m.BatchSpecs, requestinput.BatchSpec{
			Request:        req,
			GlobalScope:    m.State.GlobalScope,
			CollectionName: collectionName,
			CollectionPath: path.Dir(entry.RelPath),
		})
	}
	m.BatchVariables = make(map[string]string)
	m.BatchNext = 0
	if m.BatchChain {
		return m.nextChainedRequest()
	}
	cmds := make([]tea.Cmd, 0, len(m.BatchSpecs))
	for index, spec := range m.BatchSpecs {
		index, spec := index, spec
		retries := m.BatchRetries
		cmds = append(cmds, func() tea.Msg {
			return batchResponseMsg{index: index, response: requestinput.RunBatchRequestWithRetry(spec, retries)}
		})
	}
	return tea.Batch(cmds...)
}

func (m *AppModel) nextChainedRequest() tea.Cmd {
	if m == nil || m.BatchNext >= len(m.BatchSpecs) {
		return nil
	}
	index := m.BatchNext
	m.BatchNext++
	spec := applyBatchVariables(m.BatchSpecs[index], m.BatchVariables)
	retries := m.BatchRetries
	if token := firstUnresolvedTemplate(spec); token != "" {
		return func() tea.Msg {
			return batchResponseMsg{index: index, response: requestinput.ResponseMsg{Method: spec.Request.Method, URL: spec.Request.URL, RequestName: spec.Request.Name, CollectionName: spec.CollectionName, CollectionPath: spec.CollectionPath, Err: "unresolved chained variable " + token, Attempts: 1}}
		}
	}
	return func() tea.Msg {
		return batchResponseMsg{index: index, response: requestinput.RunBatchRequestWithRetry(spec, retries)}
	}
}

var unresolvedTemplate = regexp.MustCompile(`\{\{[^{}]+\}\}`)

func firstUnresolvedTemplate(spec requestinput.BatchSpec) string {
	req := spec.Request
	values := []string{
		req.URL,
		req.BodySpec.Raw,
		req.BodySpec.File,
		req.Auth.Token,
		req.Auth.Username,
		req.Auth.Password,
		req.Auth.Key,
		req.Auth.Value,
	}
	for _, rows := range [][]collectionstore.KVRow{req.ParamsList, req.HeadersList, req.Cookies, req.BodySpec.Fields} {
		for _, row := range rows {
			values = append(values, row.Key, row.Value, row.Description)
		}
	}
	for _, value := range values {
		if token := unresolvedTemplate.FindString(value); token != "" {
			return token
		}
	}
	return ""
}

func applyBatchVariables(spec requestinput.BatchSpec, values map[string]string) requestinput.BatchSpec {
	replacements := make([]string, 0, len(values)*2)
	for key, value := range values {
		replacements = append(replacements, "{{"+key+"}}", value)
	}
	replacer := strings.NewReplacer(replacements...)
	req := spec.Request
	req.URL = replacer.Replace(req.URL)
	req.ParamsList = replaceBatchRows(replacer, req.ParamsList)
	req.HeadersList = replaceBatchRows(replacer, req.HeadersList)
	req.Cookies = replaceBatchRows(replacer, req.Cookies)
	req.BodySpec.Raw = replacer.Replace(req.BodySpec.Raw)
	req.BodySpec.File = replacer.Replace(req.BodySpec.File)
	req.BodySpec.Fields = replaceBatchRows(replacer, req.BodySpec.Fields)
	req.Auth.Token = replacer.Replace(req.Auth.Token)
	req.Auth.Username = replacer.Replace(req.Auth.Username)
	req.Auth.Password = replacer.Replace(req.Auth.Password)
	req.Auth.Key = replacer.Replace(req.Auth.Key)
	req.Auth.Value = replacer.Replace(req.Auth.Value)
	spec.Request = req
	return spec
}

func replaceBatchRows(replacer *strings.Replacer, rows []collectionstore.KVRow) []collectionstore.KVRow {
	out := make([]collectionstore.KVRow, len(rows))
	for i, row := range rows {
		row.Key = replacer.Replace(row.Key)
		row.Value = replacer.Replace(row.Value)
		row.Description = replacer.Replace(row.Description)
		out[i] = row
	}
	return out
}

func (m *AppModel) captureChainedVariables(response requestinput.ResponseMsg) {
	if m.BatchVariables == nil {
		m.BatchVariables = make(map[string]string)
	}
	if m.State != nil && m.State.SessionVariables == nil {
		m.State.SessionVariables = make(map[string]string)
	}
	if response.Err != "" || response.Code < 200 || response.Code >= 400 {
		return
	}
	domain := ""
	if parsed, err := url.Parse(response.URL); err == nil {
		domain = parsed.Hostname()
	}
	for _, variable := range historydb.ExtractVariables(response.RawBody, domain) {
		m.BatchVariables[variable.Name] = variable.Value
		if m.State != nil {
			m.State.SessionVariables[variable.Name] = variable.Value
		}
	}
}

func collectRequests(folder collectionstore.Entry, depth, maxDepth int, out *[]collectionstore.Entry) {
	for _, child := range folder.Children {
		if child.IsFolder {
			if depth < maxDepth {
				collectRequests(child, depth+1, maxDepth, out)
			}
			continue
		}
		if child.Request != nil {
			*out = append(*out, child)
		}
	}
}
