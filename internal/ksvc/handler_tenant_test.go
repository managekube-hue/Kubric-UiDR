package ksvc

// CI-safe unit tests for tenant handler input validation.
// These tests exercise the validation code paths that return 400/422 before
// any database call is made.  No Postgres or NATS connection is required.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestHandler returns a tenantHandler with nil store and pub.
// This is only safe for tests that trigger validation errors before any
// store method is called.
func newTestHandler() *tenantHandler {
	return &tenantHandler{store: nil, pub: nil}
}

func TestCreate_BadJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPost, "/tenants", strings.NewReader("not-json"))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_EmptyTenantID(t *testing.T) {
	h := newTestHandler()
	body := `{"tenant_id":"","name":"Acme","plan":"starter"}`
	req := httptest.NewRequest(http.MethodPost, "/tenants", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty tenant_id, got %d", w.Code)
	}
}

func TestCreate_InvalidTenantID(t *testing.T) {
	invalidIDs := []string{"UPPERCASE", "has space", "has_underscore", "-bad"}
	for _, id := range invalidIDs {
		h := newTestHandler()
		body := `{"tenant_id":"` + id + `","name":"Acme","plan":"starter"}`
		req := httptest.NewRequest(http.MethodPost, "/tenants", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.create(w, req)
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("tenant_id=%q: expected 422, got %d", id, w.Code)
		}
	}
}

func TestCreate_EmptyName(t *testing.T) {
	h := newTestHandler()
	body := `{"tenant_id":"acme-corp","name":"","plan":"starter"}`
	req := httptest.NewRequest(http.MethodPost, "/tenants", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.create(w, req)
	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("expected 422 for empty name, got %d", w.Code)
	}
}

func TestUpdate_BadJSON(t *testing.T) {
	h := newTestHandler()
	req := httptest.NewRequest(http.MethodPatch, "/tenants/acme-corp", strings.NewReader("{{bad"))
	w := httptest.NewRecorder()
	h.update(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
