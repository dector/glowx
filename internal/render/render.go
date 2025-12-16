package render

import (
	"bytes"
	"embed"
	"slices"
	"text/template"

	"github.com/dector/glowx/internal/parser"
	. "github.com/dector/glowx/internal/try"
	"github.com/sivukhin/godjot/djot_parser"
	"github.com/sivukhin/godjot/html_writer"
)

//go:embed all:assets/*
var assets embed.FS

func RenderDjot(content string) string {
	ast := djot_parser.BuildDjotAst([]byte(content))
	html := djot_parser.NewConversionContext(
		"html",
		djot_parser.DefaultConversionRegistry,
		map[djot_parser.DjotNode]djot_parser.Conversion{},
	).ConvertDjotToHtml(&html_writer.HtmlWriter{}, ast[0])
	return html
}

func toSortedReverse(_entries []parser.LogEntryFile) []parser.LogEntryFile {
	entries := make([]parser.LogEntryFile, len(_entries))
	copy(entries, _entries)

	slices.SortFunc(entries, func(a, b parser.LogEntryFile) int {
		return -a.Meta.CreatedAt.Compare(b.Meta.CreatedAt)
	})

	return entries
}

func BuildLogIndexHtmlFileContent(buildId string, _entries []parser.LogEntryFile) []byte {
	entries := toSortedReverse(_entries)

	pEntries := []struct {
		Url   string
		Title string
	}{}
	for _, entry := range entries {
		pEntries = append(pEntries, struct {
			Url   string
			Title string
		}{
			Url:   "/log/" + entry.Index,
			Title: entry.Meta.Title,
		})
	}

	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log",
			BuildId: buildId,
		},
		"Title":   "/log",
		"Entries": pEntries,
	}

	return []byte(renderPageTemplate("log_list", params))
}

func BuildLogEntryHtmlFileContent(buildId string, entryFile parser.LogEntryFile) []byte {
	htmlContent := RenderDjot(entryFile.RawContent)

	meta := entryFile.Meta
	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log ~ " + meta.Title,
			BuildId: buildId,
		},
		"Entry": struct {
			Title   string
			Content string
			Date    string
			Tags    []string
		}{
			Title:   meta.Title,
			Content: htmlContent,
			Date:    meta.CreatedAt.Format("Mon, 02 Jan 2006 15:04"),
			Tags:    meta.Tags,
		},
	}

	return []byte(renderPageTemplate("log_entry", params))
}

func BuildLogTagHtmlFileContent(buildId string, tag string, _entries []parser.LogEntryFile) []byte {
	entries := toSortedReverse(_entries)

	pEntries := []struct {
		Url   string
		Title string
	}{}
	for _, entry := range entries {
		pEntries = append(pEntries, struct {
			Url   string
			Title string
		}{
			Url:   "/log/" + entry.Index,
			Title: entry.Meta.Title,
		})
	}

	params := map[string]interface{}{
		"Page": struct {
			Title   string
			BuildId string
		}{
			Title:   "/dector/log ~ " + tag,
			BuildId: buildId,
		},
		"Title":    "#" + tag,
		"Entries":  pEntries,
		"ShowBack": true,
	}

	return []byte(renderPageTemplate("log_list", params))
}

func renderPageTemplate(name string, params map[string]interface{}) string {
	// Render content
	func() {
		var out bytes.Buffer
		assetFile, err := assets.ReadFile("assets/templates/" + name + ".html")
		Try(err)
		tmpl, err := template.New(name).Parse(string(assetFile))
		Try(err)

		err = tmpl.ExecuteTemplate(&out, name, params)
		Try(err)
		params["Content"] = out.String()
		params["BodyClass"] = "p-log"
	}()

	// Render HTML page
	var out bytes.Buffer
	func() {
		assetFile, err := assets.ReadFile("assets/templates/__page.html")
		Try(err)
		tmpl, err := template.New(name).Parse(string(assetFile))
		Try(err)

		err = tmpl.ExecuteTemplate(&out, name, params)
		Try(err)
	}()

	return out.String()
}
