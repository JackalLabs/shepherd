package main

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

func mdToHTML(md []byte, title string) []byte {
	// create markdown parser with extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs | parser.NoEmptyLineBeforeBlock
	p := parser.NewWithExtensions(extensions)
	doc := p.Parse(md)

	// create HTML renderer with extensions
	htmlFlags := html.CommonFlags | html.HrefTargetBlank | html.CompletePage
	opts := html.RendererOptions{Flags: htmlFlags, Head: []byte("<style>html {text-align: center; max-width: 60vw; margin-left: auto; margin-right: auto;}</style><link rel=\"stylesheet\" href=\"https://unpkg.com/marx-css/css/marx.min.css\">"), Title: title}
	renderer := html.NewRenderer(opts)

	return markdown.Render(doc, renderer)
}
