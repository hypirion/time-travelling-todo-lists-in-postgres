package main

import (
	"database/sql"
	"errors"
	"time"
)

type TodoListRevisionBase struct {
	HistoryID TodoListHistoryID `db:"history_id"`
	SysLower  time.Time         `db:"sys_lower"`
	SysUpper  *time.Time        `db:"sys_upper"`
}

type TodoListRevision struct {
	TodoListRevisionBase
	TodoList
	Todos TodoRevisions
}

// hmm, the OnAlias idea broke down here :(
var todoListRevisionBaseCols = TableColumns{
	"tlh.history_id", "LOWER(tlh.systime) AS sys_lower", "UPPER(tlh.systime) AS sys_upper",
}

func GetTodoListRevisions(tx *Tx, tlid TodoListID) ([]TodoListRevisionBase, error) {
	var revs []TodoListRevisionBase
	err := tx.Select(&revs, `
SELECT `+todoListRevisionBaseCols.String()+`
FROM todo_lists_history tlh
WHERE tlh.todo_list_id = :tlid
ORDER BY systime DESC`, QueryArgs{
		"tlid": tlid,
	})
	if err != nil {
		return nil, err
	}
	return revs, nil
}

var todoListRevisionCols = todoListRevisionBaseCols.Concat(todoListCols.OnAlias("tlh"))

func GetTodoListRevisionAsOf(tx *Tx, tlid TodoListID, asOf time.Time) (*TodoListRevision, error) {
	// not used anywhere in the current app, but implemented to show how it's
	// done.
	var tlr TodoListRevision
	err := tx.Get(&tlr, `
SELECT `+todoListRevisionCols.String()+`
FROM todo_lists_history tlh
WHERE tlh.todo_list_id = :tlid
  AND th.systime @> CAST(:as_of AS timestamptz)`, QueryArgs{
		"tlid":  tlid,
		"as_of": asOf,
	})
	if err != nil {
		return nil, err
	}
	err = tlr.attachTodos(tx, asOf)
	if err != nil {
		return nil, err
	}
	return &tlr, nil
}

func GetTodoListRevisionByID(tx *Tx, tlhid TodoListHistoryID) (*TodoListRevision, error) {
	var tlr TodoListRevision
	err := tx.Get(&tlr, `
SELECT `+todoListRevisionCols.String()+`
FROM todo_lists_history tlh
WHERE tlh.history_id = :tlhid`, QueryArgs{
		"tlhid": tlhid,
	})
	if err != nil {
		return nil, err
	}
	err = tlr.attachTodos(tx, tlr.SysLower)
	if err != nil {
		return nil, err
	}
	return &tlr, nil
}

func RestoreTodoListToRevision(tx *Tx, tlhid TodoListHistoryID) (*TodoListID, error) {
	tlr, err := GetTodoListRevisionByID(tx, tlhid)
	if err != nil {
		return nil, err
	}

	err = DeleteTodoList(tx, tlr.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		// allow to restore deleted todo lists
		return nil, err
	}

	// insert the todo list from the history table
	err = tx.UpdateOne(`
INSERT INTO todo_lists (`+todoListCols.String()+`)
SELECT `+todoListCols.OnAlias("tlh").String()+`
FROM todo_lists_history tlh
WHERE tlh.history_id = :tlhid`, QueryArgs{
		"tlhid": tlhid,
	})
	if err != nil {
		return nil, err
	}

	// then insert the list elements
	err = tx.Exec(`
INSERT INTO todos (`+todoCols.String()+`)
SELECT `+todoCols.OnAlias("th").String()+`
FROM todos_history th
WHERE th.todo_list_id = :list_id
  AND th.systime @> CAST(:as_of AS timestamptz)`, QueryArgs{
		"list_id": tlr.ID,
		"as_of":   tlr.SysLower,
	})

	if err != nil {
		return nil, err
	}
	return &tlr.ID, nil
}

type TodoRevision struct {
	HistoryID TodoHistoryID `db:"history_id"`
	SysLower  time.Time     `db:"sys_lower"`
	SysUpper  *time.Time    `db:"sys_upper"`
	Todo
}

type TodoRevisions []TodoRevision

func (ts TodoRevisions) FilterByCompleted(completed bool) TodoRevisions {
	var res TodoRevisions
	for _, todo := range ts {
		if todo.Completed == completed {
			res = append(res, todo)
		}
	}
	return res
}

var todoRevisionBaseCols = TableColumns{
	"th.history_id", "LOWER(th.systime) AS sys_lower", "UPPER(th.systime) AS sys_upper",
}

var todoRevisionCols = todoRevisionBaseCols.Concat(todoCols.OnAlias("th"))

func (tlr *TodoListRevision) attachTodos(tx *Tx, asOf time.Time) error {
	// passing in asOf doesn't really do anthing valuable in this implementation:
	// Since we always update the todo list's updated_at field whenever we do
	// something with its todos, we may as well use the creation time of the
	// revision. However, if you don't need to maintain a list of revisions for
	// the list itself, but is rather interested in the state at a specific point
	// in time, you need the asOf as input.
	err := tx.Select(&tlr.Todos, `
SELECT `+todoRevisionCols.String()+`
FROM todos_history th
WHERE th.todo_list_id = :tlid
  AND th.systime @> CAST(:as_of AS timestamptz)
ORDER BY th.description ASC`, QueryArgs{
		"tlid":  tlr.ID,
		"as_of": asOf,
	})
	return err
}

func GetTodoRevisionByID(tx *Tx, thid TodoHistoryID) (*TodoRevision, error) {
	var tr TodoRevision
	err := tx.Get(&tr, `
SELECT `+todoRevisionCols.String()+`
FROM todos_history th
WHERE th.history_id = :thid`, QueryArgs{
		"thid": thid,
	})
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

func GetTodoRevisionAsOf(tx *Tx, tid TodoID, asOf time.Time) (*TodoRevision, error) {
	// not used anywhere in the app as of right now, but present to show you how
	// it's implemented.
	var tr TodoRevision
	err := tx.Get(&tr, `
SELECT `+todoRevisionCols.String()+`
FROM todos_history th
WHERE th.todo_id = :tid
  AND th.systime @> CAST(:as_of AS timestamptz)`, QueryArgs{
		"tid":   tid,
		"as_of": asOf,
	})
	if err != nil {
		return nil, err
	}
	return &tr, nil
}

func RestoreTodoToRevision(tx *Tx, thid TodoHistoryID) (*TodoListID, error) {
	tr, err := GetTodoRevisionByID(tx, thid)
	if err != nil {
		return nil, err
	}

	// delete it (if it still is in the list)
	err = DeleteTodo(tx, tr.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	// then insert it back into the list
	err = tx.Exec(`
INSERT INTO todos (`+todoCols.String()+`)
SELECT `+todoCols.OnAlias("th").String()+`
FROM todos_history th
WHERE th.history_id = :thid`, QueryArgs{
		"thid": thid,
	})

	if err != nil {
		return nil, err
	}
	return &tr.ListID, nil
}
