# Rust factory example

This extension exposes the conventional `new_extension()` factory in
`src/lib.rs`. It registers the `hello` tool and `count_words`, a tool with a required `text` parameter that answers an error result when it is missing.

Validate it:

```bash
ompig install --validate-only --json examples/extensions/rust-factory
```

Ompig imports the library factory into a generated Rust runner. A Rust factory can
share a Rust runtime cell. Use `src/main.rs` without a factory library for an
isolated standalone.
