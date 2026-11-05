# TermAPI - TUI API Client Architecture & Design

TermAPI is a lightning-fast, modern Terminal User Interface (TUI) API client built in Go using Charmbracelet (`Bubble Tea`, `Bubbles`, `Lip Gloss`, and `Bubble Zone`). It combines the elegance of Postman/Insomnia with the speed and ergonomics of terminal-driven workflows.

---

## 1. Core Architecture & Component Layout

TermAPI follows a modular, Elm-architecture pattern (Model-Update-View) via Bubble Tea.

```
TermAPI Workspace
├── Sidebar (Collections, Requests, History)
├── Main Request Panel
│   ├── Method Selector (GET, POST, PUT, DELETE, PATCH, etc.)
│   ├── URL Input / Curl Parser Bar
│   ├── Request Editors (Params, Headers, Body with Tab/Ctrl+Enter support)
│   └── Send Button
└── Response Panel (Status code, timing, size, headers, cookies, pretty/raw viewer)
```

---

## 2. Key UI/UX Principles & Features

### A. Editor Ergonomics & Keybindings
- **Tab / Shift+Tab:** Inside the request editors (Params, Headers, Body), pressing `Tab` inserts indentation (`  `) rather than shifting focus away.
- **Ctrl + Enter:** Acts like VS Code's **Insert Line Below** — moves the cursor to the end of the current line, creates a new line below, and places the cursor there without splitting the text.
- **Global Navigation:** `Tab` toggles focus between Sidebar and Main Content when not inside an editor.

### B. Response Viewer & Error Handling
- **Structured Status Codes:** Color-coded status badges (`200 OK` in green, `4xx/5xx` in red) with round-trip latency and payload size.
- **Clean Error Presentation:** When network errors, timeouts, or invalid endpoints occur, errors are displayed cleanly in the response panel without raw unformatted dumps or header/cookie mixing.

### C. Local RAG & Token-Saving Indexer (SQLite FTS5)
- **Local Workspace Indexing:** Automatically indexes project files, endpoints, collections, and markdown docs locally into `.termapi/rag.db`.
- **Semantic & Full-Text Search:** Retrieves only relevant code and request snippets during AI queries, cutting token consumption by up to 90%.

### D. Folder & Collection Management UX (`Ctrl + A`)
- **Contextual Folder Creation Prompt:** 
  1. When a user selects a folder in the sidebar (e.g., `authentication`) and presses `Ctrl + A` to create a new subfolder, a prompt modal appears.
  2. The modal **must not** blindly force nesting inside the selected folder.
  3. The modal must provide an explicit toggle/option allowing the user to choose:
     - **[x] Current Folder (`authentication`)** (nested inside the currently selected folder)
     - **[ ] Root Collection** (top-level collection root)
  4. Pressing `Enter` creates the folder at the chosen target destination.

---

## 3. Recommended Development Workflow

1. **Initialize Workspace:** Run TermAPI inside your project directory.
2. **Organize Collections:** Store API requests inside `.termapi/` or collection folders.
3. **Inspect Responses:** Use pretty/raw mode and tab between Body, Cookies, and Headers.
4. **AI & RAG Integration:** Leverage local RAG queries for instant codebase and endpoint context.
