# 🔍 Smart Code Reviewer

An AI assistant that gives code a **structured first-pass review** — for
readability, structure, and maintainability — *before* it reaches a human
reviewer. People then spend their attention on judgment calls, not nitpicks.

Built for the Careem "AI savviness" challenge (Challenge #1, which also contains
Challenge #3's prompt). Written in **Go, standard library only** — no external
dependencies.

> See [`examples/sample-review.md`](examples/sample-review.md) for a sample of
> the output, and [`PROMPT.md`](PROMPT.md) for the reviewing prompt itself.

## What it does

Paste a snippet (or pipe a file) and get back:

- a one-line **health summary** and a **1–10 score**,
- the **three highest-impact improvements**, each tagged with
  *category* (readability / structure / maintainability / correctness /
  performance / security), *severity*, *location*, and a **concrete fix**,
- **one genuine positive note**, and
- (CLI) a Markdown report.

## Why it's built this way

- **Senior-engineer prompt** framed as a *pre-human review*, so feedback stays
  high-signal instead of becoming a linter dump. See [`PROMPT.md`](PROMPT.md).
- **Structured output via a Claude tool schema** — the model returns the review
  through a forced `submit_review` tool call, so the result is always valid,
  renderable JSON (no fragile text scraping). See
  [`reviewer/reviewer.go`](reviewer/reviewer.go).
- **Anti-quota guardrails** ("be proportionate", "don't invent problems") keep
  it from manufacturing weak findings just to reach three.
- **A mandatory positive note** keeps reviews constructive and trustworthy for
  the human author.

## Run it

Requires Go 1.24+ and an Anthropic API key.

```bash
export ANTHROPIC_API_KEY=sk-ant-...

# Web app — then open http://localhost:8080
go run ./cmd/server

# CLI — review a file or pipe via stdin
go run ./cmd/review examples/sample.go
cat main.go | go run ./cmd/review
```

Set `REVIEWER_MODEL=claude-opus-4-8` for the deepest analysis; the default
`claude-sonnet-4-6` is the fast/quality sweet spot. The server port defaults to
`8080` (override with `PORT`).

Build standalone binaries:

```bash
go build -o review  ./cmd/review
go build -o server  ./cmd/server
```

## Project layout

| Path | Purpose |
|------|---------|
| `reviewer/reviewer.go` | Core logic, prompt, the `submit_review` tool schema, Markdown rendering |
| `cmd/server/` | `net/http` web UI (`html/template`, no JS deps) |
| `cmd/review/` | Terminal entry point (file arg or stdin) |
| `PROMPT.md` | Standalone copy-paste prompt (Challenge #3) |
| `SUMMARY.md` | 100-word summary of the idea and approach |
| `examples/` | A rough sample snippet and an illustrative review |

## Run with Docker

The server is a static binary on a distroless base (tiny image, no shell, runs
as non-root):

```bash
docker build -t smart-code-reviewer .
docker run --rm -p 8080:8080 -e ANTHROPIC_API_KEY=sk-ant-... smart-code-reviewer
# open http://localhost:8080
```

## Deploy a public link

The image is a single static binary, so it deploys anywhere that runs a
container (Fly.io, Render, Cloud Run, a VM). Set `ANTHROPIC_API_KEY` and `PORT`
in the environment.
