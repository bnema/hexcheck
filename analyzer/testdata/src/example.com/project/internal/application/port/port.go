package port

import "example.com/project/internal/infrastructure/sql"

type UserRepository interface {
	Find() (*sql.Row, error) // want "no-infra-types-in-ports"
}
