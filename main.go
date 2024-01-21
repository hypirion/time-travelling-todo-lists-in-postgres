package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	g "github.com/maragudk/gomponents"
	"github.com/sirupsen/logrus"
)

func main() {
	// obviously, don't store passwords etc in your source code, this is just an
	// example
	db, err := sqlx.Open("postgres", "postgres://postgres:mySecretPassword@localhost:10840/postgres?sslmode=disable")
	if err != nil {
		panic(err)
	}

	runMigrations(db.DB)

	s := newServer(db)

	s.GETWithTx("/", indexHandler)
	s.POSTWithTx("/todo-lists", postTodoListHandler)
	s.GETWithTx("/todo-lists/:tlid", getTodoListHandler)
	s.POSTWithTx("/todo-lists/:tlid/delete", deleteTodoListHandler)
	s.POSTWithTx("/todo-lists/:tlid/new-todos", newTodosHandler)
	s.GETWithTx("/todo-lists/:tlid/revisions", getTodoListRevisionsHandler)

	s.POSTWithTx("/todos/:tid/complete", completeTodoHandler)
	s.POSTWithTx("/todos/:tid/reactivate", reactivateTodoHandler)
	s.POSTWithTx("/todos/:tid/delete", deleteTodoHandler)

	s.GETWithTx("/todo-lists-history/:tlhid", getTodoListRevisionHandler)
	s.POSTWithTx("/todo-lists-history/:tlhid/restore", restoreTodoListRevisionHandler)

	s.POSTWithTx("/todos-history/:thid/restore", restoreTodoRevisionHandler)

	s.Run()
}

type Context struct {
	*gin.Context
	Tx *Tx
}

type server struct {
	router *gin.Engine
	db     *sqlx.DB
}

func newServer(db *sqlx.DB) *server {
	s := &server{
		router: gin.New(),
		db:     db,
	}
	s.router.Use(gin.Recovery())
	return s
}

func (s *server) Run() {
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: s.router,
	}

	logrus.Info("Starting HTTP API on port ", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logrus.WithError(err).Fatal()
	}
}

func (s *server) GETWithTx(path string, handler func(*Context) (g.Node, error)) {
	s.router.GET(path, s.wrapInTx(nodeHandler(handler)))
}

func (s *server) POSTWithTx(path string, handler func(*Context) error) {
	s.router.POST(path, s.wrapInTx(handler))
}

func nodeHandler(handler func(*Context) (g.Node, error)) func(*Context) error {
	return func(c *Context) error {
		node, err := handler(c)
		if err != nil {
			return err
		}
		node.Render(c.Context.Writer)
		return nil
	}
}

func (s *server) wrapInTx(handler func(*Context) error) gin.HandlerFunc {
	return func(gc *gin.Context) {
		err := RunInTx(gc.Request.Context(), s.db, func(tx *Tx) error {
			return handler(&Context{
				Context: gc,
				Tx:      tx,
			})
		})
		if err != nil {
			gc.Status(500)
			errorNode(err).Render(gc.Writer)
		}
	}
}
