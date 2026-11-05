# TermAPI collections

Each direct subfolder of `collections/` is a collection. `_folder.yaml` stores
its name and type. Request files are regular `.yaml` files in that folder;
subfolders can group requests and use their own `_folder.yaml` metadata.

```yaml
id: list-users
name: List users
method: GET
url: https://api.example.com/users
headers:
  Accept: application/json
body: ""
```

Supported request fields are `id`, `name`, `method`, `url`, `headers`, and
`body`. Selecting a request in the sidebar copies its method and URL into the
request editor. Adding a collection or subfolder from the sidebar creates the
folder and `_folder.yaml` automatically.
