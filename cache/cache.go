package cache

import (
	// "fmt"
	"context"
	"sync"
	"time"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

type GenericCache struct {
	timeout      time.Duration
	mutex        sync.RWMutex
	missFunction CacheMissFn
	keyType      string
	// ctx          context.Context
}

type CacheObject struct {
	LastUpdated time.Time
	Value       interface{}
}

type CacheMissFn func(context.Context, string) (interface{}, bool)

func NewGenericCache(timeout time.Duration, keyType string, missFunction CacheMissFn) *GenericCache {
	c := new(GenericCache)
	c.timeout = timeout
	c.missFunction = missFunction
	c.keyType = keyType
	// c.ctx = ctx

	return c
}

func (c *GenericCache) store(ctx context.Context, key string, value interface{}) {
	item := &memcache.Item{
		Key: c.keyType + ":" + key,
		Object: CacheObject{
			LastUpdated: time.Now(),
			Value:       value,
		},
	}
	err := memcache.Gob.Set(ctx, item)
	if err != nil {
		log.Infof(ctx, "Error adding err = %s", err)
	}
}

func (c *GenericCache) Retrieve(ctx context.Context, key string) (interface{}, bool) {
	// log.Infof(c.ctx, "Retrieving from memcache")
	var item0 CacheObject
	_, err := memcache.Gob.Get(ctx, c.keyType+":"+key, &item0)
	if err == nil {
		if time.Since(item0.LastUpdated) < c.timeout {
			return item0.Value, true
		}
	} else {
		// log.Infof(c.ctx, "cache err = %s", err)
	}
	// log.Infof(c.ctx, "Using miss Function")

	dataFound, found := c.missFunction(ctx, key)
	c.store(ctx, key, dataFound)

	return dataFound, found
}
