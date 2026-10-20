package mock

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestThePortalTokensAreLinkedOnlyBehindThePortal(t *testing.T) {
	s := NewServer("test", "abc1234", "2026-01-01")
	s.SetBasePath("/wled")
	h := s.Handler()

	alone := httptest.NewRecorder()
	h.ServeHTTP(alone, httptest.NewRequest(http.MethodGet, "/wled/", http.NoBody))
	if strings.Contains(alone.Body.String(), portalTokensLink) {
		t.Error("the page links the portal's tokens without the portal")
	}

	req := httptest.NewRequest(http.MethodGet, "/wled/", http.NoBody)
	req.Header.Set("X-Forwarded-Prefix", "/wled")
	behind := httptest.NewRecorder()
	h.ServeHTTP(behind, req)
	body := behind.Body.String()
	if i := strings.Index(body, portalTokensLink); i < 0 || i < strings.Index(body, "</style>") {
		t.Error("behind the portal the page does not link its tokens after its own copy")
	}
}

func TestThePageSwitchesToTheCompactHeaderInsideThePortal(t *testing.T) {
	s := NewServer("test", "", "")
	page := generateDashboardHTML(s.state, s.effectNames, "{}", "", "test", "", "")
	head, _, _ := strings.Cut(page, "</head>")
	script := strings.Index(head, "classList.add('hr-embedded')")
	if script < 0 || script > strings.Index(head, "<style>") {
		t.Error("the hr-embedded script is missing or runs after the styles")
	}
	if !strings.Contains(head, ".hr-embedded .header-bar {") {
		t.Error("no compact header styles")
	}
}
