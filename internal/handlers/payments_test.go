package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GalaDe/payments-service/internal/domain"
	"github.com/gorilla/mux"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

/* ---------- fakes & helpers ---------- */

type fakeRepo struct {
	getByIDFn    func(ctx context.Context, id string) (domain.Payment, error)
	getAllFn     func(ctx context.Context) ([]domain.Payment, error)
}

func (f *fakeRepo) GetPaymentByID(ctx context.Context, id string) (domain.Payment, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn(ctx, id)
	}
	return domain.Payment{}, sql.ErrNoRows
}

func (f *fakeRepo) GetAllPayments(ctx context.Context) ([]domain.Payment, error) {
	if f.getAllFn != nil {
		return f.getAllFn(ctx)
	}
	return nil, nil
}

func newServer(repo domain.Repository) *HttpServer {
	return NewHttpServer(zap.NewNop(), client.Client(nil), repo, nil, nil)
}

func doJSON(t *testing.T, method, url string, body any, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, url, &buf)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

/* ---------- CreatePayment tests (validation paths) ---------- */

func TestCreatePayment_BadJSON(t *testing.T) {
	repo := mocks.NewRepository(t)
	srv := newServer(&MockRepository{})

	req := httptest.NewRequest(http.MethodPost, "/payments", bytes.NewBufferString("{bad json"))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	srv.CreatePayment(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if got := rr.Body.String(); got == "" {
		t.Fatalf("expected error body, got empty")
	}
}

func TestCreatePayment_MissingFields(t *testing.T) {
	srv := newServer(&fakeRepo{})

	// Missing customer_id, payment_method_id, amount <= 0, or currency
	body := map[string]any{
		"user_id": "u1",
		// "customer_id": "c1",
		// "payment_method_id": "pm1",
		"amount":   0,
		"currency": "",
	}
	rr := doJSON(t, http.MethodPost, "/payments", body, srv.CreatePayment)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	if got := rr.Body.String(); got == "" {
		t.Fatalf("expected error body, got empty")
	}
}

/*
NOTE: We’re not testing the happy path of CreatePayment here to avoid mocking the full
Temporal client.Client interface. If you want that, I can add a lightweight mock that
implements ExecuteWorkflow and returns a stub WorkflowRun, or we can wrap client.Client
behind a smaller interface in your production code for easier testing.
*/

/* ---------- GetPaymentByID ---------- */

func TestGetPaymentByID_Success(t *testing.T) {
	// Provide a minimal payment (fields depend on your domain.Payment).
	// Using zero value is OK; we only assert 200 and JSON.
	repo := &fakeRepo{
		getByIDFn: func(ctx context.Context, id string) (domain.Payment, error) {
			var p domain.Payment
			// If your domain.Payment has ID field, set it (optional):
			// p.ID = id
			return p, nil
		},
	}
	srv := newServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/payments/abc", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "abc"})
	rr := httptest.NewRecorder()

	srv.GetPaymentByID(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want %q", ct, "application/json")
	}
	if rr.Body.Len() == 0 {
		t.Fatalf("expected JSON body, got empty")
	}
}

func TestGetPaymentByID_MissingID(t *testing.T) {
	srv := newServer(&fakeRepo{})
	req := httptest.NewRequest(http.MethodGet, "/payments/", nil)
	// no mux vars set
	rr := httptest.NewRecorder()

	srv.GetPaymentByID(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestGetPaymentByID_NotFound(t *testing.T) {
	repo := &fakeRepo{
		getByIDFn: func(ctx context.Context, id string) (domain.Payment, error) {
			return domain.Payment{}, sql.ErrNoRows
		},
	}
	srv := newServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/payments/xyz", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	rr := httptest.NewRecorder()

	srv.GetPaymentByID(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestGetPaymentByID_RepoError(t *testing.T) {
	repo := &fakeRepo{
		getByIDFn: func(ctx context.Context, id string) (domain.Payment, error) {
			return domain.Payment{}, errors.New("db boom")
		},
	}
	srv := newServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/payments/xyz", nil)
	req = mux.SetURLVars(req, map[string]string{"id": "xyz"})
	rr := httptest.NewRecorder()

	srv.GetPaymentByID(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}

/* ---------- GetPayments ---------- */

func TestGetPayments_Success(t *testing.T) {
	repo := &fakeRepo{
		getAllFn: func(ctx context.Context) ([]domain.Payment, error) {
			return []domain.Payment{{}, {}}, nil
		},
	}
	srv := newServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/payments", nil)
	rr := httptest.NewRecorder()

	srv.GetPayments(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q, want %q", ct, "application/json")
	}
	// Should be a JSON array
	var arr []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &arr); err != nil {
		t.Fatalf("invalid JSON array: %v; body=%q", err, rr.Body.String())
	}
	if len(arr) != 2 {
		t.Fatalf("len = %d, want %d", len(arr), 2)
	}
}

func TestGetPayments_RepoError(t *testing.T) {
	repo := &fakeRepo{
		getAllFn: func(ctx context.Context) ([]domain.Payment, error) {
			return nil, errors.New("db unavailable")
		},
	}
	srv := newServer(repo)

	req := httptest.NewRequest(http.MethodGet, "/payments", nil)
	rr := httptest.NewRecorder()

	srv.GetPayments(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
}
