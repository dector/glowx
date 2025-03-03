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
	index      string
	title      string
	fileName   string
	rawContent string
}

var DirOut = "out"
var FileGlowMark = ".glow_build"

func Build() {
	fmt.Println("Building...")

	prepareOutDir(DirOut)

	items := fetchLogItems()

	for _, i := range items {
		filePath := filepath.Join(DirOut, i.index, "index.html")
		os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		os.WriteFile(filePath, buildHtmlFileContent(i), os.ModePerm)
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
		fullName := f.Name()
		extension := filepath.Ext(fullName)

		fileName := strings.TrimSuffix(f.Name(), extension)
		index := strings.Split(fileName, "-")[0]
		title := strings.Split(fileName, "-")[1]

		content, _ := os.ReadFile(filePath)

		logItem := RawLogItem{
			index:      index,
			title:      title,
			fileName:   fileName,
			rawContent: string(content),
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

func buildHtmlFileContent(item RawLogItem) []byte {
	htmlContent := renderMarkdown(item.rawContent)
	return []byte(htmlContent)
}
