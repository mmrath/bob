package expr

import (
	"errors"
	"io"

	"github.com/stephenafamo/typesql/query"
)

var ErrNoLockStrength = errors.New("No lock strength specified")

const (
	LockStrengthUpdate      = "UPDATE"
	LockStrengthNoKeyUpdate = "NO KEY UPDATE"
	LockStrengthShare       = "SHARE"
	LockStrengthKeyShare    = "KEY SHARE"
)

const (
	LockWaitNoWait     = "NOWAIT"
	LockWaitSkipLocked = "SKIP LOCKED"
)

type For struct {
	Strength string
	Tables   []string
	Wait     string
}

func (f *For) SetFor(lock For) {
	f = &lock
}

func (f For) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.Strength == "" {
		return nil, nil
	}

	w.Write([]byte("FOR "))
	w.Write([]byte(f.Strength))

	args, err := query.ExpressSlice(w, d, start, f.Tables, " OF ", ", ", "")
	if err != nil {
		return nil, err
	}

	if f.Wait != "" {
		w.Write([]byte(" "))
		w.Write([]byte(f.Wait))
	}

	return args, nil
}
