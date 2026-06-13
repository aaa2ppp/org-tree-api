package memstor

import (
	"errors"
	"fmt"
	"time"

	"org-tree-api/internal/model"
)

var (
	ErrNotFound   = model.ErrNotFound
	ErrConstraint = errors.New("constraint error")
)

// employeeNode хранит запись о сотруднике.
type employeeNode struct {
	model.Employee
}

// departmentNode хранит запись об отделе. Предоставляет дочерние отделы по имени и сотрудников по ID.
// Никогда не использует содержимое записи, кроме ID в сообщениях об ошибках.
// Никогда не изменяет содержимое записи.
// Никогда не использует содержимое записей дочерних отделов и сотрудников.
type departmentNode struct {
	model.Department
	children  map[string]int   // lazy init in addChild
	employees map[int]struct{} // lazy init in addEmployee
}

func (d *departmentNode) addChild(name string, id int) error {
	if _, ok := d.children[name]; ok {
		return fmt.Errorf("%w: children with name %q already exist in department %d", ErrConstraint, name, d.ID)
	}
	if d.children == nil {
		d.children = map[string]int{}
	}
	d.children[name] = id
	return nil
}

func (d *departmentNode) addEmployee(id int) error {
	if d.ID == -1 {
		return fmt.Errorf("%w: forbidden add employee to root node", ErrConstraint)
	}
	if d.employees == nil {
		d.employees = map[int]struct{}{}
	}
	d.employees[id] = struct{}{}
	return nil
}

func (d *departmentNode) deleteChild(name string) {
	delete(d.children, name)
}

func (d *departmentNode) deleteEmployee(id int) {
	delete(d.employees, id)
}

// departmentsTree хранит записи отделов и сотрудников.
// Дерево всегда содержит виртуальный корень отделов с ID=-1.
// Корень запрещено изменять, удалять и добавлять в него сотрудникоа.
type departmentsTree struct {
	departments   map[int]*departmentNode
	employees     map[int]*employeeNode
	deptIDCounter int
	emplIDCounter int
}

func newDepathmentsTree() *departmentsTree {
	departments := map[int]*departmentNode{}
	departments[-1] = &departmentNode{Department: model.Department{
		ID:       -1,
		Name:     "__virtual_root__",
		ParentID: -1,
	}}
	return &departmentsTree{
		departments: departments,
		employees:   map[int]*employeeNode{},
	}
}

// ------- Basic methods -------

func (t *departmentsTree) CreateDepartment(dept model.Department) (model.Department, error) {
	parent := t.departments[dept.ParentID]
	if parent == nil {
		return dept, fmt.Errorf("%w: parent %d must be exists", ErrConstraint, dept.ParentID)
	}
	t.deptIDCounter++
	if err := parent.addChild(dept.Name, t.deptIDCounter); err != nil {
		return dept, err
	}
	dept.ID = t.deptIDCounter
	now := time.Now()
	dept.CreatedAt = now
	dept.UpdatedAt = now
	t.departments[t.deptIDCounter] = &departmentNode{Department: dept}
	return dept, nil
}

func (t *departmentsTree) CreateEmployee(empl model.Employee) (model.Employee, error) {
	dept := t.departments[empl.DepartmentID]
	if dept == nil {
		return empl, fmt.Errorf("%w: department %d must be exists", ErrConstraint, empl.DepartmentID)
	}
	t.emplIDCounter++
	if err := dept.addEmployee(t.emplIDCounter); err != nil {
		return empl, err
	}
	empl.ID = t.emplIDCounter
	now := time.Now()
	empl.CreatedAt = now
	empl.UpdatedAt = now
	t.employees[t.emplIDCounter] = &employeeNode{Employee: empl}
	return empl, nil
}

