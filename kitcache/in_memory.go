package kitcache

import (
	"fmt"
	"github.com/dgraph-io/ristretto"
)

type InMemoryStore struct {
	Cache *ristretto.Cache
}

func NewInMemoryStore(config *Config) (*InMemoryStore, error) {
	cfg := &ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	}

	if config.InMemoryConfig != nil {
		cfg = config.InMemoryConfig
	}

	cache, err := ristretto.NewCache(cfg)

	if err != nil {
		return nil, fmt.Errorf("error while creating in memory cache: %w", err)
	}

	return &InMemoryStore{Cache: cache}, nil
}

func (i InMemoryStore) Get(s string) (any, error) {
	a, ok := i.Cache.Get(s)
	if !ok {
		return a, ErrNotFound
	}

	return a, nil
}

func (i InMemoryStore) Set(s string, a any, options *SetOptions) error {
	var ok bool
	if options.TTL != nil {
		ok = i.Cache.SetWithTTL(s, a, 1, *options.TTL)
	} else {
		ok = i.Cache.Set(s, a, 1)
	}

	if !ok {
		return ErrUnableToSet
	}

	return nil
}

func (i InMemoryStore) Del(s string) error {
	i.Cache.Del(s)

	return nil
}

func (i InMemoryStore) Update(s string, a any, option *UpdateOption) error {
	var ok bool
	if option.TTL != nil {
		ok = i.Cache.SetWithTTL(s, a, 1, *option.TTL)
	} else {
		ok = i.Cache.Set(s, a, 1)
	}

	if !ok {
		return ErrUnableToUpdate
	}

	return nil
}

func (i InMemoryStore) Name() string {
	return "in_memory"
}
