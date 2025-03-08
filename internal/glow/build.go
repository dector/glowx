package glow

import (
	"bytes"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/sivukhin/godjot/djot_parser"
	"github.com/sivukhin/godjot/html_writer"
)

//go:embed all:assets/*
var assets embed.FS

type RawLogEntry struct {
	index      string
	title      string
	fileName   string
	rawContent string
}

type HeaderAndContent struct {
	header  string
	content string
}

var DirOut = "out"
var FileGlowMark = ".glow_build"

func Try(err error) {
	if err != nil {
		panic(err)
	}
}

func Build() {
	fmt.Println("Building...")

	prepareOutDir(DirOut)

	items := fetchLogItems()

	for _, item := range items {
		filePath := filepath.Join(DirOut, item.index, "index.html")
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		os.WriteFile(filePath, buildHtmlFileContent(item), os.ModePerm)
	}

	// iterate over files
	// render each file
	// render index
	// render tag pages
}

func fetchLogItems() []RawLogEntry {
	result := []RawLogEntry{}

	dirPath := "log"
	files, err := os.ReadDir(dirPath)
	Try(err)
	for _, f := range files {
		filePath := filepath.Join(dirPath, f.Name())
		fullName := f.Name()
		extension := filepath.Ext(fullName)

		fileName := strings.TrimSuffix(f.Name(), extension)
		index := strings.Split(fileName, "-")[0]
		title := strings.Split(fileName, "-")[1]

		content, err := os.ReadFile(filePath)
		Try(err)

		headerAndContent, err := splitHeaderAndContent(string(content))

		logItem := RawLogEntry{
			index:      index,
			title:      title,
			fileName:   fileName,
			rawContent: headerAndContent.content,
		}
		result = append(result, logItem)
	}

	return result
}

func renderMarkdown(content string) string {
	ast := djot_parser.BuildDjotAst([]byte(content))
	html := djot_parser.NewConversionContext(
		"html",
		djot_parser.DefaultConversionRegistry,
		map[djot_parser.DjotNode]djot_parser.Conversion{},
	).ConvertDjotToHtml(&html_writer.HtmlWriter{}, ast[0])
	return html
}

func prepareOutDir(dir string) {
	fmt.Printf("Preparing %s dir...\n", dir)

	// check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
		os.Create(filepath.Join(dir, FileGlowMark))
	} else {
		glowMark := filepath.Join(dir, FileGlowMark)
		_, err := os.Stat(glowMark)
		if os.IsNotExist(err) {
			fmt.Printf("Out dir '%s' exists but doesn't have Glow mark", dir)
			os.Exit(1)
		}

		files, _ := os.ReadDir(dir)
		for _, f := range files {
			if f.Name() == FileGlowMark {
				continue
			}
			os.RemoveAll(filepath.Join(dir, f.Name()))
		}
	}
}

func buildHtmlFileContent(entry RawLogEntry) []byte {
	htmlContent := renderMarkdown(entry.rawContent)

	params := map[string]interface{}{
		"Page": struct {
			Title string
		}{
			Title: entry.title,
		},
		"Entry": struct {
			Title   string
			Content string
			Date    string
			Tags    []string
		}{
			Title:   entry.title,
			Content: htmlContent,
			Date:    "",
			Tags:    []string{},
		},
	}

	return []byte(renderPageTemplate("log_entry", params))
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

func splitHeaderAndContent(content string) (*HeaderAndContent, error) {
	contentLines := strings.Split(content, "\n")

	headerStart := -1
	for i, line := range contentLines {
		if strings.HasPrefix(line, "---") {
			headerStart = i
			break
		}
	}
	if headerStart == -1 {
		return nil, fmt.Errorf("could not find header start")
	}

	headerEnd := -1
	for i, line := range contentLines[headerStart+1:] {
		if strings.HasPrefix(line, "---") {
			headerEnd = i + headerStart + 1
			break
		}
	}
	if headerEnd == -1 {
		return nil, fmt.Errorf("could not find header end")
	}

	headerContent := strings.Join(contentLines[headerStart+1:headerEnd], "\n")

	// var header LogHeader
	// err = yaml.Unmarshal([]byte(headerContent), &header)
	// if err != nil {
	// 	return fmt.Errorf("error unmarshaling YAML: %w", err)
	// }

	realContent := strings.TrimSpace(strings.Join(contentLines[headerEnd+1:], "\n"))
	return &HeaderAndContent{
		header:  headerContent,
		content: realContent,
	}, nil
}
