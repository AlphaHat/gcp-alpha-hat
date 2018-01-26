package run

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/AlphaHat/gcp-alpha-hat/db"
)

// The zero values for all of these are going to be the "defaults", so make sure
// that booleans are the correct direction (i.e. we used disabled instead of active)
// since the disabled=false is the default
type SeriesOption struct {
	Name       string `json:"name" bson:"name"`
	NewName    string `json:"new_name" bson:"new_name"`
	NewUnits   string `json:"new_units" bson:"new_units"`
	Disabled   bool   `json:"disabled" bson:"disabled"`
	LineWidth  string `json:"line_width" bson:"line_width"`
	MarkerSize string `json:"marker_size" bson:"marker_size"`
	ChartType  string `json:"chart_type" bson:"chart_type"`
}

type ChartOptions struct {
	RunId         string         `json:"run_id" bson:"run_id"`
	Title         string         `json:"title" bson:"title"`
	ChartType     string         `json:"chart_type" bson:"chart_type"`
	Commentary    string         `json:"commentary" bson:"commentary"`
	Colors        []string       `json:"colors" bson:"colors"`
	SeriesOptions []SeriesOption `json:"series_options" bson:"series_options"`
}

func (co ChartOptions) GetSeriesOptionsFromOriginalName(name string) SeriesOption {
	for _, v := range co.SeriesOptions {
		if v.Name == name {
			return v
		}
	}

	return SeriesOption{Name: name}
}

func (co ChartOptions) GetSeriesOptionsFromNewName(name string) SeriesOption {
	for _, v := range co.SeriesOptions {
		if (v.NewName == "" && v.Name == name) || (v.NewName == name) {
			return v
		}
	}

	return SeriesOption{Name: name}
}

func SaveChartOptions(ctx context.Context, c ChartOptions) string {
	return db.DatabaseInsert(ctx, db.ChartOptions, &c, "")
}

func GetChartOptions(ctx context.Context, id string) ChartOptions {
	var m ChartOptions

	db.GetFromKey(ctx, id, &m)

	return m
}

func ParseChartOptions(ctx context.Context, inline string) ChartOptions {
	var co ChartOptions

	json.Unmarshal([]byte(inline), &co)

	return co
}

func ChartOptionsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var c ChartOptions
	err := decoder.Decode(&c)

	if err == nil {
		id := SaveChartOptions(ctx, c)
		fmt.Fprintf(w, "{\"id\": \"%s\"}\n", id)
	}
}
