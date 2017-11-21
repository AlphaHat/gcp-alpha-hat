package run

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
)

var bogusData = `{"EntityData":[{"Data":[{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":43},{"Time":"2017-08-08T00:00:00Z","Data":48},{"Time":"2017-08-09T00:00:00Z","Data":34},{"Time":"2017-08-10T00:00:00Z","Data":23},{"Time":"2017-08-11T00:00:00Z","Data":34},{"Time":"2017-08-12T00:00:00Z","Data":51},{"Time":"2017-08-13T00:00:00Z","Data":42},{"Time":"2017-08-14T00:00:00Z","Data":40},{"Time":"2017-08-15T00:00:00Z","Data":27},{"Time":"2017-08-16T00:00:00Z","Data":27},{"Time":"2017-08-17T00:00:00Z","Data":27},{"Time":"2017-08-18T00:00:00Z","Data":20},{"Time":"2017-08-19T00:00:00Z","Data":43},{"Time":"2017-08-20T00:00:00Z","Data":47},{"Time":"2017-08-21T00:00:00Z","Data":32},{"Time":"2017-08-22T00:00:00Z","Data":30},{"Time":"2017-08-23T00:00:00Z","Data":32},{"Time":"2017-08-24T00:00:00Z","Data":18},{"Time":"2017-08-25T00:00:00Z","Data":20},{"Time":"2017-08-26T00:00:00Z","Data":28},{"Time":"2017-08-27T00:00:00Z","Data":47},{"Time":"2017-08-28T00:00:00Z","Data":20},{"Time":"2017-08-29T00:00:00Z","Data":20},{"Time":"2017-08-30T00:00:00Z","Data":20},{"Time":"2017-08-31T00:00:00Z","Data":27},{"Time":"2017-09-01T00:00:00Z","Data":28},{"Time":"2017-09-02T00:00:00Z","Data":39},{"Time":"2017-09-03T00:00:00Z","Data":30},{"Time":"2017-09-04T00:00:00Z","Data":31},{"Time":"2017-09-05T00:00:00Z","Data":28},{"Time":"2017-09-06T00:00:00Z","Data":24},{"Time":"2017-09-07T00:00:00Z","Data":27},{"Time":"2017-09-08T00:00:00Z","Data":22},{"Time":"2017-09-09T00:00:00Z","Data":32},{"Time":"2017-09-10T00:00:00Z","Data":43},{"Time":"2017-09-11T00:00:00Z","Data":37},{"Time":"2017-09-12T00:00:00Z","Data":31},{"Time":"2017-09-13T00:00:00Z","Data":20},{"Time":"2017-09-14T00:00:00Z","Data":26},{"Time":"2017-09-15T00:00:00Z","Data":26},{"Time":"2017-09-16T00:00:00Z","Data":30},{"Time":"2017-09-17T00:00:00Z","Data":36},{"Time":"2017-09-18T00:00:00Z","Data":26},{"Time":"2017-09-19T00:00:00Z","Data":31},{"Time":"2017-09-20T00:00:00Z","Data":29},{"Time":"2017-09-21T00:00:00Z","Data":25},{"Time":"2017-09-22T00:00:00Z","Data":48},{"Time":"2017-09-23T00:00:00Z","Data":76},{"Time":"2017-09-24T00:00:00Z","Data":95},{"Time":"2017-09-25T00:00:00Z","Data":57},{"Time":"2017-09-26T00:00:00Z","Data":58},{"Time":"2017-09-27T00:00:00Z","Data":47},{"Time":"2017-09-28T00:00:00Z","Data":37},{"Time":"2017-09-29T00:00:00Z","Data":46},{"Time":"2017-09-30T00:00:00Z","Data":63},{"Time":"2017-10-01T00:00:00Z","Data":89},{"Time":"2017-10-02T00:00:00Z","Data":56},{"Time":"2017-10-03T00:00:00Z","Data":47},{"Time":"2017-10-04T00:00:00Z","Data":22},{"Time":"2017-10-05T00:00:00Z","Data":27},{"Time":"2017-10-06T00:00:00Z","Data":30},{"Time":"2017-10-07T00:00:00Z","Data":58},{"Time":"2017-10-08T00:00:00Z","Data":66},{"Time":"2017-10-09T00:00:00Z","Data":47},{"Time":"2017-10-10T00:00:00Z","Data":42},{"Time":"2017-10-11T00:00:00Z","Data":36},{"Time":"2017-10-12T00:00:00Z","Data":28},{"Time":"2017-10-13T00:00:00Z","Data":20},{"Time":"2017-10-14T00:00:00Z","Data":61},{"Time":"2017-10-15T00:00:00Z","Data":69},{"Time":"2017-10-16T00:00:00Z","Data":45},{"Time":"2017-10-17T00:00:00Z","Data":21},{"Time":"2017-10-18T00:00:00Z","Data":30},{"Time":"2017-10-19T00:00:00Z","Data":49},{"Time":"2017-10-20T00:00:00Z","Data":38},{"Time":"2017-10-21T00:00:00Z","Data":56},{"Time":"2017-10-22T00:00:00Z","Data":48},{"Time":"2017-10-23T00:00:00Z","Data":29},{"Time":"2017-10-24T00:00:00Z","Data":36},{"Time":"2017-10-25T00:00:00Z","Data":19},{"Time":"2017-10-26T00:00:00Z","Data":25},{"Time":"2017-10-27T00:00:00Z","Data":26},{"Time":"2017-10-28T00:00:00Z","Data":33},{"Time":"2017-10-29T00:00:00Z","Data":26},{"Time":"2017-10-30T00:00:00Z","Data":5}],"Meta":{"VendorCode":"Trader Joe's","Label":"Number of Visits","Units":"#","Source":"Narrative","Upsample":4,"Downsample":3,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Trader Joe's","UniqueId":"Trader Joe's","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":64},{"Time":"2017-08-08T00:00:00Z","Data":65},{"Time":"2017-08-09T00:00:00Z","Data":50},{"Time":"2017-08-10T00:00:00Z","Data":56},{"Time":"2017-08-11T00:00:00Z","Data":63},{"Time":"2017-08-12T00:00:00Z","Data":72},{"Time":"2017-08-13T00:00:00Z","Data":51},{"Time":"2017-08-14T00:00:00Z","Data":41},{"Time":"2017-08-15T00:00:00Z","Data":35},{"Time":"2017-08-16T00:00:00Z","Data":41},{"Time":"2017-08-17T00:00:00Z","Data":57},{"Time":"2017-08-18T00:00:00Z","Data":44},{"Time":"2017-08-19T00:00:00Z","Data":54},{"Time":"2017-08-20T00:00:00Z","Data":62},{"Time":"2017-08-21T00:00:00Z","Data":41},{"Time":"2017-08-22T00:00:00Z","Data":44},{"Time":"2017-08-23T00:00:00Z","Data":39},{"Time":"2017-08-24T00:00:00Z","Data":39},{"Time":"2017-08-25T00:00:00Z","Data":49},{"Time":"2017-08-26T00:00:00Z","Data":61},{"Time":"2017-08-27T00:00:00Z","Data":74},{"Time":"2017-08-28T00:00:00Z","Data":40},{"Time":"2017-08-29T00:00:00Z","Data":42},{"Time":"2017-08-30T00:00:00Z","Data":56},{"Time":"2017-08-31T00:00:00Z","Data":59},{"Time":"2017-09-01T00:00:00Z","Data":62},{"Time":"2017-09-02T00:00:00Z","Data":81},{"Time":"2017-09-03T00:00:00Z","Data":73},{"Time":"2017-09-04T00:00:00Z","Data":72},{"Time":"2017-09-05T00:00:00Z","Data":62},{"Time":"2017-09-06T00:00:00Z","Data":59},{"Time":"2017-09-07T00:00:00Z","Data":46},{"Time":"2017-09-08T00:00:00Z","Data":57},{"Time":"2017-09-09T00:00:00Z","Data":77},{"Time":"2017-09-10T00:00:00Z","Data":88},{"Time":"2017-09-11T00:00:00Z","Data":56},{"Time":"2017-09-12T00:00:00Z","Data":58},{"Time":"2017-09-13T00:00:00Z","Data":65},{"Time":"2017-09-14T00:00:00Z","Data":52},{"Time":"2017-09-15T00:00:00Z","Data":68},{"Time":"2017-09-16T00:00:00Z","Data":74},{"Time":"2017-09-17T00:00:00Z","Data":75},{"Time":"2017-09-18T00:00:00Z","Data":45},{"Time":"2017-09-19T00:00:00Z","Data":69},{"Time":"2017-09-20T00:00:00Z","Data":52},{"Time":"2017-09-21T00:00:00Z","Data":84},{"Time":"2017-09-22T00:00:00Z","Data":121},{"Time":"2017-09-23T00:00:00Z","Data":142},{"Time":"2017-09-24T00:00:00Z","Data":195},{"Time":"2017-09-25T00:00:00Z","Data":121},{"Time":"2017-09-26T00:00:00Z","Data":95},{"Time":"2017-09-27T00:00:00Z","Data":97},{"Time":"2017-09-28T00:00:00Z","Data":86},{"Time":"2017-09-29T00:00:00Z","Data":92},{"Time":"2017-09-30T00:00:00Z","Data":93},{"Time":"2017-10-01T00:00:00Z","Data":114},{"Time":"2017-10-02T00:00:00Z","Data":87},{"Time":"2017-10-03T00:00:00Z","Data":77},{"Time":"2017-10-04T00:00:00Z","Data":63},{"Time":"2017-10-05T00:00:00Z","Data":76},{"Time":"2017-10-06T00:00:00Z","Data":75},{"Time":"2017-10-07T00:00:00Z","Data":98},{"Time":"2017-10-08T00:00:00Z","Data":94},{"Time":"2017-10-09T00:00:00Z","Data":81},{"Time":"2017-10-10T00:00:00Z","Data":85},{"Time":"2017-10-11T00:00:00Z","Data":66},{"Time":"2017-10-12T00:00:00Z","Data":91},{"Time":"2017-10-13T00:00:00Z","Data":97},{"Time":"2017-10-14T00:00:00Z","Data":111},{"Time":"2017-10-15T00:00:00Z","Data":121},{"Time":"2017-10-16T00:00:00Z","Data":100},{"Time":"2017-10-17T00:00:00Z","Data":65},{"Time":"2017-10-18T00:00:00Z","Data":74},{"Time":"2017-10-19T00:00:00Z","Data":69},{"Time":"2017-10-20T00:00:00Z","Data":47},{"Time":"2017-10-21T00:00:00Z","Data":79},{"Time":"2017-10-22T00:00:00Z","Data":111},{"Time":"2017-10-23T00:00:00Z","Data":71},{"Time":"2017-10-24T00:00:00Z","Data":69},{"Time":"2017-10-25T00:00:00Z","Data":53},{"Time":"2017-10-26T00:00:00Z","Data":44},{"Time":"2017-10-27T00:00:00Z","Data":60},{"Time":"2017-10-28T00:00:00Z","Data":68},{"Time":"2017-10-29T00:00:00Z","Data":42},{"Time":"2017-10-30T00:00:00Z","Data":8}],"Meta":{"VendorCode":"Whole Foods","Label":"Number of Visits","Units":"#","Source":"Narrative","Upsample":4,"Downsample":3,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Whole Foods","UniqueId":"Whole Foods","IsCustom":true}}],"Title":"Visit Count","Error":"","GraphicalPreference":""}`

