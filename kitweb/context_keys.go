package kitweb

type ContextKeys int

const (
	ContextKeyEnv ContextKeys = iota
	ContextKeyEngines
	ContextKeyAlreadyRendered
)
