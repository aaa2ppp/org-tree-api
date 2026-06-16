package api_test

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"org-tree-api/internal/api"
	"org-tree-api/internal/model"

	"github.com/aaa2ppp/be"
)

type stubStorage struct {
	gotRequest any
}

// CreateDepartment implements api.Service.
func (s *stubStorage) CreateDepartment(ctx context.Context, req model.Department) (model.Department, error) {
	s.gotRequest = req
	return model.Department{}, nil
}

// CreateEmployee implements api.Service.
func (s *stubStorage) CreateEmployee(ctx context.Context, req model.Employee) (model.Employee, error) {
	s.gotRequest = req
	return model.Employee{}, nil
}

// DeleteDepartment implements api.Service.
func (s *stubStorage) DeleteDepartment(ctx context.Context, req model.DeleteDepartmentRequest) error {
	s.gotRequest = req
	return nil
}

// GetDepartmentTree implements api.Service.
func (s *stubStorage) GetDepartmentTree(ctx context.Context, req model.GetDepartmentsTreeRequest) (*model.DepartmentNode, error) {
	s.gotRequest = req
	return &model.DepartmentNode{Department: &model.Department{}}, nil
}

// MoveDepartment implements api.Service.
func (s *stubStorage) MoveDepartment(ctx context.Context, req model.MoveDepartmentRequest) (model.Department, error) {
	s.gotRequest = req
	return model.Department{}, nil
}

var _ api.Service = &stubStorage{}

func date(y, m, d int) model.Date {
	return model.Date{
		NullTime: sql.NullTime{
			Time:  time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC),
			Valid: true},
	}
}

