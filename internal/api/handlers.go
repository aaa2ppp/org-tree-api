package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"org-tree-api/internal/model"
)

type Service interface {
	CreateDepartment(ctx context.Context, req model.Department) (model.Department, error)
	CreateEmployee(ctx context.Context, req model.Employee) (model.Employee, error)
	GetDepartmentTree(ctx context.Context, req model.GetDepartmentsTreeRequest) (*model.DepartmentNode, error)
	MoveDepartment(ctx context.Context, req model.MoveDepartmentRequest) (model.Department, error)
	DeleteDepartment(ctx context.Context, req model.DeleteDepartmentRequest) error
}

func New(s Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /departments", CreateDepartment(s))
	mux.HandleFunc("POST /departments/{department_id}/employees", CreateEmployee(s))
	mux.HandleFunc("GET /departments", GetTopDepartments(s))
	mux.HandleFunc("GET /departments/{department_id}", GetDepartmentTree(s))
	mux.HandleFunc("PATCH /departments/{department_id}", MoveDepartment(s))
	mux.HandleFunc("DELETE /departments/{department_id}", DeleteDepartment(s))
	return mux
}

// CreateDepartment godoc
//
//	@tags			department
//	@router			/departments [post]
//	@summary		Создать подразделение
//	@description	**Body:**
//	@description	- name: str
//	@description	- parent_id: int | null (опционально)
//	@description
//	@description	**Response:** созданное подразделение
//	@description
//	@accept		json
//	@produce	json
//	@param		req	body		CreateDepartmentRequest	true	"CreateDepartmentRequest"
//	@success	201	{object}	model.Department
//	@failure	400
//	@failure	409
//	@failure	500
func CreateDepartment(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.CreateDepartment")

		var req CreateDepartmentRequest
		if err := h.decodeAndValidateRequestBody(&req); err != nil {
			h.writeError(err)
			return
		}

		department, err := s.CreateDepartment(h.ctx(), model.Department{
			Name:     req.Name,
			ParentID: req.ParentID.Value,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		if department.ParentID == model.VirtualRoot {
			// hide virtual root id
			department.ParentID = 0
		}
		h.writeResponse(http.StatusCreated, department)
	}
}

type CreateDepartmentRequest struct {
	Name     string        `json:"name" validate:"required" example:"IT отдел" minLength:"1" maxLength:"200"`
	ParentID Nullable[int] `json:"parent_id" swaggertype:"integer" example:"1" minimum:"1"`
}

func (req *CreateDepartmentRequest) validate() error {
	var errs []error

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > model.MaxNameLength {
		errs = append(errs, errors.New("invalid name"))
	}

	if !req.ParentID.Valid {
		// null or undefined
		req.ParentID.Value = model.VirtualRoot
	} else {
		if !model.ValidID(req.ParentID.Value) {
			errs = append(errs, errors.New("invalid parent_id"))
		}
	}

	return errors.Join(errs...)
}

// CreateEmployee godoc
//
//	@tags			department
//	@router			/departments/{department_id}/employees [post]
//	@summary		Создать сотрудника в подразделении
//	@description	**Body:**
//	@description	- full_name: str
//	@description	- position: str
//	@description	- hired_at: date | null (опционально)
//	@description
//	@description	**Response:** созданный сотрудник
//	@description
//	@accept		json
//	@produce	json
//	@param		department_id	path		int						true	"ID подразделения"	minimum(1)
//	@param		req				body		CreateEmployeeRequest	true	"CreateEmployeeRequest"
//	@success	201				{object}	model.Employee
//	@failure	400
//	@failure	404
//	@failure	409
//	@failure	500
func CreateEmployee(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.CreateEmployee")

		departmentID, err := h.getDepartmentID()
		if err != nil {
			h.writeError(err)
			return
		}

		var req CreateEmployeeRequest
		if err := h.decodeAndValidateRequestBody(&req); err != nil {
			h.writeError(err)
			return
		}

		employee, err := s.CreateEmployee(h.ctx(), model.Employee{
			DepartmentID: departmentID,
			FullName:     req.FullName,
			Position:     req.Position,
			HiredAt:      req.HiredAt,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(http.StatusCreated, employee)
	}
}

type CreateEmployeeRequest struct {
	FullName string     `json:"full_name" validate:"required" example:"Василий Иванович Пупкин" minLength:"1" maxLength:"200"`
	Position string     `json:"position" validate:"required" example:"Программист" minLength:"1" maxLength:"200"`
	HiredAt  model.Date `json:"hired_at" swaggertype:"string" format:"date" example:"2026-05-30"`
}

func (req *CreateEmployeeRequest) validate() error {
	var errs []error

	req.FullName = strings.TrimSpace(req.FullName)
	if req.FullName == "" || len(req.FullName) > model.MaxFullNameLength {
		errs = append(errs, errors.New("invalid full_name"))
	}

	req.Position = strings.TrimSpace(req.Position)
	if req.Position == "" || len(req.Position) > model.MaxPositionLength {
		errs = append(errs, errors.New("invalid position"))
	}

	return errors.Join(errs...)
}

// GetDepartmentTree godoc
//
//	@tags			department
//	@router			/departments/{department_id} [get]
//	@summary		Получить подразделение (детали + сотрудники + поддерево)
//	@description	**Query:**
//	@description	- depth: int (по умолчанию 1, максимум 5) — глубина *вложенных* подразделений в ответе
//	@description	- include_employees: bool (по умолчанию true)
//	@description
//	@description	**Response:**
//	@description	- department (объект подразделения)
//	@description	- employees: [] (если include_employees=true, сортировка по created_at или full_name)
//	@description	- children: [] (*вложенные* подразделения до depth, рекурсивно)
//	@description	- sort_by: id | name | created_at (задает поле сортировки дочерних отделов и сотрудников, по умолчанию id; если sort_by=name, сотрудники сортируются по full_name)
//	@description
//	@description	*Глубина дерева определяется как **количество ребер** на самом длинном пути от корня до листового узла.*
//	@description	При значении depth=0 будет возвращено подразделение без потомков.
//	@produce		json
//	@param			department_id		path		int		true	"ID подразделения"							minimum(1)
//	@param			depth				query		int		false	"Глубина вложенных подразделений в ответе"	minimum(0),maximum(5),default(1)
//	@param			include_employees	query		bool	false	"Возвращать сотрудников подразделения"		default(true)
//	@param			sort_by				query		string	false	"Порядок сортировки"						enums(id,name,created_at),default(id)
//	@success		200					{object}	model.DepartmentNode
//	@failure		400
//	@failure		404
//	@failure		500
func GetDepartmentTree(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.GetDepartmentTree")

		departmentID, err := h.getDepartmentID()
		if err != nil {
			h.writeError(err)
			return
		}

		req, err := getDepartmentParams(h)
		if err != nil {
			h.writeError(&httpError{err.Error(), http.StatusBadRequest})
			return
		}

		if req.depth < 0 || req.depth > model.MaxDepth {
			h.writeError(&httpError{"invalid depth", http.StatusBadRequest})
			return
		}

		tree, err := s.GetDepartmentTree(h.ctx(), model.GetDepartmentsTreeRequest{
			ID:               departmentID,
			Depth:            req.depth,
			IncludeEmployees: req.includeEmployees,
			SortBy:           req.sortBy,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		if tree.Department.ParentID == model.VirtualRoot {
			// hide virtual root id
			tree.Department.ParentID = 0
		}
		h.writeResponse(http.StatusOK, tree)
	}
}

type departmentParams struct {
	depth            int
	includeEmployees bool
	sortBy           model.SortBy
}

func getDepartmentParams(h *helper) (departmentParams, error) {
	q := h.query()

	depth := q.Int("depth", optional, 1)
	includeEmployees := q.Bool("include_employees", optional, true)
	sortByStr := q.String("sort_by", optional, "id")

	if err := q.Err(); err != nil {
		return departmentParams{}, err
	}

	sortBy := model.SortByID
	if sortByStr != "" {
		var err error
		if sortBy, err = model.SortByString(sortByStr); err != nil {
			return departmentParams{}, errors.New("invalid sort_by")
		}
	}

	return departmentParams{
		depth:            depth,
		includeEmployees: includeEmployees,
		sortBy:           sortBy,
	}, nil
}

// GetTopDepartments godoc
//
//	@tags			department
//	@router			/departments [get]
//	@summary		Получить список подразделений верхнего уровня (детали + сотрудники + поддерево)
//	@description	**Query:**
//	@description	- depth: int (по умолчанию 1, максимум 5) — глубина *вложенных* подразделений в ответе (от виртуального корня)
//	@description	- include_employees: bool (по умолчанию true)
//	@description
//	@description	**Response:**
//	@description	- department (объект подразделения)
//	@description	- employees: [] (если include_employees=true, сортировка по created_at или full_name)
//	@description	- children: [] (*вложенные* подразделения до depth, рекурсивно)
//	@description	- sort_by: id | name | created_at (задает поле сортировки отделов и сотрудников, по умолчанию id; если sort_by=name, сотрудники сортируются по full_name)
//	@description
//	@description	*Глубина дерева определяется как **количество ребер** на самом длинном пути от корня до листового узла.*
//	@description	Предполагается, что подразделения верхнего уровня это дети виртуального корня.
//	@description	При значении depth=1 будет возвращен список подразделений верхнего уровня без потомков.
//	@produce		json
//	@param			depth				query	int		false	"Глубина вложенных подразделений в ответе"	minimum(1),maximum(5),default(1)
//	@param			include_employees	query	bool	false	"Возвращать сотрудников подразделения"		default(true)
//	@param			sort_by				query	string	false	"Порядок сортировки"						enums(id,name,created_at),defalut(id)
//	@success		200					{array}	model.DepartmentNode
//	@failure		400
//	@failure		404
//	@failure		500
func GetTopDepartments(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.GetTopDepartments")

		req, err := getDepartmentParams(h)
		if err != nil {
			h.writeError(&httpError{err.Error(), http.StatusBadRequest})
			return
		}

		if req.depth < 1 || req.depth > model.MaxDepth {
			h.writeError(&httpError{"invalid depth", http.StatusBadRequest})
			return
		}

		tree, err := s.GetDepartmentTree(h.ctx(), model.GetDepartmentsTreeRequest{
			ID:               model.VirtualRoot,
			Depth:            req.depth,
			IncludeEmployees: req.includeEmployees,
			SortBy:           req.sortBy,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		// hide virtual root id
		tops := tree.Children
		for i := range tops {
			tops[i].Department.ParentID = 0
		}
		// replace nil to empty slice
		if tops == nil {
			tops = []*model.DepartmentNode{}
		}
		h.writeResponse(http.StatusOK, tops)
	}
}

// MoveDepartment godoc
//
//	@tags			department
//	@router			/departments/{department_id} [patch]
//	@summary		Переместить подразделение в другое (изменить parent)
//	@description	**Body:**
//	@description	- name: str (опционально)
//	@description	- parent_id: int | null (опционально)
//	@description
//	@description	**Response:** обновлённое подразделение
//	@description
//	@description	Должен быть задан по крайней мере один из параметров name или parent_id
//	@description	Нельзя сделать подразделение родителем самого себя.
//	@description	Нельзя создать цикл в дереве (например, переместить департамент внутрь своего поддерева).
//	@description	В этом случае возвращает 409 Conflict.
//	@description
//	@description	Если параметр parent_id отсутствует, то подразделение только переименовывается без перемещения.
//	@description	Если parent_id=null, то подразделение перемещается на верхний уровень.
//	@accept			json
//	@produce		json
//	@param			department_id	path		int						true	"ID подразделения"	minimum(1)
//	@param			req				body		MoveDepartmentRequest	true	"MoveDepartmentRequest"
//	@success		200				{object}	model.Department
//	@failure		400
//	@failure		404
//	@failure		409
//	@failure		500
func MoveDepartment(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.MoveDepartment")

		departmentID, err := h.getDepartmentID()
		if err != nil {
			h.writeError(err)
			return
		}

		var req MoveDepartmentRequest
		if err := h.decodeAndValidateRequestBody(&req); err != nil {
			h.writeError(err)
			return
		}

		department, err := s.MoveDepartment(h.ctx(), model.MoveDepartmentRequest{
			ID:       departmentID,
			Name:     req.Name.Value,
			ParentID: req.ParentID.Value,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		h.writeResponse(http.StatusOK, department)
	}
}

type MoveDepartmentRequest struct {
	Name     Nullable[string] `json:"name" swaggertype:"string" example:"IT отдел" minLength:"1" maxLength:"200"`
	ParentID Nullable[int]    `json:"parent_id" swaggertype:"integer" example:"1" minimum:"1"`
}

func (r *MoveDepartmentRequest) validate() error {
	if !r.Name.Defined && !r.ParentID.Defined {
		return errors.New("no fields to update")
	}

	r.Name.Value = strings.TrimSpace(r.Name.Value)
	if r.Name.Defined && (r.Name.Value == "" || len(r.Name.Value) > model.MaxNameLength) {
		return errors.New("invalid name")
	}

	switch {
	case !r.ParentID.Defined:
		// undefined
		r.ParentID.Value = 0
	case !r.ParentID.Valid:
		// null
		r.ParentID.Value = model.VirtualRoot
	default:
		if !model.ValidID(r.ParentID.Value) {
			return errors.New("invalid parent_id")
		}
	}

	return nil
}

// DeleteDepartment godoc
//
//	@tags			department
//	@router			/departments/{department_id} [delete]
//	@summary		Удалить подразделение
//	@description	**Query:**
//	@description	- mode: str (cascade | reassign)
//	@description	cascade — удалить подразделение, всех сотрудников и все дочерние подразделения
//	@description	reassign — удалить подразделение, а сотрудников и дочерние подразделения переместить в reassign_to_department_id
//	@description	Если этот параметр не указан, подразделение удаляется только в том случае, если в нем нет дочерних подразделений или сотрудников
//	@description
//	@description	- reassign_to_department_id: int (обязателен, если mode=reassign)
//	@description
//	@description	**Response:** 204 No Content
//	@description
//	@param		department_id				path	int		true	"ID подразделения"					minimum(1)
//	@param		mode						query	string	false	"Режим удаления"					enums(cascade,reassign)
//	@param		reassign_to_department_id	query	int		false	"Расформировать подразделение в"	minimum(1)
//	@success	204
//	@failure	400
//	@failure	404
//	@failure	409
//	@failure	500
func DeleteDepartment(s Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h := newHelper(w, r, "api.DeleteDepartment")

		departmentID, err := h.getDepartmentID()
		if err != nil {
			h.writeError(err)
			return
		}

		q := h.query()
		modeStr := q.String("mode", optional, "")
		reassignTo := q.Int("reassign_to_department_id", optional, 0)
		if err := q.Err(); err != nil {
			h.writeError(&httpError{err.Error(), http.StatusBadRequest})
			return
		}

		var mode model.DeleteMode
		if modeStr != "" {
			var err error
			if mode, err = model.DeleteModeString(modeStr); err != nil {
				h.writeError(&httpError{"invalid mode", http.StatusBadRequest})
				return
			}
		}

		if mode == model.DeleteModeReassign {
			if !model.ValidID(reassignTo) {
				h.writeError(&httpError{"invalid reassign_to_department_id", http.StatusBadRequest})
				return
			}
		}

		err = s.DeleteDepartment(h.ctx(), model.DeleteDepartmentRequest{
			ID:                     departmentID,
			Mode:                   mode,
			ReassignToDepartmentID: reassignTo,
		})
		if err != nil {
			h.writeError(err)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
