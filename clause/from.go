package clause

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

type FromItems struct {
	Items []FromItem
}

func (f *FromItems) AppendFromItem(item FromItem) {
	f.Items = append(f.Items, item)
}

func (f FromItems) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, f.Items, "", ",\n", "")
}

/*
https://www.postgresql.org/docs/current/sql-select.html#SQL-WITH

where from_item can be one of:

    [ ONLY ] table_name [ * ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
                [ TABLESAMPLE sampling_method ( argument [, ...] ) [ REPEATABLE ( seed ) ] ]
    [ LATERAL ] ( select ) [ AS ] alias [ ( column_alias [, ...] ) ]
    with_query_name [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    [ LATERAL ] function_name ( [ argument [, ...] ] )
                [ WITH ORDINALITY ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    [ LATERAL ] function_name ( [ argument [, ...] ] ) [ AS ] alias ( column_definition [, ...] )
    [ LATERAL ] function_name ( [ argument [, ...] ] ) AS ( column_definition [, ...] )
    [ LATERAL ] ROWS FROM( function_name ( [ argument [, ...] ] ) [ AS ( column_definition [, ...] ) ] [, ...] )
                [ WITH ORDINALITY ] [ [ AS ] alias [ ( column_alias [, ...] ) ] ]
    from_item [ NATURAL ] join_type from_item [ ON join_condition | USING ( join_column [, ...] ) [ AS join_using_alias ] ]


SQLite: https://www.sqlite.org/syntax/table-or-subbob.html

MySQL: https://dev.mysql.com/doc/refman/8.0/en/join.html
*/

type FromItem struct {
	Table any

	// Aliases
	Alias   string
	Columns []string

	// Dialect specific modifiers
	Only           bool        // Postgres
	Lateral        bool        // Postgres & MySQL
	WithOrdinality bool        // Postgres
	IndexedBy      *string     // SQLite
	Partitions     []string    // MySQL
	IndexHints     []IndexHint // MySQL

	// Joins
	Joins []Join
}

func (f *FromItem) SetTableAlias(alias string, columns ...string) {
	f.Alias = alias
	f.Columns = columns
}

func (f *FromItem) AppendJoin(j Join) {
	f.Joins = append(f.Joins, j)
}

func (f *FromItem) AppendPartition(partitions ...string) {
	f.Partitions = append(f.Partitions, partitions...)
}

func (f *FromItem) AppendIndexHint(i IndexHint) {
	f.IndexHints = append(f.IndexHints, i)
}

func (f FromItem) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if f.Table == nil {
		return nil, nil
	}

	if f.Only {
		w.Write([]byte("ONLY "))
	}

	if f.Lateral {
		w.Write([]byte("LATERAL "))
	}

	args, err := bob.Express(w, d, start, f.Table)
	if err != nil {
		return nil, err
	}

	if f.WithOrdinality {
		w.Write([]byte(" WITH ORDINALITY"))
	}

	_, err = bob.ExpressSlice(w, d, start, f.Partitions, " PARTITION (", ", ", ")")
	if err != nil {
		return nil, err
	}

	if f.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, f.Alias)
	}

	if len(f.Columns) > 0 {
		w.Write([]byte("("))
		for k, cAlias := range f.Columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	// No args for index hints
	_, err = bob.ExpressSlice(w, d, start+len(args), f.IndexHints, "\n", " ", "")
	if err != nil {
		return nil, err
	}

	switch {
	case f.IndexedBy == nil:
		break
	case *f.IndexedBy == "":
		w.Write([]byte(" NOT INDEXED"))
	default:
		w.Write([]byte(" INDEXED BY "))
		w.Write([]byte(*f.IndexedBy))
	}

	joinArgs, err := bob.ExpressSlice(w, d, start+len(args), f.Joins, "\n", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, joinArgs...)

	return args, nil
}

type IndexHint struct {
	Type    string // USE, FORCE or IGNORE
	Indexes []string
	For     string // JOIN, ORDER BY or GROUP BY
}

func (f IndexHint) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if f.Type == "" {
		return nil, nil
	}
	fmt.Fprintf(w, "%s INDEX ", f.Type)

	_, err := bob.ExpressIf(w, d, start, f.For, f.For != "", " FOR ", "")
	if err != nil {
		return nil, err
	}

	// Always include the brackets
	fmt.Fprint(w, " (")
	_, err = bob.ExpressSlice(w, d, start, f.Indexes, "", ", ", "")
	if err != nil {
		return nil, err
	}
	fmt.Fprint(w, ")")

	return nil, nil
}
