package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestForwardCopiesJSON(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/forms/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"id":123}`)
	}))
	defer upstream.Close()

	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://localhost/api/forms/123", nil)

	Forward(recorder, req, upstream.Client(), upstream.URL+"/api/forms", "/123")

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if got := recorder.Body.String(); got != "{\"id\":123}" {
		t.Fatalf("unexpected body: %s", got)
	}
	if ct := recorder.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content type: %s", ct)
	}
}
