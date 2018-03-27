package run

import (
	"fmt"
	"math"
	"time"
	//"fmt"
	"sort"
	//"strconv"
	"strings"
	//"time"
	"github.com/AlphaHat/regression"
)

func getDataEnd(m MultiEntityData) string {
	//var dataEnd time.Time

	//for _, v := range m.EntityData {
	//	for _, v2 := range v.Data {
	//		if len(v2.Data) > 0 && dataEnd.Before(v2.Data[len(v2.Data)-1].Time) {
	//			dataEnd = v2.Data[len(v2.Data)-1].Time
	//		}
	//	}
	//}

	dataEnd := m.LastDay()

	return dataEnd.String()[0:10]
}

//func getXAxisTypeAndLabel(m MultiEntityData) (string, string) {
//	chartType := getChartType(m)
//	var xAxisType string

//	//fmt.Printf("chartType=%s\n", chartType)
//	if chartType == "column" {
//		xAxisType = "category"
//	} else {
//		xAxisType = "datetime"
//	}

//	return xAxisType, ""
//}

func getPlotLineValue(m MultiEntityData, s string) float64 {
	var data float64

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if v2.Meta.Label == s {
				for _, v3 := range v2.Data {
					data = v3.Data
				}
			}
		}
	}

	fmt.Printf("\n\nDATA = %v\n\n\n", data)

	return data
}

func getPlotLines(m MultiEntityData, chartOptions ChartOptions) []map[string]interface{} {
	plotLinesInner := make([]map[string]interface{}, 0)

	for _, v := range chartOptions.SeriesOptions {
		if v.ChartType == "plotLines" {

			plotLinesInner = append(plotLinesInner,
				map[string]interface{}{
					"color":  "#FF0000",
					"width":  2,
					"value":  getPlotLineValue(m, v.Name),
					"zIndex": 100,
					"label": map[string]interface{}{
						"text":  v.Name,
						"align": "right",
						"style": map[string]interface{}{
							"color": "#FF0000",
						},
					},
				})
		}
	}

	return plotLinesInner
}

func getYAxisHighcharts(m MultiEntityData, chartOptions ChartOptions) []map[string]interface{} {
	units := m.GetUnits()
	fields := m.GetFields()
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	var yAxis []map[string]interface{}

	if chartType != "heatmap" {
		yAxis = make([]map[string]interface{}, len(units), len(units))

		for i, v := range units {
			var text string
			var opposite bool = (i % 2) == 1

			if (len(fields)) == 1 {
				text = fields[0] + " (" + v + ") "
			} else {
				text = "(" + v + ")"
			}

			yAxis[i] = map[string]interface{}{
				"opposite": opposite,
				"type":     "linear",
				"title": map[string]interface{}{
					"text": smartBreak(text),
				},
			}

			plotLines := getPlotLines(m, chartOptions)

			if len(plotLines) > 0 {
				yAxis[i]["plotLines"] = plotLines
			}
		}

	} else {

		yAxis = make([]map[string]interface{}, 1)

		yAxis[0] = map[string]interface{}{
			"opposite": false,
			"type":     "category",
			"title": map[string]interface{}{
				"text": "",
			},
			"categories": fields,
		}

	}

	return yAxis
}

func getYAxisHistogram(m MultiEntityData) []map[string]interface{} {
	var yAxis []map[string]interface{}

	yAxis = make([]map[string]interface{}, 1)

	yAxis[0] = map[string]interface{}{
		"opposite": false,
		"type":     "linear",
		"title": map[string]interface{}{
			"text": "Count (#)",
		},
	}

	return yAxis
}

func getYAxisNum(m MultiEntityData, sMeta SeriesMeta) int {
	units := m.GetUnits()

	for i, v := range units {
		if sMeta.Units == v {
			return i
		}
	}

	return 0

	//for _, v := range m.EntityData {
	//	for _, v2 := range v.Data {
	//		if v2.Meta.Label == sMeta.Label && v.Meta.UniqueId == eMeta.UniqueId {

	//		}
	//	}
	//}

	//return yAxisNum
}

//func getYAxisTypeAndLabel(m MultiEntityData) (string, string) {
//	for _, v := range m.EntityData {
//		for _, v2 := range v.Data {
//			return "linear", v2.Meta.Label + " (" + v2.Meta.Units + ")"
//		}
//	}

//	return "linear", ""
//}

