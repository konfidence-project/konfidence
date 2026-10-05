package auth

import (
	"bytes"
	_ "embed"
	"html/template"
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
	Eyebrow                string
	Heading                string
	Message                string
	CloseDelayMilliseconds int
	Styles                 template.CSS
}

var (
	loginSuccessPage = mustRenderLoginResultPage(loginResultPageData{
		Successful:             true,
		Title:                  "Konfidence Login Successful",
		Eyebrow:                "Authentication complete",
		Heading:                "Login successful",
		Message:                "Your identity has been verified.",
		CloseDelayMilliseconds: 5000,
	})

	loginFailurePage = mustRenderLoginResultPage(loginResultPageData{
		Successful:             false,
		Title:                  "Konfidence Login Failed",
		Eyebrow:                "Authentication failed",
		Heading:                "We couldn't sign you in",
		Message:                "Authentication could not be completed. Check your terminal for details.",
		CloseDelayMilliseconds: 5000,
	})
)

func mustRenderLoginResultPage(data loginResultPageData) string {
	data.Styles = template.CSS(loginResultPageStyles)

	var page bytes.Buffer
	if err := loginResultPageTemplate.Execute(&page, data); err != nil {
		panic(err)
	}

	return page.String()
}
