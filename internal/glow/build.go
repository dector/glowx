package glow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sivukhin/godjot/djot_parser"
	"github.com/sivukhin/godjot/html_writer"
)

type RawLogItem struct {
	file    string
	content string
}

func Build() {
	fmt.Println("Building...")
	items := fetchLogItems()

	for _, i := range items {
		os.Mkdir("out", os.ModePerm)
		htmlContent := renderMarkdown(i.content)

		filePath := filepath.Join("out", fmt.Sprintf("%s.html", i.file))
		os.WriteFile(filePath, []byte(htmlContent), os.ModePerm)
	}

	// iterate over files
	// render each file
	// render index
	// render tag pages
}

func fetchLogItems() []RawLogItem {
	result := []RawLogItem{}

	dirPath := "log"
	files, _ := os.ReadDir(dirPath)
	for _, f := range files {
		filePath := filepath.Join(dirPath, f.Name())
		fileName := strings.TrimSuffix(f.Name(), filepath.Ext(f.Name()))
		content, _ := os.ReadFile(filePath)

		logItem := RawLogItem{
			file:    fileName,
			content: string(content),
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