func getChartTypeAndXAxisType(m MultiEntityData, chartOptions ChartOptions) (string, string, bool) {

	var isOneDay bool = true
	var containsPercentPositive bool = false

	// If only one day of data, this is a column chart
	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if len(v2.Data) > 1 {
				isOneDay = false
			}
			if v2.Meta.Label == "% Positive" {
				containsPercentPositive = true
			}
		}
	}

	if m.GraphicalPreference == "Scatter" || chartOptions.ChartType == "scatter" {
		return "scatter", "linear", isOneDay
	}

	if m.GraphicalPreference == "Column" || chartOptions.ChartType == "column" {
		return "column", "datetime", isOneDay
	}

	if m.GraphicalPreference == "Heatmap" || chartOptions.ChartType == "heatmap" {
		return "heatmap", "category", isOneDay
	}

	if m.GraphicalPreference == "Boxplot" || chartOptions.ChartType == "boxplot" {
		return "boxplot", "category", isOneDay
	}

	if m.GraphicalPreference == "Histogram" || chartOptions.ChartType == "histogram" {
		return "column", "category", isOneDay
	}

	if chartOptions.ChartType == "area" {
		return "area", "datetime", isOneDay
	}

	if isOneDay && containsPercentPositive {
		return "bar", "category", isOneDay
	}

	if isOneDay {
		return "column", "category", isOneDay
	}

	return "line", "datetime", isOneDay
}

func sanitizeMultiEntityData(m MultiEntityData) MultiEntityData {
	for i, _ := range m.EntityData {
		for j, _ := range m.EntityData[i].Data {
			for k, v := range m.EntityData[i].Data[j].Data {
				if math.IsInf(v.Data, 1) || math.IsInf(v.Data, -1) || math.IsNaN(v.Data) {
					m.EntityData[i].Data[j].Data[k].Data = 0
				}
			}
		}
	}

	return m
}

func getTitle(m MultiEntityData, chartOptions ChartOptions) string {
	if chartOptions.Title != "" {
		return chartOptions.Title
	}

	return m.Title
}

func isBlank(chartOptions ChartOptions) bool {
	if chartOptions.Title != "" {
		return false
	}

	if len(chartOptions.Colors) > 0 {
		return false
	}

	if chartOptions.ChartType != "" {
		return false
	}

	if len(chartOptions.SeriesOptions) > 0 {
		return false
	}

	return true
}

func editMultiEntityDataWithOptions(m MultiEntityData, chartOptions ChartOptions) MultiEntityData {
	for i, v := range m.EntityData {
		// TODO: Will need to be able to remove individual entities, but for now just remove/override
		temp := make([]Series, 0, len(v.Data))

		for _, v2 := range v.Data {
			seriesOption := chartOptions.GetSeriesOptionsFromOriginalName(v2.Meta.Label)

			if seriesOption.NewName != "" {
				v2.Meta.Label = seriesOption.NewName
			}

			if seriesOption.NewUnits != "" {
				v2.Meta.Units = seriesOption.NewUnits
			}

			if !seriesOption.Disabled {
				temp = append(temp, v2)
			}
		}

		m.EntityData[i].Data = temp
	}

	return m
}

func ConvertMultiEntityDataToHighcharts(m MultiEntityData, chartOptions ChartOptions) map[string]interface{} {
	m = sanitizeMultiEntityData(m)
	m = editMultiEntityDataWithOptions(m, chartOptions)

	hc := getDefaultHighchart()

	// Set title
	hc["title"] = map[string]interface{}{"text": getTitle(m, chartOptions)}

	hc["credits"] = map[string]interface{}{
		"text":    "Source: Alpha Hat", // + getMergedSource(m),
		"enabled": true,
		// "href":    "http://www.quandl.com",
	}

	// Set the type
	// TODO: In the future, we may need to set multiple types per series
	chartType, xAxisType, isOneDay := getChartTypeAndXAxisType(m, chartOptions)

	hc["chart"] = map[string]interface{}{
		"type":      chartType,
		"zoomType":  "xy",
		"animation": true,
	}

	hc["xAxis"] = map[string]interface{}{
		"type": xAxisType,
		"title": map[string]interface{}{
			"text": "",
		},
	}

	if m.GraphicalPreference == "Histogram" || chartOptions.ChartType == "histogram" {
		hc = getHistogram(m, chartOptions, hc)
	} else if chartType == "heatmap" {
		hc = getHeatmapSeries(m, chartOptions, hc)
	} else if chartType == "boxplot" {
		hc = getBoxplotSeries(m, chartOptions, hc)
	} else if chartType == "bar" {
		hc = getCategorySeriesEntityFirstSummaryStats(m, chartOptions, hc)
	} else if xAxisType == "category" {
		hc = getCategorySeriesEntityFirst(m, chartOptions, hc)
	} else if chartType == "scatter" && !isOneDay {
		hc = getScatterSeries(m, chartOptions, hc)
	} else if chartType == "scatter" && isOneDay {
		hc = getCrossEntityScatter(m, chartOptions, hc)
	} else if chartType == "column" {
		hc = getCategorySeriesTimeSeries(m, chartOptions, hc)
	} else {
		hc = getTimeSeries(m, chartOptions, hc)
	}

	hc = overridePlotOptions(chartOptions, hc)

	return hc
}

