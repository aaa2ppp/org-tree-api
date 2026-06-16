package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"

	"org-tree-api/internal/api"
	"org-tree-api/internal/model"
)

type Storage interface {
	Transaction(ctx context.Context, fn func(tx StorageTx) error) error
}

type StorageTx interface {
	CreateDepartment(ctx context.Context, dept model.Department) (model.Department, error)
	CreateEmployee(ctx context.Context, empl model.Employee) (model.Employee, error)
	GetDepartment(ctx context.Context, id int) (model.Department, error)
	GetDepartmentTree(ctx context.Context, id int, depth int) ([]model.Department, error)
	GetEmployeesByDepartments(ctx context.Context, deptIDs []int) ([]model.Employee, error)
	UpdateDepartment(ctx context.Context, dept model.Department) error
	DeleteDepartment(ctx context.Context, id int) error
	DeleteDepartmentCascade(ctx context.Context, id int) error
	DepartmentExists(ctx context.Context, id int) (bool, error)
	HasChildWithName(ctx context.Context, parentID int, name string) (bool, error)
	HaveCommonChildNames(ctx context.Context, deptID1, deptID2 int) (bool, error)
	IsDepartmentEmpty(ctx context.Context, id int) (bool, error)
	IsDescendantOf(ctx context.Context, descendantID int, ancestorID int) (bool, error)
	ReassignChildren(ctx context.Context, srcDeptID, dstDeptID int) error
	ReassignEmployees(ctx context.Context, srcDeptID, dstDeptID int) error
}

type Service struct {
	stor Storage
}

func New(stor Storage) *Service {
	return &Service{
		stor: stor,
	}
}

// CreateDepartment implements Service.
func (s *Service) CreateDepartment(ctx context.Context, req model.Department) (model.Department, error) {
	var resp model.Department

	if !model.ValidID(req.ParentID) && req.ParentID != model.VirtualRoot {
		return resp, fmt.Errorf("%w: invalid parent_id", model.ErrValidation)
	}
	if req.Name == "" || len(req.Name) > model.MaxNameLength {
		return resp, fmt.Errorf("%w: invalid name", model.ErrValidation)
	}

	err := s.stor.Transaction(ctx, func(tx StorageTx) (err error) {
		if err := checkDepartmentExists(ctx, tx, req.ParentID); err != nil {
			return err
		}
		if err := checkChildNameNotExists(ctx, tx, req.ParentID, req.Name); err != nil {
			return err
		}
		resp, err = tx.CreateDepartment(ctx, req)
		return err
	})
	return resp, err
}

func checkDepartmentExists(ctx context.Context, tx StorageTx, id int) error {
	exists, err := tx.DepartmentExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: department id=%d not found", model.ErrNotFound, id)
	}
	return nil
}

func ensureDepartmentExists(ctx context.Context, tx StorageTx, id int) error {
	exists, err := tx.DepartmentExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: department id=%d must be exists", model.ErrConflict, id)
	}
	return nil
}

func checkChildNameNotExists(ctx context.Context, tx StorageTx, id int, name string) error {
	exists, err := tx.HasChildWithName(ctx, id, name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("%w: child name %q already in department id=%d", model.ErrConflict, name, id)
	}
	return nil
}

// CreateEmployee implements Service.
func (s *Service) CreateEmployee(ctx context.Context, req model.Employee) (model.Employee, error) {
	var resp model.Employee

	if !model.ValidID(req.DepartmentID) {
		return resp, fmt.Errorf("%w: invalid department_id", model.ErrValidation)
	}
	if req.FullName == "" || len(req.FullName) > model.MaxFullNameLength {
		return resp, fmt.Errorf("%w: invalid full_name", model.ErrValidation)
	}
	if req.Position == "" || len(req.Position) > model.MaxPositionLength {
		return resp, fmt.Errorf("%w: invalid position", model.ErrValidation)
	}

	err := s.stor.Transaction(ctx, func(tx StorageTx) (err error) {
		if err := checkDepartmentExists(ctx, tx, req.DepartmentID); err != nil {
			return err
		}
		resp, err = tx.CreateEmployee(ctx, req)
		return err
	})
	return resp, err
}

