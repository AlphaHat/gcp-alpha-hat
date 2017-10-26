package db

import (
	"context"

	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/log"
)

const (
	RunData      = "rundata"
	RunTree      = "tree"
	Quandl       = "quandl"
	ChartOptions = "chartoptions"
)

func logError(ctx context.Context, err error) bool {
	if err == nil {
		return true
	}

	log.Warningf(ctx, "Error = %s", err)
	return false
}

// data should be a pointer
func DatabaseInsert(ctx context.Context, kind string, data interface{}, parent string) string {
	var key *datastore.Key

	if parent != "" {
		parentKey, err := datastore.DecodeKey(parent)

		if logError(ctx, err) {
			key = datastore.NewIncompleteKey(ctx, kind, parentKey)
		} else {
			key = datastore.NewIncompleteKey(ctx, kind, nil)
		}
	} else {
		key = datastore.NewIncompleteKey(ctx, kind, nil)
	}

	returnKey, err := datastore.Put(ctx, key, data)

	if err == nil {
		log.Infof(ctx, "Return key %s = %s", kind, returnKey.Encode())
		return returnKey.Encode()
	} else {
		log.Warningf(ctx, "Error inserting %s = %s", kind, err)
	}

	return ""
}

func DatabaseUpdate(ctx context.Context, data interface{}, keyString string) {

	if keyString != "" {
		key, err := datastore.DecodeKey(keyString)

		if logError(ctx, err) {
			_, err := datastore.Put(ctx, key, data)

			if err == nil {
				log.Infof(ctx, "Updated key %s", keyString)
			} else {
				log.Warningf(ctx, "Error updating key %s = %s", keyString, err)
			}
		}
	} else {
		log.Errorf(ctx, "No key provided")
	}

}

func DatabaseDelete(ctx context.Context, keyString string) {

	if keyString != "" {
		key, err := datastore.DecodeKey(keyString)

		if logError(ctx, err) {
			err := datastore.Delete(ctx, key)

			if err == nil {
				log.Infof(ctx, "Deleted key %s", keyString)
			} else {
				log.Warningf(ctx, "Error deleting key %s = %s", keyString, err)
			}
		}
	} else {
		log.Errorf(ctx, "No key provided")
	}

}

func GetFromKey(ctx context.Context, keyString string, v interface{}) error {
	key, err := datastore.DecodeKey(keyString)

	if logError(ctx, err) {
		return err
	}

	return datastore.Get(ctx, key, v)
}

func GetFromField(ctx context.Context, tableName string, fieldName string, fieldValue string, v interface{}) error {
	query := datastore.NewQuery(tableName).Filter(fieldName+" =", fieldValue)

	iter := query.Run(ctx)

	_, err := iter.Next(&v)

	if err == datastore.Done {
		return nil
	}

	return err
}

func GetAllFromField(ctx context.Context, tableName string, fieldName string, fieldValue string, v interface{}) error {
	query := datastore.NewQuery(tableName).Filter(fieldName+" =", fieldValue)

	_, err := query.GetAll(ctx, v)

	if err == datastore.Done {
		return nil
	}

	return err
}
