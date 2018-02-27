package data

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/AlphaHat/gcp-alpha-hat/quandl"

	"github.com/AlphaHat/gcp-alpha-hat/component"
	"github.com/AlphaHat/gcp-alpha-hat/list"

	"github.com/AlphaHat/gcp-alpha-hat/cache"
	"github.com/AlphaHat/gcp-alpha-hat/db"
	"github.com/AlphaHat/gcp-alpha-hat/timeseries"

	"google.golang.org/appengine/log"
)

func extractTickerFromUniverseMember(universeMember component.QueryComponent) (string, error) {
	if universeMember.QueryComponentType == component.CustomQuandlCode {
		return universeMember.QueryComponentProviderId, nil
	}

	re, _ := regexp.Compile("[0-9A-Za-z]+/([0-9A-Za-z]+)")
	tickerList := re.FindAllStringSubmatch(universeMember.QueryComponentProviderId, -1)

	var ticker string

	if len(tickerList) >= 1 && len(tickerList[0]) > 1 {
		ticker = tickerList[0][1]
		return ticker, nil
	} else {
		return "", fmt.Errorf("Unable to read %s", tickerList)
	}
}

func constructQuandlTicker(universeMember component.QueryComponent, concept component.QueryComponent) (string, string) {
	ticker, err := extractTickerFromUniverseMember(universeMember)

	if err != nil {
		return "", ""
	}

	if concept.QueryComponentSource == component.Damodaran {
		queryTicker := fmt.Sprintf("DMDRN/%s_%s", ticker, concept.QueryComponentProviderId)

		return queryTicker, concept.QueryComponentCanonicalName
	} else if concept.QueryComponentCanonicalName == "Short Interest" || concept.QueryComponentCanonicalName == "Days to Cover" {
		queryTicker := fmt.Sprintf("SI/%s_SI", ticker)

		return queryTicker, concept.QueryComponentCanonicalName
	} else if concept.QueryComponentSource == component.SECHarmonized {
		queryTicker := fmt.Sprintf("RAYMOND/%s_%s_Q", ticker, concept.QueryComponentProviderId)

		return queryTicker, concept.QueryComponentCanonicalName
	} else if concept.QueryComponentSource == component.Sharadar {
		queryTicker := fmt.Sprintf("SF1/%s_%s", ticker, concept.QueryComponentProviderId)

		return queryTicker, concept.QueryComponentCanonicalName
	} else if concept.QueryComponentSource == component.QuandlOpenData && concept.QueryComponentType == "Concept: Economic" {
		return concept.QueryComponentProviderId, concept.QueryComponentCanonicalName
	} else if universeMember.QueryComponentSource == component.QuandlOpenData {
		return universeMember.QueryComponentProviderId, concept.QueryComponentCanonicalName
	}

	return "", ""
}

func GetDataFull(ctx context.Context, universeMember component.QueryComponent, concept component.QueryComponent) (*timeseries.TimeSeries, string) {
	ticker, seriesName := constructQuandlTicker(universeMember, concept)

	if ticker == "" {
		log.Errorf(ctx, "Error extracting ticker from %s", universeMember)
		return nil, ""
	}

	return GetQuandlDataFull(ctx, ticker, seriesName), ticker
}

func GetQuandlDataFull(ctx context.Context, ticker string, seriesName string) *timeseries.TimeSeries {
	c := quandlConnect(seriesName)

	if c == nil {
		log.Errorf(ctx, "Unable to set up quandl connection")
		return nil
	}

	tsRaw, found := c.Retrieve(ctx, ticker)

	if !found {
		log.Infof(ctx, "%s not found", ticker)
		return nil
	}

	switch t := tsRaw.(type) {
	case timeseries.TimeSeries:
		return &t
	}

	log.Infof(ctx, "%s not a timeseries", ticker)

	return nil
}

func quandlConnect(columnName string) *cache.GenericCache {
	quandl.SetAuthToken(os.Getenv("QUANDL_KEY"))

	c := cache.NewGenericCache(time.Hour, "quandl", func(ctx context.Context, ticker string) (interface{}, bool) {
		log.Infof(ctx, "Cache miss")

		ts, err := getQuandlTimeSeriesFromDatabase(ctx, ticker)

		if err == nil && ts.Len() > 0 {
			return ts, true
		}

		q, err := quandl.GetAllHistory(ctx, ticker)

		if err == nil && q != nil {
			date, data := q.GetTimeSeries(ctx, columnName)

			if len(data) == 0 {
				date = q.GetTimeSeriesDate(ctx)
				data, columnName = q.GetTimeSeriesData(ctx)
			}

			if columnName != "Volume" && columnName != "Days to Cover" {
				ts := timeseries.NewTimeSeries(date, data, ticker, q.QuandlName)
				ts.Source = q.SourceName
				ts.DataEnd = q.ToDate

				// The units are already built in to the highcharts object
				if columnName == "Value" {
					ts.Units = q.GetUnits()
				} else {
					ts.Units = fmt.Sprintf("%s", columnName)
				}

				db.DatabaseInsert(ctx, "quandl", ts, "")

				return *ts, true
			}

		}

		return nil, false
	})

	// f, found := c.Retrieve(ctx, ticker)

	return c
}

