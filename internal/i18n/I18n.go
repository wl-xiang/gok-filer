package i18n

/**
Internationalisation of the Gokapi user interface.

All user-visible strings are stored in JSON resource files inside the ./locales folder.
Adding a new language therefore only requires
  - a new <code>.json</code> file in ./locales,
  - an entry in the supportedLanguages list below.
Missing keys fall back to English (DefaultLanguage) and finally to the key itself, so a
partially translated language never breaks the interface.
*/

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

//go:embed locales/*.json
var localesFS embed.FS

// CookieName is the cookie used to remember the language selected by the user
const CookieName = "gokapi_language"

// DefaultLanguage is used whenever no language has been selected by the user
const DefaultLanguage = "en"

// cookieMaxAge is the lifetime of the language cookie in seconds (one year)
const cookieMaxAge = 365 * 24 * 60 * 60

// LanguageInfo describes one supported language
type LanguageInfo struct {
	Code string // IETF language tag, e.g. "en" or "zh-CN"
	Name string // Name of the language in the language itself
}

// supportedLanguages lists all languages that are shipped with Gokapi
var supportedLanguages = []LanguageInfo{
	{Code: "en", Name: "English"},
	{Code: "zh-CN", Name: "简体中文"},
}

// translations holds the loaded resource files, keyed by language code
var translations = map[string]map[string]string{}

// Init loads all language resource files. It must be called once before the templates are parsed.
func Init() error {
	for _, lang := range supportedLanguages {
		content, err := localesFS.ReadFile("locales/" + lang.Code + ".json")
		if err != nil {
			return fmt.Errorf("unable to read locale %s: %w", lang.Code, err)
		}
		dictionary := make(map[string]string)
		if err := json.Unmarshal(content, &dictionary); err != nil {
			return fmt.Errorf("unable to parse locale %s: %w", lang.Code, err)
		}
		translations[lang.Code] = dictionary
	}
	return nil
}

// SupportedLanguages returns the list of all available languages
func SupportedLanguages() []LanguageInfo {
	result := make([]LanguageInfo, len(supportedLanguages))
	copy(result, supportedLanguages)
	return result
}

// IsSupported returns true if a resource file is available for the given language code
func IsSupported(code string) bool {
	_, ok := translations[code]
	return ok
}

// Normalize returns the given code if it is supported, otherwise DefaultLanguage.
// Matching is case-insensitive and also accepts a language tag whose primary part matches,
// e.g. "zh" or "zh-Hans" both resolve to "zh-CN" and "en-US" resolves to "en".
func Normalize(code string) string {
	if IsSupported(code) {
		return code
	}
	normalized := strings.ToLower(strings.TrimSpace(code))
	if normalized == "" {
		return DefaultLanguage
	}
	normalizedPrimary := primaryTag(normalized)
	for _, lang := range supportedLanguages {
		lowerCode := strings.ToLower(lang.Code)
		if lowerCode == normalized || lowerCode == normalizedPrimary || primaryTag(lowerCode) == normalizedPrimary {
			return lang.Code
		}
	}
	return DefaultLanguage
}

// primaryTag returns the part of an IETF language tag before the first separator,
// e.g. "zh" for "zh-CN"
func primaryTag(code string) string {
	if index := strings.IndexAny(code, "-_"); index > 0 {
		return code[:index]
	}
	return code
}

// FromRequest returns the language that should be used for rendering the given request.
// The language cookie set by the user takes precedence over the default language.
func FromRequest(r *http.Request) string {
	if r == nil {
		return DefaultLanguage
	}
	cookie, err := r.Cookie(CookieName)
	if err != nil || cookie.Value == "" {
		return DefaultLanguage
	}
	return Normalize(cookie.Value)
}

// WriteLanguageCookie stores the selected language in a cookie, so that it is remembered
// across requests and browser sessions.
func WriteLanguageCookie(w http.ResponseWriter, code string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    Normalize(code),
		Path:     "/",
		MaxAge:   cookieMaxAge,
		SameSite: http.SameSiteLaxMode,
	})
}

// Translator returns a translation function bound to the given language, suitable for templates
func Translator(language string) func(key string) string {
	return func(key string) string {
		return Translate(language, key)
	}
}

// TranslatorFormat returns a translating function bound to the given language that supports
// printf-style placeholders, e.g. TranslateFormat(lang, "upload.limit", 5)
func TranslatorFormat(language string) func(key string, args ...any) string {
	return func(key string, args ...any) string {
		return TranslateFormat(language, key, args...)
	}
}

// Translate returns the translation of the given key. It falls back to the default language and
// finally to the key itself, so that a missing translation never renders an empty string.
func Translate(language, key string) string {
	if dictionary, ok := translations[language]; ok {
		if value, ok := dictionary[key]; ok {
			return value
		}
	}
	if dictionary, ok := translations[DefaultLanguage]; ok {
		if value, ok := dictionary[key]; ok {
			return value
		}
	}
	return key
}

// TranslateFormat translates the given key and applies printf-style formatting with the arguments
func TranslateFormat(language, key string, args ...any) string {
	format := Translate(language, key)
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}
