// Command server is the web front end for the Smart Code Reviewer.
//
// Paste a snippet, get a structured pre-human review rendered in the browser.
//
//	ANTHROPIC_API_KEY=... go run ./cmd/server
//	# then open http://localhost:8080
//
// Configure with PORT (default 8080) and REVIEWER_MODEL.
package main

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/asim/careem/reviewer"
)

type pageData struct {
	Code   string
	Result *reviewer.Result
	Err    string
}

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"add1": func(i int) int { return i + 1 },
	"sevColor": func(s string) string {
		switch s {
		case "high":
			return "#d6336c"
		case "medium":
			return "#f08c00"
		default:
			return "#f59f00"
		}
	},
}).Parse(pageHTML))

func main() {
	http.HandleFunc("/", handle)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Smart Code Reviewer listening on http://localhost%s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handle(w http.ResponseWriter, r *http.Request) {
	data := pageData{}
	if r.Method == http.MethodPost {
		data.Code = r.FormValue("code")
		ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
		defer cancel()
		result, err := reviewer.Review(ctx, data.Code, reviewer.Options{
			Language: r.FormValue("language"),
			Context:  r.FormValue("context"),
		})
		if err != nil {
			data.Err = err.Error()
		} else {
			data.Result = result
		}
	}
	if err := page.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

const pageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Smart Code Reviewer</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: system-ui, sans-serif; max-width: 1100px; margin: 2rem auto; padding: 0 1rem; line-height: 1.5; }
  h1 { margin-bottom: .2rem; }
  .sub { color: #888; margin-top: 0; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; }
  @media (max-width: 800px) { .grid { grid-template-columns: 1fr; } }
  textarea { width: 100%; min-height: 360px; font-family: ui-monospace, monospace; font-size: 13px; padding: .6rem; box-sizing: border-box; }
  input[type=text] { width: 100%; padding: .4rem; box-sizing: border-box; margin-bottom: .6rem; }
  button { background: #1f7a3d; color: #fff; border: 0; padding: .7rem 1.2rem; font-size: 15px; border-radius: 6px; cursor: pointer; }
  .card { border: 1px solid #8884; border-radius: 8px; padding: 1rem; margin-bottom: 1rem; }
  .score { font-size: 2rem; font-weight: 700; }
  .sev { font-weight: 700; text-transform: uppercase; font-size: .75rem; }
  .pos { background: #1f7a3d22; border-left: 4px solid #1f7a3d; padding: .8rem 1rem; border-radius: 4px; }
  .err { background: #d6336c22; border-left: 4px solid #d6336c; padding: .8rem 1rem; border-radius: 4px; }
  pre { white-space: pre-wrap; }
  label { font-weight: 600; font-size: .85rem; }
</style>
</head>
<body>
  <h1>🔍 Smart Code Reviewer</h1>
  <p class="sub">An AI first-pass review for readability, structure, and maintainability — so human reviewers spend their time on what matters.</p>
  <form method="post" class="grid">
    <div>
      <label>Your code</label>
      <textarea name="code" placeholder="Paste a function or short file here…">{{.Code}}</textarea>
      <input type="text" name="language" placeholder="Language hint (optional — auto-detected)">
      <input type="text" name="context" placeholder="Context (optional, e.g. 'runs in a hot loop')">
      <button type="submit">Review code ▶</button>
    </div>
    <div>
      {{if .Err}}
        <div class="err"><strong>Review failed:</strong> {{.Err}}</div>
      {{else if .Result}}
        {{with .Result}}
        <div class="card">
          <span class="score">{{.OverallScore}}/10</span> &nbsp; · &nbsp; <strong>{{.Language}}</strong>
          <p>{{.Summary}}</p>
        </div>
        <h3>Top 3 improvements</h3>
        {{range $i, $imp := .Improvements}}
          <div class="card">
            <strong>{{add1 $i}}. {{$imp.Title}}</strong>
            <span style="color:#888">({{$imp.Category}}{{if $imp.Location}} · {{$imp.Location}}{{end}})</span><br>
            <span class="sev" style="color:{{sevColor $imp.Severity}}">{{$imp.Severity}}</span>
            <p>{{$imp.Explanation}}</p>
            <strong>Suggestion</strong><pre>{{$imp.Suggestion}}</pre>
          </div>
        {{end}}
        <div class="pos">✅ <strong>What's good:</strong> {{.PositiveNote}}</div>
        {{if .ExtraObservations}}
          <h4>Minor notes</h4>
          <ul>{{range .ExtraObservations}}<li>{{.}}</li>{{end}}</ul>
        {{end}}
        {{end}}
      {{else}}
        <p class="sub">Your structured review will appear here.</p>
      {{end}}
    </div>
  </form>
</body>
</html>`
