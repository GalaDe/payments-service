package utils

import (
	"database/sql"

	"github.com/gobuffalo/nulls"
	"github.com/guregu/null"
)

func NullsStringToPointer(ns nulls.String) *string {
	if ns.Valid {
		return &ns.String
	}
	return nil
}

func NullsIntToPointer(ns nulls.Int) *int {
	if ns.Valid {
		return &ns.Int
	}
	return nil
}

func NullsBoolToPointer(ns nulls.Bool) *bool {
	if ns.Valid {
		return &ns.Bool
	}
	return nil
}

func PointerToNullsString(s *string) nulls.String {
	if s != nil {
		return nulls.NewString(*s)
	}
	return nulls.String{Valid: false}
}

func PointerToNullsBool(b *bool) nulls.Bool {
	if b != nil {
		return nulls.NewBool(*b)
	}
	return nulls.Bool{Valid: false}
}

func SqlToNullString(ns sql.NullString) null.String {
    if ns.Valid {
        return null.StringFrom(ns.String)
    }
    return null.String{}
}

// Converts null.String to sql.NullString
func NullStringToSQL(s null.String) sql.NullString {
	return sql.NullString{
		String: s.String,
		Valid:  s.Valid,
	}
}

func NullBoolToSQL(b bool) sql.NullBool {
	return sql.NullBool{
		Bool:  b,
	}
}

func NullStringToStr(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func StringToNull(s string) sql.NullString {
    return sql.NullString{String: s, Valid: s != ""}
}