package zql

import "database/sql"

type Row struct {
	actual *sql.Rows
}

func (r Row) Scan(dest ...any) error {
	return r.actual.Scan(dest)
}

func (r Row) Err() error {
	return r.actual.Err()
}

// AllRows is a convenience func for leveraging the
// Rangefunc experiment to loop over sql rows as a
// traditional for range loop.
//
// so, generally instead of doing:
//
//	for rows.Next() {
//		rows.Scan()
//	}
//
// code would instead look like:
//
//	for _, row := range AllRows(rows) {
//		row.Scan()
//	}
//
// more can be found here: https://go.dev/wiki/RangefuncExperiment
func AllRows(rows *sql.Rows) func(func(int, Row) bool) {
	return func(yield func(int, Row) bool) {
		i := 0
		for rows.Next() {
			if !yield(i, Row{
				actual: rows,
			}) {
				return
			}
			i++
		}
		// not super sure if i want to implement this way.
		/*
			if err := rows.Err(); err != nil {
				yield(i, Row{
					actual: rows,
				})
			}
		*/
	}
}