func TestParseInput(t *testing.T) {
	tests := []struct {
		method         string
		url            string
		contentType    string
		body           string
		wantStatusCode int
		wantRequest    any
	}{
		// ---------- CreateDepartment ----------
		{
			"POST", "/departments", "application/json",
			`{"name":"dept","parent_id":10}`,
			201,
			model.Department{Name: "dept", ParentID: 10},
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"top"}`,
			201,
			model.Department{Name: "top", ParentID: model.VirtualRoot},
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"top","parent_id":null}`,
			201,
			model.Department{Name: "top", ParentID: model.VirtualRoot},
		},
		{
			"POST", "/departments", "application/json",
			`{"parent_id":10}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"","parent_id":10}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"` + strings.Repeat("a", 201) + `"}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"dept","parent_id":0}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"dept","parent_id":-1}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"dept","parent_id":-5}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`{"name":"dept","parent_id":9999999999}`,
			400, nil,
		},
		{
			"POST", "/departments", "application/json",
			`invalid json`,
			400, nil,
		},
		{
			"POST", "/departments", "text/plain",
			`{"name":"dept"}`,
			400, nil,
		},
		{
			"POST", "/departments", "",
			`{"name":"dept"}`,
			400, nil,
		},

		// ---------- CreateEmployee ----------
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван Иванов","position":"менеджер"}`,
			201,
			model.Employee{DepartmentID: 5, FullName: "Иван Иванов", Position: "менеджер", HiredAt: model.Date{}},
		},
		{
			"POST", "/departments/10/employees", "application/json",
			`{"full_name":"Петр Петров","position":"разработчик","hired_at":"2025-01-15"}`,
			201,
			model.Employee{DepartmentID: 10, FullName: "Петр Петров", Position: "разработчик", HiredAt: date(2025, 1, 15)},
		},
		{
			"POST", "/departments/0/employees", "application/json",
			`{"full_name":"ФИО","position":"должность"}`,
			400, nil,
		},
		{
			"POST", "/departments/abc/employees", "application/json",
			`{"full_name":"ФИО","position":"должность"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"","position":"должность"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"` + strings.Repeat("b", 201) + `","position":"должность"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван","position":""}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван","position":"` + strings.Repeat("c", 201) + `"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван","position":"менеджер","hired_at":"bad-date"}`,
			400, nil,
		},
		{
			"POST", "/departments/5/employees", "application/json",
			`{"full_name":"Иван","position":"менеджер","hired_at":null}`,
			201,
			model.Employee{DepartmentID: 5, FullName: "Иван", Position: "менеджер", HiredAt: model.Date{}},
		},

		// ---------- GetDepartmentTree ----------
		{
			"GET", "/departments/5?depth=2&include_employees=false&sort_by=id", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: 5, Depth: 2, IncludeEmployees: false},
		},
		{
			"GET", "/departments/5?depth=2&include_employees=false&sort_by=name", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: 5, Depth: 2, IncludeEmployees: false, SortBy: model.SortByName},
		},
		{
			"GET", "/departments/5?depth=2&include_employees=false&sort_by=created_at", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: 5, Depth: 2, IncludeEmployees: false, SortBy: model.SortByCreatedAt},
		},
		{
			"GET", "/departments/5?depth=0&include_employees=true", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: 5, Depth: 0, IncludeEmployees: true},
		},
		{
			"GET", "/departments/5?depth=5", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: 5, Depth: 5, IncludeEmployees: true},
		},
		{
			"GET", "/departments/5?depth=-1", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments/5?depth=6", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments/5?sort_by=invalid", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments/0", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments/abc", "",
			"",
			400, nil,
		},

		// ---------- GetTopDepartments ----------
		{
			"GET", "/departments?depth=2&include_employees=false&sort_by=name", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: -1, Depth: 2, IncludeEmployees: false, SortBy: model.SortByName},
		},
		{
			"GET", "/departments?depth=2&include_employees=false&sort_by=created_at", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: -1, Depth: 2, IncludeEmployees: false, SortBy: model.SortByCreatedAt},
		},
		{
			"GET", "/departments?depth=5", "",
			"",
			200,
			model.GetDepartmentsTreeRequest{ID: -1, Depth: 5, IncludeEmployees: true},
		},
		{
			"GET", "/departments?depth=0", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments?depth=-1", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments?depth=6", "",
			"",
			400, nil,
		},
		{
			"GET", "/departments?sort_by=invalid", "",
			"",
			400, nil,
		},

		// ---------- MoveDepartment ----------
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name"}`,
			200,
			model.MoveDepartmentRequest{ID: 10, Name: "new name"},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"parent_id":20}`,
			200,
			model.MoveDepartmentRequest{ID: 10, ParentID: 20},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":null,"parent_id":20}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"parent_id":null}`,
			200,
			model.MoveDepartmentRequest{ID: 10, ParentID: model.VirtualRoot},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":null}`,
			200,
			model.MoveDepartmentRequest{ID: 10, Name: "new name", ParentID: model.VirtualRoot},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name"}`,
			200,
			model.MoveDepartmentRequest{ID: 10, Name: "new name"},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":20}`,
			200,
			model.MoveDepartmentRequest{ID: 10, Name: "new name", ParentID: 20},
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"","parent_id":20}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"` + strings.Repeat("a", 201) + `"}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":0}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":-1}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":-3}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":9999999999}`,
			400, nil,
		},
		{
			"PATCH", "/departments/10", "application/json",
			`{"name":"new name","parent_id":"string"}`,
			400, nil,
		},
		{
			"PATCH", "/departments/0", "application/json",
			`{"name":"x"}`,
			400, nil,
		},

		// ---------- DeleteDepartment ----------
		{
			"DELETE", "/departments/5?mode=cascade", "",
			"",
			204, model.DeleteDepartmentRequest{ID: 5, Mode: model.DeleteModeCascade},
		},
		{
			"DELETE", "/departments/5?mode=reassign&reassign_to_department_id=10", "",
			"",
			204, model.DeleteDepartmentRequest{ID: 5, Mode: model.DeleteModeReassign, ReassignToDepartmentID: 10},
		},
		{
			"DELETE", "/departments/5?mode=reassign", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/5?mode=reassign&reassign_to_department_id=0", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/5?mode=reassign&reassign_to_department_id=-1", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/5?mode=reassign&reassign_to_department_id=9999999999", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/5?mode=unknown", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/5", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/0?mode=cascade", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/-1?mode=cascade", "",
			"",
			400, nil,
		},
		{
			"DELETE", "/departments/abc?mode=cascade", "",
			"",
			400, nil,
		},
	}
	for _, tt := range tests {
		ok := t.Run(tt.url, func(t *testing.T) {
			srv := &stubStorage{}
			api := api.New(srv)

			ts := httptest.NewServer(api)
			defer ts.Close()

			req, err := http.NewRequest(tt.method, ts.URL+tt.url, strings.NewReader(tt.body))
			be.Err(t, err, nil)
			if tt.contentType != "" {
				req.Header.Set("content-type", tt.contentType)
			}

			resp, err := http.DefaultClient.Do(req)
			be.Err(t, err, nil)
			defer resp.Body.Close()

			if resp.StatusCode != tt.wantStatusCode {
				body, _ := io.ReadAll(resp.Body)
				t.Errorf("got: code:%d msg:%q, want code:%d", resp.StatusCode, string(body), tt.wantStatusCode)
			}
			if v := resp.StatusCode; 200 <= v && v < 300 {
				// Запрос дошел до сервиса - проверяем, что он правильно сформирован
				be.Equal(t, srv.gotRequest, tt.wantRequest)
			}
		})
		if !ok {
			t.Logf("%s:%s:%s:%s", tt.method, tt.url, tt.contentType, tt.body)
		}
	}
}