func (t *departmentsTree) GetDepartment(id int) (model.Department, error) {
	node := t.departments[id]
	if node == nil {
		return model.Department{}, fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	return node.Department, nil
}

func (t *departmentsTree) GetDepartmentChildren(id int) ([]model.Department, error) {
	node := t.departments[id]
	if node == nil {
		return nil, fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	var departments []model.Department
	for _, id := range node.children {
		child := t.departments[id]
		departments = append(departments, child.Department)
	}
	return departments, nil
}

func (t *departmentsTree) GetDepartmentEmployees(id int) ([]model.Employee, error) {
	node := t.departments[id]
	if node == nil {
		return nil, fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	var employees []model.Employee
	for id := range node.employees {
		empl := t.employees[id]
		employees = append(employees, empl.Employee)
	}
	return employees, nil
}

func (t *departmentsTree) GetEmployee(id int) (model.Employee, error) {
	node := t.employees[id]
	if node == nil {
		return model.Employee{}, fmt.Errorf("%w: employee %d not found", ErrNotFound, id)
	}
	return node.Employee, nil
}

func (t *departmentsTree) UpdateDepartment(dept model.Department) error {
	if dept.ID == -1 {
		return fmt.Errorf("%w: forbidden update root node", ErrConstraint)
	}
	node := t.departments[dept.ID]
	if node == nil {
		return fmt.Errorf("%w: department %d must be exists", ErrConstraint, dept.ID)
	}
	if dept.ParentID != node.ParentID || dept.Name != node.Name {
		if err := t.reassignDepartment(node, dept.ParentID, dept.Name); err != nil {
			return err
		}
	}
	dept.CreatedAt = node.CreatedAt
	dept.UpdatedAt = time.Now()
	node.Department = dept
	return nil
}

func (t *departmentsTree) reassignDepartment(node *departmentNode, toParentID int, withName string) error {
	oldParent := t.departments[node.ParentID]
	newParent := t.departments[toParentID]
	if newParent == nil {
		return fmt.Errorf("%w: new parent %d must be exists", ErrConstraint, toParentID)
	}
	if t.isDescendantOf(newParent, node.ID) {
		return fmt.Errorf("%w: new parent %d is descendant or is itself department %d", ErrConstraint, newParent.ID, node.ID)
	}
	oldParent.deleteChild(node.Name)
	if err := newParent.addChild(withName, node.ID); err != nil {
		oldParent.addChild(node.Name, node.ID)
		return err
	}
	return nil
}

// isDescendantOf проверяет, является ли node потомком ancestorID. Считает, что узел является потомком самого себя.
func (t *departmentsTree) isDescendantOf(node *departmentNode, ancestorID int) bool {
	for node != nil {
		if node.ID == ancestorID {
			return true
		}
		if node.ID == -1 {
			return false
		}
		node = t.departments[node.ParentID]
	}
	return false
}

func (t *departmentsTree) UpdateEmployee(empl model.Employee) error {
	node := t.employees[empl.ID]
	if node == nil {
		return fmt.Errorf("%w: employee %d not found", ErrConstraint, empl.ID)
	}
	if empl.DepartmentID != node.DepartmentID {
		if err := t.reassignEmployee(node, empl.DepartmentID); err != nil {
			return err
		}
	}
	empl.CreatedAt = node.CreatedAt
	empl.UpdatedAt = time.Now()
	node.Employee = empl
	return nil
}

func (t *departmentsTree) reassignEmployee(node *employeeNode, toDeptID int) error {
	oldDept := t.departments[node.DepartmentID]
	newDept := t.departments[toDeptID]
	if newDept == nil {
		return fmt.Errorf("%w: new department %d must be exists", ErrConstraint, toDeptID)
	}
	oldDept.deleteEmployee(node.ID)
	if err := newDept.addEmployee(node.ID); err != nil {
		oldDept.addEmployee(node.ID)
		return err
	}
	return nil
}

func (t *departmentsTree) DeleteDepartment(id int) error {
	if id == -1 {
		return fmt.Errorf("%w: forbidden delete root node", ErrConstraint)
	}
	node := t.departments[id]
	if node == nil {
		return fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	if len(node.children) != 0 {
		return fmt.Errorf("%w: department %d contains children", ErrConstraint, id)
	}
	if len(node.employees) != 0 {
		return fmt.Errorf("%w: department %d contains employees", ErrConstraint, id)
	}
	parent := t.departments[node.ParentID]
	parent.deleteChild(node.Name)
	delete(t.departments, id)
	return nil
}

func (t *departmentsTree) DeleteDepartmentCascade(id int) error {
	if id == -1 {
		return fmt.Errorf("%w: forbidden delete root node", ErrConstraint)
	}
	node := t.departments[id]
	if node == nil {
		return fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	parent := t.departments[node.ParentID]
	parent.deleteChild(node.Name)
	t.deleteDepartmentCascade(node)
	return nil
}

func (t *departmentsTree) deleteDepartmentCascade(node *departmentNode) {
	for _, id := range node.children {
		child := t.departments[id]
		t.deleteDepartmentCascade(child)
	}
	for id := range node.employees {
		delete(t.employees, id)
	}
	delete(t.departments, node.ID)
}

func (t *departmentsTree) DeleteEmployee(id int) error {
	node := t.employees[id]
	if node == nil {
		return fmt.Errorf("%w: employee %d not found", ErrNotFound, id)
	}
	dept := t.departments[node.DepartmentID]
	dept.deleteEmployee(id)
	delete(t.employees, id)
	return nil
}

// ------- Additional methods -------

func (t *departmentsTree) DepartmentExists(id int) (bool, error) {
	return t.departments[id] != nil, nil
}

func (t *departmentsTree) GetDepartmentTree(id int, depth int) ([]model.Department, error) {
	node := t.departments[id]
	if node == nil {
		return nil, fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	return t.collectDepartments(nil, node, depth), nil
}

func (t *departmentsTree) collectDepartments(departments []model.Department, node *departmentNode, depth int) []model.Department {
	departments = append(departments, node.Department)
	if depth > 0 {
		for _, id := range node.children {
			child := t.departments[id]
			departments = t.collectDepartments(departments, child, depth-1)
		}
	}
	return departments
}

func (t *departmentsTree) ChildNameExists(id int, name string) (bool, error) {
	node := t.departments[id]
	if node == nil {
		return false, fmt.Errorf("%w: department %d not found", ErrNotFound, id)
	}
	_, ok := node.children[name]
	return ok, nil
}

func (t *departmentsTree) ChildNamesIntersects(id1, id2 int) (bool, error) {
	node1 := t.departments[id1]
	if node1 == nil {
		return false, fmt.Errorf("%w: department %d not found", ErrNotFound, id1)
	}
	node2 := t.departments[id2]
	if node2 == nil {
		return false, fmt.Errorf("%w: department %d not found", ErrNotFound, id2)
	}
	for name := range node1.children {
		if _, ok := node2.children[name]; ok {
			return true, nil
		}
	}
	return false, nil
}

func (t *departmentsTree) IsDescendantOf(descendantID int, ancestorID int) (bool, error) {
	descendant := t.departments[descendantID]
	if descendant == nil {
		return false, fmt.Errorf("%w: descendant %d not found", ErrNotFound, descendantID)
	}
	ancestor := t.departments[ancestorID]
	if ancestor == nil {
		return false, fmt.Errorf("%w: ancestor %d not found", ErrNotFound, ancestorID)
	}
	return t.isDescendantOf(descendant, ancestorID), nil
}
