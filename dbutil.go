package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// this file contains helper functions to make the SQL queries easier to read.

type Tx struct {
	tx *sqlx.Tx
}

type TableColumns []string

func (tb TableColumns) Concat(other TableColumns) TableColumns {
	ret := make(TableColumns, len(tb)+len(other))
	copy(ret, tb)
	copy(ret[len(tb):], other)
	return ret
}

func (tb TableColumns) OnAlias(alias string) TableColumns {
	ret := make(TableColumns, len(tb))
	for i, val := range tb {
		ret[i] = alias + "." + val
	}
	return ret
}

func (tb TableColumns) String() string {
	return strings.Join(tb, ", ")
}

type QueryArgs map[string]any

func (tx *Tx) Exec(query string, args QueryArgs) error {
	translatedQuery, sliceArgs, err := tx.tx.BindNamed(query, args)
	if err != nil {
		return err
	}
	_, err = tx.tx.Exec(translatedQuery, sliceArgs...)
	return err
}

func (tx *Tx) Get(dest any, query string, args QueryArgs) error {
	translatedQuery, sliceArgs, err := tx.tx.BindNamed(query, args)
	if err != nil {
		return err
	}
	return tx.tx.Get(dest, translatedQuery, sliceArgs...)
}

func (tx *Tx) Select(dest any, query string, args QueryArgs) error {
	translatedQuery, sliceArgs, err := tx.tx.BindNamed(query, args)
	if err != nil {
		return err
	}
	return tx.tx.Select(dest, translatedQuery, sliceArgs...)
}

func (tx *Tx) DeleteOne(query string, args QueryArgs) error {
	return tx.UpdateOne(query, args)
}

func (tx *Tx) UpdateOne(query string, args QueryArgs) error {
	translatedQuery, sliceArgs, err := tx.tx.BindNamed(query, args)
	if err != nil {
		return err
	}

	res, err := tx.tx.Exec(translatedQuery, sliceArgs...)
	if err != nil {
		return err
	}

	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// to allow to match on ErrNoRows.
	if affected == 0 {
		return fmt.Errorf("query affected 0 rows, should affect one: %w", sql.ErrNoRows)
	}

	if affected != 1 {
		return fmt.Errorf("query affected %d rows, but should only affect one", affected)
	}

	return nil
}

func RunInTx(ctx context.Context, db *sqlx.DB, f func(tx *Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	didPanic := true
	defer func() {
		if didPanic {
			err := tx.Rollback()
			if err != nil {
				logrus.WithError(err).Info("failed to rollback transaction during panic")
			}
		}
	}()
	err = f(&Tx{tx: tx})

	didPanic = false
	if err != nil {
		err2 := tx.Rollback()
		if err2 != nil {
			logrus.WithError(err).Info("failed to rollback transaction after error")
		}

		return err
	}

	err = tx.Commit()
	return err
}
