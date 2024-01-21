package main

import (
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jxskiss/base62"
)

// trick to get Stripe-like identifiers. Uses base62 because - and _ can mess up
// copypasting etc.
type idUtil uuid.UUID

func (id idUtil) str(prefix string) string {
	return prefix + "_" + base62.StdEncoding.EncodeToString(id[:])
}

func (id *idUtil) fromStr(prefix string, data string) error {
	if !strings.HasPrefix(data, prefix+"_") {
		return fmt.Errorf("id %s did not start with %s", data, prefix)
	}
	idStr := data[len(prefix)+1:]
	bs, err := base62.StdEncoding.DecodeString(idStr)
	if err != nil {
		return err
	}
	if len(bs) != 16 {
		return fmt.Errorf("id %s has unexpected length", data)
	}
	copy(id[:], bs)
	return nil
}

type TodoListID uuid.UUID

// Value implements the sql.Valuer interface
func (id TodoListID) Value() (driver.Value, error) {
	return uuid.UUID(id).Value()
}

// Scan implements the sql.Scanner interface
func (id *TodoListID) Scan(value interface{}) error {
	return (*uuid.UUID)(id).Scan(value)
}

// String implements the Stringer interface.
func (id TodoListID) String() string {
	return idUtil(id).str("tl")
}

func (id *TodoListID) Parse(str string) error {
	return (*idUtil)(id).fromStr("tl", str)
}

func (id TodoListID) Href() string {
	return fmt.Sprintf("/todo-lists/%s", id)
}

func (id TodoListID) HrefTo(action string) string {
	return fmt.Sprintf("/todo-lists/%s/%s", id, action)
}

type TodoListHistoryID uuid.UUID

// Value implements the sql.Valuer interface
func (id TodoListHistoryID) Value() (driver.Value, error) {
	return uuid.UUID(id).Value()
}

// Scan implements the sql.Scanner interface
func (id *TodoListHistoryID) Scan(value interface{}) error {
	return (*uuid.UUID)(id).Scan(value)
}

// String implements the Stringer interface.
func (id TodoListHistoryID) String() string {
	return idUtil(id).str("tl_hist")
}

func (id *TodoListHistoryID) Parse(str string) error {
	return (*idUtil)(id).fromStr("tl_hist", str)
}

func (id TodoListHistoryID) Href() string {
	return fmt.Sprintf("/todo-lists-history/%s", id)
}

func (id TodoListHistoryID) HrefTo(action string) string {
	return fmt.Sprintf("/todo-lists-history/%s/%s", id, action)
}

type TodoID uuid.UUID

// Value implements the sql.Valuer interface
func (id TodoID) Value() (driver.Value, error) {
	return uuid.UUID(id).Value()
}

// Scan implements the sql.Scanner interface
func (id *TodoID) Scan(value interface{}) error {
	return (*uuid.UUID)(id).Scan(value)
}

// String implements the Stringer interface.
func (id TodoID) String() string {
	return idUtil(id).str("todo")
}

func (id *TodoID) Parse(str string) error {
	return (*idUtil)(id).fromStr("todo", str)
}

func (id TodoID) Href() string {
	return fmt.Sprintf("/todos/%s", id)
}

func (id TodoID) HrefTo(action string) string {
	return fmt.Sprintf("/todos/%s/%s", id, action)
}

type TodoHistoryID uuid.UUID

// Value implements the sql.Valuer interface
func (id TodoHistoryID) Value() (driver.Value, error) {
	return uuid.UUID(id).Value()
}

// Scan implements the sql.Scanner interface
func (id *TodoHistoryID) Scan(value interface{}) error {
	return (*uuid.UUID)(id).Scan(value)
}

// String implements the Stringer interface.
func (id TodoHistoryID) String() string {
	return idUtil(id).str("todo_hist")
}

func (id *TodoHistoryID) Parse(str string) error {
	return (*idUtil)(id).fromStr("todo_hist", str)
}

func (id TodoHistoryID) Href(action string) string {
	return fmt.Sprintf("/todos-history/%s", id)
}

func (id TodoHistoryID) HrefTo(action string) string {
	return fmt.Sprintf("/todos-history/%s/%s", id, action)
}
