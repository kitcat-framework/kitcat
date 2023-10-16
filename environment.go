package kitcat

import "errors"

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
		*e = Development
	case "production":
		*e = Production
	default:
		return ErrInvalidEnvironment
	}

	return nil
}

func (e *Environment) String() string {
	return e.Name
}

var (
	Development = Environment{Name: "development"}
	Production  = Environment{Name: "production"}
)