// DeleteDepartment implements Service.
func (s *Service) DeleteDepartment(ctx context.Context, req model.DeleteDepartmentRequest) error {
	if !model.ValidID(req.ID) {
		return fmt.Errorf("%w: invalid department id", model.ErrValidation)
	}

	switch req.Mode {
	case model.DeleteModeCascade:
		return s.deleteDepartmentCascade(ctx, req.ID)

	case model.DeleteModeReassign:
		reassignTo := req.ReassignToDepartmentID
		if !model.ValidID(reassignTo) {
			return fmt.Errorf("%w: invalid reassign_to_department_id", model.ErrValidation)
		}
		if req.ID == reassignTo {
			return fmt.Errorf("%w: cannot reassign department into itself", model.ErrValidation)
		}
		return s.deleteDepartmentWithReassign(ctx, req.ID, reassignTo)
	}

	return s.deleteDepartmentIfEmpty(ctx, req.ID)
}

func (s *Service) deleteDepartmentCascade(ctx context.Context, id int) error {
	return s.stor.Transaction(ctx, func(tx StorageTx) error {
		if err := checkDepartmentExists(ctx, tx, id); err != nil {
			return err
		}
		return tx.DeleteDepartmentCascade(ctx, id)
	})
}

func (s *Service) deleteDepartmentWithReassign(ctx context.Context, id int, reassignTo int) error {
	return s.stor.Transaction(ctx, func(tx StorageTx) error {
		dept, err := tx.GetDepartment(ctx, id)
		if err != nil {
			return err
		}

		if err := ensureDepartmentExists(ctx, tx, reassignTo); err != nil {
			return err
		}

		if dept.ParentID == reassignTo {
			// костыль, чтобы избежать конфликта имен с удаляемым подразделением при переназначении
			dept.Name = fmt.Sprintf("%x", rand.Uint64())
			if err := tx.UpdateDepartment(ctx, dept); err != nil {
				return err
			}
		} else {
			isDescendant, err := tx.IsDescendantOf(ctx, reassignTo, id)
			if err != nil {
				return err
			}
			if isDescendant {
				return fmt.Errorf("%w: cannot reassign department into sub-department", model.ErrConflict)
			}
		}

		nameConflict, err := tx.HaveCommonChildNames(ctx, reassignTo, id)
		if err != nil {
			return err
		}
		if nameConflict {
			return fmt.Errorf("%w: can't move children: name conflict", model.ErrConflict)
		}

		if err := tx.ReassignChildren(ctx, id, reassignTo); err != nil {
			return err
		}
		if err := tx.ReassignEmployees(ctx, id, reassignTo); err != nil {
			return err
		}

		return tx.DeleteDepartment(ctx, id)
	})
}

func (s *Service) deleteDepartmentIfEmpty(ctx context.Context, id int) error {
	return s.stor.Transaction(ctx, func(tx StorageTx) error {
		if err := checkDepartmentExists(ctx, tx, id); err != nil {
			return err
		}

		isEmpty, err := tx.IsDepartmentEmpty(ctx, id)
		if err != nil {
			return err
		}
		if !isEmpty {
			return fmt.Errorf("%w: department must be empty", model.ErrConflict)
		}

		return tx.DeleteDepartment(ctx, id)
	})
}