func getQuandlTimeSeriesFromDatabase(ctx context.Context, ticker string) (*timeseries.TimeSeries, error) {
	var ts timeseries.TimeSeries

	_, err := db.GetFromField(ctx, "quandl", "Name", ticker, &ts)

	return &ts, err
}

func GetSP500Constituents() ([]component.QueryComponent, []float64) {

	// Since quandl doesn't have time-varying constituents, we always just return the current S&P 500 constituents
	// utnil we have a better data source
	identifier, description := list.GetSP500Constituents()

	universeMembers := make([]component.QueryComponent, len(identifier), len(identifier))
	weights := make([]float64, len(identifier), len(identifier))

	for i, _ := range identifier {
		universeMembers[i] = component.QueryComponent{i, description[i], description[i], "Universe", identifier[i], component.QuandlOpenData, "", nil}
		weights[i] = 1
	}

	return universeMembers, weights
}

func GetCurrencies(ctx context.Context) ([]component.QueryComponent, []float64) {
	identifier, description := quandl.GetCurrenciesList(ctx)

	universeMembers := make([]component.QueryComponent, len(identifier), len(identifier))
	weights := make([]float64, len(identifier), len(identifier))

	for i, _ := range identifier {
		universeMembers[i] = component.QueryComponent{i, description[i], description[i], "Universe", identifier[i], component.QuandlOpenData, "", nil}
		weights[i] = 1
	}

	return universeMembers, weights
}

func GetCommodities(ctx context.Context) ([]component.QueryComponent, []float64) {
	identifier, description := quandl.GetCommoditiesList(ctx)

	universeMembers := make([]component.QueryComponent, len(identifier), len(identifier))
	weights := make([]float64, len(identifier), len(identifier))

	for i, _ := range identifier {
		universeMembers[i] = component.QueryComponent{i, description[i], description[i], "Universe", identifier[i], component.QuandlOpenData, "", nil}
		weights[i] = 1
	}

	return universeMembers, weights
}

func GetISharesPopular() ([]component.QueryComponent, []float64) {
	identifier, description := quandl.GetISharesPopular()

	universeMembers := make([]component.QueryComponent, len(identifier), len(identifier))
	weights := make([]float64, len(identifier), len(identifier))

	for i, _ := range identifier {
		universeMembers[i] = component.QueryComponent{i, description[i], description[i], "Universe", identifier[i], component.QuandlOpenData, "", nil}
		weights[i] = 1
	}

	return universeMembers, weights
}

func GetSectorETFs() ([]component.QueryComponent, []float64) {
	identifier, description := quandl.GetSectorETFs()

	universeMembers := make([]component.QueryComponent, len(identifier), len(identifier))
	weights := make([]float64, len(identifier), len(identifier))

	for i, _ := range identifier {
		universeMembers[i] = component.QueryComponent{i, description[i], description[i], "Universe", identifier[i], component.QuandlOpenData, "", nil}
		weights[i] = 1
	}

	return universeMembers, weights
}

func GetSP500SectorConstituents(sector string, level list.SectorLevel) ([]component.QueryComponent, []float64) {
	// identifier, sectorNames, stockNames := list.GetSP500SectorMappings()
	var identifier, sectorNames, stockNames []string

	if level == list.Sector || level == list.IndustryGroup {
		identifier, sectorNames, stockNames = list.GetSectorMappings(level, 500)
	} else {
		identifier, sectorNames, stockNames = list.GetSectorMappings(level, list.MAX_RANK)
	}

	universeMembers := make([]component.QueryComponent, 0, len(identifier))
	weights := make([]float64, 0, len(identifier))

	for i, v := range identifier {
		if sectorNames[i] == sector {
			universeMembers = append(universeMembers, component.QueryComponent{i, stockNames[i], stockNames[i], "Universe", v, component.QuandlOpenData, "", nil})
			weights = append(weights, 1)
		}
	}

	return universeMembers, weights
}

func GetAllSectorMappings(level list.SectorLevel) ([]component.QueryComponent, []string, []float64) {
	// identifier, sectorNames, stockNames := quandl.GetSP500SectorMappings()
	identifier, sectorNames, stockNames := list.GetSectorMappings(level, list.MAX_RANK)

	universeMembers := make([]component.QueryComponent, 0, len(identifier))
	weights := make([]float64, 0, len(identifier))

	for i, v := range identifier {
		//if sectorNames[i] == sector {
		universeMembers = append(universeMembers, component.QueryComponent{i, stockNames[i], stockNames[i], "Universe", v, component.QuandlOpenData, "", nil})
		weights = append(weights, 1)
		//}
	}

	return universeMembers, sectorNames, weights
}