var sqlDb *sql.DB

func connect() (*sql.DB, error) {
	var db *sql.DB

	connectionName := os.Getenv("CLOUDSQL_CONNECTION_NAME")
	user := os.Getenv("CLOUDSQL_USER")
	password := os.Getenv("CLOUDSQL_PASSWORD") // NOTE: password may be empty

	var err error
	db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@cloudsql(%s)/location", user, password, connectionName))
	if err != nil {
		return nil, err
	}

	// rows, err := db.Query("SHOW DATABASES")
	// if err != nil {
	// 	return nil, err
	// }
	// defer rows.Close()

	return db, nil
}

func GetVisitCounts(ctx context.Context) MultiEntityData {
	var err error
	if sqlDb == nil {
		sqlDb, err = connect()
	}
	if err != nil {
		log.Errorf(ctx, "Unable to connect to database")
	}

	var m MultiEntityData

	if appengine.IsDevAppServer() {
		log.Infof(ctx, "CANNOT DO A SQL CONNECT ON THE DEV SERVER")
		json.Unmarshal([]byte(bogusData), &m)
		return m
	}

	rows, err := sqlDb.Query(`select date, sum(num_visit), brand
from wow_change
group by date, brand`)

	if err != nil {
		log.Errorf(ctx, "query error = %s", err)
		return m
	}

	m.Title = "Visit Count"
	var date, brand string
	var numVisit int64

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&date, &numVisit, &brand)

		if err != nil {
			log.Errorf(ctx, "query error = %s", err)
		} else {
			t, _ := time.Parse("2006-01-02", date)

			m = m.Insert(
				EntityMeta{
					Name:     brand,
					UniqueId: brand,
					IsCustom: true,
				},
				SeriesMeta{
					VendorCode:    brand,
					Label:         "Number of Visits",
					Units:         "#",
					Source:        "Narrative",
					Upsample:      ResampleZero,
					Downsample:    ResampleArithmetic,
					IsTransformed: false,
				},
				DataPoint{
					Time: t,
					Data: float64(numVisit),
				},
				CategoryLabel{
					Id:    0,
					Label: "",
				},
				false)
		}
	}

	return m
}
