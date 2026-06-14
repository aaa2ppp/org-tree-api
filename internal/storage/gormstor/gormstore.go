package gormstor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"unsafe"

	"gorm.io/gorm"

	"org-tree-api/internal/model"
	"org-tree-api/internal/service"
)

type DepartmentDB model.Department

func (DepartmentDB) TableName() string {
	return "department"
}

type EmployeeDB model.Employee

func (EmployeeDB) TableName() string {
	return "employee"
}

func modelDepartments(a []DepartmentDB) []model.Department {
	return unsafe.Slice((*model.Department)(unsafe.SliceData(a)), len(a))
}

func modelEmployees(a []EmployeeDB) []model.Employee {
	return unsafe.Slice((*model.Employee)(unsafe.SliceData(a)), len(a))
}

type GormStorage gorm.DB

func New(db *gorm.DB) *GormStorage {
	return (*GormStorage)(db)
}

func (g *GormStorage) db() *gorm.DB {
	return (*gorm.DB)(g)
}

// Close implements io.Closer.
func (g *GormStorage) Close() error {
	sqlDB, err := g.db().DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Transaction implements service.Storage.
func (g GormStorage) Transaction(ctx context.Context, fn func(tx service.StorageTx) error) error {
	return g.db().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn((*GormStorage)(tx))
	})
}

// CheckDepartmentExists implements service.StorageTx.
func (g *GormStorage) DepartmentExists(ctx context.Context, id int) (bool, error) {
	count, err := gorm.G[DepartmentDB](g.db()).Where("id = ?", id).Count(ctx, "*")
	return count != 0, err
}

// CreateDepartment implements service.StorageTx.
func (g *GormStorage) CreateDepartment(ctx context.Context, dept model.Department) (model.Department, error) {
	resp := DepartmentDB(dept)
	err := gorm.G[DepartmentDB](g.db()).Create(ctx, &resp)
	return model.Department(resp), err
}

// CreateEmployee implements service.StorageTx.
func (g *GormStorage) CreateEmployee(ctx context.Context, empl model.Employee) (model.Employee, error) {
	resp := EmployeeDB(empl)
	err := gorm.G[EmployeeDB](g.db()).Create(ctx, &resp)
	return model.Employee(resp), err
}

// DeleteDepartment implements service.StorageTx.
func (g *GormStorage) DeleteDepartment(ctx context.Context, id int) error {
	_, err := gorm.G[DepartmentDB](g.db()).Where("id = ?", id).Delete(ctx)
	return err
}

// DeleteDepartmentCascade DANGER! удаляет отдел, все подчиненые отделы и всех сотрудников в этих отделах.
func (g *GormStorage) DeleteDepartmentCascade(ctx context.Context, id int) error {
	return gorm.G[DepartmentDB](g.db()).Exec(ctx, `
		WITH RECURSIVE tree (id) AS (
			SELECT d.id FROM department AS d WHERE d.id = $1
			UNION ALL
			SELECT d.id FROM tree AS t JOIN department AS d ON t.id = d.parent_id 
		), delete_employees AS (
			DELETE FROM employee AS e WHERE e.department_id IN (select id FROM tree)
		)
		DELETE FROM department AS d WHERE d.id IN (select id FROM tree);`,
		id,
	)
}

// GetDepartment implements service.StorageTx.
func (g *GormStorage) GetDepartment(ctx context.Context, id int) (model.Department, error) {
	resp, err := gorm.G[DepartmentDB](g.db()).Where("id = ?", id).First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = fmt.Errorf("%w: department not found", model.ErrNotFound)
	}
	return model.Department(resp), nil
}

// GetDepartmentDescendants Возвращает узел (если найден) и всех его потомков до depth.
// Если если depth <= 0 возвращается только сам узел.
func (g *GormStorage) GetDepartmentTree(ctx context.Context, id int, depth int) ([]model.Department, error) {
	resp, err := gorm.G[DepartmentDB](g.db()).Raw(`
		WITH RECURSIVE tree (depth, id) AS (
			SELECT $2+0, d.id FROM department AS d WHERE d.id = $1
			UNION ALL
			SELECT t.depth-1, d.id FROM tree AS t JOIN department AS d ON t.id = d.parent_id 
			WHERE t.depth > 0 AND d.id <> -1
		)
		SELECT d.* FROM tree AS t JOIN department AS d ON t.id = d.id;`,
		id, depth,
	).Find(ctx)
	return modelDepartments(resp), err
}

// GetEmployees implements service.StorageTx.
func (g *GormStorage) GetEmployeesByDepartments(ctx context.Context, deptIDs []int) ([]model.Employee, error) {
	resp, err := gorm.G[EmployeeDB](g.db()).Where("department_id IN ?", deptIDs).Find(ctx)
	return modelEmployees(resp), err
}

// CheckNameUnique implements service.StorageTx.
func (g *GormStorage) HasChildWithName(ctx context.Context, parentID int, name string) (bool, error) {
	count, err := gorm.G[DepartmentDB](g.db()).Where("parent_id = ? AND name = ?", parentID, name).Count(ctx, "*")
	return count != 0, err
}

// HaveCommonChildNames implements service.StorageTx.
func (g *GormStorage) HaveCommonChildNames(ctx context.Context, deptID1 int, deptID2 int) (bool, error) {
	maxCount, err := gorm.G[int](g.db()).Raw(`
		WITH names (name, cnt) AS (
			SELECT name, count(name) FROM department AS d
			WHERE d.parent_id = $1 OR d.parent_id = $2
			GROUP BY name
		) 
		SELECT max(cnt) FROM names;`,
		deptID1, deptID2,
	).First(ctx)
	return maxCount > 1, err
}

// IsDescendantOf проверяет истинность утверждения. Считает, что узел является потомком самого себя.
func (g *GormStorage) IsDescendantOf(ctx context.Context, descendantID int, ancestorID int) (bool, error) {
	count, err := gorm.G[int](g.db()).Raw(`
		WITH RECURSIVE ancestor (id) AS (
			SELECT id FROM department WHERE id = $1
			UNION ALL
			SELECT d.parent_id FROM ancestor AS a JOIN department AS d ON a.id = d.id
			WHERE a.id != $2 AND a.id != $3
		)
		SELECT count(*) FROM ancestor WHERE id = $2;`,
		descendantID, ancestorID, model.VirtualRoot,
	).First(ctx)
	return count != 0, err
}

// ReassignChildren implements service.StorageTx.
func (g *GormStorage) ReassignChildren(ctx context.Context, srcDeptID int, dstDeptID int) error {
	_, err := gorm.G[DepartmentDB](g.db()).Where("parent_id = ?", srcDeptID).Update(ctx, "parent_id", dstDeptID)
	return err
}

// ReassignEmployees implements service.StorageTx.
func (g *GormStorage) ReassignEmployees(ctx context.Context, srcDeptID int, dstDeptID int) error {
	_, err := gorm.G[EmployeeDB](g.db()).Where("department_id = ?", srcDeptID).Update(ctx, "department_id", dstDeptID)
	return err
}

// UpdateDepartment implements service.StorageTx.
func (g *GormStorage) UpdateDepartment(ctx context.Context, dept model.Department) error {
	_, err := gorm.G[DepartmentDB](g.db()).Select("*").Updates(ctx, DepartmentDB(dept))
	return err
}

var _ io.Closer = &GormStorage{}
var _ service.Storage = &GormStorage{}
var _ service.StorageTx = &GormStorage{}
