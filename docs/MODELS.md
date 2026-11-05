# TermAPI — Peta Model, Relasi, dan Alur Data

Dokumen internal untuk maintainer. Fokus: **model apa yang ada, siapa memiliki state, bagaimana relasinya, dan file mana yang harus disentuh** saat mengubah fitur.

- Dokumentasi pengguna: `docs/index.html`
- Sumber kebenaran: source code saat ini (bukan `graphify-out/GRAPH_REPORT.md`, yang bisa stale)
- Konvensi rujukan: `file.go:baris`; `pkg.Type` berarti tipe di paket `internal/<pkg>`

Daftar isi:

1. [Ringkasan 60 detik](#1-ringkasan-60-detik)
2. [Peta dependensi](#2-peta-dependensi)
3. [Core state](#3-core-state-modelsharedstate)
4. [Model persisten](#4-model-persisten)
5. [Model UI](#5-model-ui)
6. [Diagram relasi](#6-diagram-relasi)
7. [Ownership state](#7-matriks-ownership-state)
8. [Matriks perubahan](#8-matriks-mau-ubah-x-sentuh-file-y)
9. [Invariant & jebakan](#9-invariant--jebakan)
10. [Indeks simbol](#10-indeks-simbol-kunci)
11. [Verifikasi](#11-verifikasi)

---

## 1. Ringkasan 60 detik

TermAPI memakai Elm Architecture (`Model–Update–View`) lewat Bubble Tea.

```
cmd/termapi/main.go
  └── ui.NewAppModel(model.NewSharedState())
        └── ui.AppModel            ← root aggregate / router
              ├── sidebar.Model             → collectionstore.Store
              ├── requestinput.Model        → SharedState, collectionstore
              ├── requestinput.ResponsePanel
              ├── historyscreen.Model       → historydb.Run
              ├── envscreen.Model           → envstore.Store / EnvFile
              ├── scopescreen.Model         → config.GlobalConfig
              ├── diffscreen.Model          → DiffData
              ├── wirescreen.Model          → ws.Client / grpcx.Client
              ├── newscreen.Model           → collectionstore.Store
              └── helpscreen.Model
```

Hal penting:

1. **`AppModel` adalah satu-satunya router.** Semua event masuk ke `AppModel.Update`, yang memilih overlay aktif sebelum meneruskan ke sub-model.
2. **`model.SharedState` adalah pointer yang dibagi** ke `AppModel`, `sidebar`, `requestinput`, dan `wirescreen`. Perubahan di satu tempat terlihat di tempat lain.
3. **UI tidak menyimpan data sendiri.** Sumber persisten tetap file/SQLite; model UI hanyalah representasi sementara.
4. **Domain package tidak boleh mengimpor `ui`.** Arah dependensi: `cmd → ui → domain`.
5. **Tidak ada import cycle** saat ini.

Alur request singkat:

```
editor → requestSpec → performRequest → ResponseMsg
       → ResponsePanel + historydb.Save + SharedState.SessionVariables
```

**Versi lisan saat diminta menjelaskan:**

> “TermAPI itu TUI Bubble Tea dengan satu root `AppModel` yang merutekan event ke sub-model per layar. State lintas layar ada di `SharedState` berbentuk pointer. Data persisten dipisah per domain: collections YAML, environment YAML, config global YAML, dan history SQLite. Saat request dikirim, editor diubah jadi `requestSpec`, variabel session/env di-resolve, scope dicek, lalu `performRequest` mengembalikan `ResponseMsg`. Response ditampilkan, disimpan ke history, dan scalar JSON-nya bisa jadi session variable. Untuk mengubah fitur, ikuti arah dependensi `ui → domain` dan perbarui routing di `Update` serta `View`.”

---

## 2. Peta dependensi

```
cmd/termapi ──> ui ──> requestinput ──> collectionstore
                 │          │            model
                 │          └──────────> styles
                 ├──> sidebar ─────────> collectionstore, model
                 ├──> historyscreen ───> historydb, requestinput
                 ├──> envscreen ───────> envstore
                 ├──> scopescreen ─────> config
                 ├──> diffscreen ──────> styles, go-udiff
                 ├──> wirescreen ──────> proto/ws, proto/grpcx, model
                 └──> newscreen ───────> collectionstore
```

| Paket | Tanggung jawab | Tipe kunci | Dependensi |
|---|---|---|---|
| `internal/model` | state global yang dibagi | `SharedState` | — |
| `internal/config` | `.termapi/global.yaml` | `GlobalConfig` | yaml |
| `internal/collectionstore` | baca/tulis collections YAML | `Store`, `Entry`, `Request`, `KVRow`, `BodySpec`, `AuthSpec`, `FolderMeta` | yaml |
| `internal/envstore` | `.termapi/envs/*.yaml` | `Store`, `EnvFile` | yaml |
| `internal/historydb` | riwayat request di SQLite | `Store`, `Run`, `CapturedVariable` | sqlite3 |
| `internal/proto/ws` | WebSocket client + event | `Client`, `WSEvent` | coder/websocket |
| `internal/proto/grpcx` | gRPC client + event | `Client`, `GRPCEvent` | grpc |
| `internal/ui` | root model, routing, layout | `AppModel` | semua UI + domain |
| `internal/ui/requestInput` | bar request, editor, response | `Model`, `KVEditor`, `BodyEditor`, `AuthEditor`, `ResponsePanel`, `ResponseMsg` | collectionstore, model |
| `internal/ui/sidebar` | tree collections, search, create/delete | `Model`, `TreeNode`, `SearchSuggestion` | collectionstore, model |
| `internal/ui/*screen` | overlay satu fitur | `Model` masing-masing | domain terkait |
| `internal/ui/styles` | warna, border, ikon, spacing | `theme.go`, `styles.go` | lipgloss |

Aturan praktis:

- `styles` adalah leaf: jangan menaruh logika domain di sana.
- Package layar boleh mengimpor `requestInput` (contoh: `historyscreen` memakai `ResponsePanel`), tetapi `requestInput` tidak boleh mengimpor layar.
- Jangan membuat `collectionstore`/`envstore`/`historydb` mengimpor `ui`; itu akan menciptakan cycle.

---

## 3. Core state: `model.SharedState`

`internal/model/model.go:4`

| Field | Tipe | Arti | Penulis utama |
|---|---|---|---|
| `TabActive` | `int` | fokus global: `0` = request, `1` = sidebar | `update_keys.go` |
| `Widht` | `int` | lebar window (nama memang typo) | `update_resize.go`, handler resize |
| `Height` | `int` | tinggi window | idem |
| `SessionVariables` | `map[string]string` | variabel hasil capture response / batch chaining | `batch.go`, `AppModel.captureChainedVariables` |
| `GlobalScope` | `string` | origin allowlist (satu per baris) | `NewAppModel`, `update_scope.go` |
| `ActiveEnv` | `string` | nama environment aktif | `NewAppModel`, `update_env.go` |
| `EnvVars` | `map[string]string` | variabel environment aktif | `NewAppModel`, `update_env.go` |

Siklus hidup:

1. `main` membuat `NewSharedState()`.
2. `NewAppModel` memuat `global.yaml` → `GlobalScope`.
3. `NewAppModel` memilih environment pertama (alfabetis) → `ActiveEnv` + `EnvVars`.
4. `envscreen` mengubah env aktif → `update_env.go` menyalin ke `SharedState`.
5. `scopescreen` menyimpan origin → `SharedState.GlobalScope`.
6. Request berikutnya memakai `variableReplacer()` untuk mengganti `{{var}}`.

> Jangan “memperbaiki” `Widht` sepihak. Ganti nama field berarti mengganti semua referensi di `ui`, layar, dan test.

---

## 4. Model persisten

### 4.1 `collectionstore`

`internal/collectionstore/types.go`

| Tipe | Baris | Fungsi |
|---|---:|---|
| `KVRow` | 14 | satu baris key/value: params, headers, cookies, form fields |
| `BodySpec` | 27 | deskripsi body: `type`, `raw`, `fields`, `file`, `content_type` |
| `AuthSpec` | 36 | auth: `type` (`none/bearer/basic/api_key`), token, user/pass, key/value, `add_to` |
| `Request` | 46 | representasi persisted satu request |
| `FolderMeta` | 120 | isi `_folder.yaml` |
| `Entry` | 126 | node in-memory hasil `Load()` (folder atau request) |
| `Store` | 135 | root direktori collections |

`Request` punya dua generasi field:

| Generasi | Field | Status |
|---|---|---|
| Legacy | `Params string`, `Headers map[string]string`, `Body string` | dibaca saat load; **dikosongkan saat save** |
| Terstruktur | `ParamsList []KVRow`, `HeadersList []KVRow`, `BodySpec`, `Auth`, `Cookies` | format utama |

`Request.Normalize()` (`types.go:64`) idempoten:

| Kondisi | Konversi |
|---|---|
| `ParamsList` kosong + `Params` terisi | pecah per newline/`&` → `KVRow` |
| `HeadersList` kosong + `Headers` terisi | map → `KVRow`, key diurutkan |
| `BodySpec.Type` kosong | body kosong → `none`; JSON valid → `json`; lainnya → `text` |
| `Auth.Type` kosong | `none` |

Relasi entry:

```
Entry (folder)
 ├── ID, Name, RelPath, IsFolder=true
 └── Children []Entry
      ├── Entry (folder) ...
      └── Entry (request)
           ├── ID, Name, RelPath, IsFolder=false
           └── Request *Request
```

Layout disk:

```
collections/
  auth/
    _folder.yaml          # FolderMeta{id,name,type}
    login.yaml            # Request
    folder-06/
      _folder.yaml
      nss.yaml
```

Contoh YAML request:

```yaml
id: login
name: Login
method: POST
url: https://api.example.com/auth/login
params_list:
  - key: q
    value: hello
headers_list:
  - key: Authorization
    value: Bearer {{token}}
body_spec:
  type: json
  raw: '{"username":"demo"}'
auth:
  type: bearer
  token: "{{token}}"
cookies:
  - key: session
    value: abc
```

Operasi:

| Operasi | File | Catatan |
|---|---|---|
| `Load()` | `load.go:13` | baca root → rekursif; abaikan dotfile, non-YAML, `_folder.yaml`; panggil `Normalize()` |
| `SaveRequest()` | `save.go:13` | update/create; kosongkan field legacy; tulis atomik via temp + rename |
| `CreateFolder()` | `create.go:13` | buat direktori + `_folder.yaml`; suffix `-2`, `-3` bila bentrok |
| `CreateRequestWithMethod()` | `create.go:73` | buat YAML dengan `O_EXCL`; suffix bila bentrok |
| `Delete()` | `delete.go:13` | pindahkan ke `.termapi/trash/`, bukan hapus permanen |
| `resolve()` | `delete.go:33` | cegah path keluar dari root collections |

Root collections ditentukan `sidebar.defaultCollectionStore()` (`sidebar.go:89`):

1. env `TERMAPI_COLLECTIONS_DIR` bila di-set;
2. direktori `collections/` di samping `go.mod` terdekat;
3. fallback `collections/` relatif cwd.

### 4.2 `envstore`

`internal/envstore/store.go`

| Tipe | Baris | Isi |
|---|---:|---|
| `EnvFile` | 14 | `Name string`, `Variables map[string]string` |
| `Store` | 20 | direktori `.termapi/envs/` |

- Path: `.termapi/envs/<nama>.yaml`
- API: `List()` (terurut), `Load()`, `Save()`, `Delete()`, `Create()`
- Satu environment aktif disalin ke `SharedState.ActiveEnv` + `EnvVars`; `envscreen` tetap memegang `Store` dan `Current *EnvFile` untuk panel kanan.
- Variabel environment dipakai oleh `requestinput.variableReplacer()`.

### 4.3 `historydb`

`internal/historydb/types.go`

| Tipe | Baris | Isi |
|---|---:|---|
| `Store` | 8 | wrapper `*sql.DB` |
| `Run` | 10 | satu eksekusi request + response |
| `CapturedVariable` | 24 | variabel hasil flatten JSON |

Skema SQLite (`.termapi/history.db`, `open.go:12`):

```
collections 1 ──── N history_runs 1 ──── N captured_variables
```

| Tabel | Kolom penting | Relasi |
|---|---|---|
| `collections` | `id`, `name`, `folder_path UNIQUE`, `base_url` | — |
| `history_runs` | `collection_id`, `request_name`, `target_domain`, `url`, `method`, `status_code`, `response_time`, `response_body`, `executed_at` | `collection_id → collections.id ON DELETE SET NULL` |
| `captured_variables` | `run_id`, `domain_scope`, `var_name`, `var_value` | `run_id → history_runs.id ON DELETE CASCADE` |

Catatan:

- `Store.Save()` (`save.go:9`) memakai transaksi; `collections` di-upsert berdasarkan `folder_path`.
- `response_time` disimpan dalam **milidetik** (`Duration.Milliseconds()`).
- `ExtractVariables()` (`variables.go:11`) hanya mengambil **scalar JSON** (string/number/bool). Object, array, dan null tidak menghasilkan variabel. Batas: 1000 variabel, depth 32.
- `List(limit)` (`list.go:8`) default limit 500, maksimum 1000; urut `executed_at DESC, id DESC`.
- `ListForRequest(method,url,limit)` (`list.go:21`) dipakai Diff Viewer supaya baseline relevan.
- `Delete(runID)` ada di `delete.go:6`.
- SQLite membutuhkan CGO (`mattgo/go-sqlite3`).

### 4.4 `config`

`internal/config/config.go`

| Tipe/Fungsi | Baris | Isi |
|---|---:|---|
| `GlobalConfig` | 10 | `Scope string`, `ResponseRatio float64` |
| `LoadGlobalScope()` | 22 | baca `scope` dari `.termapi/global.yaml` |
| `SaveGlobalScope()` | 39 | tulis `scope`, pertahankan field lain |
| `LoadResponseRatio()` | 47 | default `0.72`; valid hanya `0.1 < r < 0.9` |
| `SaveResponseRatio()` | 56 | simpan rasio, pertahankan scope |

`Scope` adalah daftar origin dipisah newline. Request yang membawa credential (auth, cookie, header sensitif) hanya diizinkan jika origin persis ada di allowlist; redirect lintas origin juga diblokir (`perform.go`, `url_security.go`).

### 4.5 Lokasi file

| Path | Isi | Ditulis oleh |
|---|---|---|
| `collections/<collection>/...` | request + folder YAML | `collectionstore` |
| `.termapi/envs/<env>.yaml` | environment | `envstore` |
| `.termapi/global.yaml` | scope + response ratio | `config` |
| `.termapi/history.db` | history SQLite | `historydb` |
| `.termapi/trash/` | item terhapus | `collectionstore.Delete` |
| `~/.termapi/sessions/*.jsonl` | export wire inspector | `wirescreen.exportSession` |

---

## 5. Model UI

### 5.1 `ui.AppModel`

`internal/ui/model.go:23`

| Kelompok | Field | Model/tipe |
|---|---|---|
| Shared | `State` | `*model.SharedState` |
| Request | `ReqState`, `Response` | `*requestinput.Model`, `*requestinput.ResponsePanel` |
| Sidebar | `Sidebar` | `*sidebar.Model` |
| Store | `HistoryStore`, `EnvStore` | `*historydb.Store`, `*envstore.Store` |
| Overlay | `HistoryScreen`, `ScopeScreen`, `EnvScreen`, `DiffScreen`, `WireScreen`, `HelpScreen`, `NewItemScreen` | pointer model layar |
| Mode flag | `HistoryMode`, `ScopeMode`, `EnvMode`, `DiffMode`, `WireMode`, `HelpMode`, `NewItemMode`, `BatchMode`, `BatchChoosing`, `SearchMode`, `SaveDialog` | `bool` |
| Batch | `BatchResults`, `BatchNames`, `BatchSpecs`, `BatchVariables`, `BatchNext`, `BatchChain`, `BatchRetries`, `BatchSelected`, `BatchError` | state runner |
| Save | `SaveName`, `SaveFolder`, `SaveFocus`, `SaveError` | dialog simpan |
| Search | `SearchInput`, `SearchHistory`, `SearchSelectedIndex` | pencarian file |
| Layout | `ScreenWidth/Height`, `ContentWidth`, `MainHeight`, `RequestHeight`, `ResponseHeight`, `ResponseRatio` | hasil resize |

Sifat penting:

- `Update` dan `View` memakai **value receiver** (`func (m AppModel)`), tetapi field-model adalah pointer. Mutasi `m.Sidebar.X` terlihat; penggantian `m.Sidebar = ...` hanya berlaku pada salinan yang dikembalikan.
- Semua sub-model dibuat sekali di `NewAppModel` (`model.go:78`) dan dipakai ulang.

Urutan routing `Update` (`app.go:19`):

```
NewItem → Save → BatchChoosing → Sidebar create prompt → Batch
→ History → Scope → Env → Diff → Search → Wire → updateEvents
```

Urutan `View` (`view.go:19`):

```
window too small → NewItem → Save → Search → BatchChoosing → Batch
→ History → Scope → Env → Help → Diff → Wire → main
```

> Urutan tidak identik. Saat menambah mode baru, daftarkan di **keduanya** dan pastikan input overlay tidak bocor ke editor. `Help` ditampilkan lewat `View` dan ditangani di `handleGlobalKeys` (`update_keys.go:10`, F1/Esc), bukan cabang routing sendiri.

Shortcut global utama (`update_keys.go:141`):

| Key | Aksi |
|---|---|
| `Tab` | pindah fokus sidebar ↔ request |
| `Ctrl+F` | pencarian file |
| `Ctrl+S` | simpan request |
| `Ctrl+D` | hapus file (saat sidebar fokus) / buka diff |
| `Ctrl+O` | history |
| `Ctrl+G` | global scope |
| `Ctrl+E` | environment |
| `Ctrl+W` | wire inspector |
| `Ctrl+Q` | keluar |

### 5.2 `requestinput.Model`

`internal/ui/requestInput/model.go:14`

| Field | Tipe | Peran |
|---|---|---|
| `State` | `*model.SharedState` | sumber variabel & layout |
| `Input` | `textinput.Model` | URL |
| `Method`, `Methods` | `string`, `[]string` | method aktif + pilihan |
| `FocusArea` | `int` | 0 method, 1 URL, 2 send, 3 editor |
| `Tab` | `int` | tab editor: params/headers/body/auth/cookies |
| `Params`, `Headers`, `Cookies` | `KVEditor` | tabel key/value |
| `Body` | `BodyEditor` | tipe body + editor |
| `Auth` | `AuthEditor` | skema auth |
| `CollectionName`, `CollectionPath`, `RequestFilePath`, `RequestName` | `string` | metadata request yang sedang dibuka |
| `cancel` | `context.CancelFunc` | pembatalan request in-flight |

Editor:

| Tipe | File | Isi |
|---|---|---|
| `KVEditor` | `kv_editor.go:21` | `Rows []collectionstore.KVRow`, mode navigasi/edit, raw mode |
| `BodyEditor` | `body_editor.go:50` | `Type`, `Raw`, `Fields KVEditor`, `File` |
| `AuthEditor` | `auth_editor.go:49` | `Type`, `values map[string]string` |

Konversi utama:

```
LoadRequest(*collectionstore.Request)   focus.go:255
Snapshot() (collectionstore.Request)    focus.go:271
```

`Snapshot()` adalah satu-satunya jalur UI → format persisted: commit edit aktif, sinkronkan params dari URL, ambil `EffectiveRows()`, lalu isi `Request`.

### 5.3 Alur kirim request

```
sendRequest()                                  sender.go:74
 ├─ commitActiveEdit + syncParamsFromURL
 ├─ Params/Headers/Cookies.EffectiveRows()
 ├─ Body.Spec() + Auth.Spec()
 ├─ variableReplacer()                        sender.go:160
 │    ├─ {{session}} dari SharedState.SessionVariables
 │    └─ {{env}} dari SharedState.EnvVars
 ├─ token session otomatis → header Authorization
 ├─ requestSpec                               sender.go:59
 └─ performRequestWithRetry(spec, 2, true)    sender.go:256
      └─ performRequest(spec)                 perform.go:31
           ├─ buildRequestURL (path :id / {id}, query)
           ├─ scope allowlist check
           ├─ buildRequestBody (json/xml/text/form/multipart/binary/graphql)
           ├─ http.Client timeout 30s + cookie jar + max 10 redirect
           ├─ baca body maksimum 2 MiB
           └─ ResponseMsg
```

`ResponseMsg` (`sender.go:36`) adalah kontrak antara eksekusi dan UI:

| Field | Isi |
|---|---|
| `Method`, `URL`, `RequestName`, `CollectionName`, `CollectionPath` | metadata |
| `Status`, `Code` | status HTTP |
| `Duration`, `Timings`, `TLS`, `Redirects`, `Attempts` | metadata koneksi |
| `Body`, `RawBody`, `Headers`, `Cookies`, `Size`, `ContentType` | payload |
| `Err`, `Retryable` | error + boleh retry |

Retry: manual request hanya untuk method idempotent (`GET/HEAD/OPTIONS/PUT/DELETE`); status `408/429/5xx` dan transport error dapat di-retry. Batch memakai `RunBatchRequestWithRetry` dengan `idempotentOnly=false`.

### 5.4 `requestinput.ResponsePanel`

`internal/ui/requestInput/response.go:18`

- Menampung hasil `ResponseMsg` lewat `SetResponse()` (`response.go:197`).
- `Loading`, `Tab`, `RawMode`, `PreviewMode`, `Find`, `Filter`, selection, dan timing semuanya state tampilan.
- `SetResponse` menyalin `ResponseMsg` ke field panel; `Body`/`RawBody` dipakai untuk pretty/raw.
- `historyscreen` juga membuat `ResponsePanel` dan mengisi `ResponseMsg` dari `historydb.Run` (durasi: `ResponseTime` ms).

### 5.5 `sidebar.Model`

`internal/ui/sidebar/types.go:77`

| Tipe | Baris | Peran |
|---|---:|---|
| `TreeNode` | 30 | node UI: `Name`, `IsFolder`, `Open`, `ZoneID`, `StorePath`, `RequestName`, `Method`, `RequestURL`, `Children` |
| `flatItem` | 20 | hasil render datar sementara (tidak disimpan) |
| `SearchSuggestion` | 51 | hasil pencarian file/folder |
| `Model` | 77 | state sidebar + `CollectionStore` |

Alur data:

```
disk YAML
  → collectionstore.Store.Load() []Entry
  → treeNodesFromEntries() []TreeNode        tree.go:13
  → getFlatItems() []flatItem                tree.go:175
  → View()
```

Konversi `Entry → TreeNode` (`tree.go:21`):

- folder: `ZoneID = "folder_" + RelPath` (`/` → `__`)
- request: `Name = "<METHOD> <route>"`; `ZoneID = "file_" + RelPath`
- `StorePath` adalah **join key** ke disk/Entry; jangan mengubahnya tanpa mengubah `Entry.RelPath`.

Operasi penting:

| Fungsi | File:baris | Fungsi |
|---|---|---|
| `SelectedRequestEntry()` | `actions.go:242` | load ulang store, cari `Entry` berdasarkan `StorePath`, kembalikan request + root collection |
| `ReloadCollections()` | `actions.go:285` | rebuild `Nodes` setelah save/create/delete |
| `SelectPath()` | `actions.go:320` | pilih node berdasarkan path |
| `SelectedFolderPath()` | `actions.go:342` | folder tujuan untuk save/create |
| `GetSearchSuggestions()` | `actions.go:41` | pencarian nama/path/method/URL |
| `DeleteSelectedRequest()` | `actions.go:187` | hapus file terpilih lewat `Store.Delete` |

`AppModel.openSelectedRequest()` (`open.go:4`) menjembatani sidebar → editor:

```
Sidebar.SelectedRequestEntry()
  → Request.Normalize()
  → ReqState.LoadRequest(request)
  → isi CollectionName/CollectionPath/RequestFilePath/RequestName
  → fokus pindah ke editor
```

### 5.6 Layar overlay

| Paket | Model | State kunci | Relasi |
|---|---|---|---|
| `historyscreen` | `Model` (`model.go:15`) | `Runs []historydb.Run`, `Cursor`, `Query`, `Response *requestinput.ResponsePanel` | `historydb.List` → `Run` → preview `ResponseMsg` |
| `envscreen` | `Model` (`model.go:12`) | `Store *envstore.Store`, `Envs`, `Current *envstore.EnvFile`, `ActiveEnv`, cursor/pane | `envstore` ↔ `SharedState.EnvVars` |
| `scopescreen` | `Model` (`scope.go:11`) | `Origins []string`, `Input`, `Adding` | `config.GlobalConfig.Scope` (newline list) |
| `diffscreen` | `Model` (`diff.go:24`) | `OriginalText`, `ModifiedText`, `Mode`, `DiffResult *DiffData`, `Options []BaselineOption` | `historydb.Run` → baseline A/B |
| `wirescreen` | `Model` (`model.go:40`) | `Protocol`, `Client *ws.Client`, `GrpcClient *grpcx.Client`, `Ring *RingBuffer`, `Composer` | `proto/ws`, `proto/grpcx` |
| `newscreen` | `Model` (`model.go:49`) | `Kind`, `Method`, `Focus`, `Name`, `Store *collectionstore.Store`, `folders` | `collectionstore.CreateFolder/CreateRequestWithMethod` |
| `helpscreen` | `Model` | konten bantuan + scroll | hanya UI |
| `nav` | fungsi navbar | `NavbarContent()`, `NavbarWidth` | `AppModel.View` |
| `styles` | `theme.go`, `styles.go` | warna/ikon/box | semua view |

Detail diff:

```
Model.OriginalText (A/baseline)  ─┐
Model.ModifiedText (B/terbaru)   ─┴→ DiffData
                                      ├── StatsAdd / StatsDel / StatsChange
                                      └── Hunks []Hunk
                                            └── Lines []DiffLine
                                                  └── Intra []IntraSegment
```

`AppModel.openDiffViewer()` (`update_diff.go:45`):

1. Ambil method + `ResolvedURL()` dari request aktif.
2. `HistoryStore.ListForRequest(method,url,50)`; fallback `List(50)`.
3. `runs[1]` = baseline A, `runs[0]` = B; fallback response aktif.
4. `diffscreen.NewModel(orig, mod)` + `BaselineOption` dari history.

### 5.7 Wire clients

| Tipe | File:baris | Isi |
|---|---|---|
| `ws.Client` | `proto/ws/client.go:14` | koneksi, `Events() <-chan WSEvent`, `Send`, `RTT`, `Disconnect` |
| `ws.WSEvent` | `client.go:27` | `Direction` (`1 TX, 2 RX, 3 Control, 4 Error`), subtype, payload, size, RTT |
| `grpcx.Client` | `proto/grpcx/client.go:36` | koneksi, `Events() <-chan GRPCEvent`, `UnaryCall` |
| `grpcx.GRPCEvent` | `client.go:25` | direction, subtype, payload, status, error |
| `wirescreen.Event` | `wirescreen/events.go:26` | event internal: `ID`, `Direction`, `Type`, `CorrelationID`, `Dropped` |
| `wirescreen.RingBuffer` | `events.go:41` | buffer 10.000 event, drop terlama saat penuh |

Alur:

```
ws.Client / grpcx.Client (goroutine)
  → channel Events()
  → waitForWsEvent / waitForGrpcEvent
  → WsEventMsg / GrpcEventMsg
  → wirescreen.Update
  → RingBuffer.Push(Event)
  → timeline / inspector / sparkline
```

`wirescreen.Model.exportSession()` menulis `~/.termapi/sessions/session-<timestamp>.jsonl`.

---

## 6. Diagram relasi

### 6.1 Graf in-memory

```
                         ┌──────────────────────┐
                         │ model.SharedState    │
                         │ TabActive, Widht,    │
                         │ Height, SessionVars, │
                         │ GlobalScope, EnvVars │
                         └───────┬──────────────┘
                                 │ pointer
        ┌────────────────────────┼─────────────────────────┐
        │                        │                         │
┌───────▼────────┐      ┌────────▼────────┐       ┌────────▼────────┐
│ ui.AppModel    │─────>│ requestinput    │       │ sidebar.Model   │
│ (router)       │      │ .Model          │       │                 │
└───┬───┬───┬────┘      │ ├ Params KV     │       │ Nodes []TreeNode│
    │   │   │           │ ├ Headers KV    │       │ CollectionStore │
    │   │   │           │ ├ BodyEditor    │       └────────┬────────┘
    │   │   │           │ ├ AuthEditor    │                │
    │   │   │           │ └ Cookies KV    │                │
    │   │   │           └────────┬────────┘                │
    │   │   │                    │                         │
    │   │   └──> ResponsePanel <──┘                         │
    │   │                                                   │
    │   ├──> historyscreen ──> historydb.Store ──> SQLite   │
    │   ├──> envscreen ─────> envstore.Store ──> YAML       │
    │   ├──> scopescreen ───> config ──────────> YAML       │
    │   ├──> diffscreen ────> historydb.Run                │
    │   ├──> wirescreen ────> ws / grpcx clients           │
    │   └──> newscreen ─────> collectionstore.Store <──────┘
    │                              │
    └──────────────────────────────┴──> collections/*.yaml
```

### 6.2 Relasi entitas tersimpan

```
collections/<root>/
  _folder.yaml ──> FolderMeta
  <request>.yaml ──> Request
                       ├── params_list []KVRow
                       ├── headers_list []KVRow
                       ├── cookies []KVRow
                       ├── body_spec BodySpec
                       │     └── fields []KVRow
                       └── auth AuthSpec

.termapi/envs/<env>.yaml ──> EnvFile
                              └── variables map[string]string

.termapi/global.yaml ──> GlobalConfig {scope, response_ratio}

.termapi/history.db
  collections 1 ──< history_runs 1 ──< captured_variables
  (folder_path)     (collection_id)     (run_id)
```

### 6.3 Siklus request

```
[editor] Params/Headers/Body/Auth/Cookies
      │
      ▼
requestinput.Snapshot() ──> collectionstore.Request ──> SaveRequest() ──> YAML
      │
      ▼
sendRequest() ──> requestSpec ──> performRequest()
                                      │
                    ┌─────────────────┼──────────────────┐
                    ▼                 ▼                  ▼
              buildRequestURL   scope check        buildRequestBody
                    │                 │                  │
                    └─────────────────┼──────────────────┘
                                      ▼
                              http.Client (30s, jar)
                                      │
                                      ▼
                                 ResponseMsg
                                      │
              ┌───────────────────────┼────────────────────────┐
              ▼                       ▼                        ▼
      ResponsePanel.SetResponse   HistoryStore.Save()   captureChainedVariables()
              │                       │                        │
              ▼                       ▼                        ▼
           tampilan              Run + CapturedVariable   SharedState.SessionVariables
```

### 6.4 Load & buka request

```
collections/*.yaml
      │
      ▼
collectionstore.Store.Load() ──> []Entry (tree)
      │
      ▼
sidebar.treeNodesFromEntries() ──> []TreeNode
      │
      ▼
sidebar.getFlatItems() ──> []flatItem ──> View()
      │
      ▼ (user memilih)
SelectedRequestEntry() ──> *collectionstore.Request
      │
      ▼
ReqState.LoadRequest() ──> editor terisi
```

### 6.5 Environment & scope

```
.termapi/envs/*.yaml
      │ envstore.List/Load
      ▼
envscreen.Model.Current *EnvFile
      │ update_env.go
      ▼
SharedState.ActiveEnv + EnvVars
      │
      ▼
requestinput.variableReplacer() ──> {{var}} diganti

.termapi/global.yaml
      │ config.LoadGlobalScope
      ▼
SharedState.GlobalScope ──> performRequest() scope check
      ▲
      │ update_scope.go
scopescreen.Model.Origins
```

### 6.6 Wire inspector

```
ws:// / grpc://
      │
      ▼
ws.Client / grpcx.Client  (goroutine read/ping)
      │ Events() channel
      ▼
WsEventMsg / GrpcEventMsg
      │
      ▼
wirescreen.Model.Update
      │
      ├── RingBuffer.Push(Event)
      ├── timeline / inspector / sparkline
      └── exportSession() ──> ~/.termapi/sessions/*.jsonl
```

---

## 7. Matriks ownership state

| State | Pemilik | Penulis | Pembaca | Persisten? |
|---|---|---|---|---|
| `SharedState.SessionVariables` | `AppModel` | `captureChainedVariables`, batch | `requestinput.variableReplacer` | tidak |
| `SharedState.EnvVars` | `AppModel` | `NewAppModel`, `update_env` | `variableReplacer` | via env YAML |
| `SharedState.GlobalScope` | `AppModel` | `NewAppModel`, `update_scope` | `performRequest` | global.yaml |
| `ReqState` (editor) | `AppModel` | `requestinput.Update`, `LoadRequest` | `Snapshot`, send | via save |
| `ResponsePanel` | `AppModel` | `SetResponse`, `SetLoading` | response view | via history |
| `Sidebar.Nodes` | `AppModel` | `ReloadCollections` | tree/flat view, search | collections YAML |
| `HistoryScreen.Runs` | `AppModel` | `reloadHistory` | history view | SQLite |
| `EnvScreen.Current` | `AppModel` | `Load`/`refreshRight` | env view | env YAML |
| `DiffScreen.DiffResult` | `AppModel` | `ComputeDiff` | diff view | tidak |
| `WireScreen.Ring` | `AppModel` | event handler | wire view | export JSONL |
| `Batch*` | `AppModel` | batch runner | batch view | tidak |

Aturan: **jangan menyimpan salinan state persisten di dua tempat.** Misal, `Sidebar.Nodes` adalah cache tampilan dari `collectionstore.Load()`; setelah save/delete, panggil `ReloadCollections()`.

---

## 8. Matriks “mau ubah X, sentuh file Y”

| Kebutuhan | File utama | File pendukung | Catatan |
|---|---|---|---|
| Tambah field di request | `collectionstore/types.go` (`Request`) | `load.go`, `save.go`, `requestinput/focus.go` (`Snapshot`/`LoadRequest`), `sender.go`, editor + view, test | update YAML + normalisasi bila field legacy |
| Tambah tipe body | `requestinput/body_editor.go` (`bodyTypes`, `Spec`, `SetSpec`, label) | `requestinput/perform.go` (`buildRequestBody`), `sender.go` | pastikan content-type default |
| Tambah tipe auth | `requestinput/auth_editor.go` (`authTypes`, `authFieldsFor`, `Spec`, `SetSpec`) | `requestinput/perform.go` (`applyHeaderAuth`, `applyQueryAuth`), `url_security.go` | credential memicu scope check |
| Tambah editor/tab request | `requestinput/model.go` (`Tab`), `view.go`, `update.go`, `focus.go` | `collectionstore/types.go` | simpan ke `Snapshot` |
| Tambah layar/overlay | package baru `internal/ui/<nama>/` | `ui/model.go`, `ui/app.go`, `ui/view.go`, `ui/update_keys.go`, `ui/open.go` | daftarkan mode flag + routing Update & View |
| Ubah routing overlay | `ui/app.go` | `ui/view.go` | urutan harus konsisten; cegah input bocor |
| Ubah layout/resize | `ui/update_resize.go`, `ui/view.go`, `ui/styles` | `requestinput/model.go` (`SetWidth`), `sidebar/sidebar.go` (`SetSize`) | `SharedState.Widht/Height` sumber ukuran |
| Ubah format collection | `collectionstore/*.go` | `sidebar/tree.go`, `ui/save.go`, `ui/open.go`, `newscreen/model.go` | jaga kompatibilitas YAML lama |
| Ubah history/schema | `historydb/open.go`, `types.go`, `save.go`, `list.go` | `ui/update_events.go`, `historyscreen/model.go`, `ui/update_diff.go` | migrasi tabel + CGO |
| Ubah eksekusi HTTP | `requestinput/sender.go`, `perform.go` | `url_security.go`, `config`, test | timeout, retry, cap 2 MiB |
| Ubah environment | `envstore/store.go`, `envscreen/*` | `ui/model.go`, `ui/update_env.go`, `ui/open.go`, `requestinput/sender.go` | sinkron ke `SharedState` |
| Ubah scope/keamanan | `config/config.go`, `ui/scopescreen/*` | `requestinput/url_security.go`, `perform.go` | origin harus exact match |
| Ubah wire inspector | `proto/ws`, `proto/grpcx` | `wirescreen/*`, `ui/update_wire.go`, `ui/open.go` | map event → `wirescreen.Event` |
| Ubah tema/warna | `ui/styles/theme.go`, `styles.go` | semua view | jangan hardcode hex baru |
| Ubah shortcut | `ui/update_keys.go` | `sidebar/types.go` (shortcut lokal), help | shortcut kontekstual di paket masing-masing |

---

## 9. Invariant & jebakan

1. **Legacy request fields hanya untuk load.** `SaveRequest` mengosongkan `Params`, `Headers`, `Body` (`save.go:26`). Jangan menulis logika baru ke field itu.
2. **`Normalize()` harus idempoten.** Dipanggil di load, open, save, dan `BatchSpec.internalSpec`. Menambah konversi harus aman bila dijalankan berulang.
3. **`Widht` typo.** Nama field di `SharedState` memang `Widht`. Ganti nama = grep seluruh repo + test.
4. **Pointer sharing.** `AppModel.Update` value receiver; field pointer tetap dishare. Mengganti pointer di salinan tidak otomatis mengubah state global kecuali salinan dikembalikan.
5. **Routing Update vs View tidak identik.** Search di View lebih awal daripada di Update; Help hanya di View. Mode baru wajib didaftarkan di kedua tempat.
6. **Input overlay harus di-swallow.** Saat mode aktif, jangan teruskan key ke `ReqState`/`Sidebar` kecuali memang diinginkan.
7. **StorePath = kunci join.** `Entry.RelPath` ↔ `TreeNode.StorePath` ↔ `flatItem`/`SearchSuggestion.Path`. Ubah salah satu, ubah semuanya.
8. **ZoneID diturunkan dari path.** Format: `file_`/`folder_` + path dengan `/` → `__`. Dipakai bubblezone untuk klik.
9. **Save atomik.** `SaveRequest` menulis temp file lalu `os.Rename`; jangan ganti dengan tulis langsung.
10. **Delete tidak permanen.** Item dipindah ke `.termapi/trash/`.
11. **Scope itu allowlist credential.** Hanya origin persis yang diizinkan; redirect lintas origin diblokir. Menambah origin di UI berarti menulis `global.yaml`.
12. **Batas response.** Timeout 30 detik, redirect maksimum 10, body maksimum 2 MiB (dipotong dengan penanda).
13. **History hanya scalar JSON.** `ExtractVariables` tidak menangkap object/array/null; batas 1000 variabel dan depth 32.
14. **Waktu history dalam ms.** `Run.ResponseTime` = `Duration.Milliseconds()`.
15. **SQLite butuh CGO.** Build/test perlu `CGO_ENABLED=1` dan compiler C.
16. **Root collections bisa dari env.** `TERMAPI_COLLECTIONS_DIR` menang atas deteksi `go.mod`.
17. **Graph report bisa stale.** `graphify-out/GRAPH_REPORT.md` bukan sumber kebenaran; verifikasi ke source.
18. **Jangan campur dokumentasi pengguna dan internal.** `docs/index.html` untuk pengguna; file ini untuk maintainer.

---

## 10. Indeks simbol kunci

| Simbol | Lokasi |
|---|---|
| `SharedState` | `internal/model/model.go:4` |
| `KVRow`, `BodySpec`, `AuthSpec`, `Request` | `internal/collectionstore/types.go:14,27,36,46` |
| `Request.Normalize` | `internal/collectionstore/types.go:64` |
| `EnabledRows` | `internal/collectionstore/types.go:106` |
| `Entry`, `Store` | `internal/collectionstore/types.go:126,135` |
| `Store.Load` | `internal/collectionstore/load.go:13` |
| `Store.SaveRequest` | `internal/collectionstore/save.go:13` |
| `Store.CreateFolder` / `CreateRequestWithMethod` | `internal/collectionstore/create.go:13,73` |
| `Store.Delete` / `resolve` | `internal/collectionstore/delete.go:13,33` |
| `EnvFile`, `Store` | `internal/envstore/store.go:14,20` |
| `historydb.Store`, `Run`, `CapturedVariable` | `internal/historydb/types.go:8,10,24` |
| `OpenProject` | `internal/historydb/open.go:12` |
| `Store.Save`, `ExtractVariables` | `internal/historydb/save.go:9`, `variables.go:11` |
| `Store.List`, `ListForRequest` | `internal/historydb/list.go:8,21` |
| `GlobalConfig` | `internal/config/config.go:10` |
| `LoadGlobalScope`, `SaveGlobalScope` | `internal/config/config.go:22,39` |
| `LoadResponseRatio`, `SaveResponseRatio` | `internal/config/config.go:47,56` |
| `AppModel`, `NewAppModel` | `internal/ui/model.go:23,78` |
| `AppModel.Update`, `View` | `internal/ui/app.go:19`, `internal/ui/view.go:19` |
| `AppModel.openSelectedRequest` | `internal/ui/open.go:4` |
| `AppModel.saveRequest` | `internal/ui/save.go:82` |
| `AppModel.openDiffViewer` | `internal/ui/update_diff.go:45` |
| `AppModel.captureChainedVariables` | `internal/ui/batch.go:228` |
| `requestinput.Model`, `New` | `internal/ui/requestInput/model.go:14,37` |
| `LoadRequest`, `Snapshot` | `internal/ui/requestInput/focus.go:255,271` |
| `sendRequest`, `variableReplacer` | `internal/ui/requestInput/sender.go:74,160` |
| `performRequest` | `internal/ui/requestInput/perform.go:31` |
| `performRequestWithRetry` | `internal/ui/requestInput/sender.go:256` |
| `ResponseMsg`, `BatchSpec` | `internal/ui/requestInput/sender.go:36,222` |
| `ResponsePanel` | `internal/ui/requestInput/response.go:18` |
| `ResponsePanel.SetResponse` | `internal/ui/requestInput/response.go:197` |
| `sidebar.Model`, `TreeNode` | `internal/ui/sidebar/types.go:77,30` |
| `treeNodeFromEntry` | `internal/ui/sidebar/tree.go:21` |
| `SelectedRequestEntry` | `internal/ui/sidebar/actions.go:242` |
| `ReloadCollections` | `internal/ui/sidebar/actions.go:285` |
| `defaultCollectionStore` | `internal/ui/sidebar/sidebar.go:89` |
| `historyscreen.Model` | `internal/ui/historyscreen/model.go:15` |
| `envscreen.Model` | `internal/ui/envscreen/model.go:12` |
| `scopescreen.Model` | `internal/ui/scopescreen/scope.go:11` |
| `diffscreen.Model`, `DiffData` | `internal/ui/diffscreen/diff.go:24,53` |
| `wirescreen.Model`, `RingBuffer` | `internal/ui/wirescreen/model.go:40`, `events.go:41` |
| `newscreen.Model` | `internal/ui/newscreen/model.go:49` |
| `ws.Client`, `WSEvent` | `internal/proto/ws/client.go:14,27` |
| `grpcx.Client`, `GRPCEvent` | `internal/proto/grpcx/client.go:36,25` |

---

## 11. Verifikasi

Dokumen ini hanya dokumentasi, tidak mengubah kode. Untuk memastikan tetap akurat setelah perubahan:

```bash
# Pastikan simbol dan baris masih ada
grep -n "type AppModel" internal/ui/model.go
grep -n "func (s \*Store) SaveRequest" internal/collectionstore/save.go
grep -n "func performRequest" internal/ui/requestInput/perform.go

# Build & test (SQLite butuh CGO)
CGO_ENABLED=1 go build ./...
CGO_ENABLED=1 go test ./...
```

Jika menambah model/fitur baru, perbarui minimal:

1. tabel tipe di bagian 4/5;
2. diagram relasi di bagian 6;
3. matriks ownership (bagian 7) dan change matrix (bagian 8);
4. indeks simbol (bagian 10).
