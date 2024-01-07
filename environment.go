package kitcat

import (
	"errors"
)

var (
	ErrInvalidEnvironment = errors.New("invalid environment")
)

type Environment struct {
	Name string
}

func (e *Environment) UnmarshalText(text []byte) error {
	name := string(text)
	switch name {
	case "development":
		*e = EnvironmentDevelopment
	case "production":
		*e = EnvironmentProduction
	case "test":
		*e = EnvironmentTest
	default:
		return ErrInvalidEnvironment
	}

	return nil
}

func (e *Environment) String() string {
	return e.Name
}

func (e *Environment) Equal(development Environment) bool {
	return e.Name == development.Name
}

var (
	EnvironmentDevelopment = Environment{Name: "development"}
	EnvironmentProduction  = Environment{Name: "production"}
	EnvironmentTest        = Environment{Name: "test"}
)

var AllEnvironments = []Environment{
	EnvironmentDevelopment,
	EnvironmentProduction,
	EnvironmentTest,
}