func overridePlotOptions(chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	var plotOptions map[string]interface{}
	var series map[string]interface{}

	if _, found := hc["plotOptions"]; found {
		plotOptions = hc["plotOptions"].(map[string]interface{})
	} else {
		plotOptions = make(map[string]interface{})
	}

	if _, found := plotOptions["series"]; found {
		series = hc["series"].(map[string]interface{})
	} else {
		series = make(map[string]interface{})
	}

	series["animation"] = isBlank(chartOptions)

	showMarker := false

	for _, v := range chartOptions.SeriesOptions {
		if v.MarkerSize != "" {
			showMarker = true
		}
	}

	series["marker"] = map[string]interface{}{
		"enabled": showMarker,
	}

	plotOptions["series"] = series

	hc["plotOptions"] = plotOptions
	return hc
}

func getBoxplotSeries(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	var boxPlotAggregator map[string][]float64 = make(map[string][]float64)

	// Loop through entities and accumulate all the data for a given label (time)
	for _, v := range m.EntityData {
		for _, v2 := range v.Data[0].Data {
			label := v2.Time.String()[0:10]

			_, found := boxPlotAggregator[label]
			if !found {
				boxPlotAggregator[label] = make([]float64, 0)
			}
			boxPlotAggregator[label] = append(boxPlotAggregator[label], v2.Data)
		}
	}

	var categories []string = make([]string, 0)
	for k, _ := range boxPlotAggregator {
		categories = append(categories, k)
	}

	sort.Strings(categories)

	data := make([][]interface{}, 0)

	for _, k := range categories {
		v2 := boxPlotAggregator[k]
		tempBoxplot := make([]interface{}, len(v2))
		for j, v3 := range v2 {
			tempBoxplot[j] = v3
		}
		data = append(data, tempBoxplot)

	}
	fieldNames := m.GetFields()
	hc["xAxis"].(map[string]interface{})["categories"] = categories

	hc["series"] = make([]map[string]interface{}, 1, 1)

	hc["series"].([]map[string]interface{})[0] = map[string]interface{}{
		"name":  fieldNames[0],
		"type":  chartType,
		"data":  data,
		"yAxis": 0,
	}

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	return hc
}

func getMaxMinFloat(d []float64) (float64, float64) {
	var max float64 = math.Inf(-1)
	var min float64 = math.Inf(1)

	for _, v := range d {
		if v > max {
			max = v
		}

		if v < min {
			min = v
		}
	}

	return max, min
}

func getBinBounds(numBins int64, d []float64) ([]float64, []float64) {
	var binLowerBound []float64 = make([]float64, numBins, numBins)
	var binUpperBound []float64 = make([]float64, numBins, numBins)

	// Get the max and min
	max, min := getMaxMinFloat(d)

	for i, _ := range binLowerBound {
		binLowerBound[i] = min + (float64(i) * (max - min) / float64(numBins))
		binUpperBound[i] = min + (float64(i+1) * (max - min) / float64(numBins))
	}

	return binLowerBound, binUpperBound
}

func countBinData(d []float64, binLowerBound []float64, binUpperBound []float64) []float64 {
	var binCount []float64 = make([]float64, len(binLowerBound))

	for i, _ := range binCount {
		for _, v := range d {
			if i == 0 {
				if v <= binUpperBound[i] {
					binCount[i]++
				}
			} else {
				if binLowerBound[i] < v && v <= binUpperBound[i] {
					binCount[i]++
				}
			}
		}
	}

	return binCount
}

