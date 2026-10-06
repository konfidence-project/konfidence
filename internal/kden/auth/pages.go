package auth

import (
	"bytes"
	_ "embed"
	"html/template"

	designsystem "github.com/konfidence-project/konfidence/packages/konfidence-design-system"
)

//go:embed pages/login_result.gohtml
var loginResultPageSource string

//go:embed pages/login_result.css
var loginResultPageStyles string

var loginResultPageTemplate = template.Must(
	template.New("login_result.gohtml").Parse(loginResultPageSource),
)

type loginResultPageData struct {
	Successful             bool
	Title                  string
	Heading                string
	Message                string
	CloseDelayMilliseconds int
	Styles                 template.CSS
}

var (
	loginSuccessPage = mustRenderLoginResultPage(loginResultPageData{
		Successful:             true,
		Title:                  "Konfidence Login Successful",
		Heading:                "Login successful",
		Message:                "Your identity has been verified.",
		CloseDelayMilliseconds: 5000,
	})

	loginFailurePage = mustRenderLoginResultPage(loginResultPageData{
		Successful:             false,
		Title:                  "Konfidence Login Failed",
		Heading:                "Login failed",
		Message:                "Authentication could not be completed. Check your terminal for details.",
		CloseDelayMilliseconds: 5000,
	})
)

func mustRenderLoginResultPage(data loginResultPageData) string {
	data.Styles = template.CSS(
		designsystem.TokensCSS() + "\n" + loginResultPageStyles,
	)

	var page bytes.Buffer
	if err := loginResultPageTemplate.Execute(&page, data); err != nil {
		panic(err)
	}

	return page.String()
}
