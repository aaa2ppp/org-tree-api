package tests

import (
	"context"
	"org-tree-api/internal/model"
	"org-tree-api/internal/service"
	"strings"
	"testing"
	"time"

	"github.com/aaa2ppp/be"
)

func zeroedTimeD(v model.Department) model.Department {
	v.CreatedAt = time.Time{}
	v.UpdatedAt = time.Time{}
	return v
}

func zeroedTimeE(v model.Employee) model.Employee {
	v.CreatedAt = time.Time{}
	v.UpdatedAt = time.Time{}
	return v
}

func zeroedTimeN(node *model.DepartmentNode) *model.DepartmentNode {
	if node == nil {
		return nil
	}
	node.Department.CreatedAt = time.Time{}
	node.Department.UpdatedAt = time.Time{}
	for _, empl := range node.Employees {
		empl.CreatedAt = time.Time{}
		empl.UpdatedAt = time.Time{}
	}
	for _, child := range node.Children {
		zeroedTimeN(child)
	}
	return node
}

type (
	D  = model.Department
	E  = model.Employee
	N  = model.DepartmentNode
	GR = model.GetDepartmentsTreeRequest
	MR = model.MoveDepartmentRequest
	DR = model.DeleteDepartmentRequest
)

type NewStorageFunc func(t *testing.T) service.Storage

