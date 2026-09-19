package admin

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerServesConsole(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/admin/index.html", nil)
	rec := httptest.NewRecorder()

	Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "NanoMDM Console") {
		t.Fatal("console index was not served")
	}
}
