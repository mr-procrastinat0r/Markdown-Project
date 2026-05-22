package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var gm = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
)

func ToHTML(md []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := gm.Convert(md, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func WrapDocument(title string, body []byte) []byte {
	if title == "" {
		title = "Note"
	}
	var buf bytes.Buffer
	buf.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	buf.WriteString("<meta charset=\"utf-8\">\n")
	buf.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	buf.WriteString("<title>")
	buf.WriteString(escapeHTML(title))
	buf.WriteString("</title>\n<style>body{font-family:system-ui,sans-serif;max-width:48rem;margin:2rem auto;padding:0 1rem;line-height:1.6}pre{background:#f4f4f5;padding:1rem;overflow:auto}code{background:#f4f4f5;padding:.1em .3em;border-radius:3px}</style>\n")
	buf.WriteString("</head>\n<body>\n")
	buf.Write(body)
	buf.WriteString("\n</body>\n</html>\n")
	return buf.Bytes()
}

func escapeHTML(s string) string {
	var b bytes.Buffer
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
