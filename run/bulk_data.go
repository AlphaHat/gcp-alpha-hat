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

var bogusData2 = `{"EntityData":[{"Data":[{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":64},{"Time":"2017-08-08T00:00:00Z","Data":65},{"Time":"2017-08-09T00:00:00Z","Data":50},{"Time":"2017-08-10T00:00:00Z","Data":56},{"Time":"2017-08-11T00:00:00Z","Data":63},{"Time":"2017-08-12T00:00:00Z","Data":72},{"Time":"2017-08-13T00:00:00Z","Data":51},{"Time":"2017-08-14T00:00:00Z","Data":41},{"Time":"2017-08-15T00:00:00Z","Data":35},{"Time":"2017-08-16T00:00:00Z","Data":41},{"Time":"2017-08-17T00:00:00Z","Data":57},{"Time":"2017-08-18T00:00:00Z","Data":44},{"Time":"2017-08-19T00:00:00Z","Data":54},{"Time":"2017-08-20T00:00:00Z","Data":62},{"Time":"2017-08-21T00:00:00Z","Data":41},{"Time":"2017-08-22T00:00:00Z","Data":44},{"Time":"2017-08-23T00:00:00Z","Data":39},{"Time":"2017-08-24T00:00:00Z","Data":39},{"Time":"2017-08-25T00:00:00Z","Data":49},{"Time":"2017-08-26T00:00:00Z","Data":61},{"Time":"2017-08-27T00:00:00Z","Data":74},{"Time":"2017-08-28T00:00:00Z","Data":40},{"Time":"2017-08-29T00:00:00Z","Data":42},{"Time":"2017-08-30T00:00:00Z","Data":56},{"Time":"2017-08-31T00:00:00Z","Data":59},{"Time":"2017-09-01T00:00:00Z","Data":62},{"Time":"2017-09-02T00:00:00Z","Data":81},{"Time":"2017-09-03T00:00:00Z","Data":73},{"Time":"2017-09-04T00:00:00Z","Data":72},{"Time":"2017-09-05T00:00:00Z","Data":62},{"Time":"2017-09-06T00:00:00Z","Data":59},{"Time":"2017-09-07T00:00:00Z","Data":46},{"Time":"2017-09-08T00:00:00Z","Data":57},{"Time":"2017-09-09T00:00:00Z","Data":77},{"Time":"2017-09-10T00:00:00Z","Data":88},{"Time":"2017-09-11T00:00:00Z","Data":56},{"Time":"2017-09-12T00:00:00Z","Data":58},{"Time":"2017-09-13T00:00:00Z","Data":65},{"Time":"2017-09-14T00:00:00Z","Data":52},{"Time":"2017-09-15T00:00:00Z","Data":68},{"Time":"2017-09-16T00:00:00Z","Data":74},{"Time":"2017-09-17T00:00:00Z","Data":75},{"Time":"2017-09-18T00:00:00Z","Data":45},{"Time":"2017-09-19T00:00:00Z","Data":69},{"Time":"2017-09-20T00:00:00Z","Data":52},{"Time":"2017-09-21T00:00:00Z","Data":84},{"Time":"2017-09-22T00:00:00Z","Data":121},{"Time":"2017-09-23T00:00:00Z","Data":142},{"Time":"2017-09-24T00:00:00Z","Data":195},{"Time":"2017-09-25T00:00:00Z","Data":121},{"Time":"2017-09-26T00:00:00Z","Data":95},{"Time":"2017-09-27T00:00:00Z","Data":97},{"Time":"2017-09-28T00:00:00Z","Data":86},{"Time":"2017-09-29T00:00:00Z","Data":92},{"Time":"2017-09-30T00:00:00Z","Data":93},{"Time":"2017-10-01T00:00:00Z","Data":114},{"Time":"2017-10-02T00:00:00Z","Data":87},{"Time":"2017-10-03T00:00:00Z","Data":77},{"Time":"2017-10-04T00:00:00Z","Data":63},{"Time":"2017-10-05T00:00:00Z","Data":76},{"Time":"2017-10-06T00:00:00Z","Data":75},{"Time":"2017-10-07T00:00:00Z","Data":98},{"Time":"2017-10-08T00:00:00Z","Data":94},{"Time":"2017-10-09T00:00:00Z","Data":81},{"Time":"2017-10-10T00:00:00Z","Data":85},{"Time":"2017-10-11T00:00:00Z","Data":66},{"Time":"2017-10-12T00:00:00Z","Data":91},{"Time":"2017-10-13T00:00:00Z","Data":97},{"Time":"2017-10-14T00:00:00Z","Data":111},{"Time":"2017-10-15T00:00:00Z","Data":121},{"Time":"2017-10-16T00:00:00Z","Data":100},{"Time":"2017-10-17T00:00:00Z","Data":65},{"Time":"2017-10-18T00:00:00Z","Data":74},{"Time":"2017-10-19T00:00:00Z","Data":69},{"Time":"2017-10-20T00:00:00Z","Data":47},{"Time":"2017-10-21T00:00:00Z","Data":79},{"Time":"2017-10-22T00:00:00Z","Data":111},{"Time":"2017-10-23T00:00:00Z","Data":71},{"Time":"2017-10-24T00:00:00Z","Data":69},{"Time":"2017-10-25T00:00:00Z","Data":53},{"Time":"2017-10-26T00:00:00Z","Data":44},{"Time":"2017-10-27T00:00:00Z","Data":60},{"Time":"2017-10-28T00:00:00Z","Data":68},{"Time":"2017-10-29T00:00:00Z","Data":42},{"Time":"2017-10-30T00:00:00Z","Data":8}],"Meta":{"VendorCode":"Number of Visits","Label":"Number of Visits","Units":"#","Source":"Narrative","Upsample":4,"Downsample":3,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":6},{"Time":"2017-08-08T00:00:00Z","Data":10},{"Time":"2017-08-09T00:00:00Z","Data":8},{"Time":"2017-08-10T00:00:00Z","Data":10},{"Time":"2017-08-11T00:00:00Z","Data":10},{"Time":"2017-08-12T00:00:00Z","Data":10},{"Time":"2017-08-13T00:00:00Z","Data":9},{"Time":"2017-08-14T00:00:00Z","Data":9},{"Time":"2017-08-15T00:00:00Z","Data":8},{"Time":"2017-08-16T00:00:00Z","Data":8},{"Time":"2017-08-17T00:00:00Z","Data":8},{"Time":"2017-08-18T00:00:00Z","Data":10},{"Time":"2017-08-19T00:00:00Z","Data":9},{"Time":"2017-08-20T00:00:00Z","Data":9},{"Time":"2017-08-21T00:00:00Z","Data":9},{"Time":"2017-08-22T00:00:00Z","Data":8},{"Time":"2017-08-23T00:00:00Z","Data":10},{"Time":"2017-08-24T00:00:00Z","Data":8},{"Time":"2017-08-25T00:00:00Z","Data":10},{"Time":"2017-08-26T00:00:00Z","Data":9},{"Time":"2017-08-27T00:00:00Z","Data":8},{"Time":"2017-08-28T00:00:00Z","Data":8},{"Time":"2017-08-29T00:00:00Z","Data":9},{"Time":"2017-08-30T00:00:00Z","Data":10},{"Time":"2017-08-31T00:00:00Z","Data":9},{"Time":"2017-09-01T00:00:00Z","Data":9},{"Time":"2017-09-02T00:00:00Z","Data":9},{"Time":"2017-09-03T00:00:00Z","Data":8},{"Time":"2017-09-04T00:00:00Z","Data":7},{"Time":"2017-09-05T00:00:00Z","Data":8},{"Time":"2017-09-06T00:00:00Z","Data":9},{"Time":"2017-09-07T00:00:00Z","Data":9},{"Time":"2017-09-08T00:00:00Z","Data":8},{"Time":"2017-09-09T00:00:00Z","Data":9},{"Time":"2017-09-10T00:00:00Z","Data":10},{"Time":"2017-09-11T00:00:00Z","Data":8},{"Time":"2017-09-12T00:00:00Z","Data":9},{"Time":"2017-09-13T00:00:00Z","Data":9},{"Time":"2017-09-14T00:00:00Z","Data":9},{"Time":"2017-09-15T00:00:00Z","Data":8},{"Time":"2017-09-16T00:00:00Z","Data":10},{"Time":"2017-09-17T00:00:00Z","Data":9},{"Time":"2017-09-18T00:00:00Z","Data":8},{"Time":"2017-09-19T00:00:00Z","Data":10},{"Time":"2017-09-20T00:00:00Z","Data":7},{"Time":"2017-09-21T00:00:00Z","Data":9},{"Time":"2017-09-22T00:00:00Z","Data":8},{"Time":"2017-09-23T00:00:00Z","Data":9},{"Time":"2017-09-24T00:00:00Z","Data":9},{"Time":"2017-09-25T00:00:00Z","Data":9},{"Time":"2017-09-26T00:00:00Z","Data":10},{"Time":"2017-09-27T00:00:00Z","Data":8},{"Time":"2017-09-28T00:00:00Z","Data":10},{"Time":"2017-09-29T00:00:00Z","Data":8},{"Time":"2017-09-30T00:00:00Z","Data":9},{"Time":"2017-10-01T00:00:00Z","Data":10},{"Time":"2017-10-02T00:00:00Z","Data":10},{"Time":"2017-10-03T00:00:00Z","Data":10},{"Time":"2017-10-04T00:00:00Z","Data":9},{"Time":"2017-10-05T00:00:00Z","Data":10},{"Time":"2017-10-06T00:00:00Z","Data":9},{"Time":"2017-10-07T00:00:00Z","Data":9},{"Time":"2017-10-08T00:00:00Z","Data":9},{"Time":"2017-10-09T00:00:00Z","Data":10},{"Time":"2017-10-10T00:00:00Z","Data":10},{"Time":"2017-10-11T00:00:00Z","Data":9},{"Time":"2017-10-12T00:00:00Z","Data":10},{"Time":"2017-10-13T00:00:00Z","Data":10},{"Time":"2017-10-14T00:00:00Z","Data":9},{"Time":"2017-10-15T00:00:00Z","Data":9},{"Time":"2017-10-16T00:00:00Z","Data":10},{"Time":"2017-10-17T00:00:00Z","Data":10},{"Time":"2017-10-18T00:00:00Z","Data":9},{"Time":"2017-10-19T00:00:00Z","Data":9},{"Time":"2017-10-20T00:00:00Z","Data":8},{"Time":"2017-10-21T00:00:00Z","Data":10},{"Time":"2017-10-22T00:00:00Z","Data":10},{"Time":"2017-10-23T00:00:00Z","Data":10},{"Time":"2017-10-24T00:00:00Z","Data":10},{"Time":"2017-10-25T00:00:00Z","Data":9},{"Time":"2017-10-26T00:00:00Z","Data":9},{"Time":"2017-10-27T00:00:00Z","Data":7},{"Time":"2017-10-28T00:00:00Z","Data":10},{"Time":"2017-10-29T00:00:00Z","Data":8},{"Time":"2017-10-30T00:00:00Z","Data":6}],"Meta":{"VendorCode":"Number of Stores","Label":"Number of Stores","Units":"#","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-08-07T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Whole Foods","UniqueId":"Whole Foods","IsCustom":true}}],"Title":"Whole Foods Market, Inc. â†’ Traffic per Store Last Five Years","Error":"","GraphicalPreference":""}`