func Service_Create_Get_Departments(t *testing.T, ctx context.Context, newStorage NewStorageFunc) {
	svc := service.New(newStorage(t))

	// ------ GREATE ------

	// 1
	gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 1, Name: "top_1", ParentID: -1})

	// empty name
	_, err = svc.CreateDepartment(ctx, D{Name: "", ParentID: -1})
	be.Err(t, err, model.ErrValidation)

	// 2
	gotD, err = svc.CreateDepartment(ctx, D{Name: "top_2", ParentID: -1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 2, Name: "top_2", ParentID: -1})

	// duplicate name
	_, err = svc.CreateDepartment(ctx, D{Name: "top_2", ParentID: -1})
	be.Err(t, err, model.ErrConflict)

	// name too long
	_, err = svc.CreateDepartment(ctx, D{Name: strings.Repeat("a", model.MaxFullNameLength+1), ParentID: -1})
	be.Err(t, err, model.ErrValidation)

	// empty parent
	_, err = svc.CreateDepartment(ctx, D{Name: "top_3", ParentID: 0})
	be.Err(t, err, model.ErrValidation)

	// invalid parent (negative)
	_, err = svc.CreateDepartment(ctx, D{Name: "top_3", ParentID: -100})
	be.Err(t, err, model.ErrValidation)

	// invalid parent (too large)
	_, err = svc.CreateDepartment(ctx, D{Name: "top_3", ParentID: 9999999999})
	be.Err(t, err, model.ErrValidation)

	// unknow parent
	_, err = svc.CreateDepartment(ctx, D{Name: "top_3", ParentID: 100})
	be.Err(t, err, model.ErrNotFound) // хм?.. not found?

	// 3
	gotD, err = svc.CreateDepartment(ctx, D{Name: "child_3", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 3, Name: "child_3", ParentID: 1})

	// 4
	gotD, err = svc.CreateDepartment(ctx, D{Name: "child_2", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, gotD.ID, 4)

	// 5
	gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, gotD.ID, 5)

	// 6
	gotD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_3", ParentID: 5})
	be.Err(t, err, nil)
	be.Equal(t, gotD.ID, 6)

	// 7
	gotD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_2", ParentID: 5})
	be.Err(t, err, nil)
	be.Equal(t, gotD.ID, 7)

	// 8
	gotD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_1", ParentID: 5})
	be.Err(t, err, nil)
	be.Equal(t, gotD.ID, 8)

	// ------ GET -------

	// get top_1
	gotT, err := svc.GetDepartmentTree(ctx, GR{ID: 1})
	be.Err(t, err, nil)
	be.Equal(t, (zeroedTimeN(gotT)), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
	})

	// get top_2
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 2})
	be.Err(t, err, nil)
	be.Equal(t, (zeroedTimeN(gotT)), &N{
		Department: &D{ID: 2, Name: "top_2", ParentID: -1},
	})

	// root
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: -1, Depth: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), (zeroedTimeN(gotT)), &N{
		Department: &D{ID: -1, Name: "__virtual_root__", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 1, Name: "top_1", ParentID: -1}},
			{Department: &D{ID: 2, Name: "top_2", ParentID: -1}},
		},
	})

	// depth = 2
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotT), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 3, Name: "child_3", ParentID: 1}},
			{Department: &D{ID: 4, Name: "child_2", ParentID: 1}},
			{Department: &D{ID: 5, Name: "child_1", ParentID: 1}},
		},
	})

	// depth = 2, sort by name
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 1, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotT), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 5, Name: "child_1", ParentID: 1}},
			{Department: &D{ID: 4, Name: "child_2", ParentID: 1}},
			{Department: &D{ID: 3, Name: "child_3", ParentID: 1}},
		},
	})

	// depth = 3
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 2})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotT), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 3, Name: "child_3", ParentID: 1}},
			{Department: &D{ID: 4, Name: "child_2", ParentID: 1}},
			{Department: &D{ID: 5, Name: "child_1", ParentID: 1},
				Children: []*N{
					{Department: &D{ID: 6, Name: "grandchild_3", ParentID: 5}},
					{Department: &D{ID: 7, Name: "grandchild_2", ParentID: 5}},
					{Department: &D{ID: 8, Name: "grandchild_1", ParentID: 5}},
				},
			},
		},
	})

	// depth = 3, sort by name
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 3, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotT), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 5, Name: "child_1", ParentID: 1},
				Children: []*N{
					{Department: &D{ID: 8, Name: "grandchild_1", ParentID: 5}},
					{Department: &D{ID: 7, Name: "grandchild_2", ParentID: 5}},
					{Department: &D{ID: 6, Name: "grandchild_3", ParentID: 5}},
				},
			},
			{Department: &D{ID: 4, Name: "child_2", ParentID: 1}},
			{Department: &D{ID: 3, Name: "child_3", ParentID: 1}},
		},
	})

	// get child
	gotT, err = svc.GetDepartmentTree(ctx, GR{ID: 5, Depth: 5, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotT), &N{
		Department: &D{ID: 5, Name: "child_1", ParentID: 1},
		Children: []*N{
			{Department: &D{ID: 8, Name: "grandchild_1", ParentID: 5}},
			{Department: &D{ID: 7, Name: "grandchild_2", ParentID: 5}},
			{Department: &D{ID: 6, Name: "grandchild_3", ParentID: 5}},
		},
	})

	// empty ID (zero)
	_, err = svc.GetDepartmentTree(ctx, GR{})
	be.Err(t, err, model.ErrValidation)

	// invalid ID (negative)
	_, err = svc.GetDepartmentTree(ctx, GR{ID: -5})
	be.Err(t, err, model.ErrValidation)

	// invalid ID (too large)
	_, err = svc.GetDepartmentTree(ctx, GR{ID: 9999999999})
	be.Err(t, err, model.ErrValidation)

	// unknown ID
	_, err = svc.GetDepartmentTree(ctx, GR{ID: 100})
	be.Err(t, err, model.ErrNotFound)

	// invalid Depth (negative)
	_, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: -1})
	be.Err(t, err, model.ErrValidation)

	// invalid Depth (too large)
	_, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: model.MaxDepth + 1})
	be.Err(t, err, model.ErrValidation)
}

