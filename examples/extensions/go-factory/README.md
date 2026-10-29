# go-factory: minimal Go extension

A conventional manifest-free Ompig extension written as an importable Go package.
It registers the `hello` tool and `count_words`, a tool with a required `text` parameter that answers an error result when it is missing.

## Structure

| File | Purpose |
|------|---------|
| `go.mod` | Go module and SDK dependency |
| `extension.go` | `Extension() *sdk.Extension` factory |

## Validate or run

```bash
ompig install --validate-only --json examples/extensions/go-factory
ompig -e examples/extensions/go-factory
```

Ompig discovers the factory from its exact Go signature and generates the runner.
The same package can run in an isolated subprocess, a packed subprocess cell, or
a fused Ompiglet Binary without source changes.
