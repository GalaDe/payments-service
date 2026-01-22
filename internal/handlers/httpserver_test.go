package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

// minimal constructor for tests
func newTestServer() *HttpServer {
	return NewHttpServer(zap.NewNop(), client.Client(nil), nil, nil, nil)
}

func readJSONBody(t *testing.T, rr *httptest.ResponseRecorder) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(rr.Body.Bytes(), &v); err != nil {
		t.Fatalf("response body is not valid JSON: %v\nbody=%q", err, rr.Body.String())
	}
	return v
}

func TestRespondWithJSON_SetsHeadersStatusAndBody(t *testing.T) {
	srv := newTestServer()

	tests := []struct {
		name       string
		status     int
		payload    any
		wantBody   any // compare after JSON unmarshal
	}{
		{
			name:   "map payload",
			status: http.StatusCreated,
			payload: map[string]any{
				"id":   "ch_123",
				"ok":   true,
				"amt":  12500,
				"meta": map[string]any{"k": "v"},
			},
			wantBody: map[string]any{
				"id":   "ch_123",
				"ok":   true,
				"amt":  float64(12500), // numbers unmarshal as float64
				"meta": map[string]any{"k": "v"},
			},
		},
		{
			name:   "struct payload",
			status: http.StatusOK,
			payload: struct {
				Msg string `json:"msg"`
				N   int    `json:"n"`
			}{Msg: "hello", N: 7},
			wantBody: map[string]any{"msg": "hello", "n": float64(7)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			srv.respondWithJSON(rr, tc.status, tc.payload)

			// status
			if rr.Code != tc.status {
				t.Fatalf("status = %d, want %d", rr.Code, tc.status)
			}
			// header
			if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
			}
			// body
			got := readJSONBody(t, rr)
			if !reflect.DeepEqual(got, tc.wantBody) {
				t.Fatalf("body = %#v, want %#v", got, tc.wantBody)
			}
			_ = req // req unused but kept for pattern symmetry
		})
	}
}

func TestRespondWithError_WrapsErrorInJSON(t *testing.T) {
	srv := newTestServer()

	rr := httptest.NewRecorder()
	_ = httptest.NewRequest(http.MethodGet, "/", nil)

	srv.respondWithError(rr, http.StatusBadRequest, "invalid input")

	// status
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	// header
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want %q", ct, "application/json")
	}
	// body
	got := readJSONBody(t, rr)
	want := map[string]any{"error": "invalid input"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}