var bogusData3 = `{"EntityData":[{"Data":[{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":0.034763805}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":-0.5}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"4800 El Camino WF","UniqueId":"4800 El Camino WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.088822355}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.8333333333333334}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"777 WF","UniqueId":"777 WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.178310046}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.9285714285714286}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Augustine","UniqueId":"Augustine","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.13023952}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.8}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Blossom WF","UniqueId":"Blossom WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.037258815}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.6666666666666667}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Boscom WF","UniqueId":"Boscom WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":0.130572188}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":0.16666666666666674}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-29T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Emerson","UniqueId":"Emerson","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":0.087990685}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":-0.1428571428571429}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"jefferson WF","UniqueId":"jefferson WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":0.017465069}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":0}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-28T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Los Gatos WF","UniqueId":"Los Gatos WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.217897538}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.8823529411764706}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Park WF","UniqueId":"Park WF","IsCustom":true}},{"Data":[{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":0.076679973}],"Meta":{"VendorCode":"% of Traffic","Label":"% of Traffic","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false},{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":-0.33333333333333337}],"Meta":{"VendorCode":"WoW Change","Label":"WoW Change","Units":"%","Source":"Narrative","Upsample":1,"Downsample":1,"IsTransformed":false},"IsWeight":false}],"Category":{"Data":[{"Time":"2017-10-30T00:00:00Z","Data":1}],"Labels":[{"Id":1,"Label":""}]},"Meta":{"Name":"Stephens Cr. WF","UniqueId":"Stephens Cr. WF","IsCustom":true}}],"Title":"Whole Foods Market, Inc. → Traffic Contribution Last Five Years","Error":"","GraphicalPreference":""}`

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