func getHistogram(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	categories := m.GetEntities()

	fields := m.GetFields()

	hc["series"] = make([]map[string]interface{}, len(fields), len(fields))
	var labels []string

	for fieldNum, v := range fields {
		data, _ := getDataFromCategoryAndField(m, categories, v)
		var numBins int64 = int64(math.Sqrt(float64(len(data))))
		binLowerBound, binUpperBound := getBinBounds(numBins, data)
		data = countBinData(data, binLowerBound, binUpperBound)

		//fmt.Printf("data=%s\n", data)

		hc["series"].([]map[string]interface{})[fieldNum] = map[string]interface{}{
			"name":  v,
			"type":  chartType,
			"data":  data,
			"yAxis": 0,
		}

		labels = make([]string, len(data))
		for i, _ := range binLowerBound {
			labels[i] = prettyLabel(binLowerBound[i], v) + " to " + prettyLabel(binUpperBound[i], v)
		}
	}

	hc["xAxis"].(map[string]interface{})["categories"] = labels
	//hc["xAxis"].(map[string]interface{})["title"] = map[string]interface{}{
	//	"text": fields[0] + " (" + units[0] + ")",
	//}

	hc["yAxis"] = getYAxisHistogram(m)

	hc["plotOptions"] = map[string]interface{}{
		"column": map[string]interface{}{
			"pointPadding": 0,
			"borderWidth":  0,
			"groupPadding": 0,
			"shadow":       false,
		},
	}

	return hc

}

func prettyLabel(d float64, seriesLabel string) string {
	if d > 1e9 {
		return fmt.Sprintf("%.1f B", d/1e9)
	}

	if d > 1e6 {
		return fmt.Sprintf("%.1f MM", d/1e6)
	}

	if d > 1e3 {
		return fmt.Sprintf("%.1f k", d/1e3)
	}

	if strings.Contains(seriesLabel, "%") {
		return fmt.Sprintf("%.1f %%", d*100)
	}
	return fmt.Sprintf("%.2f", d)
}

func getCategorySeriesTimeSeries(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	uniqueDates := m.UniqueDates()

	categories := make([]string, len(uniqueDates))

	for i, _ := range uniqueDates {
		categories[i] = uniqueDates[i].Format("Jan 02 '06")
	}

	numFields := len(m.GetFields())

	showUnits := numFields > 1

	hc["series"] = make([]map[string]interface{}, 0, m.NumSeries())

	var seriesNum int = 0

	for _, v2 := range m.EntityData {
		for _, v := range v2.Data {

			var data []interface{} = make([]interface{}, len(uniqueDates), len(uniqueDates))

			for i, _ := range uniqueDates {
				dataFound, _ := v.Find(uniqueDates[i])
					data[i] = dataFound.Data
			}

			axisNum := getYAxisNum(m, v.Meta)

			var suffix string
			if showUnits {
				suffix = " " + v.Meta.Label
			}

			seriesTemp := map[string]interface{}{
				"name":  v2.Meta.Name + suffix,
				"type":  chartType,
				"data":  data,
				"yAxis": axisNum,
			}

			hc["series"] = append(hc["series"].([]map[string]interface{}), seriesTemp)

			seriesNum = seriesNum + 1
		}
	}

	hc["xAxis"].(map[string]interface{})["categories"] = categories

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	return hc
}

func getCategorySeriesEntityFirst(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	categories := m.GetEntities()

	fields := m.GetFields()

	hc["series"] = make([]map[string]interface{}, len(fields), len(fields))

	for fieldNum, v := range fields {
		data, yAxisNum := getDataFromCategoryAndField(m, categories, v)

		hc["series"].([]map[string]interface{})[fieldNum] = map[string]interface{}{
			"name":  v,
			"type":  chartType,
			"data":  data,
			"yAxis": yAxisNum,
		}
	}

	hc["xAxis"].(map[string]interface{})["categories"] = categories

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	return hc
}

func getCategorySeriesEntityFirstSummaryStats(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	categories := m.GetEntities()

	fields := m.GetFields()

	newFields := make([]string, 0)
	for _, v := range fields {
		if v != "Count" && v != "% Positive" {
			newFields = append(newFields, v)
		}
	}
	fields = newFields

	hc["series"] = make([]map[string]interface{}, len(fields)+1)

	var maxData float64 = 0
	var minData float64 = math.Inf(1)
	var yAxisNum int
	var data []float64

	for fieldNum, v := range fields {
		data, yAxisNum = getDataFromCategoryAndField(m, categories, v)

		for _, v2 := range data {
			if v2 > maxData {
				maxData = v2
			}
			if v2 < minData {
				minData = v2
			}
		}

		hc["series"].([]map[string]interface{})[fieldNum] = map[string]interface{}{
			"name":       v,
			"type":       chartType,
			"data":       data,
			"yAxis":      yAxisNum,
			"dataLabels": map[string]interface{}{"enabled": true, "inside": true, "align": "right"},
		}
	}

	data, yAxisNumCount := getDataFromCategoryAndField(m, categories, "Count")

	hc["series"].([]map[string]interface{})[len(fields)] = map[string]interface{}{
		"name":  "Count",
		"type":  chartType,
		"data":  data,
		"yAxis": yAxisNumCount,
		"color": "rgba(255, 255, 255, 0)",
		"dataLabels": map[string]interface{}{
			"enabled": true,
			"inside":  true,
			"align":   "left",
			"format":  "{y} instances",
		},
	}

	hc["series"] = append(hc["series"].([]map[string]interface{}), getPieSeriesPositive(m, categories)...)

	hc["xAxis"].(map[string]interface{})["categories"] = categories

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)
	hc["yAxis"].([]map[string]interface{})[yAxisNumCount]["visible"] = false

	if maxData > 0 {
		hc["yAxis"].([]map[string]interface{})[yAxisNum]["max"] = maxData * 1.50
	} else {
		hc["yAxis"].([]map[string]interface{})[yAxisNum]["max"] = minData * -0.5
	}

	return hc
}

