# JSON To Go Model Generator

Generate typed Go structs from an example JSON object or a JSON Schema file:

```bash
gokub add model
gokub add model user
gokub add model user --from user.json
gokub add model payment-event --from payment.schema.json
```

With no `--from`, GOKUB searches the project for JSON files and opens an arrow-key
selector. With no model name, it suggests one from the selected filename. This makes
`gokub add model` the simplest interactive workflow; keep `--from` for scripts and
CI.

For a project with `internal/domain`, GOKUB writes
`internal/domain/<name>/model_gen.go`. Other project layouts use
`internal/<name>/model_gen.go`.

## Supported Types

| JSON | Go |
|---|---|
| String | `string` |
| RFC3339 string or schema `date-time` | `time.Time` |
| Integer | `int64` |
| Number | `float64` |
| Boolean | `bool` |
| Object | Named nested struct |
| Array | Typed slice |
| Null or unknown | `any` |

JSON Schema optional scalar fields use pointers and `omitempty`. Nullable scalar
types also use pointers. Property names are preserved exactly in `json` tags while
Go field names are exported and normalized.

## Options

```bash
gokub add model user --from user.json --package account
gokub add model user --from user.json --output internal/account/user_gen.go
gokub add model user --from user.json --force
```

GOKUB refuses to replace an existing output file unless `--force` is passed. Output
is formatted with Go's standard formatter and starts with a generated-code notice.

The generator supports inline JSON Schema `properties`, `required`, `items`, type
arrays containing `null`, and `format: date-time`. External `$ref` resolution and
schema code execution are intentionally not performed.

## VS Code And MCP

Use `GOKUB: Generate Model from JSON` in the VS Code Command Palette to select a
file visually. AI agents can call `gokub_generate_model` through the GOKUB MCP
server with a project-relative input path.
