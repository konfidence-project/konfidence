package designsystem

import _ "embed"

//go:embed src/styles/tokens.css
var tokensCSS string

// TokensCSS returns the canonical Konfidence design tokens.
func TokensCSS() string {
	return tokensCSS
}
