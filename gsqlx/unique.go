package gsqlx

import (
	"errors"

	"github.com/lib/pq"
	"modernc.org/sqlite"
	sqliteLib "modernc.org/sqlite/lib"
)

//nolint:gochecknoglobals // not sure why it's complaining about an error
var ErrUniqueViolation = pq.ErrorCode("23505") // 'unique_violation'

func IsDuplicateKey(err error) bool {
	var pgerr *pq.Error
	if errors.As(err, &pgerr) {
		return pgerr.Code == ErrUniqueViolation
	}

	var sqliteerr *sqlite.Error
	if errors.As(err, &sqliteerr) {
		return sqliteerr.Code() == sqliteLib.SQLITE_CONSTRAINT_UNIQUE ||
			sqliteerr.Code() == sqliteLib.SQLITE_CONSTRAINT_PRIMARYKEY
	}

	return false
}