func getPieSeriesPositive(m MultiEntityData, categories []string) []map[string]interface{} {
	data, yAxisNum := getDataFromCategoryAndField(m, categories, "% Positive")

	size := (1.0 / float64(len(data))) / 2.0

	rv := make([]map[string]interface{}, len(data))

	for i, v := range data {
		rv[i] = map[string]interface{}{
			"data": []map[string]interface{}{
				map[string]interface{}{
					"name": "% Positive",
					"y":    v,
					"dataLabels": map[string]interface{}{
						"enabled": true,
						"format":  "{point.percentage:.0f}%<br>Positive",
					},
					"color": "#20BF55",
				},
				map[string]interface{}{
					"name": "% Negative",
					"y":    1 - v,
					"dataLabels": map[string]interface{}{
						"enabled": false,
					},
					"color": "#F6511D",
				},
			},
			"size":      fmt.Sprintf("%0.1f%%", size*100),
			"center":    []string{"90%", fmt.Sprintf("%0.1f%%", 100*(size+(float64(i)*2.0*size)))},
			"innerSize": "50%",
			"name":      "% Positive",
			"type":      "pie",
			"yAxis":     yAxisNum,
			"dataLabels": map[string]interface{}{
				"enabled":  true,
				"distance": 0,
			},
		}
	}

	return rv
}

func getHeatmapSeries(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	categories := m.GetEntities()

	fields := m.GetFields()

	hc["series"] = make([]map[string]interface{}, len(fields), len(fields))

	for fieldNum, v := range fields {
		data, _ := getDataFromCategoryAndField(m, categories, v)

		data2 := make([][]interface{}, len(data))
		for i, _ := range data {
			if data[i] >= 1 || data[i] <= -1 {
				data2[i] = []interface{}{fieldNum, fmt.Sprintf("%.2f", data[i])}
			} else {
				data2[i] = []interface{}{fieldNum, fmt.Sprintf("%.4f", data[i])}
			}
		}

		hc["series"].([]map[string]interface{})[fieldNum] = map[string]interface{}{
			"name":  v,
			"type":  chartType,
			"data":  data2,
			"yAxis": 0,
			"dataLabels": map[string]interface{}{
				"enabled": true,
				"color":   "black",
				"style": map[string]interface{}{
					"textShadow": "none",
				},
			},
		}
	}

	hc["xAxis"].(map[string]interface{})["categories"] = categories

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	//yAxisType, yAxisLabel := getYAxisTypeAndLabel(m)

	//hc["yAxis"] = make([]map[string]interface{}, 1, 1)

	//hc["yAxis"].([]map[string]interface{})[0] = map[string]interface{}{
	//	"opposite": false,
	//	"type":     yAxisType,
	//	"title": map[string]interface{}{
	//		"text": yAxisLabel,
	//	},
	//}

	return hc
}

func getDataFromCategoryAndField(m MultiEntityData, entities []string, field string) ([]float64, int) {
	data := make([]float64, len(entities), len(entities))
	var yAxisNum int

	for i, v := range entities {
		for _, v2 := range m.EntityData {
			if v2.Meta.Name == v {
				for _, v3 := range v2.Data {
					if len(v3.Data) > 0 && v3.Meta.Label == field {
						data[i] = v3.Data[0].Data
						yAxisNum = getYAxisNum(m, v3.Meta)
					}
				}

			}
		}

	}

	return data, yAxisNum
}

