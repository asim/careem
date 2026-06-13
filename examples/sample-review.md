<!-- Genuine, unedited output from `go run ./cmd/review examples/sample.go`
     (model: claude-sonnet-4-6). Reproduce it with your own ANTHROPIC_API_KEY. -->

# Smart Code Review

**Language:** Go  |  **Score:** 4/10  |  **Model:** claude-sonnet-4-6

> A small utility function that filters a slice of loosely-typed maps and extracts "name" values where "status" is "active". The logic is correct for the happy path but carries a latent panic risk and uses several non-idiomatic Go patterns that would make a teammate pause in review.

## Top 3 Improvements

### 1. 🔴 Unsafe type assertion will panic on missing or wrong-typed "name" `correctness` _(at Line 14: `name := data[i]["name"].(string)`)_

If the "name" key is absent, the map returns a nil interface{}. If it is present but not a string (e.g. a number), the bare assertion panics at runtime with no way to recover gracefully. Because the input type is `[]map[string]interface{}`, both situations are entirely possible.

**Suggestion:** Use the comma-ok form and decide on a skip-or-error policy explicitly:
```go
name, ok := data[i]["name"].(string)
if !ok || name == "" {
    continue // or collect an error
}
result = append(result, name)
```

### 2. 🟠 Prefer a range loop and typed input over an index loop with interface{} `readability` _(at Lines 9–17: the for-loop and `[]map[string]interface{}` signature)_

`for i := 0; i < len(data); i++` is a C-style loop that Go's `range` replaces cleanly. More importantly, `interface{}` (or `any` in modern Go) as the map value type throws away all compile-time safety. If a concrete type or even a small struct is feasible, the assertion on line 14 — and its panic risk — disappears entirely.

**Suggestion:** At minimum, switch to range:
```go
for _, row := range data {
    if row["status"] == "active" { ... }
}
```
Ideally, define a struct:
```go
type Record struct {
    Name   string
    Status string
}
func Process(data []Record) []string { ... }
```

### 3. 🟡 Initialise the result slice with var, not a composite literal `maintainability` _(at Line 9: `result := []string{}`)_

`[]string{}` allocates a non-nil empty slice immediately. The Go idiom for "I will append to this" is `var result []string`, which starts as nil and is handled identically by `append`, `len`, and `range`. The composite literal form is only preferred when you need a non-nil empty slice to distinguish it from nil in JSON serialisation or equality checks — which this function does not.

**Suggestion:**
```go
var result []string
```
If a non-nil empty slice is genuinely needed (e.g. for JSON `[]` vs `null`), add a comment explaining why so the next reader doesn't "fix" it.

## ✅ What's good

The nil-map guard (`if data[i] != nil`) shows defensive awareness — in Go a nil map read is safe, but an explicit check signals intentional handling of sparse input, which is a good instinct in a loosely-typed data pipeline.

## Minor notes

- The `//go:build ignore` tag is a clean way to keep demo/scratch code in-tree without polluting the build — good use of the build constraint system.
- Consider renaming the parameter from `data` to something more descriptive like `records` or `entries` to make the function signature self-documenting.
- In Go 1.18+, `interface{}` should be written as `any` for consistency with the standard library and current style guides.
