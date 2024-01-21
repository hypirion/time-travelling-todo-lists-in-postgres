package main

import "time"

type TodoListBase struct {
	ID        TodoListID `db:"todo_list_id"`
	Name      string     `db:"name"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
}

type TodoList struct {
	TodoListBase
	Todos Todos
}

var todoListCols = TableColumns{"todo_list_id", "name", "created_at", "updated_at"}

func GetAllTodoLists(tx *Tx) ([]TodoListBase, error) {
	var tls []TodoListBase
	err := tx.Select(&tls, `
SELECT `+todoListCols.OnAlias("tl").String()+`
FROM todo_lists tl
ORDER BY name`, QueryArgs{})
	if err != nil {
		return nil, err
	}
	return tls, nil
}

func GetTodoListByID(tx *Tx, tlid TodoListID) (*TodoList, error) {
	var tl TodoList
	err := tx.Get(&tl, `
SELECT `+todoListCols.OnAlias("tl").String()+`
FROM todo_lists tl
WHERE todo_list_id = :tlid`,
		QueryArgs{"tlid": tlid})

	if err != nil {
		return nil, err
	}
	err = tl.attachTodos(tx)
	if err != nil {
		return nil, err
	}
	return &tl, err
}

func NewTodoList(tx *Tx, name string) (*TodoList, error) {
	var tlid TodoListID
	err := tx.Get(&tlid, `
INSERT INTO todo_lists (name)
VALUES (:name)
RETURNING todo_list_id`, QueryArgs{
		"name": name,
	})
	if err != nil {
		return nil, err
	}
	return GetTodoListByID(tx, tlid)
}

func UpdateTodoList(tx *Tx, tl TodoList) (*TodoList, error) {
	err := tx.UpdateOne(`
UPDATE todo_lists
  SET name = :name
    , updated_at = NOW()
WHERE todo_list_id = :id`, QueryArgs{
		"id":   tl.ID,
		"name": tl.Name,
	})
	if err != nil {
		return nil, err
	}
	return GetTodoListByID(tx, tl.ID)
}

func DeleteTodoList(tx *Tx, tlid TodoListID) error {
	err := tx.DeleteOne(`
DELETE FROM todo_lists
WHERE todo_list_id = :id`, QueryArgs{
		"id": tlid,
	})
	return err
}

type Todo struct {
	ID          TodoID     `db:"todo_id"`
	ListID      TodoListID `db:"todo_list_id"`
	Description string     `db:"description"`
	CreatedAt   time.Time  `db:"created_at"`
	Completed   bool       `db:"completed"`
}

type Todos []Todo

func (ts Todos) FilterByCompleted(completed bool) Todos {
	var res Todos
	for _, todo := range ts {
		if todo.Completed == completed {
			res = append(res, todo)
		}
	}
	return res
}

var todoCols = TableColumns{"todo_id", "todo_list_id", "description", "created_at", "completed"}

func (tl *TodoList) attachTodos(tx *Tx) error {
	err := tx.Select(&tl.Todos, `
SELECT `+todoCols.OnAlias("t").String()+`
FROM todos t
WHERE t.todo_list_id = :tlid
ORDER BY t.description ASC`, QueryArgs{
		"tlid": tl.ID,
	})
	return err
}

func GetTodoByID(tx *Tx, tid TodoID) (*Todo, error) {
	var todo Todo
	err := tx.Get(&todo, `
SELECT `+todoCols.OnAlias("t").String()+`
FROM todos t
WHERE todo_id = :tid`, QueryArgs{
		"tid": tid,
	})
	if err != nil {
		return nil, err
	}
	return &todo, nil
}

func NewTodo(tx *Tx, tlid TodoListID, description string) error {
	var tid TodoID
	err := tx.Get(&tid, `
INSERT INTO todos (todo_list_id, description)
VALUES (:list_id, :description)
RETURNING todo_id`,
		QueryArgs{
			"list_id":     tlid,
			"description": description,
		})
	if err != nil {
		return err
	}
	return touchList(tx, tid)
}

func SetTodoCompleted(tx *Tx, tid TodoID, completed bool) error {
	err := tx.UpdateOne(`
UPDATE todos
   SET completed = :completed
WHERE todo_id = :id`, QueryArgs{
		"id":        tid,
		"completed": completed,
	})
	if err != nil {
		return err
	}
	return touchList(tx, tid)
}

func DeleteTodo(tx *Tx, tid TodoID) error {
	err := touchList(tx, tid)
	if err != nil {
		return err
	}
	err = tx.DeleteOne(`
DELETE FROM todos
WHERE todo_id = :id`, QueryArgs{
		"id": tid,
	})
	return err
}

// touchList bumps the updated_at field on the list this todo is in, forcing a
// new version of the todo list to be stored in the history table.
func touchList(tx *Tx, tid TodoID) error {
	err := tx.UpdateOne(`
UPDATE todo_lists
  SET updated_at = NOW()
WHERE todo_list_id = (SELECT todo_list_id
                      FROM todos
                      WHERE todo_id = :tid)`,
		QueryArgs{
			"tid": tid,
		})
	return err
}
