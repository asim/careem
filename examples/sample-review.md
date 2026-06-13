# Smart Code Review

**Language:** Go  |  **Score:** 5/10  |  **Model:** claude-sonnet-4-6

> Filters a slice of records down to the names of "active" entries. The logic is
> correct but the implementation is unidiomatic and will panic on malformed input.

*(Illustrative output, lightly trimmed, showing the shape the tool produces.)*

## Top 3 Improvements

### 1. 🔴 Unchecked type assertion will panic `correctness` _(at `data[i]["name"].(string)`)_

`data[i]["name"].(string)` panics if `name` is missing or not a string — one bad
record takes down the whole call.

**Suggestion:** Use the comma-ok form and skip/handle the bad row:
```go
name, ok := data[i]["name"].(string)
if !ok {
    continue
}
```

### 2. 🟠 Index loop instead of range `readability` _(at the `for` loop)_

`for i := 0; i < len(data); i++ { data[i]... }` is C-style noise in Go and repeats
the indexing. `range` reads cleaner and avoids off-by-one risk.

**Suggestion:** `for _, record := range data {` and reference `record`.

### 3. 🟡 Prefer a concrete type over `map[string]interface{}` `maintainability` _(at the signature)_

`[]map[string]interface{}` pushes type checks to runtime everywhere. A small
struct (`type Record struct { Name, Status string }`) makes the contract explicit
and removes the assertions entirely.

**Suggestion:** Define a `Record` type and accept `[]Record`.

## ✅ What's good

The function has a single, clear responsibility and a descriptive name — it's easy
to tell at a glance that it returns the names of active records.