// GetDepartmentTree implements Service.
func (s *Service) GetDepartmentTree(ctx context.Context, req model.GetDepartmentsTreeRequest) (*model.DepartmentNode, error) {
	if !model.ValidID(req.ID) && req.ID != model.VirtualRoot {
		return nil, fmt.Errorf("%w: invalid department id", model.ErrValidation)
	}
	if req.Depth < 0 || req.Depth > model.MaxDepth {
		return nil, fmt.Errorf("%w: invalid depth", model.ErrValidation)
	}

	var departments []model.Department
	var employees []model.Employee

	err := s.stor.Transaction(ctx, func(tx StorageTx) (err error) {
		departments, err = tx.GetDepartmentTree(ctx, req.ID, req.Depth)
		if err != nil {
			return err
		}
		if len(departments) == 0 {
			return fmt.Errorf("%w: department id=%d not found", model.ErrNotFound, req.ID)
		}
		if req.IncludeEmployees {
			deptIDs := make([]int, 0, len(departments))
			for i := range departments {
				deptIDs = append(deptIDs, departments[i].ID)
			}
			employees, err = tx.GetEmployeesByDepartments(ctx, deptIDs)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	nodesMap := buildTree(departments)
	linkEmployees(nodesMap, employees)

	for _, node := range nodesMap {
		sortChildren(node.Children, req.SortByName)
		sortEmployees(node.Employees, req.SortByName)
	}

	root := nodesMap[req.ID]
	return root, nil
}

func buildTree(departments []model.Department) map[int]*model.DepartmentNode {
	nodes := make(map[int]*model.DepartmentNode, len(departments))
	for _, department := range departments {
		node, exists := nodes[department.ID]
		if !exists {
			node = &model.DepartmentNode{}
			nodes[department.ID] = node
		}
		node.Department = &department
		parent, exists := nodes[department.ParentID]
		if !exists {
			parent = &model.DepartmentNode{}
			nodes[department.ParentID] = parent
		}
		if department.ID != model.VirtualRoot {
			parent.Children = append(parent.Children, node)
		}
	}
	return nodes
}

func linkEmployees(nodes map[int]*model.DepartmentNode, employees []model.Employee) {
	for _, employee := range employees {
		if node := nodes[employee.DepartmentID]; node != nil {
			node.Employees = append(node.Employees, &employee)
		}
	}
}

func sortChildren(children []*model.DepartmentNode, sortByName bool) {
	if sortByName {
		slices.SortFunc(children, func(a, b *model.DepartmentNode) int {
			if v := strings.Compare(a.Department.Name, b.Department.Name); v != 0 {
				return v
			}
			return a.Department.ID - b.Department.ID
		})
		return
	}
	slices.SortFunc(children, func(a, b *model.DepartmentNode) int {
		return a.Department.ID - b.Department.ID
	})
}

func sortEmployees(employees []*model.Employee, sortByFullName bool) {
	if sortByFullName {
		slices.SortFunc(employees, func(a, b *model.Employee) int {
			if v := strings.Compare(a.FullName, b.FullName); v != 0 {
				return v
			}
			return a.ID - b.ID
		})
		return
	}
	slices.SortFunc(employees, func(a, b *model.Employee) int {
		return a.ID - b.ID
	})
}

// MoveDepartment implements Service.
func (s *Service) MoveDepartment(ctx context.Context, req model.MoveDepartmentRequest) (model.Department, error) {
	var dept model.Department

	if !model.ValidID(req.ID) {
		return dept, fmt.Errorf("%w: invalid department id", model.ErrValidation)
	}
	if !model.ValidID(req.ParentID) && req.ParentID != model.VirtualRoot && req.ParentID != 0 {
		return dept, fmt.Errorf("%w: invalid parent_id", model.ErrValidation)
	}
	if len(req.Name) > model.MaxNameLength {
		return dept, fmt.Errorf("%w: invalid name", model.ErrValidation)
	}
	if req.Name == "" && req.ParentID == 0 {
		return dept, fmt.Errorf("%w: name or parent_id must be defined, or both", model.ErrValidation)
	}
	if req.ID == req.ParentID {
		return dept, fmt.Errorf("%w: cannot move department into itself", model.ErrValidation)
	}

	err := s.stor.Transaction(ctx, func(tx StorageTx) (err error) {
		if dept, err = tx.GetDepartment(ctx, req.ID); err != nil {
			return err
		}

		newName := dept.Name
		if req.Name != "" {
			newName = req.Name
		}

		newParentID := dept.ParentID
		if req.ParentID != 0 {
			newParentID = req.ParentID
		}

		if newName == dept.Name && newParentID == dept.ParentID {
			// nothing to change
			return nil
		}

		if newParentID != dept.ParentID {
			if err := ensureDepartmentExists(ctx, tx, newParentID); err != nil {
				return err
			}
			isDescendant, err := tx.IsDescendantOf(ctx, newParentID, req.ID)
			if err != nil {
				return err
			}
			if isDescendant {
				return fmt.Errorf("%w: cannot move department into descendant or into itself", model.ErrConflict)
			}
		}

		if err := checkChildNameNotExists(ctx, tx, newParentID, newName); err != nil {
			return err
		}

		dept.Name = newName
		dept.ParentID = newParentID
		return tx.UpdateDepartment(ctx, dept)
	})
	return dept, err
}

var _ api.Service = &Service{}
