package track

import (
	"context"
	"encoding/json"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/memcache"
)

type TrackData struct {
	Message         string
	PercentComplete float64
}

func Update(ctx context.Context, query string, message string, percentComplete float64) {
	item := &memcache.Item{
		Key:    "track:" + query,
		Object: TrackData{message, percentComplete},
	}
	err := memcache.Gob.Set(ctx, item)
	if err != nil {
		log.Errorf(ctx, "track.Update err = %s", err)
	}

}

func GetData(ctx context.Context, query string) []byte {
	var item0 TrackData
	_, err := memcache.Gob.Get(ctx, "track:"+query, &item0)

	if err != nil {
		log.Errorf(ctx, "track.GetData err = %s", err)
	}

	m, _ := json.Marshal(item0)

	return m
}

func GetDetails(ctx context.Context, query string) TrackData {
	var item0 TrackData
	_, err := memcache.Gob.Get(ctx, "track:"+query, &item0)

	if err != nil {
		log.Errorf(ctx, "track.GetDetails err = %s", err)
	}

	return item0
}
