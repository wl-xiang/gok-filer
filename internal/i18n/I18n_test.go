package i18n

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/forceu/gokapi/internal/test"
)

func TestInitAndTranslate(t *testing.T) {
	err := Init()
	test.IsEqualBool(t, err == nil, true)
	test.IsEqualInt(t, len(translations), len(supportedLanguages))

	// Every language has to define exactly the same keys as the reference language,
	// otherwise a phrase would silently fall back to English.
	reference := translations[DefaultLanguage]
	for _, lang := range supportedLanguages {
		dictionary, ok := translations[lang.Code]
		test.IsEqualBool(t, ok, true)
		test.IsEqualInt(t, len(dictionary), len(reference))
		for key := range reference {
			value, exists := dictionary[key]
			if !exists {
				t.Errorf("language %s is missing the key %q", lang.Code, key)
				continue
			}
			if value == "" {
				t.Errorf("language %s has an empty value for %q", lang.Code, key)
			}
		}
	}

	// Translation, including the fallback behaviour
	test.IsEqualString(t, Translate("en", "login.title"), "Login")
	test.IsEqualString(t, Translate("zh-CN", "login.title"), "登录")
	test.IsEqualString(t, Translate("en", "does.not.exist"), "does.not.exist")
	test.IsEqualString(t, TranslateFormat("en", "does.not.exist"), "does.not.exist")
}

func TestNormalize(t *testing.T) {
	Init()
	test.IsEqualString(t, Normalize("zh-CN"), "zh-CN")
	test.IsEqualString(t, Normalize("ZH-cn"), "zh-CN")
	test.IsEqualString(t, Normalize("zh"), "zh-CN")
	test.IsEqualString(t, Normalize("en-US"), "en")
	test.IsEqualString(t, Normalize(""), DefaultLanguage)
	test.IsEqualString(t, Normalize("kl-GL"), DefaultLanguage)
	test.IsEqualBool(t, IsSupported("zh-CN"), true)
	test.IsEqualBool(t, IsSupported("kl-GL"), false)
}

func TestFromRequestAndCookie(t *testing.T) {
	Init()
	request := httptest.NewRequest(http.MethodGet, "/admin", nil)
	test.IsEqualString(t, FromRequest(request), DefaultLanguage)
	test.IsEqualString(t, FromRequest(nil), DefaultLanguage)

	request.AddCookie(&http.Cookie{Name: CookieName, Value: "zh-CN"})
	test.IsEqualString(t, FromRequest(request), "zh-CN")

	// An unsupported value must not break the rendering
	invalidRequest := httptest.NewRequest(http.MethodGet, "/admin", nil)
	invalidRequest.AddCookie(&http.Cookie{Name: CookieName, Value: "kl-GL"})
	test.IsEqualString(t, FromRequest(invalidRequest), DefaultLanguage)

	recorder := httptest.NewRecorder()
	WriteLanguageCookie(recorder, "zh-CN")
	cookies := recorder.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, CookieName)
	test.IsEqualString(t, cookies[0].Value, "zh-CN")
	test.IsEqualInt(t, cookies[0].MaxAge, cookieMaxAge)
}

func TestSupportedLanguages(t *testing.T) {
	Init()
	languages := SupportedLanguages()
	test.IsEqualInt(t, len(languages), 2)
	test.IsEqualString(t, languages[0].Code, "en")
	test.IsEqualString(t, languages[1].Code, "zh-CN")
}
