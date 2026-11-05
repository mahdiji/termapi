# TermAPI

A terminal API client — Postman and Insomnia, but in the terminal. Built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea), Lip Gloss, and BubbleZone.

Requests are stored as plain YAML files, so a collection can live in your repo, be reviewed in a pull request, and be edited without leaving the keyboard.

## Features

- **Collections and folders** — organise requests in a tree; every request is a readable YAML file.
- **Full request editor** — method, URL, query params, headers, body, auth, and cookies.
- **Response viewer** — status badge, latency, payload size, pretty/raw body, headers, and cookies in separate tabs.
- **History** — every request and response recorded in a local SQLite database.
- **Environments** — named variable sets, switchable per request.
- **WebSocket and gRPC** — dedicated screens for streaming and unary calls.
- **Batch runner** — fire a set of saved requests in sequence.
- **Diff view** — compare responses side by side.
- **Local index** — project files and requests indexed into SQLite FTS5 for fast offline search.
- **Mock server** — a small helper binary for testing requests without a real backend.

## Requirements

- Go 1.27 or newer
- A terminal with truecolor support
- Linux, macOS, or Windows

## Build and run

```bash
git clone https://github.com/mahdiji/termapi.git
cd termapi
go build -o termapi ./cmd/termapi
./termapi
```

Run it from the directory you want to work in. TermAPI keeps its local state in `.termapi/` inside that directory.

## Keybindings

| Key | Action |
| --- | --- |
| `Tab` | Move focus between sidebar and main panel (outside an editor) |
| `Tab` / `Shift+Tab` | Insert indentation inside the body editor |
| `Ctrl+Enter` | Insert a new line below the cursor |
| `Ctrl+A` | Create a folder — choose current folder or collection root |
| `Ctrl+S` | Save the current request |
| `Ctrl+O` | Open history |
| `Ctrl+E` | Open environment manager |
| `Ctrl+G` | Open scope manager |
| `Ctrl+F` | Search files / find inside the response |
| `Ctrl+Y` | Copy the whole response |
| `/` | Jump to the response filter |
| `F1` | Help |
| `Ctrl+Q` | Quit |

## Workspace layout

```
.termapi/
├── global.yaml      # workspace settings
├── envs/            # environment variable sets
├── history.db       # request history (SQLite)
└── trash/           # deleted requests
```

Collections live wherever you put them — commonly `collections/`, with one YAML file per request:

```yaml
id: bass
name: bass
method: GET
url: localhost:8083/
params_list:
  - key: q
    value: search
    description: free text query
headers_list: []
body_spec:
  type: none
auth:
  type: none
  add_to: header
cookies: []
```

## Project structure

```
cmd/
  termapi/          # main application
  mockserver/       # helper mock server
  wirescreen-demo/  # WebSocket screen demo
internal/
  ui/               # screens, layout, key handling
  model/            # shared application state
  engine/           # request execution
  collectionstore/  # YAML collection persistence
  envstore/         # environment variables
  historydb/        # SQLite history
  proto/            # WebSocket and gRPC clients
  config/           # workspace paths and settings
```

## Documentation

- [`cmd/termapi/DESIGN.md`](cmd/termapi/DESIGN.md) — architecture, UI layout, and UX rules
- [`docs/MODELS.md`](docs/MODELS.md) — data models, state ownership, and which files to touch when changing a feature

## Status

Early development. The core request/response flow works; expect breaking changes to the on-disk format and keybindings.

## License

No license file yet. Until one is added, all rights are reserved.
