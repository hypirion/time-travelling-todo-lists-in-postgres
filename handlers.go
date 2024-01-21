package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	g "github.com/maragudk/gomponents"
	c "github.com/maragudk/gomponents/components"
	. "github.com/maragudk/gomponents/html"
)

func indexHandler(ctx *Context) (g.Node, error) {
	tls, err := GetAllTodoLists(ctx.Tx)
	if err != nil {
		return nil, err
	}

	return pageNode("Todo Lists",
		[]g.Node{
			H1(g.Text("Your Todo Lists")),
			todoListTable(tls),
			newTodoListForm(),
		},
	), nil
}

func newTodoListForm() g.Node {
	return FormEl(Method("post"), Action("/todo-lists"),
		H3(g.Text("Make a new list")),
		Label(For("name"), g.Text("Name of new list:")),
		Input(Type("text"), Name("name"), Required()),
		Button(g.Text("Create")))
}

func todoListTable(tls []TodoListBase) g.Node {
	return table(
		[]string{"Name", "Created", "Last Updated", ""},
		g.Map(tls, todoListRow),
	)
}

func todoListRow(tl TodoListBase) g.Node {
	return Tr(
		Td(A(Href(tl.ID.Href()), g.Text(tl.Name))),
		Td(g.Text(fmtTime(tl.CreatedAt))),
		Td(g.Text(fmtTime(tl.UpdatedAt))),
		Td(postButton(tl.ID.HrefTo("delete"), "delete")),
	)
}

func postTodoListHandler(ctx *Context) error {
	name, ok := ctx.GetPostForm("name")
	name = strings.TrimSpace(name)
	if !ok || name == "" {
		return errors.New("must have a nonempty name")
	}

	tl, err := NewTodoList(ctx.Tx, name)
	if err != nil {
		return err
	}

	ctx.Redirect(http.StatusSeeOther, tl.ID.Href())
	return nil
}

func getTodoListHandler(ctx *Context) (g.Node, error) {
	var tlid TodoListID
	err := tlid.Parse(ctx.Param("tlid"))
	if err != nil {
		return nil, err
	}

	tl, err := GetTodoListByID(ctx.Tx, tlid)
	if err != nil {
		return nil, err
	}

	completed := tl.Todos.FilterByCompleted(true)
	unfinished := tl.Todos.FilterByCompleted(false)

	return pageNode("Todo List - "+tl.Name,
		[]g.Node{
			H1(g.Text(tl.Name)),
			newTodosForm(tlid),
			g.If(len(unfinished) != 0, g.Group([]g.Node{
				H3(g.Text("Todos")),
				Table(TBody(g.Map(unfinished, unfinishedTodoRow)...)),
			})),
			g.If(len(completed) != 0, g.Group([]g.Node{
				H3(g.Text("Completed")),
				Table(TBody(g.Map(completed, completedTodoRow)...)),
			})),
			P(A(Href(tl.ID.HrefTo("revisions")), g.Text("Revisions"))),
		},
	), nil
}

func newTodosForm(tlid TodoListID) g.Node {
	return FormEl(Method("post"), Action(tlid.HrefTo("new-todos")),
		Label(For("new-todos"), g.Text("Make new todos (comma separated):")),
		Input(Type("text"), Name("new-todos"), Required()),
		Button(g.Text("Add")))
}

func unfinishedTodoRow(todo Todo) g.Node {
	return Tr(
		Td(g.Text(todo.Description)),
		Td(postButton(todo.ID.HrefTo("complete"), "Complete"),
			postButton(todo.ID.HrefTo("delete"), "Delete")),
	)
}
func completedTodoRow(todo Todo) g.Node {
	return Tr(
		Td(S(g.Text(todo.Description))),
		Td(postButton(todo.ID.HrefTo("reactivate"), "Reactivate"),
			postButton(todo.ID.HrefTo("delete"), "Delete")),
	)
}

func deleteTodoListHandler(ctx *Context) error {
	var tlid TodoListID
	err := tlid.Parse(ctx.Param("tlid"))
	if err != nil {
		return err
	}

	err = DeleteTodoList(ctx.Tx, tlid)
	if err != nil {
		return err
	}
	ctx.Redirect(http.StatusSeeOther, "/")
	return nil
}

