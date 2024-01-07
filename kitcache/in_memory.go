package kitcache

import (
	"fmt"
	"github.com/dgraph-io/ristretto"
	"github.com/kitcat-framework/kitcat"
	"github.com/spf13/viper"
	"go.uber.org/dig"
)

type InMemoryStoreConfig struct {
	NumCounters int64 `cfg:"num_counters"`
	MaxCost     int64 `cfg:"max_cost"`
	BufferItems int64 `cfg:"buffer_items"`
}

func (i *InMemoryStoreConfig) InitConfig(prefix string) kitcat.ConfigUnmarshal {
	prefix = fmt.Sprintf("%s.config_stores.in_memory", prefix)

	viper.SetDefault(prefix+".num_counters", 1e7)
	viper.SetDefault(prefix+".max_cost", 1<<30)
	viper.SetDefault(prefix+".buffer_items", 64)

	return kitcat.ConfigUnmarshalHandler(prefix, i, "unable to unmarshal in memory store config: %w")
}

func init() {
	kitcat.RegisterConfig(new(InMemoryStoreConfig))
}

type InMemoryStore struct {
	Cache *ristretto.Cache
}

type InMemoryStoreParams struct {
	dig.In

	Config *InMemoryStoreConfig

	// To set it manually, just provide a *ristretto.Config in the kitcat.App.Provides() method
	RistrettoConfig *ristretto.Config `optional:"true"`
}

func NewInMemoryStore(params InMemoryStoreParams) (*InMemoryStore, error) {
	ristrettoConfig := &ristretto.Config{
		NumCounters: params.Config.NumCounters,
		MaxCost:     params.Config.MaxCost,
		BufferItems: params.Config.BufferItems,
	}

	if params.RistrettoConfig != nil {
		ristrettoConfig = params.RistrettoConfig
	}

	cache, err := ristretto.NewCache(ristrettoConfig)

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
