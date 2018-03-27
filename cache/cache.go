package cache

import (
	// "fmt"
	"context"
	"fmt"
	"hash/fnv"
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

func hash(key string) string {
	first := fnv.New32()
	first.Write([]byte(key))

	return fmt.Sprintf("%x", first.Sum(nil))
}

func (c *GenericCache) store(ctx context.Context, key string, value interface{}) {
	item := &memcache.Item{
		Key: c.keyType + ":" + hash(key),
		Object: CacheObject{
			LastUpdated: time.Now(),
			Value:       value,
		},
	}
	err := memcache.Gob.Set(ctx, item)
	if err != nil {
		log.Infof(ctx, "Error adding err = %s", err)
	} else {
		log.Infof(ctx, "Storing cache key "+c.keyType+":"+hash(key))
	}
}

func (c *GenericCache) Retrieve(ctx context.Context, key string) (interface{}, bool) {
	// log.Infof(c.ctx, "Retrieving from memcache")
	var item0 CacheObject
	_, err := memcache.Gob.Get(ctx, c.keyType+":"+hash(key), &item0)
	if err == nil {
		if time.Since(item0.LastUpdated) < c.timeout {
			return item0.Value, true
		}
		log.Infof(ctx, "cache expired for "+c.keyType+":"+hash(key))
	} else {
		log.Infof(ctx, "cache Retrieve "+c.keyType+":"+hash(key)+" err = %s", err)
	}
	log.Infof(ctx, "Using miss Function")

	dataFound, found := c.missFunction(ctx, key)
	c.store(ctx, key, dataFound)

	return dataFound, found
}
