package kitcache

import (
	"errors"
	"github.com/expectedsh/dig"
	"github.com/expectedsh/kitcat"
	"github.com/expectedsh/kitcat/kitdi"
	"time"
)

var (
	ErrNotFound       = errors.New("kitcache: key not found")
	ErrUnableToSet    = errors.New("kitcache: unable to set key")
	ErrUnableToUpdate = errors.New("kitcache: unable to update key")
)

type (
	Store interface {
		// Get returns the value associated with the key parameter.
		Get(string) (any, error)

		// Set adds the key-value pair to the Map or updates the value if it's
		// already present. The key-value pair is passed as a pointer to an
		// item object.
		Set(string, any, *SetOptions) error

		// Del deletes the key-value pair from the Map.
		Del(string) error

		// Update attempts to update the key with a new value and returns true if
		// successful.
		Update(string, any, *UpdateOption) error

		kitcat.Nameable
	}

	// Cache is a convenient wrapper around a Store with generics.
	Cache[V any] interface {
		// Get returns the value associated with the key parameter.
		Get(string) (V, error)

		// Set adds the key-value pair to the Map or updates the value if it's
		// already present. The key-value pair is passed as a pointer to an
		// item object.
		Set(string, V, *SetOptions) error

		// Del deletes the key-value pair from the Map.
		Del(string) error

		// Update attempts to update the key with a new value and returns true if
		// successful.
		Update(string, V, *UpdateOption) (V, error)

		kitcat.Nameable
	}

	// SetOptions is used to pass options to the Set method.
	SetOptions struct {
		// TTL is the time-to-live for the key-value pair.
		TTL *time.Duration
	}

	// UpdateOption is used to pass options to the Update method.
	UpdateOption struct {
		// TTL is the time-to-live for the key-value pair.
		TTL *time.Duration
	}

	stores struct {
		dig.In
		Stores []Store `group:"kitcache.store"`
	}
)

func ProvideStore(store any) *kitdi.Annotation {
	return kitdi.Annotate(store, kitdi.As(new(Store)), kitdi.Group("kitcache.store"))
}

func NewSetOptions() *SetOptions {
	return &SetOptions{}
}

func (o *SetOptions) WithTTL(ttl time.Duration) *SetOptions {
	o.TTL = &ttl

	return o
}

func NewUpdateOption() *UpdateOption {
	return &UpdateOption{}
}

func (o *UpdateOption) WithTTL(ttl time.Duration) *UpdateOption {
	o.TTL = &ttl

	return o
}

type storeToCache[T any] struct {
	cache Store
}

func (c storeToCache[T]) Get(s string) (T, error) {
	get, err := c.cache.Get(s)
	if errors.Is(err, ErrNotFound) {
		return *new(T), err
	}

	return get.(T), err
}

func (c storeToCache[T]) Set(s string, v T, options *SetOptions) error {
	return c.cache.Set(s, v, options)
}

func (c storeToCache[T]) Del(u string) error {
	return c.cache.Del(u)
}

func (c storeToCache[T]) Update(s string, v T, option *UpdateOption) (T, error) {
	newT := new(T)
	err := c.cache.Update(s, v, option)
	return *newT, err
}

func (c storeToCache[T]) Name() string {
	return c.cache.Name()
}

func NewCache[T any](store Store) Cache[T] {
	return &storeToCache[T]{cache: store}
}