func getCategorySeriesDataTypeFirst(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	fields := m.GetFields()
	hc["series"] = make([]map[string]interface{}, m.NumEntities(), m.NumEntities())

	hc["xAxis"].(map[string]interface{})["categories"] = fields

	for entityNum, v := range m.EntityData {
		data, yAxisNum := getFieldArray(m, fields, v.Data)

		hc["series"].([]map[string]interface{})[entityNum] = map[string]interface{}{
			"name":  v.Meta.Name,
			"type":  chartType,
			"data":  data,
			"yAxis": yAxisNum,
		}
	}

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	//yAxisType, yAxisLabel := getYAxisTypeAndLabel(m)

	//hc["yAxis"] = make([]map[string]interface{}, 1, 1)

	//hc["yAxis"].([]map[string]interface{})[0] = map[string]interface{}{
	//	"opposite": false,
	//	"type":     yAxisType,
	//	"title": map[string]interface{}{
	//		"text": yAxisLabel,
	//	},
	//}

	return hc
}

func getFieldArray(m MultiEntityData, fields []string, s []Series) ([]float64, int) {
	data := make([]float64, len(fields))
	var yAxisNum int

	for i, _ := range fields {
		for _, v2 := range s {
			if v2.Meta.Label == fields[i] && len(v2.Data) > 0 {
				data[i] = v2.Data[0].Data
				yAxisNum = getYAxisNum(m, v2.Meta)
			}
		}
	}

	return data, yAxisNum
}

func getCrossEntityScatter(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	var r regression.Regression
	var r2 regression.Regression

	categories := m.GetEntities()

	fields := m.GetFields()
	units := m.GetUnitsFromFields(fields)

	hc["series"] = make([]map[string]interface{}, 2, 2)

	if len(fields) < 2 {
		return hc
	}

	dataX, _ := getDataFromCategoryAndField(m, categories, fields[0])
	dataY, _ := getDataFromCategoryAndField(m, categories, fields[1])

	var minX float64 = math.MaxFloat64
	var maxX float64 = -math.MaxFloat64

	multiplier := 100.0

	var data []map[string]interface{} = make([]map[string]interface{}, len(dataX))
	for i, _ := range dataX {
		point := make(map[string]interface{})

		point["name"] = categories[i]
		point["y"] = dataY[i]
		point["x"] = dataX[i]

		if dataX[i] < minX {
			minX = dataX[i]
		}
		if dataX[i] > maxX {
			maxX = dataX[i]
		}

		data[i] = point
		r.AddDataPoint(regression.DataPoint{Observed: dataY[i], Variables: []float64{dataX[i]}})
		r2.AddDataPoint(regression.DataPoint{Observed: multiplier * dataY[i], Variables: []float64{dataX[i], dataX[i] * dataX[i]}})
	}

	hc["series"].([]map[string]interface{})[0] = map[string]interface{}{
		"name":  fields[1] + "<br> vs. " + fields[0],
		"type":  chartType,
		"data":  data,
		"yAxis": 0,
	}

	r.RunLinearRegression()
	r2.RunLinearRegression()

	if r.Rsquared > r2.Rsquared {
		regressionLine := []map[string]float64{
			map[string]float64{
				"y": getCorrespondingY(minX, r),
				"x": minX,
			},
			map[string]float64{
				"y": getCorrespondingY(maxX, r),
				"x": maxX,
			},
		}

		hc["series"].([]map[string]interface{})[1] = map[string]interface{}{
			"name":  fmt.Sprintf("y = %.2f + %.2f x (R²=%.2f)", r.GetRegCoeff(0), r.GetRegCoeff(1), r.Rsquared),
			"type":  "line",
			"data":  regressionLine,
			"yAxis": 0,
		}
	} else {
		regressionLine := make([]map[string]float64, len(data))

		sort.Float64s(dataX)

		for i, v := range dataX {
			regressionLine[i] = map[string]float64{
				"y": getCorrespondingYsquare(v, r2) / multiplier,
				"x": v,
			}
		}
		hc["series"].([]map[string]interface{})[1] = map[string]interface{}{
			"name":  fmt.Sprintf("y = %.2f + %.2f x + %.2f x² (R²=%.2f)\n", r2.GetRegCoeff(0), r2.GetRegCoeff(1), r2.GetRegCoeff(2), r2.Rsquared),
			"type":  "line",
			"data":  regressionLine,
			"yAxis": 0,
		}
	}

	hc["yAxis"] = make([]map[string]interface{}, 1)
	hc["yAxis"].([]map[string]interface{})[0] = map[string]interface{}{
		"opposite": false,
		"type":     "linear",
		"title": map[string]interface{}{
			"text": smartBreak(fields[1] + " (" + units[1] + ")"),
		},
	}

	hc["xAxis"] = map[string]interface{}{
		"opposite": false,
		"type":     "linear",
		"title": map[string]interface{}{
			"text": fields[0] + " (" + units[0] + ")",
		},
	}

	hc["plotOptions"] = map[string]interface{}{
		"scatter": map[string]interface{}{
			"tooltip": map[string]interface{}{
				"headerFormat": "",
				"pointFormat":  "<b>{point.name}</b><br>x: {point.x}<br>y: {point.y}",
			},
		},
	}

	hc["plotOptions"].(map[string]interface{})["scatter"].(map[string]interface{})["dataLabels"] = map[string]interface{}{
		"enabled": true,
		"format":  "{point.name}",
	}

	return hc
}