func QueryStringArray(ctx context.Context, query string, args ...interface{}) []string {
	var err error
	if sqlDb == nil {
		sqlDb, err = connect()
	}
	if err != nil {
		log.Errorf(ctx, "Unable to connect to database")
	}

	sArr := make([]string, 0)
	rows, err := sqlDb.Query(query, args...)

	if err != nil {
		log.Errorf(ctx, "query error = %s", err)
		return sArr
	}

	var s string

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&s)

		if err != nil {
			log.Errorf(ctx, "query error = %s", err)
		} else {
			sArr = append(sArr, s)
		}
	}

	return sArr
}

func GenericQuery(ctx context.Context, query string, metaMap map[string]SeriesMeta) MultiEntityData {
	var err error
	if sqlDb == nil {
		sqlDb, err = connect()
	}
	if err != nil {
		log.Errorf(ctx, "Unable to connect to database")
	}

	var m MultiEntityData

	rows, err := sqlDb.Query(query)

	if err != nil {
		log.Errorf(ctx, "query error = %s", err)
		return m
	}

	m.Title = ""
	var date, entity, field string
	var data float64

	defer rows.Close()
	for rows.Next() {
		err := rows.Scan(&date, &entity, &field, &data)

		if err != nil {
			log.Errorf(ctx, "query error = %s", err)
		} else {
			t, _ := time.Parse("2006-01-02", date)

			m = m.Insert(
				EntityMeta{
					Name:     entity,
					UniqueId: entity,
					IsCustom: true,
				},
				metaMap[field],
				DataPoint{
					Time: t,
					Data: data,
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

func GetSQLData(ctx context.Context, entities []string, queryName string, startDate time.Time, endDate time.Time) MultiEntityData {
	metaMap := map[string]SeriesMeta{
		"Number of Visits": SeriesMeta{
			VendorCode:    "Number of Visits",
			Label:         "Number of Visits",
			Units:         "#",
			Source:        "Narrative",
			Upsample:      ResampleZero,
			Downsample:    ResampleArithmetic,
			IsTransformed: false,
		},
		"Number of Stores": SeriesMeta{
			VendorCode:    "Number of Stores",
			Label:         "Number of Stores",
			Units:         "#",
			Source:        "Narrative",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
		"% of Traffic": SeriesMeta{
			VendorCode:    "% of Traffic",
			Label:         "% of Traffic",
			Units:         "%",
			Source:        "Narrative",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
		"WoW Change": SeriesMeta{
			VendorCode:    "WoW Change",
			Label:         "WoW Change",
			Units:         "%",
			Source:        "Narrative",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: false,
		},
	}

	var query string
	switch queryName {
	case "Number of Visits":
		query = `select date, brand, "Number of Visits" as field, sum(num_visit) from wow_change group by date, brand`
	case "Traffic per Store":
		query = `select date, brand, "Number of Visits" as field_name, sum(num_visit) as field_value
from wow_change
where brand = "Whole Foods"
group by date, brand
union
select date, brand, "Number of Stores" as field_name, count(distinct name) as field_value
from wow_change
where brand = "Whole Foods"
group by date, brand`
	case "Traffic Contribution":
		query = `(
select max(date) as date, name, "% of Traffic" as field, sum(num_visit)/(select sum(num_visit) from wow_change b where b.brand=a.brand) from
wow_change a
where brand = 'Whole Foods'
group by name
)
union
(
select date, name, "WoW Change" as field, wow_change from
wow_change a
where brand = 'Whole Foods'
and a.date = (select max(date) from wow_change b where b.brand = a.brand and a.name = b.name)
)`
	}

	if appengine.IsDevAppServer() {
		var m MultiEntityData
		log.Infof(ctx, "CANNOT DO A SQL CONNECT ON THE DEV SERVER")
		switch queryName {
		case "Number of Visits":
			json.Unmarshal([]byte(bogusData), &m)
		case "Traffic per Store":
			json.Unmarshal([]byte(bogusData2), &m)
		case "Traffic Contribution":
			json.Unmarshal([]byte(bogusData3), &m)
		}
		return m
	}

	return GenericQuery(ctx, query, metaMap)
}

func GetVisitCounts(ctx context.Context) MultiEntityData {
	return GenericQuery(ctx, `select date, brand, "Number of Visits" as field, sum(num_visit)
from wow_change
group by date, brand`,
		map[string]SeriesMeta{
			"Number of Visits": SeriesMeta{
				VendorCode:    "Number of Visits",
				Label:         "Number of Visits",
				Units:         "#",
				Source:        "Narrative",
				Upsample:      ResampleZero,
				Downsample:    ResampleArithmetic,
				IsTransformed: false,
			},
		})

	// 	var err error
	// 	if sqlDb == nil {
	// 		sqlDb, err = connect()
	// 	}
	// 	if err != nil {
	// 		log.Errorf(ctx, "Unable to connect to database")
	// 	}
	//
	// 	var m MultiEntityData
	//
	// 	if appengine.IsDevAppServer() {
	// 		log.Infof(ctx, "CANNOT DO A SQL CONNECT ON THE DEV SERVER")
	// 		json.Unmarshal([]byte(bogusData), &m)
	// 		return m
	// 	}
	//
	// 	rows, err := sqlDb.Query(`select date, sum(num_visit), brand
	// from wow_change
	// group by date, brand`)
	//
	// 	if err != nil {
	// 		log.Errorf(ctx, "query error = %s", err)
	// 		return m
	// 	}
	//
	// 	m.Title = "Visit Count"
	// 	var date, brand string
	// 	var numVisit int64
	//
	// 	defer rows.Close()
	// 	for rows.Next() {
	// 		err := rows.Scan(&date, &numVisit, &brand)
	//
	// 		if err != nil {
	// 			log.Errorf(ctx, "query error = %s", err)
	// 		} else {
	// 			t, _ := time.Parse("2006-01-02", date)
	//
	// 			m = m.Insert(
	// 				EntityMeta{
	// 					Name:     brand,
	// 					UniqueId: brand,
	// 					IsCustom: true,
	// 				},
	// 				SeriesMeta{
	// 					VendorCode:    brand,
	// 					Label:         "Number of Visits",
	// 					Units:         "#",
	// 					Source:        "Narrative",
	// 					Upsample:      ResampleZero,
	// 					Downsample:    ResampleArithmetic,
	// 					IsTransformed: false,
	// 				},
	// 				DataPoint{
	// 					Time: t,
	// 					Data: float64(numVisit),
	// 				},
	// 				CategoryLabel{
	// 					Id:    0,
	// 					Label: "",
	// 				},
	// 				false)
	// 		}
	// 	}
	//
	// 	return m
}
