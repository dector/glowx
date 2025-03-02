package glow

import (
	"fmt"

	"github.com/sivukhin/godjot/djot_parser"
	"github.com/sivukhin/godjot/html_writer"
)

func Build() {
	fmt.Println("Building...")
	ast := djot_parser.BuildDjotAst([]byte("_Hello_"))
	html := djot_parser.NewConversionContext(
		"html",
		djot_parser.DefaultConversionRegistry,
		map[djot_parser.DjotNode]djot_parser.Conversion{},
	).ConvertDjotToHtml(&html_writer.HtmlWriter{}, ast[0])
	fmt.Print(html)
}
