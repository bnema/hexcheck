package usecase

import (
	"example.com/project/internal/application/port"
	"example.com/project/internal/infrastructure/sql" // want "prefer-generated-mocks"
)

type fakeUserRepository struct{} // want "no-local-fakes-for-ports"

func (fakeUserRepository) Find() (*sql.Row, error) { return nil, nil }

var _ port.UserRepository = fakeUserRepository{}