func newTodosHandler(ctx *Context) error {
	var tlid TodoListID
	err := tlid.Parse(ctx.Param("tlid"))
	if err != nil {
		return err
	}

	newTodosStr, ok := ctx.GetPostForm("new-todos")
	newTodosStr = strings.TrimSpace(newTodosStr)
	if !ok || newTodosStr == "" {
		return errors.New("must have some todos")
	}

	todos := strings.Split(newTodosStr, ",")
	for _, todo := range todos {
		todo = strings.TrimSpace(todo)
		if todo == "" {
			continue // yeah sure, may end up with no more todos this way
		}
		err = NewTodo(ctx.Tx, tlid, todo)
		if err != nil {
			return err
		}
	}

	ctx.Redirect(http.StatusSeeOther, tlid.Href())

	return nil
}

func getTodoListRevisionsHandler(ctx *Context) (g.Node, error) {
	var tlid TodoListID
	err := tlid.Parse(ctx.Param("tlid"))
	if err != nil {
		return nil, err
	}

	revs, err := GetTodoListRevisions(ctx.Tx, tlid)
	if err != nil {
		return nil, err
	}

	if len(revs) == 0 {
		return nil, fmt.Errorf("no revisions for todo list with ID %s", tlid)
	}

	rows := make([]g.Node, len(revs))
	for i, rev := range revs {
		revID := len(revs) - i
		rows[i] = todoListRevisionRow(rev, revID)
	}

	return pageNode("Todo List Revisions for "+tlid.String(),
		[]g.Node{
			H1(g.Text("Todo List Revisions for " + tlid.String())),
			table([]string{"Revision", "Valid from", "Valid to"},
				rows),
		},
	), nil
}

func todoListRevisionRow(tlhb TodoListRevisionBase, versionID int) g.Node {
	sysUpper := ""
	if tlhb.SysUpper != nil {
		sysUpper = fmtTime(*tlhb.SysUpper)
	}

	return Tr(
		Td(A(Href(tlhb.HistoryID.Href()), g.Text("#"+strconv.Itoa(versionID)))),
		Td(g.Text(fmtTime(tlhb.SysLower))),
		Td(g.Text(sysUpper)))
}

func completeTodoHandler(ctx *Context) error {
	var tid TodoID
	err := tid.Parse(ctx.Param("tid"))
	if err != nil {
		return err
	}

	err = SetTodoCompleted(ctx.Tx, tid, true)
	if err != nil {
		return err
	}

	todo, err := GetTodoByID(ctx.Tx, tid)
	if err != nil {
		return err
	}

	ctx.Redirect(http.StatusSeeOther, todo.ListID.Href())
	return nil
}

func reactivateTodoHandler(ctx *Context) error {
	var tid TodoID
	err := tid.Parse(ctx.Param("tid"))
	if err != nil {
		return err
	}

	err = SetTodoCompleted(ctx.Tx, tid, false)
	if err != nil {
		return err
	}

	todo, err := GetTodoByID(ctx.Tx, tid)
	if err != nil {
		return err
	}

	ctx.Redirect(http.StatusSeeOther, todo.ListID.Href())
	return nil
}

func deleteTodoHandler(ctx *Context) error {
	var tid TodoID
	err := tid.Parse(ctx.Param("tid"))
	if err != nil {
		return err
	}

	todo, err := GetTodoByID(ctx.Tx, tid)
	if err != nil {
		return err
	}

	err = DeleteTodo(ctx.Tx, tid)
	if err != nil {
		return err
	}

	ctx.Redirect(http.StatusSeeOther, todo.ListID.Href())
	return nil
}

