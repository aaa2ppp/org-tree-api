// FOR TESTS ONLY
package memstor

import (
	"context"
	"org-tree-api/internal/model"
	"org-tree-api/internal/service"
	"sync"
)

type MemStorage struct {
	mu sync.Mutex
	tx memStorTx
}

func New() *MemStorage {
	return &MemStorage{
		tx: memStorTx{newDepathmentsTree()},
	}
}

func (m *MemStorage) Transaction(_ context.Context, fn func(tx service.StorageTx) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(&m.tx)
}

type memStorTx struct {
	tree *departmentsTree
}

func (m *memStorTx) CreateDepartment(_ context.Context, dept model.Department) (model.Department, error) {
	return m.tree.CreateDepartment(dept)
}

func (m *memStorTx) CreateEmployee(_ context.Context, empl model.Employee) (model.Employee, error) {
	return m.tree.CreateEmployee(empl)
}

func (m *memStorTx) DepartmentExists(_ context.Context, id int) (bool, error) {
	return m.tree.DepartmentExists(id)
}

func (m *memStorTx) GetDepartment(_ context.Context, id int) (model.Department, error) {
	return m.tree.GetDepartment(id)
}

func (m *memStorTx) GetDepartmentTree(_ context.Context, id int, depth int) ([]model.Department, error) {
	return m.tree.GetDepartmentTree(id, depth)
}

func (m *memStorTx) GetEmployeesByDepartments(_ context.Context, deptIDs []int) ([]model.Employee, error) {
	var employees []model.Employee
	for _, id := range deptIDs {
		deptEmpls, err := m.tree.GetDepartmentEmployees(id)
		if err != nil {
			return nil, err
		}
		employees = append(employees, deptEmpls...)
	}
	return employees, nil
}

func (m *memStorTx) HasChildWithName(_ context.Context, parentID int, name string) (bool, error) {
	return m.tree.ChildNameExists(parentID, name)
}

func (m *memStorTx) HaveCommonChildNames(_ context.Context, deptID1 int, deptID2 int) (bool, error) {
	return m.tree.ChildNamesIntersects(deptID1, deptID2)
}

func (m *memStorTx) IsDescendantOf(_ context.Context, descendantID int, ancestorID int) (bool, error) {
	return m.tree.IsDescendantOf(descendantID, ancestorID)
}

func (m *memStorTx) UpdateDepartment(_ context.Context, dept model.Department) error {
	return m.tree.UpdateDepartment(dept)
}

func (m *memStorTx) DeleteDepartment(_ context.Context, id int) error {
	return m.tree.DeleteDepartment(id)
}

func (m *memStorTx) DeleteDepartmentCascade(_ context.Context, id int) error {
	return m.tree.DeleteDepartmentCascade(id)
}

func (m *memStorTx) ReassignChildren(_ context.Context, srcDetpID int, dstDepthID int) error {
	children, err := m.tree.GetDepartmentChildren(srcDetpID)
	if err != nil {
		return err
	}
	for _, child := range children {
		child.ParentID = dstDepthID
		if err := m.tree.UpdateDepartment(child); err != nil {
			return err
		}
	}
	return nil
}

func (m *memStorTx) ReassignEmployees(_ context.Context, srcDeptID int, dstDeptID int) error {
	employees, err := m.tree.GetDepartmentEmployees(srcDeptID)
	if err != nil {
		return err
	}
	for _, empl := range employees {
		empl.DepartmentID = dstDeptID
		if err := m.tree.UpdateEmployee(empl); err != nil {
			return err
		}
	}
	return nil
}

var _ service.StorageTx = &memStorTx{}
