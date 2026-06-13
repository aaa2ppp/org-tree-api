package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"reflect"
	"strconv"

	"org-tree-api/internal/lib/getenv"
	"org-tree-api/internal/lib/logger"
	"org-tree-api/internal/model"
)

const (
	required = true
	optional = false
)

const applicationJSON = "application/json"

type helper struct {
	w    http.ResponseWriter
	r    *http.Request
	op   string
	_ctx context.Context
	_log *slog.Logger
	_qry *getenv.Getenv
}

func newHelper(w http.ResponseWriter, r *http.Request, op string) *helper {
	return &helper{
		w:  w,
		r:  r,
		op: op,
	}
}

func (h *helper) ctx() context.Context {
	if h._ctx == nil {
		h._ctx = h.r.Context()
	}
	return h._ctx
}

func (h *helper) log() *slog.Logger {
	if h._log == nil {
		h._log = logger.FromContext(h.ctx()).With("op", h.op)
	}
	return h._log
}

func (h *helper) writeError(err error) {
	if err == nil { // хм?..
		h.log().Error("writeError called with nil error")
		h.writeHTTPError(&httpError{"internal error", http.StatusInternalServerError})
		return
	}
	if httpErr := mapError(err); httpErr != nil {
		h.writeHTTPError(httpErr)
		return
	}
	h.log().Warn("unmapped error", "type", reflect.TypeOf(err).String(), "error", err)
	h.writeHTTPError(&httpError{"internal error", http.StatusInternalServerError})
}

func (h *helper) writeHTTPError(err *httpError) {
	http.Error(h.w, strconv.Itoa(err.Code)+" "+err.Msg, err.Code)
}

func (h *helper) checkContentType(wantType string) error {
	ct, _, _ := mime.ParseMediaType(h.r.Header.Get("Content-Type"))
	if ct != wantType {
		return &httpError{"want " + wantType, http.StatusBadRequest}
	}
	return nil
}

func (h *helper) decodeRequestBody(req any) error {
	if err := h.checkContentType(applicationJSON); err != nil {
		return err
	}
	d := json.NewDecoder(h.r.Body)
	d.UseNumber() // удобно, если декодируем в мапу
	if err := d.Decode(req); err != nil {
		return &httpError{"bad json: " + err.Error(), http.StatusBadRequest}
	}
	return nil
}

type validator interface {
	validate() error
}

func (h *helper) decodeAndValidateRequestBody(req validator) error {
	if err := h.decodeRequestBody(req); err != nil {
		return err
	}
	if err := req.validate(); err != nil {
		return &httpError{err.Error(), http.StatusBadRequest}
	}
	return nil
}

func (h *helper) writeResponse(statusCode int, resp any) error {
	h.w.Header().Set("Content-Type", applicationJSON)
	h.w.WriteHeader(statusCode)

	if err := json.NewEncoder(h.w).Encode(resp); err != nil {
		h.log().Error("write response", "error", err)
		return err
	}
	return nil
}

func (h *helper) getDepartmentID() (int, error) {
	s := h.r.PathValue("department_id")
	if s == "" {
		return 0, &httpError{"department_id cannot be empty", http.StatusBadRequest}
	}
	id, err := strconv.Atoi(s)
	if err != nil || !model.ValidID(id) {
		return 0, &httpError{"invalid department_id", http.StatusBadRequest}
	}
	return id, nil
}

func (h *helper) query() *getenv.Getenv {
	if h._qry == nil {
		q := h.r.URL.Query()
		h._qry = getenv.New(func(key string) (string, bool) {
			if q.Has(key) {
				return q.Get(key), true
			}
			return "", false
		})
	}
	return h._qry
}
