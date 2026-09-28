//go:build !integration && test

package webserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/forceu/gokapi/internal/i18n"
	"github.com/forceu/gokapi/internal/test"
)

// renderLoginPage renders the login template for the given request
func renderLoginPage(t *testing.T, r *http.Request, view LoginView) string {
	t.Helper()
	view.PublicName = "Gokapi"
	recorder := httptest.NewRecorder()
	err := renderTemplate(recorder, r, "login", view)
	test.IsEqualBool(t, err == nil, true)
	return recorder.Body.String()
}

// TestLocalisedRendering makes sure that the language selected via the cookie is used for
// rendering and that the default language stays English.
func TestLocalisedRendering(t *testing.T) {
	// No cookie -> English, which keeps backwards compatibility
	english := renderLoginPage(t, httptest.NewRequest(http.MethodGet, "/login", nil), LoginView{})
	test.IsEqualBool(t, strings.Contains(english, "Forgot password"), true)

	// Cookie set to Chinese -> Chinese phrases are rendered
	chineseRequest := httptest.NewRequest(http.MethodGet, "/login", nil)
	chineseRequest.AddCookie(&http.Cookie{Name: i18n.CookieName, Value: "zh-CN"})
	chinese := renderLoginPage(t, chineseRequest, LoginView{})
	test.IsEqualBool(t, strings.Contains(chinese, "忘记密码"), true)
	test.IsEqualBool(t, strings.Contains(chinese, "Forgot password"), false)

	// Conditional phrases are translated as well
	failedLogin := LoginView{IsFailedLogin: true}
	test.IsEqualBool(t, strings.Contains(renderLoginPage(t, httptest.NewRequest(http.MethodGet, "/login", nil), failedLogin),
		"Incorrect username or password"), true)
	test.IsEqualBool(t, strings.Contains(renderLoginPage(t, chineseRequest, failedLogin),
		"用户名或密码错误"), true)
}

// TestHeaderContainsToolbarAndTheme verifies that the theme and language switcher are present
// on every rendered page and that the glass theme stylesheet is loaded.
func TestHeaderContainsToolbarAndTheme(t *testing.T) {
	page := renderLoginPage(t, httptest.NewRequest(http.MethodGet, "/login", nil), LoginView{})
	test.IsEqualBool(t, strings.Contains(page, `id="gk-toolbar"`), true)
	test.IsEqualBool(t, strings.Contains(page, `id="gk-theme-toggle"`), true)
	test.IsEqualBool(t, strings.Contains(page, `gokapiToggleTheme()`), true)
	test.IsEqualBool(t, strings.Contains(page, "./css/theme.css"), true)
	test.IsEqualBool(t, strings.Contains(page, "./js/theme.js"), true)
	test.IsEqualBool(t, strings.Contains(page, `data-bs-theme="dark"`), true)

	// Both languages are offered and the active one is marked
	test.IsEqualBool(t, strings.Contains(page, `href="./setLanguage?lang=en"`), true)
	test.IsEqualBool(t, strings.Contains(page, `href="./setLanguage?lang=zh-CN"`), true)
	test.IsEqualBool(t, strings.Contains(page, `class="gk-lang-link active"`), true)
	test.IsEqualBool(t, strings.Contains(page, "简体中文"), true)
}

// TestSetLanguageHandler verifies that selecting a language stores it in a cookie
func TestSetLanguageHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/setLanguage?lang=zh-CN", nil)
	setLanguage(recorder, request)

	cookies := recorder.Result().Cookies()
	test.IsEqualInt(t, len(cookies), 1)
	test.IsEqualString(t, cookies[0].Name, i18n.CookieName)
	test.IsEqualString(t, cookies[0].Value, "zh-CN")
	test.IsEqualInt(t, recorder.Code, http.StatusTemporaryRedirect)

	// An unsupported language falls back to the default language
	recorder = httptest.NewRecorder()
	setLanguage(recorder, httptest.NewRequest(http.MethodGet, "/setLanguage?lang=kl-GL", nil))
	test.IsEqualString(t, recorder.Result().Cookies()[0].Value, i18n.DefaultLanguage)
}

// TestLanguageRedirectTarget verifies that only same-host referrers are used as redirect target
func TestLanguageRedirectTarget(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/setLanguage?lang=en", nil)
	test.IsEqualString(t, languageRedirectTarget(request), "./admin")

	request.Header.Set("Referer", "http://evil.example/setLanguage")
	// Different host -> ignore the referrer
	test.IsEqualString(t, languageRedirectTarget(request), "./admin")

	request.Header.Set("Referer", "http://"+request.Host+"/logs?filter=all")
	test.IsEqualString(t, languageRedirectTarget(request), "/logs?filter=all")
}

// TestThemeAssetsAreServed makes sure the new static assets are embedded and reachable
func TestThemeAssetsAreServed(t *testing.T) {
	assets := map[string][]string{
		"http://localhost:53843/css/theme.css": {"--gk-radius-lg", "backdrop-filter", `data-bs-theme="light"`},
		"http://localhost:53843/js/theme.js":   {"gokapiToggleTheme", "gokapi-theme"},
	}
	for url, expected := range assets {
		response, err := http.Get(url)
		test.IsEqualBool(t, err == nil, true)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(response.Body)
		_ = response.Body.Close()
		test.IsEqualBool(t, readErr == nil, true)
		test.IsEqualInt(t, response.StatusCode, http.StatusOK)
		for _, snippet := range expected {
			if !strings.Contains(string(body), snippet) {
				t.Errorf("%s does not contain %q", url, snippet)
			}
		}
	}
}
