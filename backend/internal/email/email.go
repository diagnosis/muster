package email

import (
	"bytes"
	"embed"
	"html/template"

	"github.com/diagnosis/go-toolkit/v3/apperr"
)

//go:embed templates/*.html
var templateFolder embed.FS

// Content struct holds the content of specific email
type Content struct {
	Subject  string
	Heading  string
	Body     string
	CTALabel string
	CTAURL   string
}

var baseTmpl = template.Must(template.ParseFS(templateFolder, "templates/base.html"))

// Render renders email content provided by services
func Render(c Content) (string, error) {
	var buff bytes.Buffer
	if err := baseTmpl.Execute(&buff, c); err != nil {
		return "", apperr.Internal("failed to render email", "template execute failed", err)
	}
	return buff.String(), nil
}
