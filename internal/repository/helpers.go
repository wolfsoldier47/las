package repository

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// uuidPlaceholders builds a ($n,$n+1,...) placeholder list and matching args
// for a slice of UUIDs, so batch queries can use IN clauses without relying
// on driver-specific array types.
func uuidPlaceholders(start int, ids []uuid.UUID) (string, []interface{}) {
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", start+i)
		args[i] = id
	}
	return "(" + strings.Join(placeholders, ",") + ")", args
}