func getScatterSeries(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	var r regression.Regression

	hc["series"] = make([]map[string]interface{}, 2, 2)

	var series1 Series
	var entity1 EntityMeta
	var series2 Series
	var entity2 EntityMeta

	// Extract the first two series that we encounter
	for _, v2 := range m.EntityData {
		for _, v := range v2.Data {
			if entity1.UniqueId == "" {
				entity1 = v2.Meta
				series1 = v
				if entity1.UniqueId == "" {
					entity1.UniqueId = time.Now().String()
				}
			} else if entity2.UniqueId == "" {
				entity2 = v2.Meta
				series2 = v
				if entity2.UniqueId == "" {
					entity2.UniqueId = time.Now().String()
				}
			} else {
				break
			}
		}
	}

	var minX float64 = math.MaxFloat64
	var maxX float64 = -math.MaxFloat64

	// Line up the dates and add series to the data map
	var data []map[string]interface{} = make([]map[string]interface{}, 0, 0)
	for _, v := range series1.Data {

		for _, v2 := range series2.Data {
			if v.Time.Equal(v2.Time) {
				point := make(map[string]interface{})

				point["name"] = v.Time.String()[0:10]
				point["y"] = v.Data
				point["x"] = v2.Data

				if v2.Data < minX {
					minX = v2.Data
				}
				if v2.Data > maxX {
					maxX = v2.Data
				}

				r.AddDataPoint(regression.DataPoint{Observed: v.Data, Variables: []float64{v2.Data}})

				data = append(data, point)
			}
		}

	}

	r.RunLinearRegression()

	hc["series"].([]map[string]interface{})[0] = map[string]interface{}{
		"name":  entity1.Name + " " + series1.Meta.Label + "<br> vs. " + entity2.Name + " " + series2.Meta.Label,
		"type":  chartType,
		"data":  data,
		"yAxis": 0,
	}

	regressionLine := []map[string]float64{
		map[string]float64{
			"y": getCorrespondingY(minX, r),
			"x": minX,
		},
		map[string]float64{
			"y": getCorrespondingY(maxX, r),
			"x": maxX,
		},
	}

	hc["series"].([]map[string]interface{})[1] = map[string]interface{}{
		"name":  fmt.Sprintf("y = %.2f + %.2f x (R²=%.2f)", r.GetRegCoeff(0), r.GetRegCoeff(1), r.Rsquared),
		"type":  "line",
		"data":  regressionLine,
		"yAxis": 0,
	}

	hc["yAxis"] = make([]map[string]interface{}, 1)
	hc["yAxis"].([]map[string]interface{})[0] = map[string]interface{}{
		"opposite": false,
		"type":     "linear",
		"title": map[string]interface{}{
			"text": smartBreak(entity1.Name + " " + series1.Meta.Label + " (" + series1.Meta.Units + ")"),
		},
	}

	hc["xAxis"] = map[string]interface{}{
		"opposite": false,
		"type":     "linear",
		"title": map[string]interface{}{
			"text": entity2.Name + " " + series2.Meta.Label + "<br> (" + series2.Meta.Units + ")",
		},
	}

	hc["plotOptions"] = map[string]interface{}{
		"scatter": map[string]interface{}{
			"tooltip": map[string]interface{}{
				"headerFormat": "",
				"pointFormat":  "<b>{point.name}</b><br>x: {point.x}<br>y: {point.y}",
			},
		},
	}

	//if len(series1.Data) <= 20 {
	hc["plotOptions"].(map[string]interface{})["scatter"].(map[string]interface{})["dataLabels"] = map[string]interface{}{
		"enabled": true,
		"format":  "{point.name}",
	}
	//}

	return hc
}

func getCorrespondingY(x float64, r regression.Regression) float64 {
	return r.GetRegCoeff(0) + r.GetRegCoeff(1)*x
}