func Service_Create_Get_Employees(t *testing.T, ctx context.Context, newStorage NewStorageFunc) {
	svc := service.New(newStorage(t))

	// ------ GREATE ------

	// d1
	getD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
	be.Err(t, err, nil)
	be.Equal(t, getD.ID, 1)

	// e1
	getE, err := svc.CreateEmployee(ctx, E{FullName: "empl_3", Position: "staff", DepartmentID: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeE(getE), E{ID: 1, FullName: "empl_3", Position: "staff", DepartmentID: 1})

	// e2
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_2", Position: "staff", DepartmentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 2)

	// invalid parent (zero)
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: 0})
	be.Err(t, err, model.ErrValidation)

	// invalid parent (negative)
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: -1})
	be.Err(t, err, model.ErrValidation)

	// invalid parent (unknown)
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: 100})
	be.Err(t, err, model.ErrNotFound) // хм?.. not found?

	// invalid parent (too large)
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: 9999999999})
	be.Err(t, err, model.ErrValidation)

	// empty name
	getE, err = svc.CreateEmployee(ctx, E{FullName: "", Position: "staff", DepartmentID: 1})
	be.Err(t, err, model.ErrValidation)

	// name too long
	getE, err = svc.CreateEmployee(ctx, E{FullName: strings.Repeat("b", model.MaxFullNameLength+1), Position: "staff", DepartmentID: 1})
	be.Err(t, err, model.ErrValidation)

	// empty position
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "", DepartmentID: 1})
	be.Err(t, err, model.ErrValidation)

	// position too long
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: strings.Repeat("c", model.MaxFullNameLength+1), DepartmentID: 1})
	be.Err(t, err, model.ErrValidation)

	// e3
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 3)

	gotN, err := svc.GetDepartmentTree(ctx, GR{ID: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
	})

	// d2
	getD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, getD.ID, 2)

	// e4
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_6", Position: "staff", DepartmentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 4)

	// e5
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_5", Position: "staff", DepartmentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 5)

	// e6
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_4", Position: "staff", DepartmentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 6)

	// d3
	getD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_1", ParentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, getD.ID, 3)

	// e7
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_9", Position: "staff", DepartmentID: 3})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 7)

	// e8
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_8", Position: "staff", DepartmentID: 3})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 8)

	// e9
	getE, err = svc.CreateEmployee(ctx, E{FullName: "empl_7", Position: "staff", DepartmentID: 3})
	be.Err(t, err, nil)
	be.Equal(t, getE.ID, 9)

	// ------- GET TREE --------

	gotN, err = svc.GetDepartmentTree(ctx, GR{ID: 1, IncludeEmployees: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Employees: []*E{
			{ID: 1, FullName: "empl_3", Position: "staff", DepartmentID: 1},
			{ID: 2, FullName: "empl_2", Position: "staff", DepartmentID: 1},
			{ID: 3, FullName: "empl_1", Position: "staff", DepartmentID: 1},
		},
	})

	gotN, err = svc.GetDepartmentTree(ctx, GR{ID: 1, IncludeEmployees: true, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Employees: []*E{
			{ID: 3, FullName: "empl_1", Position: "staff", DepartmentID: 1},
			{ID: 2, FullName: "empl_2", Position: "staff", DepartmentID: 1},
			{ID: 1, FullName: "empl_3", Position: "staff", DepartmentID: 1},
		},
	})

	gotN, err = svc.GetDepartmentTree(ctx, GR{ID: 1, IncludeEmployees: true, Depth: 5})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Employees: []*E{
			{ID: 1, FullName: "empl_3", Position: "staff", DepartmentID: 1},
			{ID: 2, FullName: "empl_2", Position: "staff", DepartmentID: 1},
			{ID: 3, FullName: "empl_1", Position: "staff", DepartmentID: 1},
		},
		Children: []*N{
			{
				Department: &D{ID: 2, Name: "child_1", ParentID: 1},
				Employees: []*E{
					{ID: 4, FullName: "empl_6", Position: "staff", DepartmentID: 2},
					{ID: 5, FullName: "empl_5", Position: "staff", DepartmentID: 2},
					{ID: 6, FullName: "empl_4", Position: "staff", DepartmentID: 2},
				},
				Children: []*N{
					{
						Department: &D{ID: 3, Name: "grandchild_1", ParentID: 2},
						Employees: []*E{
							{ID: 7, FullName: "empl_9", Position: "staff", DepartmentID: 3},
							{ID: 8, FullName: "empl_8", Position: "staff", DepartmentID: 3},
							{ID: 9, FullName: "empl_7", Position: "staff", DepartmentID: 3},
						},
					},
				},
			},
		},
	})

	gotN, err = svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 5, IncludeEmployees: true, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Employees: []*E{
			{ID: 3, FullName: "empl_1", Position: "staff", DepartmentID: 1},
			{ID: 2, FullName: "empl_2", Position: "staff", DepartmentID: 1},
			{ID: 1, FullName: "empl_3", Position: "staff", DepartmentID: 1},
		},
		Children: []*N{
			{
				Department: &D{ID: 2, Name: "child_1", ParentID: 1},
				Employees: []*E{
					{ID: 6, FullName: "empl_4", Position: "staff", DepartmentID: 2},
					{ID: 5, FullName: "empl_5", Position: "staff", DepartmentID: 2},
					{ID: 4, FullName: "empl_6", Position: "staff", DepartmentID: 2},
				},
				Children: []*N{
					{
						Department: &D{ID: 3, Name: "grandchild_1", ParentID: 2},
						Employees: []*E{
							{ID: 9, FullName: "empl_7", Position: "staff", DepartmentID: 3},
							{ID: 8, FullName: "empl_8", Position: "staff", DepartmentID: 3},
							{ID: 7, FullName: "empl_9", Position: "staff", DepartmentID: 3},
						},
					},
				},
			},
		},
	})
}

