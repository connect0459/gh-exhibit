package persistence

import "html/template"

var viewTemplate = template.Must(template.New("view").Parse(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} · {{.Repo}}#{{.Number}}</title>
<style>
:root {
  --bg: #ffffff; --fg: #1f2328; --muted: #59636e; --border: #d1d9e0;
  --card-head: #f6f8fa; --link: #0969da; --code-bg: #eff1f3;
  --open: #1f883d; --closed: #cf222e; --merged: #8250df;
  --add-bg: #dafbe1; --del-bg: #ffebe9; --hunk-bg: #ddf4ff;
}
@media (prefers-color-scheme: dark) {
  :root {
    --bg: #0d1117; --fg: #f0f6fc; --muted: #9198a1; --border: #3d444d;
    --card-head: #151b23; --link: #4493f8; --code-bg: #202830;
    --add-bg: #12261e; --del-bg: #2d1214; --hunk-bg: #121d2f;
  }
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--fg); font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans", Helvetica, Arial, sans-serif; }
main { max-width: 920px; margin: 0 auto; padding: 24px 16px 64px; }
a { color: var(--link); }
h1.title { font-size: 28px; font-weight: 400; margin: 0 0 8px; }
h1.title .number { color: var(--muted); font-weight: 300; }
.meta { display: flex; align-items: center; gap: 8px; color: var(--muted); padding-bottom: 16px; margin-bottom: 24px; border-bottom: 1px solid var(--border); }
.state { color: #fff; border-radius: 2em; padding: 4px 12px; font-weight: 500; }
.state.open { background: var(--open); } .state.closed { background: var(--closed); } .state.merged { background: var(--merged); }
.card { border: 1px solid var(--border); border-radius: 6px; margin-bottom: 16px; overflow: hidden; }
.card > header { display: flex; align-items: center; gap: 8px; background: var(--card-head); border-bottom: 1px solid var(--border); padding: 8px 16px; color: var(--muted); }
.card > header strong { color: var(--fg); }
.card > header a { color: var(--muted); text-decoration: none; }
.card > header a:hover { color: var(--link); }
.avatar { display: inline-flex; align-items: center; justify-content: center; width: 28px; height: 28px; border-radius: 50%; background: var(--border); color: var(--fg); font-weight: 600; flex: none; }
.badge { border: 1px solid var(--border); border-radius: 2em; padding: 0 8px; font-size: 12px; }
.badge.approved { color: var(--open); border-color: var(--open); }
.badge.changes { color: var(--closed); border-color: var(--closed); }
.badge.outdated { color: var(--muted); }
.card .body { padding: 16px; overflow-wrap: anywhere; }
.card .body > :first-child { margin-top: 0; } .card .body > :last-child { margin-bottom: 0; }
.card .body img { max-width: 100%; }
.card .body pre { background: var(--code-bg); padding: 12px 16px; border-radius: 6px; overflow: auto; }
.card .body code { background: var(--code-bg); padding: 0.2em 0.4em; border-radius: 6px; font-size: 85%; }
.card .body pre code { background: none; padding: 0; }
.card .body blockquote { margin: 0; padding: 0 1em; color: var(--muted); border-left: 0.25em solid var(--border); }
.card .body table { border-collapse: collapse; display: block; overflow: auto; }
.card .body th, .card .body td { border: 1px solid var(--border); padding: 6px 13px; }
.inline-path { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: 12px; color: var(--fg); }
.diff { margin: 0; font: 12px/1.45 ui-monospace, SFMono-Regular, Menlo, monospace; border-bottom: 1px solid var(--border); overflow: auto; }
.diff div { padding: 0 16px; white-space: pre; }
.diff .add { background: var(--add-bg); } .diff .del { background: var(--del-bg); } .diff .hunk { background: var(--hunk-bg); color: var(--muted); }
.card .note { padding: 8px 16px; color: var(--muted); border-bottom: 1px solid var(--border); }
.file + .file { border-top: 1px solid var(--border); }
.file-head { display: flex; align-items: center; gap: 8px; padding: 8px 16px; background: var(--card-head); }
.file-head .counts { margin-left: auto; color: var(--muted); font-size: 12px; }
.file .diff { border-bottom: 0; }
.commit { display: flex; align-items: baseline; gap: 12px; padding: 8px 16px; }
.commit + .commit { border-top: 1px solid var(--border); }
.commit .sha { color: var(--muted); }
.commit details, .commit .subject { flex: 1; min-width: 0; overflow-wrap: anywhere; }
.commit summary { cursor: pointer; }
.commit pre { margin: 8px 0 0; white-space: pre-wrap; color: var(--muted); }
.commit .who { color: var(--muted); font-size: 12px; white-space: nowrap; }
</style>
</head>
<body>
<main>
<h1 class="title">{{.Title}} <span class="number">#{{.Number}}</span></h1>
<div class="meta"><span class="state {{.State}}">{{.StateLabel}}</span><span>{{.Repo}}</span></div>
{{range .Cards}}<section class="card kind-{{.Kind}}">
{{if .Author}}<header><span class="avatar" aria-hidden="true">{{.Initial}}</span><strong>{{.Author}}</strong>{{if .Badge}}<span class="badge {{.BadgeClass}}">{{.Badge}}</span>{{end}}<a href="{{.URL}}">{{.When}}</a></header>{{end}}
{{if .Path}}<header><span class="inline-path">{{.Path}}</span>{{if .Lines}}<span>{{.Lines}}</span>{{end}}{{if .Outdated}}<span class="badge outdated">Outdated</span>{{end}}</header>{{end}}
{{if .Title}}<header><strong>{{.Title}}</strong>{{if .Summary}}<span>{{.Summary}}</span>{{end}}</header>{{end}}
{{if .Note}}<div class="note">{{.Note}}</div>{{end}}
{{range .Files}}<div class="file"><div class="file-head"><span class="inline-path">{{.Name}}</span><span class="badge">{{.Status}}</span><span class="counts">{{.Counts}}</span></div>{{if .Diff}}<div class="diff">{{range .Diff}}<div class="{{.Class}}">{{.Text}}</div>{{end}}</div>{{end}}</div>
{{end}}{{range .Commits}}<div class="commit"><code class="sha">{{.SHA}}</code>{{if .Body}}<details><summary>{{.Subject}}</summary><pre>{{.Body}}</pre></details>{{else}}<span class="subject">{{.Subject}}</span>{{end}}<span class="who">{{.Author}} · {{.When}}</span></div>
{{end}}{{if .Diff}}<div class="diff">{{range .Diff}}<div class="{{.Class}}">{{.Text}}</div>{{end}}</div>{{end}}
{{if .HTML}}<div class="body">{{.HTML}}</div>{{end}}
</section>
{{end}}</main>
</body>
</html>
`))