func getTodoListRevisionHandler(ctx *Context) (g.Node, error) {
	var tlhid TodoListHistoryID
	err := tlhid.Parse(ctx.Param("tlhid"))
	if err != nil {
		return nil, err
	}

	todoListRev, err := GetTodoListRevisionByID(ctx.Tx, tlhid)
	if err != nil {
		return nil, err
	}

	current, err := GetTodoListByID(ctx.Tx, todoListRev.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	completed := todoListRev.Todos.FilterByCompleted(true)
	unfinished := todoListRev.Todos.FilterByCompleted(false)

	revRenderer := newTodoRevisionRenderer(current)

	return pageNode("Revision of "+todoListRev.Name,
		[]g.Node{
			H1(g.Text(todoListRev.Name)),
			P(A(Href(todoListRev.ID.Href()), g.Text("[current version]")), g.Text(" "),
				A(Href(todoListRev.ID.HrefTo("revisions")), g.Text("[list revisions]")), g.Text(" "),
				g.If(!revRenderer.equalTodos(todoListRev.Todos),
					postButton(todoListRev.HistoryID.HrefTo("restore"), "Restore list to this revision"))),
			g.If(len(unfinished) != 0, g.Group([]g.Node{
				H3(g.Text("Todos")),
				Table(TBody(g.Map(unfinished, revRenderer.unfinishedRow)...)),
			})),
			g.If(len(completed) != 0, g.Group([]g.Node{
				H3(g.Text("Completed")),
				Table(TBody(g.Map(completed, revRenderer.completedRow)...)),
			})),
		},
	), nil
}

func newTodoRevisionRenderer(current *TodoList) todoRevisionRenderer {
	trr := todoRevisionRenderer{
		canRestoreTodos: current != nil,
	}
	if current != nil {
		trr.todoMap = map[TodoID]Todo{}
		for _, todo := range current.Todos {
			trr.todoMap[todo.ID] = todo
		}
	}
	return trr
}

type todoRevisionRenderer struct {
	canRestoreTodos bool
	todoMap         map[TodoID]Todo
}

func (trr todoRevisionRenderer) equalTodos(todos TodoRevisions) bool {
	if len(trr.todoMap) != len(todos) {
		return false
	}
	for _, todo := range todos {
		if trr.revisionIsStale(todo) {
			return false
		}
	}
	return true
}

func (trr todoRevisionRenderer) revisionIsStale(todo TodoRevision) bool {
	// Here we match on contents and the identity, so e.g. making a new todo item
	// with the same description and complete status will mean you can end up with
	// duplicates. Anything not comparing on identity/primary key is gonna make
	// the restore procedure a bit more complicated.
	if todo.SysUpper == nil {
		return false
	}
	curTodo, ok := trr.todoMap[todo.ID]

	// check for presence and whether they are identical
	return !ok || curTodo != todo.Todo
}

func (trr todoRevisionRenderer) unfinishedRow(todo TodoRevision) g.Node {
	return Tr(
		Td(g.Text(todo.Description)),
		Td(g.If(trr.canRestoreTodos && trr.revisionIsStale(todo),
			postButton(todo.HistoryID.HrefTo("restore"), "Restore Todo to this state"))),
	)
}
func (trr todoRevisionRenderer) completedRow(todo TodoRevision) g.Node {
	return Tr(
		Td(S(g.Text(todo.Description))),
		Td(g.If(trr.canRestoreTodos && trr.revisionIsStale(todo),
			postButton(todo.HistoryID.HrefTo("restore"), "Restore Todo to this state"))),
	)
}

func restoreTodoListRevisionHandler(ctx *Context) error {
	var tlhid TodoListHistoryID
	err := tlhid.Parse(ctx.Param("tlhid"))
	if err != nil {
		return err
	}

	listID, err := RestoreTodoListToRevision(ctx.Tx, tlhid)
	if err != nil {
		return err
	}

	ctx.Redirect(http.StatusSeeOther, listID.Href())
	return nil
}

func restoreTodoRevisionHandler(ctx *Context) error {
	var thid TodoHistoryID
	err := thid.Parse(ctx.Param("thid"))
	if err != nil {
		return err
	}

	_, err = RestoreTodoToRevision(ctx.Tx, thid)
	if err != nil {
		return err
	}

	// remain at the same location. It's not possible to deduce that without
	// passing in timestamp or the todo list history id, so we just use Referer.
	ctx.Redirect(http.StatusSeeOther, ctx.Request.Referer())
	return nil
}

func errorNode(err error) g.Node {
	return pageNode("Error",
		[]g.Node{
			H1(g.Text("an error occurred")),
			P(g.Text(err.Error())),
		},
	)
}

func pageNode(title string, body []g.Node) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    title,
		Language: "en",
		Head: []g.Node{
			Link(Rel("stylesheet"), Href("https://cdn.jsdelivr.net/npm/sakura.css/css/sakura.css"), Type("text/css")),
			StyleEl(g.Text(`
.inline-form { display: inline; padding-right: 1em; }
`)),
		},
		Body: []g.Node{
			Nav(A(Href("/"), g.Text("Home"))),
			g.Group(body),
		},
	})
}

func postButton(url, text string) g.Node {
	return FormEl(Class("inline-form"),
		Method("post"), Action(url), Button(g.Text(text)))
}

func fmtTime(t time.Time) string {
	return t.Local().Format(time.DateTime)
}

func table(headers []string, body []g.Node) g.Node {
	return Table(
		THead(Tr(g.Map(headers, func(s string) g.Node { return Th(g.Text(s)) })...)),
		TBody(body...),
	)
}
