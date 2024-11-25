package errs

import (
	"errors"
	"fmt"

	"github.com/lib/pq"
)

var (
	ErrNotFound       = errors.New("Entry not found!")
	ErrInternalServer = errors.New("Internal server error!")
)

type ErrUniqueValueViolation struct {
	Field string `json:"field"`
	Value any    `json:"value"`
}

func (e *ErrUniqueValueViolation) Error() string {
	return fmt.Sprintf("Duplicate value for %s: %v", e.Field, e.Value)
}

var pqUniqueConstraintsMap map[string]string = map[string]string{
	"users_email_key":    "email",
	"users_username_key": "username",
}

func MapPostgresError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) {
		return err
	}

	switch pqErr.Code {
	case "23505": // unique_violation
		return &ErrUniqueValueViolation{
			Field: getFieldFromConstraint(pqErr.Constraint),
			Value: pqErr.Detail,
		}
	default:
		return err
	}
}

func getFieldFromConstraint(constraint string) string {
	if field, ok := pqUniqueConstraintsMap[constraint]; ok {
		return field
	}
	return constraint
}
