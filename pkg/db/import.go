package db

import (
	"bytes"
	"database/sql"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
)

type dbInserter interface {
	insert() (string, []any, error)
}

var ErrImportDone = errors.New("sentinel to stop csv import")

func importCSV[T dbInserter](db *sql.DB, in []byte, skip int, rows []T, parserFn func(*sql.DB, []string) (T, error)) ([]T, error) {
	r := csv.NewReader(bytes.NewReader(in))
	// skip header
	for range skip {
		if _, err := r.Read(); err != nil {
			return nil, err
		}
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if allEmpty(record) {
			continue
		}
		if Verbose {
			log.Println("csv:", record)
		}
		row, err := parserFn(db, record)
		if err == ErrImportDone {
			// do not import more from this csv
			break
		}
		if err != nil {
			return nil, fmt.Errorf("parsing record %v: %w", record, err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}
