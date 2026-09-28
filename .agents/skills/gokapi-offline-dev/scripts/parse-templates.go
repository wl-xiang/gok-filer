// Parse all Gokapi html/template files with the production func map to catch syntax errors
// and unknown functions. Lives outside the repository so `go build ./...` is unaffected.
//
// Usage:
//   cd <this dir> && go run . <repo>/internal/webserver/web/templates
//
// The stubs only need to exist so the parser can resolve the function names; the real
// implementations live in internal/i18n and internal/webserver.
package main

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: tplcheck <templates-dir>")
		os.Exit(2)
	}
	pattern := filepath.Join(os.Args[1], "*.tmpl")

	funcMap := template.FuncMap{
		"newAdminButtonContext": func(a, b any) any { return nil },
		"tr":                    func(key string) string { return key },
		"trf":                   func(key string, args ...any) string { return key },
		"availableLanguages":    func() []map[string]string { return nil },
		"currentLanguage":       func() string { return "en" },
	}

	templates, err := template.New("").Funcs(funcMap).ParseGlob(pattern)
	if err != nil {
		fmt.Println("TEMPLATE PARSE ERROR:", err)
		os.Exit(1)
	}
	fmt.Println("Templates parsed successfully.")
	fmt.Println(templates.DefinedTemplates())
}
