package vdr

// CI-safe unit tests for finding handler input validation.
// Tests the validation paths that return 400/422 before any database call.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestFindingHandler() *findingHandler {
	return &findingHandler{store: nil, pub: nil}
}

func TestFindingCreate_BadJSON(t *testing.T) {
	h := newTestFindingHandler()
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFindingCreate_InvalidTenantID(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"tenant_id":"INVALID","target":"nginx","scanner":"trivy","severity":"high","title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid tenant_id, got %d", w.Code)
	}
}

func TestFindingCreate_EmptyTarget(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"tenant_id":"acme-corp","target":"","scanner":"trivy","severity":"high","title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty target, got %d", w.Code)
	}
}

func TestFindingCreate_InvalidScanner(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"tenant_id":"acme-corp","target":"nginx","scanner":"unknown-scanner","severity":"high","title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid scanner, got %d", w.Code)
	}
}

func TestFindingCreate_InvalidSeverity(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"tenant_id":"acme-corp","target":"nginx","scanner":"trivy","severity":"extreme","title":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid severity, got %d", w.Code)
	}
}

func TestFindingCreate_EmptyTitle(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"tenant_id":"acme-corp","target":"nginx","scanner":"trivy","severity":"high","title":""}`
	req := httptest.NewRequest(http.MethodPost, "/findings", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty title, got %d", w.Code)
	}
}

func TestFindingUpdateStatus_InvalidStatus(t *testing.T) {
	h := newTestFindingHandler()
	body := `{"status":"invalid-status"}`
	req := httptest.NewRequest(http.MethodPatch, "/findings/abc", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.updateStatus(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for invalid status, got %d", w.Code)
	}
}

func TestFindingList_MissingTenantID(t *testing.T) {
	h := newTestFindingHandler()
	req := httptest.NewRequest(http.MethodGet, "/findings", nil)
	w := httptest.NewRecorder()
	h.list(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for missing tenant_id, got %d", w.Code)
	}
}