func Service_MoveDepartment(t *testing.T, ctx context.Context, newStorage NewStorageFunc) {
	svc := service.New(newStorage(t))

	gotID, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
	be.Err(t, err, nil)
	be.Equal(t, gotID.ID, 1)

	gotID, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(t, gotID.ID, 2)

	gotID, err = svc.CreateDepartment(ctx, D{Name: "grandchild_1", ParentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, gotID.ID, 3)

	gotID, err = svc.CreateDepartment(ctx, D{Name: "grandchild_2", ParentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, gotID.ID, 4)

	gotID, err = svc.CreateDepartment(ctx, D{Name: "grandchild_3", ParentID: 2})
	be.Err(t, err, nil)
	be.Equal(t, gotID.ID, 5)

	// cannot move to to itself
	_, err = svc.MoveDepartment(ctx, MR{ID: 1, ParentID: 1})
	be.Err(t, err, model.ErrValidation, model.ErrConflict)

	// cannot move to to descendant (child)
	_, err = svc.MoveDepartment(ctx, MR{ID: 1, ParentID: 2})
	be.Err(t, err, model.ErrConflict)

	// cannot move to to descendant (grandchild)
	_, err = svc.MoveDepartment(ctx, MR{ID: 1, ParentID: 3})
	be.Err(t, err, model.ErrConflict)

	// unknown parent
	_, err = svc.MoveDepartment(ctx, MR{ID: 3, ParentID: 100})
	be.Err(t, err, model.ErrConflict)

	gotD, err := svc.MoveDepartment(ctx, MR{ID: 3, Name: "child_2", ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 3, Name: "child_2", ParentID: 1})

	gotD, err = svc.MoveDepartment(ctx, MR{ID: 4, ParentID: 1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 4, Name: "grandchild_2", ParentID: 1})

	// duplicate name
	_, err = svc.MoveDepartment(ctx, MR{ID: 4, Name: "child_2"})
	be.Err(t, err, model.ErrConflict)

	gotD, err = svc.MoveDepartment(ctx, MR{ID: 4, Name: "child_3"})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 4, Name: "child_3", ParentID: 1})

	gotD, err = svc.MoveDepartment(ctx, MR{ID: 5, Name: "top_2", ParentID: -1})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeD(gotD), D{ID: 5, Name: "top_2", ParentID: -1})

	gotN, err := svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 5, SortByName: true})
	be.Err(t, err, nil)
	be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
		Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		Children: []*N{
			{Department: &D{ID: 2, Name: "child_1", ParentID: 1}},
			{Department: &D{ID: 3, Name: "child_2", ParentID: 1}},
			{Department: &D{ID: 4, Name: "child_3", ParentID: 1}},
		},
	})
}

