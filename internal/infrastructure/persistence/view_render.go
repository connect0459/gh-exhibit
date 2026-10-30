package persistence

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"

	"github.com/connect0459/gh-exhibit/internal/domain/valueobjects"
)

type viewModel struct {
	Title      string
	Repo       string
	Number     int
	State      string
	StateLabel string
	Cards      []viewCard
}

type viewCard struct {
	Kind       string
	Initial    string
	Author     string
	When       string
	URL        string
	Badge      string
	BadgeClass string
	Path       string
	Lines      string
	Outdated   bool
	Diff       []diffLine
	HTML       template.HTML
}

type diffLine struct {
	Class string
	Text  string
}

var markdownConverter = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithRendererOptions(html.WithUnsafe(), html.WithHardWraps()),
)

var (
	localAssetSrc = regexp.MustCompile(`^assets/[^/\\]+$`)
	codeLanguage  = regexp.MustCompile(`^language-[A-Za-z0-9_+-]+$`)
	checkboxType  = regexp.MustCompile(`^checkbox$`)
	cellAlign     = regexp.MustCompile(`^(left|right|center)$`)
	dimension     = regexp.MustCompile(`^[0-9]{1,4}%?$`)
)

var viewPolicy = newViewPolicy()

func newViewPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()

	p.AllowElements(
		"h1", "h2", "h3", "h4", "h5", "h6", "p", "br", "hr", "blockquote",
		"pre", "code", "em", "strong", "del", "ins", "sub", "sup", "kbd",
		"ul", "ol", "li", "dl", "dt", "dd", "details", "summary",
		"table", "thead", "tbody", "tfoot", "tr",
	)
	p.AllowAttrs("class").Matching(codeLanguage).OnElements("code")
	p.AllowAttrs("align").Matching(cellAlign).OnElements("th", "td")
	p.AllowElements("th", "td")
	p.AllowAttrs("type").Matching(checkboxType).OnElements("input")
	p.AllowAttrs("checked", "disabled").OnElements("input")

	p.AllowURLSchemes("http", "https", "mailto")
	p.AllowRelativeURLs(true)
	p.RequireParseableURLs(true)
	p.AllowAttrs("href").OnElements("a")
	p.RequireNoFollowOnLinks(true)
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)

	p.AllowAttrs("src").Matching(localAssetSrc).OnElements("img")
	p.AllowAttrs("alt").Matching(bluemonday.Paragraph).OnElements("img")
	p.AllowAttrs("width", "height").Matching(dimension).OnElements("img")

	return p
}

func renderView(ref valueobjects.IssueRef, doc valueobjects.Document, rewrite func(string) string) ([]byte, error) {
	model := viewModel{
		Title:      doc.Title(),
		Repo:       ref.Owner() + "/" + ref.Repo(),
		Number:     ref.Number(),
		State:      "open",
		StateLabel: "Open",
	}

	for _, entry := range doc.Entries() {
		switch e := entry.(type) {
		case valueobjects.Body:
			model.State, model.StateLabel = stateOf(e)
			model.Cards = append(model.Cards, authoredCard("body", e.Attribution(), markdownHTML(e.Content(), rewrite)))
		case valueobjects.IssueComment:
			model.Cards = append(model.Cards, authoredCard("comment", e.Attribution(), markdownHTML(e.Body(), rewrite)))
		case valueobjects.PullRequestReview:
			card := authoredCard("review", e.Attribution(), markdownHTML(e.Body(), rewrite))
			card.Badge, card.BadgeClass = reviewBadge(e.State())
			model.Cards = append(model.Cards, card)
		case valueobjects.InlineReviewComment:
			card := authoredCard("inline", e.Attribution(), markdownHTML(e.Body(), rewrite))
			ctx := e.Context()
			card.Path = ctx.Path()
			card.Lines = lineRange(ctx)
			card.Outdated = ctx.Outdated()
			card.Diff = diffLines(ctx.DiffHunk())
			model.Cards = append(model.Cards, card)
		case valueobjects.PullRequestChecks:
			continue
		default:
			var buf bytes.Buffer
			if err := entry.Render(&buf); err != nil {
				return nil, fmt.Errorf("render an entry to Markdown: %w", err)
			}
			if rendered := markdownHTML(buf.String(), rewrite); rendered != "" {
				model.Cards = append(model.Cards, viewCard{Kind: "generic", HTML: rendered})
			}
		}
	}

	var out bytes.Buffer
	if err := viewTemplate.Execute(&out, model); err != nil {
		return nil, fmt.Errorf("render the view template: %w", err)
	}
	return out.Bytes(), nil
}

func markdownHTML(markdown string, rewrite func(string) string) template.HTML {
	if strings.TrimSpace(markdown) == "" {
		return ""
	}

	var rendered bytes.Buffer
	if err := markdownConverter.Convert([]byte(rewrite(markdown)), &rendered); err != nil {
		return template.HTML(template.HTMLEscapeString(markdown))
	}
	sanitized := bytes.TrimSpace(viewPolicy.SanitizeBytes(rendered.Bytes()))
	return template.HTML(sanitized)
}

func authoredCard(kind string, a valueobjects.Attribution, body template.HTML) viewCard {
	initial, _ := utf8.DecodeRuneInString(strings.ToUpper(a.Author()))
	return viewCard{
		Kind:    kind,
		Initial: string(initial),
		Author:  a.Author(),
		When:    a.CreatedAt().UTC().Format("2006-01-02 15:04 UTC"),
		URL:     a.URL().String(),
		HTML:    body,
	}
}

func stateOf(b valueobjects.Body) (state, label string) {
	switch {
	case b.MergedAt() != nil:
		return "merged", "Merged"
	case b.ClosedAt() != nil:
		return "closed", "Closed"
	default:
		return "open", "Open"
	}
}

func reviewBadge(s valueobjects.ReviewState) (label, class string) {
	switch s {
	case valueobjects.ReviewStateApproved:
		return "Approved", "approved"
	case valueobjects.ReviewStateChangesRequested:
		return "Changes requested", "changes"
	case valueobjects.ReviewStateCommented:
	}
	return "Commented", "commented"
}

func lineRange(c valueobjects.InlineContext) string {
	line := c.Line()
	if line == nil {
		return ""
	}
	if start := c.StartLine(); start != nil {
		return fmt.Sprintf("L%d-L%d", *start, *line)
	}
	return fmt.Sprintf("L%d", *line)
}

func diffLines(hunk string) []diffLine {
	if hunk == "" {
		return nil
	}

	var lines []diffLine
	for _, text := range strings.Split(hunk, "\n") {
		class := "ctx"
		switch {
		case strings.HasPrefix(text, "@@"):
			class = "hunk"
		case strings.HasPrefix(text, "+"):
			class = "add"
		case strings.HasPrefix(text, "-"):
			class = "del"
		}
		lines = append(lines, diffLine{Class: class, Text: text})
	}
	return lines
}