func getCorrespondingYsquare(x float64, r regression.Regression) float64 {
	return r.GetRegCoeff(0) + r.GetRegCoeff(1)*x + r.GetRegCoeff(2)*x*x
}

func getTimeSeries(m MultiEntityData, chartOptions ChartOptions, hc map[string]interface{}) map[string]interface{} {
	chartType, _, _ := getChartTypeAndXAxisType(m, chartOptions)

	numFields := len(m.GetFields())

	showUnits := numFields > 1

	hc["series"] = make([]map[string]interface{}, 0, m.NumSeries())

	var seriesNum int = 0

	for _, v2 := range m.EntityData {
		for _, v := range v2.Data {

			var data [][]interface{} = make([][]interface{}, len(v.Data), len(v.Data))

			for i, d := range v.Data {

				data[i] = make([]interface{}, 2)
				if d.Time.Before(time.Date(1850, 01, 01, 0, 0, 0, 0, time.UTC)) {
					data[i][0] = d.Time.UTC().Sub(zeroDay()).Hours() / 24
					data[i][1] = d.Data
					hc["xAxis"].(map[string]interface{})["type"] = "linear"
					hc["xAxis"].(map[string]interface{})["title"] = map[string]string{
						"text": "Days",
					}
				} else {
					data[i][0] = d.Time.UTC().String()[0:10]
					data[i][1] = d.Data
				}
			}

			axisNum := getYAxisNum(m, v.Meta)

			var suffix string
			if showUnits {
				suffix = " " + v.Meta.Label
			}

			lineWidth := chartOptions.GetSeriesOptionsFromNewName(v.Meta.Label).LineWidth
			thisChartType := chartOptions.GetSeriesOptionsFromNewName(v.Meta.Label).ChartType

			if thisChartType == "" {
				thisChartType = chartType
			}

			if thisChartType != "plotLines" {
				seriesTemp := map[string]interface{}{
					"name":  v2.Meta.Name + suffix,
					"type":  thisChartType,
					"data":  data,
					"yAxis": axisNum,
				}

				if lineWidth != "" {
					seriesTemp["lineWidth"] = lineWidth
				}

				markerSize := chartOptions.GetSeriesOptionsFromNewName(v.Meta.Label).MarkerSize

				if markerSize != "" {
					seriesTemp["marker"] = map[string]interface{}{
						"radius": markerSize,
					}
				}

				hc["series"] = append(hc["series"].([]map[string]interface{}), seriesTemp)

				seriesNum = seriesNum + 1
			}
		}
	}

	hc["yAxis"] = getYAxisHighcharts(m, chartOptions)

	return hc
}

func axisIsPrice(label string) bool {
	if strings.Contains(label, "Close") {
		return true
	}
	if strings.Contains(label, "close") {
		return true
	}
	if strings.Contains(label, "Price") {
		return true
	}
	if strings.Contains(label, "price") {
		return true
	}

	return false
}

func getDefaultHighchart() map[string]interface{} {
	hc := map[string]interface{}{
		"subtitle": map[string]interface{}{
			"text": "",
		},
		"navigation": map[string]interface{}{
			"buttonOptions": map[string]interface{}{
				"enabled": false,
			},
		},
	}
	return hc
}

func getMergedSource(m MultiEntityData) string {
	return strings.Join(getUniqueSources(m), ",  ")
}

func getUniqueSources(m MultiEntityData) []string {
	nonUnique := make([]string, 0, m.NumSeries())
	uniqueStrings := make([]string, 0, len(nonUnique))

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			nonUnique = append(nonUnique, v2.Meta.Source)
		}
	}

	sort.Strings(nonUnique)

	for i, v := range nonUnique {
		if (i == 0 || v != nonUnique[i-1]) && v != "" {
			uniqueStrings = append(uniqueStrings, v)
		}
	}

	return uniqueStrings
}

func smartBreak(s string) string {
	// If it's short, no break
	if len(s) < 60 {
		return s
	}
	sArr := strings.Split(s, " ")

	// If there's a paren, and the first several words make it less than 60,
	// put the break at the paren
	var lengthToParen = 0
	var parenIndex = -1
	for i, v := range sArr {
		if strings.Contains(v, "(") {
			parenIndex = i
		}
		lengthToParen = lengthToParen + len(v) + 1
	}

	if parenIndex > 0 && lengthToParen < 60 {
		return strings.Join(sArr[:parenIndex], " ") + " <br> " + strings.Join(sArr[parenIndex:], " ")
	}

	medianIndex := int(len(sArr) / 2)

	return strings.Join(sArr[:medianIndex], " ") + " <br> " + strings.Join(sArr[medianIndex:], " ")

}
