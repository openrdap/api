package caching

import (
	"context"
	"time"

	"github.com/openrdap/rdap/bootstrap/cache"

	"google.golang.org/appengine/memcache"
)

// BootstrapMemcache implements a Service Registry cache on Google App Engine's
// Memcache service.
type MemcacheCache struct {
	Timeout time.Duration

	ctx context.Context
}

func NewMemcacheCache(ctx context.Context) *MemcacheCache {
	return &MemcacheCache{
		Timeout: time.Hour,
		ctx:     ctx,
	}
}

func (m *MemcacheCache) Load(filename string) ([]byte, error) {
	item, err := memcache.Get(m.ctx, filename)

	if err != nil && err != memcache.ErrCacheMiss {
		return nil, err
	}

	if err == nil {
		return item.Value, nil
	} else {
		return nil, nil
	}
}

func (m *MemcacheCache) Save(filename string, data []byte) error {
	item := &memcache.Item{
		Key:        filename,
		Value:      data,
		Expiration: m.Timeout,
	}

	err := memcache.Set(m.ctx, item)

	return err
}

func (m *MemcacheCache) State(filename string) cache.FileState {
	_, err := memcache.Get(m.ctx, filename)

	if err == nil {
		return cache.ShouldReload
	}

	return cache.Absent
}

func (m *MemcacheCache) SetTimeout(timeout time.Duration) {
	m.Timeout = timeout
}
