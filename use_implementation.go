package kitcat

import (
	"fmt"
	"strings"
)

type UseImplementationParams[T Nameable] struct {
	ModuleName                string
	ImplementationTerminology string
	ConfigImplementationName  string
	Implementations           []T
}

// UseImplementation is a helper function to choose an implementation for a module.
func UseImplementation[T Nameable](params UseImplementationParams[T]) (T, error) {
	retDefault := new(T)

	var availableImplems []string

	for _, implem := range params.Implementations {
		availableImplems = append(availableImplems, implem.Name())
	}

	if params.ConfigImplementationName == "" && len(availableImplems) > 1 {
		return *retDefault, fmt.Errorf(
			"%s: you must set a %s, available: %s",
			params.ModuleName,
			params.ImplementationTerminology,
			strings.Join(availableImplems, ", "))
	}

	impl := new(T)

	if len(availableImplems) == 1 {
		impl = &params.Implementations[0]
	} else {
		for _, c := range params.Implementations {
			if c.Name() == params.ConfigImplementationName {
				impl = &c
				break
			}
		}
	}

	if impl == nil {
		return *retDefault, fmt.Errorf(
			"%s: invalid %s %q, available: %s",
			params.ModuleName,
			params.ImplementationTerminology,
			params.ConfigImplementationName,
			strings.Join(availableImplems, ", "),
		)
	}

	return *impl, nil
}
