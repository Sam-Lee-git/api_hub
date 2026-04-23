package sqlite

import (
	"fmt"
	"time"
)

// timeFormats covers both SQLite's default datetime('now') output and explicit RFC3339 strings.
var timeFormats = []string{
	"2006-01-02 15:04:05",
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
}

// sqlTime scans a TEXT SQLite column into time.Time.
type sqlTime struct{ T time.Time }

func (st *sqlTime) Scan(src interface{}) error {
	if src == nil {
		return nil
	}
	var s string
	switch v := src.(type) {
	case string:
		s = v
	case []byte:
		s = string(v)
	case time.Time:
		st.T = v.UTC()
		return nil
	default:
		return fmt.Errorf("sqlTime: cannot scan %T", src)
	}
	for _, f := range timeFormats {
		if t, err := time.Parse(f, s); err == nil {
			st.T = t.UTC()
			return nil
		}
	}
	return fmt.Errorf("sqlTime: cannot parse %q as time", s)
}

// sqlNullTime scans a nullable TEXT SQLite column into *time.Time.
type sqlNullTime struct{ T *time.Time }

func (snt *sqlNullTime) Scan(src interface{}) error {
	if src == nil {
		snt.T = nil
		return nil
	}
	var st sqlTime
	if err := st.Scan(src); err != nil {
		return err
	}
	snt.T = &st.T
	return nil
}