func Service_DeleteDepartment(t *testing.T, ctx context.Context, newStorage NewStorageFunc) {
	t.Run("cascade", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 2)

		err = svc.DeleteDepartment(ctx, DR{ID: 2, Cascade: true})
		be.Err(t, err, nil)

		gotN, err := svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 5, SortByName: true})
		be.Err(t, err, nil)
		be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
			Department: &D{ID: 1, Name: "top_1", ParentID: -1},
		})
	})

	t.Run("reassign", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 2)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_1", ParentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 3)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "grandchild_2", ParentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 4)

		gotE, err := svc.CreateEmployee(ctx, E{FullName: "empl_1", Position: "staff", DepartmentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotE.ID, 1)

		gotE, err = svc.CreateEmployee(ctx, E{FullName: "empl_2", Position: "staff", DepartmentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotE.ID, 2)

		err = svc.DeleteDepartment(ctx, DR{ID: 2, ReassignToDepartmentID: 1})
		be.Err(t, err, nil)

		gotN, err := svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 5, IncludeEmployees: true, SortByName: true})
		be.Err(t, err, nil)
		be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
			Department: &D{ID: 1, Name: "top_1", ParentID: -1},
			Employees: []*E{
				{ID: 1, FullName: "empl_1", Position: "staff", DepartmentID: 1},
				{ID: 2, FullName: "empl_2", Position: "staff", DepartmentID: 1},
			},
			Children: []*N{
				{Department: &D{ID: 3, Name: "grandchild_1", ParentID: 1}},
				{Department: &D{ID: 4, Name: "grandchild_2", ParentID: 1}},
			},
		})
	})

	t.Run("fail reassign to self", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		err = svc.DeleteDepartment(ctx, DR{ID: 1, ReassignToDepartmentID: 1})
		be.Err(t, err, model.ErrValidation, model.ErrConflict)
	})

	t.Run("fail reassign to child", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 2)

		err = svc.DeleteDepartment(ctx, DR{ID: 1, ReassignToDepartmentID: 2})
		be.Err(t, err, model.ErrConflict)
	})

	t.Run("fail reassign with duplicate name", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 2)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "duplicate", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 3)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "duplicate", ParentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 4)

		err = svc.DeleteDepartment(ctx, DR{ID: 2, ReassignToDepartmentID: 1})
		be.Err(t, err, model.ErrConflict)
	})

	t.Run("reassign to parent", func(t *testing.T) {
		svc := service.New(newStorage(t))

		gotD, err := svc.CreateDepartment(ctx, D{Name: "top_1", ParentID: -1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 1)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 1})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 2)

		gotD, err = svc.CreateDepartment(ctx, D{Name: "child_1", ParentID: 2})
		be.Err(t, err, nil)
		be.Equal(t, gotD.ID, 3)

		err = svc.DeleteDepartment(ctx, DR{ID: 2, ReassignToDepartmentID: 1})
		be.Err(t, err, nil)
		gotN, err := svc.GetDepartmentTree(ctx, GR{ID: 1, Depth: 5, IncludeEmployees: true, SortByName: true})
		be.Err(t, err, nil)
		be.Equal(be.Diff(t), zeroedTimeN(gotN), &N{
			Department: &D{ID: 1, Name: "top_1", ParentID: -1},
			Children: []*N{
				{Department: &D{ID: 3, Name: "child_1", ParentID: 1}},
			},
		})
	})
}
