package api

import (
	"bytes"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aaa2ppp/be"

	"org-tree-api/internal/lib/logger"
	"org-tree-api/internal/model"
)

func Test_writeError(t *testing.T) {

	tests := []struct {
		name           string
		err            error
		wantStatusCode int
	}{
		{
			"not found",
			model.ErrNotFound,
			http.StatusNotFound,
		},
		{
			"validation error",
			model.ErrValidation,
			http.StatusBadRequest,
		},
		{
			"conflict",
			model.ErrConflict,
			http.StatusConflict,
		},
		{
			"internal",
			model.ErrInternal,
			http.StatusInternalServerError,
		},
		{
			"not implemented",
			model.ErrNotImplemented,
			http.StatusNotImplemented,
		},
		{
			"unknown error",
			errors.New("unknown error"),
			http.StatusInternalServerError,
		},
		{
			"nil error",
			nil,
			http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			r, _ := http.NewRequest("GET", "/", nil)
			r = r.WithContext(logger.Context(r.Context(), slog.New(slog.DiscardHandler)))

			h := newHelper(w, r, "Test_writeError")
			h.writeError(tt.err)

			gotStatusCode := w.Result().StatusCode
			be.Equal(t, w.Result().StatusCode, tt.wantStatusCode)

			if gotStatusCode >= 500 && tt.err != nil && !errors.Is(tt.err, model.ErrInternal) && !errors.Is(tt.err, model.ErrNotImplemented) {
				body, _ := io.ReadAll(w.Body)
				be.True(t, !bytes.Contains(body, []byte(tt.err.Error())))
			}
		})
	}
}

func Test_getDepartmentID(t *testing.T) {
	tests := []struct {
		url     string
		wantID  int
		wantErr reflect.Type
	}{
		{
			"/departments/5",
			5,
			nil,
		},
		{
			"/departments/5/employees",
			5,
			nil,
		},
		{
			"/departments/abc/employees",
			0,
			reflect.TypeOf(&httpError{}),
		},
		{
			"/departments/0/employees",
			0,
			reflect.TypeOf(&httpError{}),
		},
		{
			"/departments/-1/employees",
			0,
			reflect.TypeOf(&httpError{}),
		},
		{
			"/departments/9999999999/employees",
			0,
			reflect.TypeOf(&httpError{}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			var gotID int
			var gotErr error

			mux := http.NewServeMux()
			mux.HandleFunc("GET /departments/{department_id}/", func(w http.ResponseWriter, r *http.Request) {
				h := newHelper(w, r, "test")
				gotID, gotErr = h.getDepartmentID()
			})

			ts := httptest.NewServer(mux)
			defer ts.Close()

			httpReq, err := http.NewRequest("GET", ts.URL+tt.url, nil)
			be.Err(t, err, nil)

			httpResp, err := http.DefaultClient.Do(httpReq)
			be.Err(t, err, nil)
			defer httpResp.Body.Close()

			be.Equal(t, gotID, tt.wantID)
			be.Err(t, gotErr, tt.wantErr)
			if tt.wantErr != nil {
				be.Equal(t, gotErr.(*httpError).Code, 400)
			}
		})
	}
}
