package run

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AlphaHat/gcp-alpha-hat/build"
	"github.com/AlphaHat/gcp-alpha-hat/component"
	"github.com/AlphaHat/gcp-alpha-hat/data"
	"github.com/AlphaHat/gcp-alpha-hat/db"
	"github.com/AlphaHat/gcp-alpha-hat/list"
	"github.com/AlphaHat/gcp-alpha-hat/term"
	"github.com/AlphaHat/gcp-alpha-hat/timeseries"
	"github.com/AlphaHat/gcp-alpha-hat/track"
	"github.com/AlphaHat/gcp-alpha-hat/zstats"
	"github.com/AlphaHat/regression"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
)

// Type Definitions
type ResampleType int

const (
	ResampleNone ResampleType = iota
	ResampleLastValue
	ResampleGeometric
	ResampleArithmetic
	ResampleZero
)

type DataPoint struct {
	Time time.Time
	Data float64
}

type CategoryPoint struct {
	Time time.Time
	Data int64
}

type CategoryLabel struct {
	Id    int64
	Label string
}

type SeriesMeta struct {
	VendorCode    string
	Label         string
	Units         string
	Source        string
	Upsample      ResampleType
	Downsample    ResampleType
	IsTransformed bool
}

type Series struct {
	Data     []DataPoint
	Meta     SeriesMeta
	IsWeight bool
}

type CategorySeries struct {
	Data   []CategoryPoint
	Labels []CategoryLabel
	//Meta   SeriesMeta
}

type EntityMeta struct {
	Name     string
	UniqueId string
	IsCustom bool
}

type SingleEntityData struct {
	Data     []Series
	Category CategorySeries
	Meta     EntityMeta
}

type MultiEntityData struct {
	EntityData          []SingleEntityData
	Title               string
	Error               string
	GraphicalPreference string
}

type ExecutionNode struct {
	Type      string
	Arguments []component.QueryComponent
	Children  []ExecutionNode
}

type EntityPlusDataPoint struct {
	EntityM  EntityMeta
	Data     DataPoint
	Category CategoryLabel
	Weight   DataPoint
}

type DataForAggregation struct {
	SeriesM SeriesMeta
	Data    []EntityPlusDataPoint
}

type StepFnType func(context.Context, []MultiEntityData) MultiEntityData

func (s Series) Len() int {
	return len(s.Data)
}

func (s Series) Less(i, j int) bool {
	return s.Data[i].Time.Before(s.Data[j].Time)
}

func (s Series) Swap(i, j int) {
	s.Data[i], s.Data[j] = s.Data[j], s.Data[i]
}

func (s Series) Find(date time.Time) (DataPoint, bool) {
	foundIdx := sort.Search(s.Len(), func(i int) bool {
		return !s.Data[i].Time.Before(date)
	})

	if foundIdx < s.Len() && s.Data[foundIdx].Time.Equal(date) {
		return s.Data[foundIdx], true
	}

	return DataPoint{}, false
}

// Methods on the types above
func (d DataForAggregation) Len() int {
	return len(d.Data)
}

func (d DataForAggregation) Less(i, j int) bool {
	return d.Data[i].Data.Data > d.Data[j].Data.Data
}

func (d DataForAggregation) Swap(i, j int) {
	d.Data[i], d.Data[j] = d.Data[j], d.Data[i]
}

func (d DataForAggregation) Chop(n int) DataForAggregation {
	if n < d.Len() {
		//d.Data = d.Data[:n]
		for i := n; i < d.Len(); i++ {
			d.Data[i].Weight.Data = 0
		}
	}

	return d
}

type ByWeight struct {
	DataForAggregation
}

func (d ByWeight) Less(i, j int) bool {
	return d.Data[i].Weight.Data > d.Data[j].Weight.Data
}

func (c CategorySeries) Insert(date time.Time, label CategoryLabel) CategorySeries {
	var id int64
	id, c.Labels = c.GetCategoryId(label)

	if len(c.Data) == 0 || c.Data[len(c.Data)-1].Data != id {
		// New label to be inserted differs from the current label
		if c.Data == nil {
			c.Data = make([]CategoryPoint, 0)
		}

		c.Data = append(c.Data, CategoryPoint{date, id})
	}

	return c
}

func (c CategorySeries) GetCategoryId(label CategoryLabel) (int64, []CategoryLabel) {
	var labelString string = label.Label
	var categoryId int64
	var maxCategoryId int64

	for _, v := range c.Labels {
		if v.Id > maxCategoryId {
			maxCategoryId = v.Id
		}
		if v.Label == labelString {
			categoryId = v.Id
		}
	}

	if categoryId == 0 {
		// New label has to be inserted
		categoryId = maxCategoryId + 1
		c.Labels = append(c.Labels, CategoryLabel{categoryId, label.Label})
	}

	return categoryId, c.Labels
}

func (c CategorySeries) GetCategoryLabel(id int64) string {
	for i, _ := range c.Labels {
		if c.Labels[i].Id == id {
			return c.Labels[i].Label
		}
	}

	return ""
}

func (c CategorySeries) CheckCategory(date time.Time, categoryName string) (bool, CategoryLabel) {
	var categoryLabel CategoryLabel

	if categoryName == "" {
		return true, categoryLabel
	}

	var categoryId int64 = -1

	for _, v := range c.Labels {
		if v.Label == categoryName {
			categoryId = v.Id
			categoryLabel = v
		}
	}

	// This label doesn't exist
	if categoryId == -1 {
		return false, categoryLabel
	}

	for i, v := range c.Data {
		if v.Data == categoryId && (date.After(v.Time) || date.Equal(v.Time)) {
			// The category matches and the date is after the current date we're checking

			// Now we need to check the next data point
			if (i+1) == len(c.Data) || c.Data[i+1].Time.After(date) {
				return true, categoryLabel
			}
		}
	}

	return false, categoryLabel
}

func (c CategorySeries) LookupCategory(date time.Time) string {
	var categoryId int64 = -1

	for _, v := range c.Data {
		if date.After(v.Time) || date.Equal(v.Time) {
			// The date is after the current date we're checking
			categoryId = v.Data
		}
	}

	for _, v := range c.Labels {
		if v.Id == categoryId {
			return v.Label
		}
	}

	// fmt.Printf("category lookup date = %s\n", date)
	// marshalOutput("category series", c)

	return ""
}

//func (a AggregateLabel) Insert(s SeriesMeta, e EntityMeta, d DataPoint) AggregateLabel {
//	var i = 0
//	for ; i < len(a); i++ {
//		if s.Label == a[i].SeriesM.Label {
//			a[i].Data = append(a[i].Data, EntityPlusDataPoint{e, d})
//			return a
//		}
//	}

//	eArr := make([]EntityPlusDataPoint, 1)
//	eArr[0].Data = d
//	eArr[0].EntityM = e

//	a = append(a, DataForAggregation{s, eArr})

//	return a
//}

func (e ExecutionNode) GetTitle() string {
	if len(e.Children) == 0 {
		if len(e.Arguments) > 0 {
			return e.Arguments[0].QueryComponentOriginalString
		}
		return ""
	}

	if len(e.Children) == 1 {
		//if e.Type == string(component.GetUniverse) ||
		//	e.Type == string(component.GetData) ||
		//	e.Type == string(component.Classification) {
		//	return e.Children[0].GetTitle() + " " + e.Arguments[0].QueryComponentOriginalString
		//} else {
		//	return e.Arguments[0].QueryComponentOriginalString + " " + e.Children[0].GetTitle()
		//}
		if len(e.Arguments) > 1 {
			return e.Children[0].GetTitle() + " → " + e.Arguments[0].QueryComponentOriginalString + " " + e.Arguments[1].QueryComponentOriginalString
		}
		return e.Children[0].GetTitle() + " → " + e.Arguments[0].QueryComponentOriginalString
	}

	if len(e.Children) == 2 {
		if e.Arguments[0].QueryComponentOriginalString == "Union" {
			return e.Children[0].GetTitle() + ", " + e.Children[1].GetTitle()
		} else {
			return e.Arguments[0].QueryComponentOriginalString + " of (" + e.Children[0].GetTitle() + ") against (" + e.Children[1].GetTitle() + ")"
		}
	}

	return ""
}

func (e *ExecutionNode) ParseTree(ctx context.Context, terms *term.TermData) {
	e.Arguments = parseArguments(ctx, e.Arguments, terms)

	for i, _ := range e.Children {
		(&e.Children[i]).ParseTree(ctx, terms)
	}
}

func parseArguments(ctx context.Context, c []component.QueryComponent, terms *term.TermData) []component.QueryComponent {
	for i, v := range c {
		//parsed := build.ExtractQueryComponents(v.QueryComponentOriginalString, terms)

		//if len(parsed) > 0 {
		//	c[i] = parsed[0]
		//}

		// marshalOutput("v", v)

		if c[i].QueryComponentType != component.GetBulkData && c[i].QueryComponentType != component.CustomQuandlCode && c[i].QueryComponentType != component.TimeSeriesFormula && c[i].QueryComponentType != component.RemoveData && c[i].QueryComponentType != component.FreeText && c[i].QueryComponentType != component.RenameEntity {
			// log.Infof(ctx, "c[i].QueryComponentType = %s\n", c[i].QueryComponentType)
			c[i] = build.ExtractQueryComponentExact(v.QueryComponentOriginalString, terms, nil)
		} else if c[i].QueryComponentType == component.CustomQuandlCode {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.CustomQuandlCode,
				v.QueryComponentOriginalString,
				component.QuandlOpenData,
				v.QueryComponentOriginalString,
				nil,
			}
		} else if c[i].QueryComponentType == component.TimeSeriesFormula {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.TimeSeriesFormula,
				v.QueryComponentOriginalString,
				component.QuandlOpenData,
				v.QueryComponentOriginalString,
				nil,
			}
		} else if c[i].QueryComponentType == component.RemoveData {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.RemoveData,
				v.QueryComponentOriginalString,
				component.QuandlOpenData,
				v.QueryComponentOriginalString,
				nil,
			}
		} else if c[i].QueryComponentType == component.FreeText {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.FreeText,
				v.QueryComponentOriginalString,
				component.QuandlOpenData,
				v.QueryComponentOriginalString,
				v.QueryComponentParams,
			}
		} else if c[i].QueryComponentType == component.RenameEntity {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.RenameEntity,
				v.QueryComponentOriginalString,
				component.QuandlOpenData,
				v.QueryComponentOriginalString,
				nil,
			}
		} else if c[i].QueryComponentType == component.GetBulkData {
			c[i] = component.QueryComponent{
				0,
				v.QueryComponentOriginalString,
				v.QueryComponentOriginalString,
				component.GetBulkData,
				v.QueryComponentOriginalString,
				"Alternative Data",
				v.QueryComponentOriginalString,
				nil,
			}
		}
	}

	return c
}

func (e ExecutionNode) Execute(ctx context.Context, id string, Title string) MultiEntityData {
	stepFn := findComputationStep(e.Type, e.Arguments)

	var data []MultiEntityData = make([]MultiEntityData, len(e.Children), len(e.Children))

	for i, v := range e.Children {
		data[i] = v.Execute(ctx, id, "")
	}

	if len(e.Arguments) > 0 {
		track.Update(ctx, id, e.Type+": "+e.Arguments[0].QueryComponentOriginalString, 0)
		log.Infof(ctx, "Execute: %s : %s", e.Type, e.Arguments[0].QueryComponentOriginalString)
	} else {
		track.Update(ctx, id, e.Type, 0)
		log.Infof(ctx, "Execute: %s", e.Type)
	}

	var med MultiEntityData
	if stepFn != nil {
		timer := time.Now()
		med = stepFn(e.Arguments)(ctx, data)
		log.Infof(ctx, "time taken was %v", time.Since(timer))
	} else if len(data) > 0 {
		med = data[0]
	}

	if Title == "" {
		med.Title = e.GetTitle()
	} else {
		med.Title = Title
	}

	return med
}

func createSeriesArray(dates []time.Time, data []float64) []DataPoint {
	var dp = make([]DataPoint, len(dates))

	for i, v := range dates {
		dp[i] = DataPoint{v, data[i]}
	}

	return dp
}

func (m MultiEntityData) Duplicate() MultiEntityData {
	marhsalled, err := json.Marshal(m)

	var newData MultiEntityData

	if err == nil {
		json.Unmarshal(marhsalled, &newData)
	}

	return newData
}

func (m MultiEntityData) NumEntities() int {
	return len(m.EntityData)
}

func (m MultiEntityData) NumSeries() int {
	var numSeries = 0

	for _, v := range m.EntityData {
		numSeries = numSeries + len(v.Data)
	}

	return numSeries
}

func (m MultiEntityData) GetFields() []string {
	fieldMap := make(map[string]bool)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if !v2.IsWeight {
				fieldMap[v2.Meta.Label] = true
			}
		}
	}

	fields := make([]string, 0)

	for k, _ := range fieldMap {
		fields = append(fields, k)
	}

	sort.Strings(fields)

	return fields
}

func (m MultiEntityData) RemoveSuperflousDataAndWeights(keepNonBlankEntityNames bool) MultiEntityData {
	var newData MultiEntityData

	newData.Title = m.Title
	newData.Error = m.Error
	newData.GraphicalPreference = m.GraphicalPreference
	newData.EntityData = make([]SingleEntityData, 0)

	// If the whole weight series is 1, get rid of the weight series

	// If the whole weight series is 0, get rid of the entity

	// js, _ := json.Marshal(m)
	// fmt.Printf("before=%s\n", js)

	for _, v := range m.EntityData {
		var keepEntity bool = false
		var entityHasWeight bool = false
		var entityHasData bool = false

		for j, v2 := range v.Data {
			entityHasData = true

			if v2.IsWeight {
				var keepWeightSeries bool = false

				entityHasWeight = true
				for _, v3 := range v2.Data {
					if v3.Data != 0 {
						keepEntity = true
					}
					if v3.Data != 1 {
						keepWeightSeries = true
					}
				}

				if !keepWeightSeries {
					v.Data = append(v.Data[:j], v.Data[j+1:]...)
					break
				}
			}
		}

		if keepNonBlankEntityNames {
			if v.Meta.Name != "" {
				keepEntity = true
			}
		}

		if !entityHasWeight && entityHasData {
			keepEntity = true
		}

		if keepEntity {
			newData.EntityData = append(newData.EntityData, v)
		}
	}

	// js, _ = json.Marshal(m)
	// fmt.Printf("after=%s\n", js)

	return newData
}
func (m MultiEntityData) GetSources() []string {
	sourceMap := make(map[string]bool)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			sourceMap[v2.Meta.Source] = true
		}
	}

	sources := make([]string, 0)

	for k, _ := range sourceMap {
		sources = append(sources, k)
	}

	sort.Strings(sources)

	return sources
}

func (m MultiEntityData) GetEntities() []string {
	entities := make([]string, m.NumEntities(), m.NumEntities())

	for i, v := range m.EntityData {
		entities[i] = v.Meta.Name
	}

	return entities
}

func (m MultiEntityData) GetCategories() []string {
	categoryMap := make(map[string]bool)

	for _, v := range m.EntityData {
		for _, v2 := range v.Category.Labels {
			categoryMap[v2.Label] = true
		}
	}

	categories := make([]string, 0)

	for k, _ := range categoryMap {
		categories = append(categories, k)
	}

	if len(categories) == 0 {
		categories = append(categories, "")
	}

	sort.Strings(categories)

	return categories
}

func (m MultiEntityData) GetUnits() []string {
	unitsMap := make(map[string]bool)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			unitsMap[v2.Meta.Units] = true
		}
	}

	units := make([]string, 0)

	for k, _ := range unitsMap {
		units = append(units, k)
	}

	if len(units) == 0 {
		units = append(units, "")
	}

	sort.Strings(units)

	return units
}

func (m MultiEntityData) GetUnitsFromFields(fields []string) []string {
	units := make([]string, len(fields))

	for i, _ := range fields {
		units[i] = findMeta(m, fields[i]).Units
	}

	return units
}

func findMeta(m MultiEntityData, field string) SeriesMeta {
	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if v2.Meta.Label == field {
				return v2.Meta
			}
		}
	}

	return SeriesMeta{}
}

func (m MultiEntityData) LastDay() time.Time {
	var lastDay time.Time

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if len(v2.Data) > 0 {
				if v2.Data[len(v2.Data)-1].Time.After(lastDay) {
					lastDay = v2.Data[len(v2.Data)-1].Time
				}
			}
		}
	}

	return lastDay
}

func (m MultiEntityData) Insert(eMeta EntityMeta, sMeta SeriesMeta, d DataPoint, c CategoryLabel, isWeight bool) MultiEntityData {
	if m.EntityData == nil {
		m.EntityData = make([]SingleEntityData, 0)
	}

	if isWeight {
		sMeta.Label = "Weights"
		sMeta.Units = "Weight"

		//fmt.Printf("Weight Data Point = %q\n", d)
	}

	newSeries := Series{
		Meta:     sMeta,
		Data:     []DataPoint{d},
		IsWeight: isWeight,
	}

	for i, v := range m.EntityData {

		if v.Meta.UniqueId == eMeta.UniqueId {
			if !isWeight {
				m.EntityData[i].Category = m.EntityData[i].Category.Insert(d.Time, c)
			}

			if v.Data == nil {
				v.Data = make([]Series, 0)
			}

			for j, v2 := range v.Data {
				if v2.Meta.Label == sMeta.Label || (isWeight && v2.IsWeight) {
					if v2.Data == nil {
						m.EntityData[i].Data[j].Data = make([]DataPoint, 0)
					}
					m.EntityData[i].Data[j].Data = append(m.EntityData[i].Data[j].Data, d)

					return m
				}
			}
			// Series not found. Inserting new series

			m.EntityData[i].Data = append(m.EntityData[i].Data, newSeries)

			return m
		}
	}

	// Entity not found. Inserting a new entity
	newEntity := SingleEntityData{
		Data: []Series{newSeries},
		Meta: eMeta,
	}

	newEntity.Category = newEntity.Category.Insert(d.Time, c)

	m.EntityData = append(m.EntityData, newEntity)

	return m
}

func (s Series) GetDates() []time.Time {
	tArr := make([]time.Time, len(s.Data))

	for i, v := range s.Data {
		tArr[i] = v.Time
	}

	return tArr
}

func unionDates(a []time.Time, b []time.Time) []time.Time {
	var uDates = make([]time.Time, 0, len(a)+len(b))

	var i, j int

	for i < len(a) && j < len(b) {
		if a[i].Equal(b[j]) {
			uDates = append(uDates, a[i])
			i++
			j++
		} else if a[i].Before(b[j]) {
			uDates = append(uDates, a[i])
			i++
		} else {
			uDates = append(uDates, b[j])
			j++
		}
	}

	for ; i < len(a); i++ {
		uDates = append(uDates, a[i])
	}

	for ; j < len(b); j++ {
		uDates = append(uDates, b[j])
	}

	return uDates
}

func zeroDay() time.Time {
	return time.Date(1776, 0, 0, 0, 0, 0, 0, time.UTC)
}

func (m MultiEntityData) NumDates() int {
	return len(m.UniqueDates())
}

func (m MultiEntityData) UniqueDates() []time.Time {
	uniqueDates := make([]time.Time, 0)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			uniqueDates = unionDates(uniqueDates, v2.GetDates())
		}
	}

	return uniqueDates
}

func (m MultiEntityData) Summarize() string {
	return fmt.Sprintf("%v Entities, %v Series, %v Dates", m.NumEntities(), m.NumSeries(), m.NumDates())
}

// Structs to hold dynamically-called functions
type ComputationStep struct {
	Type          component.MajorType
	Name          string
	DefaultString string
	ArgCheckFn    func(MultiEntityData, []component.QueryComponent) ([]component.QueryComponent, error)
	ComputeFn     func([]component.QueryComponent) StepFnType
}

var ComputationsSteps []ComputationStep = []ComputationStep{
	ComputationStep{
		Type:          component.GetUniverse,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyUniverse,
		ComputeFn:     WrapUniverseArgument(getUniverse),
	},
	// ComputationStep{
	// 	Type:          component.GetPortfolio,
	// 	Name:          "",
	// 	DefaultString: "",
	// 	ArgCheckFn:    verifyPortfolio,
	// 	ComputeFn:     WrapPortfolioArgument(getUniverse, getUniverseWeights),
	// },
	ComputationStep{
		Type:          component.CustomQuandlCode,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyQuandlCode,
		ComputeFn:     WrapUniverseArgument(getUniverse),
	},
	ComputationStep{
		Type:          component.TimeSeriesFormula,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyFormula,
		ComputeFn:     WrapStringArgumentTS2(formulaToFunction),
	},
	ComputationStep{
		Type:          component.RemoveData,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyRemoveData,
		ComputeFn:     WrapStringArgumentTS(removeNamedField),
	},
	ComputationStep{
		Type:          component.KeepData,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyRemoveData,
		ComputeFn:     WrapStringArgumentTS(keepNamedField),
	},
	ComputationStep{
		Type:          component.RenameEntity,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyRenameEntity,
		ComputeFn:     WrapStringArgumentTS(renameEntity),
	},
	ComputationStep{
		Type:          component.TimeSlice,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyTimeRange,
		ComputeFn:     TimeSlicer,
	},
	ComputationStep{
		Type:          component.GetData,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyDataField,
		ComputeFn:     WrapUniverseData(GetUniverseData, getData),
	},
	ComputationStep{
		Type:          component.GetBulkData,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyDataField,
		ComputeFn:     GetBulkData,
	},
	ComputationStep{
		Type:          component.SetWeights,
		Name:          "",
		DefaultString: "",
		ArgCheckFn:    verifyDataField,
		ComputeFn:     WrapUniverseData(SetWeights, getData),
	},
	// ComputationStep{
	// 	Type:          component.MacroEvent,
	// 	Name:          "",
	// 	DefaultString: "",
	// 	ArgCheckFn:    verifyMacroEvent,
	// 	ComputeFn:     WrapEvent(SetWeightsAndCategory, getMacroEvent),
	// },
	// ComputationStep{
	// 	Type:          component.StockEvent,
	// 	Name:          "",
	// 	DefaultString: "",
	// 	ArgCheckFn:    verifyStockEvent,
	// 	ComputeFn:     WrapEvent(SetWeightsAndCategory, getStockEvent),
	// },
	ComputationStep{
		Type:          component.GraphicalPreference,
		Name:          "Scatter",
		DefaultString: "Scatter",
		ArgCheckFn:    verifyNoArguments("Scatter", "Scatter"),
		ComputeFn:     WrapNoArguments(GraphicalPreference("Scatter")),
	},
	ComputationStep{
		Type:          component.GraphicalPreference,
		Name:          "Histogram",
		DefaultString: "Histogram",
		ArgCheckFn:    verifyNoArguments("Histogram", "Histogram"),
		ComputeFn:     WrapNoArguments(GraphicalPreference("Histogram")),
	},
	ComputationStep{
		Type:          component.GraphicalPreference,
		Name:          "Column",
		DefaultString: "Column",
		ArgCheckFn:    verifyNoArguments("Column", "Column"),
		ComputeFn:     WrapNoArguments(GraphicalPreference("Column")),
	},
	ComputationStep{
		Type:          component.GraphicalPreference,
		Name:          "Heatmap",
		DefaultString: "Heatmap",
		ArgCheckFn:    verifyNoArguments("Heatmap", "Heatmap"),
		ComputeFn:     WrapNoArguments(GraphicalPreference("Heatmap")),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Lag {Number}",
		DefaultString: "Lag 1",
		ArgCheckFn:    verifyNoArguments("Lag {Number}", "Lag 1"),
		ComputeFn:     WrapNumericalArgumentTS(tsLag),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "> {Number}",
		DefaultString: "> 0",
		ArgCheckFn:    verifyNoArguments("> {Number}", "> 0"),
		ComputeFn:     WrapNumericalArgument(greaterXaggregator),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "< {Number}",
		DefaultString: "< 0",
		ArgCheckFn:    verifyNoArguments("< {Number}", "< 0"),
		ComputeFn:     WrapNumericalArgument(lessXaggregator),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "= {Number}",
		DefaultString: "= 0",
		ArgCheckFn:    verifyNoArguments("= {Number}", "= 0"),
		ComputeFn:     WrapNumericalArgument(equalXaggregator),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Annualized {Number}-Day Standard Deviation",
		DefaultString: "Annualized 30-Day Standard Deviation",
		ArgCheckFn:    verifyNoArguments("Annualized {Number}-Day Standard Deviation", "Annualized 30-Day Standard Deviation"),
		ComputeFn:     WrapNumericalArgumentTS(tsStdDev),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Remove Data, Keep Universe Weights",
		DefaultString: "Remove Data, Keep Universe Weights",
		ArgCheckFn:    verifyNoArguments("Remove Data, Keep Universe Weights", "Remove Data, Keep Universe Weights"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsRemoveData)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Remove Weights, Keep Data",
		DefaultString: "Remove Weights, Keep Data",
		ArgCheckFn:    verifyNoArguments("Remove Weights, Keep Data", "Remove Weights, Keep Data"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsRemoveWeights)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Remove Filtered Data",
		DefaultString: "Remove Filtered Data",
		ArgCheckFn:    verifyNoArguments("Remove Filtered Data", "Remove Filtered Data"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsRemoveFilteredData)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Align Data To Zero",
		DefaultString: "Align Data To Zero",
		ArgCheckFn:    verifyNoArguments("Align Data To Zero", "Align Data To Zero"),
		ComputeFn:     WrapNoArgumentTSMulti(AlignEvent),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Align {Number} Days Before, {Number} Days After",
		DefaultString: "Align 15 Days Before, 30 Days After",
		ArgCheckFn:    verifyNoArguments("Align {Number} Days Before, {Number} Days After", "Align 15 Days Before, 30 Days After"),
		ComputeFn:     WrapNumericalArgumentTS2Multi(AlignEventBeforeAfter(false)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Market Align {Number} Days Before, {Number} Days After",
		DefaultString: "Market Align 15 Days Before, 30 Days After",
		ArgCheckFn:    verifyNoArguments("Market Align {Number} Days Before, {Number} Days After", "Market Align 15 Days Before, 30 Days After"),
		ComputeFn:     WrapNumericalArgumentTS2Multi(AlignEventBeforeAfter(true)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Hacky Align Quarter for SSS",
		DefaultString: "Hacky Align Quarter for SSS",
		ArgCheckFn:    verifyNoArguments("Hacky Align Quarter for SSS", "Hacky Align Quarter for SSS"),
		ComputeFn:     WrapNoArguments(AlignEventBeforeAfterPreserveDates),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Union",
		DefaultString: "Union",
		ArgCheckFn:    verifyNoArguments("Union", "Union"),
		ComputeFn:     WrapNoArguments(UnionData),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Difference",
		DefaultString: "Difference",
		ArgCheckFn:    verifyNoArguments("Difference", "Difference"),
		ComputeFn:     WrapNoArguments(DifferenceData),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Intersection",
		DefaultString: "Intersection",
		ArgCheckFn:    verifyNoArguments("Intersection", "Intersection"),
		ComputeFn:     WrapNoArguments(IntersectData),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Exclusion",
		DefaultString: "Exclusion",
		ArgCheckFn:    verifyNoArguments("Exclusion", "Exclusion"),
		ComputeFn:     WrapNoArguments(ExcludeData),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Regression",
		DefaultString: "Regression",
		ArgCheckFn:    verifyNoArguments("Regression", "Regression"),
		ComputeFn:     WrapNoArguments(Regression),
	},
	ComputationStep{
		Type:          component.CombineData,
		Name:          "Alpha using {Number}-Month Regression",
		DefaultString: "Alpha using 12-Month Regression",
		ArgCheckFn:    verifyNoArguments("Alpha using {Number}-Month Regression", "Alpha using 12-Month Regression"),
		ComputeFn:     WrapNumericalArgumentCombine(Alpha),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Percentage Change",
		DefaultString: "Percentage Change",
		ArgCheckFn:    verifyNoArguments("Percentage Change", "Percentage Change"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsTotalReturn())),
	},
	// ComputationStep{
	// 	Type:          component.TimeSeriesTransformation,
	// 	Name:          "Difference",
	// 	DefaultString: "Difference",
	// 	ArgCheckFn:    verifyNoArguments("Difference", "Difference"),
	// 	ComputeFn:     WrapNoArguments(ComputeTS(tsDifference())),
	// },
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Cumulative Change",
		DefaultString: "Cumulative Change",
		ArgCheckFn:    verifyNoArguments("Cumulative Change", "Cumulative Change"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsCumulativeChange())),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "{Number}-Day Forward Return",
		DefaultString: "30-Day Forward Return",
		ArgCheckFn:    verifyNoArguments("{Number}-Day Forward Return", "30-Day Forward Return"),
		ComputeFn:     WrapNumericalArgumentTS(tsForwardReturn),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.Daily,
		DefaultString: component.Daily,
		ArgCheckFn:    verifyNoArguments(component.Daily, component.Daily),
		ComputeFn:     WrapNoArguments(AlignCalendar(tsDaily)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Sample Every {Number} Periods",
		DefaultString: "Sample Every 7 Periods",
		ArgCheckFn:    verifyNoArguments("Sample Every {Number} Periods", "Sample Every 7 Periods"),
		ComputeFn:     WrapNumericalArgumentTS(tsSampleEveryNumPeriods),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.Weekly,
		DefaultString: component.Weekly,
		ArgCheckFn:    verifyNoArguments(component.Weekly, component.Weekly),
		ComputeFn:     WrapNoArguments(AlignCalendar(tsByWeek)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.Monthly,
		DefaultString: component.Monthly,
		ArgCheckFn:    verifyNoArguments(component.Monthly, component.Monthly),
		ComputeFn:     WrapNoArguments(AlignCalendar(tsByMonth)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.Quarterly,
		DefaultString: component.Quarterly,
		ArgCheckFn:    verifyNoArguments(component.Quarterly, component.Quarterly),
		ComputeFn:     WrapNoArguments(AlignCalendar(tsByQuarter)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.Yearly,
		DefaultString: component.Yearly,
		ArgCheckFn:    verifyNoArguments(component.Yearly, component.Yearly),
		ComputeFn:     WrapNoArguments(AlignCalendar(tsByYear)),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          component.AllTime,
		DefaultString: component.AllTime,
		ArgCheckFn:    verifyNoArguments(component.AllTime, component.AllTime),
		ComputeFn:     WrapNoArguments(ComputeTS(tsAllTime())),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Latest Data",
		DefaultString: "Latest Data",
		ArgCheckFn:    verifyNoArguments("Latest Data", "Latest Data"),
		ComputeFn:     WrapNoArguments(WrapLatestData()),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Align Last Day",
		DefaultString: "Align Last Day",
		ArgCheckFn:    verifyNoArguments("Align Last Day", "Align Last Day"),
		ComputeFn:     WrapNoArguments(AlignLastDay()),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "All Data Available",
		DefaultString: "All Data Available",
		ArgCheckFn:    verifyNoArguments("All Data Available", "All Data Available"),
		ComputeFn:     WrapNoArguments(AlignFirstDay()),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "CAGR",
		DefaultString: "CAGR",
		ArgCheckFn:    verifyNoArguments("CAGR", "CAGR"),
		ComputeFn:     WrapNoArguments(ComputeTS(tsCagr)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Use Latest Weights Historically",
		DefaultString: "Use Latest Weights Historically",
		ArgCheckFn:    verifyNoArguments("Use Latest Weights Historically", "Use Latest Weights Historically"),
		ComputeFn:     WrapNoArguments(ComputeTS(latestWeightsHistorical)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Rebalance Weekly",
		DefaultString: "Rebalance Weekly",
		ArgCheckFn:    verifyNoArguments("Rebalance Weekly", "Rebalance Weekly"),
		ComputeFn:     WrapNoArguments(ComputeTS(rebalanceWeekly)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Rebalance Monthly",
		DefaultString: "Rebalance Monthly",
		ArgCheckFn:    verifyNoArguments("Rebalance Monthly", "Rebalance Monthly"),
		ComputeFn:     WrapNoArguments(ComputeTS(rebalanceMonthly)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Rebalance Quarterly",
		DefaultString: "Rebalance Quarterly",
		ArgCheckFn:    verifyNoArguments("Rebalance Quarterly", "Rebalance Quarterly"),
		ComputeFn:     WrapNoArguments(ComputeTS(rebalanceQuarterly)),
	},
	ComputationStep{
		Type:          component.TransformWeights,
		Name:          "Rebalance Yearly",
		DefaultString: "Rebalance Yearly",
		ArgCheckFn:    verifyNoArguments("Rebalance Yearly", "Rebalance Yearly"),
		ComputeFn:     WrapNoArguments(ComputeTS(rebalanceYearly)),
	},
	//ComputationStep{
	//	Type:          component.SameEntityAggregation,
	//	Name:          "Multiply Field By {Number}",
	//	DefaultString: "Multiply Field By 5",
	//	ArgCheckFn:    requireNumberOfFields(1),
	//	ComputeFn:     WrapFieldAndNumber(tsMultiplyFieldByNumber),
	//},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Number of Days Since < {Number}",
		DefaultString: "Number of Days Since < 5",
		ArgCheckFn:    verifyNoArguments("Number of Days Since < {Number}", "Number of Days Since < 5"),
		ComputeFn:     WrapNumericalArgumentTS(tsDaysSinceDrop),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Spread",
		DefaultString: "Spread",
		ArgCheckFn:    verifyNoArguments("Spread", "Spread"),
		ComputeFn:     WrapNoArguments(ComputeTS(spread)),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Ratio",
		DefaultString: "Ratio",
		ArgCheckFn:    verifyNoArguments("Ratio", "Ratio"),
		ComputeFn:     WrapNoArguments(ComputeTS(ratio)),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Ratio",
		DefaultString: "Ratio",
		ArgCheckFn:    verifyNoArguments("Ratio", "Ratio"),
		ComputeFn:     WrapNoArguments(ComputeTS(ratio)),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Divide Field {Number} By Field {Number}",
		DefaultString: "Divide Field 1 By Field 2",
		ArgCheckFn:    verifyNoArguments("Divide Field {Number} By Field {Number}", "Divide Field 1 By Field 2"),
		ComputeFn:     WrapNumericalArgumentTS2(tsDividedBy),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Multiply Field {Number} By Field {Number}",
		DefaultString: "Multiply Field 1 By Field 2",
		ArgCheckFn:    verifyNoArguments("Multiply Field {Number} By Field {Number}", "Multiply Field 1 By Field 2"),
		ComputeFn:     WrapNumericalArgumentTS2(tsMultiply),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Add Field {Number} And Field {Number}",
		DefaultString: "Add Field 1 And Field 2",
		ArgCheckFn:    verifyNoArguments("Add Field {Number} And Field {Number}", "Add Field 1 And Field 2"),
		ComputeFn:     WrapNumericalArgumentTS2(tsAdd),
	},
	ComputationStep{
		Type:          component.SameEntityAggregation,
		Name:          "Subtract Field {Number} From Field {Number}",
		DefaultString: "Subtract Field 1 From Field 2",
		ArgCheckFn:    verifyNoArguments("Subtract Field {Number} From Field {Number}", "Subtract Field 1 From Field 2"),
		ComputeFn:     WrapNumericalArgumentTS2(tsSubtract),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Sum",
		DefaultString: "Sum",
		ArgCheckFn:    verifyNoArguments("Sum", "Sum"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(sumAggregator)),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Average",
		DefaultString: "Average",
		ArgCheckFn:    verifyNoArguments("Average", "Average"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(averageAggregator)),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Flatten",
		DefaultString: "Flatten",
		ArgCheckFn:    verifyNoArguments("Flatten", "Flatten"),
		ComputeFn:     WrapNoArguments(flatten),
	},
	ComputationStep{
		Type:          component.TimeSeriesTransformation,
		Name:          "Company Event to Macro Event",
		DefaultString: "Company Event to Macro Event",
		ArgCheckFn:    verifyNoArguments("Company Event to Macro Event", "Company Event to Macro Event"),
		ComputeFn:     WrapNoArguments(WeightsToMacro),
	},
	//ComputationStep{
	//	Type:          component.SameEntityAggregation,
	//	Name:          "Sum over Fields",
	//	DefaultString: "Sum over Fields",
	//	ArgCheckFn:    verifyNoArguments("Sum over Fields", "Sum over Fields"),
	//	ComputeFn:     WrapNoArguments(ComputeTS(addFields)),
	//},
	//ComputationStep{
	//	Type:          component.SameEntityAggregation,
	//	Name:          "Average over Fields",
	//	DefaultString: "Average over Fields",
	//	ArgCheckFn:    verifyNoArguments("Average over Fields", "Average over Fields"),
	//	ComputeFn:     WrapNoArguments(ComputeTS(averageFields)),
	//},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Count",
		DefaultString: "Count",
		ArgCheckFn:    verifyNoArguments("Count", "Count"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(countAggregator)),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Percent",
		DefaultString: "Percent",
		ArgCheckFn:    verifyNoArguments("Percent", "Percent"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(percentAggregator)),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Median",
		DefaultString: "Median",
		ArgCheckFn:    verifyNoArguments("Median", "Median"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(medianAggregator)),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Summary Stats",
		DefaultString: "Summary Stats",
		ArgCheckFn:    verifyNoArguments("Summary Stats", "Summary Stats"),
		ComputeFn:     WrapNoArguments(SummaryStats),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Top {Number}",
		DefaultString: "Top 5",
		ArgCheckFn:    verifyNoArguments("Top {Number}", "Top 5"),
		ComputeFn:     WrapNumericalArgument(topXaggregator),
	},
	ComputationStep{
		Type:          component.CrossEntityAggregation,
		Name:          "Bottom {Number}",
		DefaultString: "Bottom 5",
		ArgCheckFn:    verifyNoArguments("Bottom {Number}", "Bottom 5"),
		ComputeFn:     WrapNumericalArgument(bottomXaggregator),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "No Classification",
		DefaultString: "No Classification",
		ArgCheckFn:    verifyNoArguments("No Classification", "No Classification"),
		ComputeFn:     WrapNoArguments(ComputeTS(noClassification)),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "By Company",
		DefaultString: "By Company",
		ArgCheckFn:    verifyNoArguments("By Company", "By Company"),
		ComputeFn:     WrapNoArguments(ComputeTS(categorizeByCompany)),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "By Sector",
		DefaultString: "By Sector",
		ArgCheckFn:    verifyNoArguments("By Sector", "By Sector"),
		ComputeFn:     WrapNoArguments(ComputeTS(categorizeBySector(list.Sector))),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "By Industry Group",
		DefaultString: "By Industry Group",
		ArgCheckFn:    verifyNoArguments("By Industry Group", "By Industry Group"),
		ComputeFn:     WrapNoArguments(ComputeTS(categorizeBySector(list.IndustryGroup))),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "By Industry",
		DefaultString: "By Industry",
		ArgCheckFn:    verifyNoArguments("By Industry", "By Industry"),
		ComputeFn:     WrapNoArguments(ComputeTS(categorizeBySector(list.Industry))),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "By Sub Industry",
		DefaultString: "By Sub Industry",
		ArgCheckFn:    verifyNoArguments("By Sub Industry", "By Sub Industry"),
		ComputeFn:     WrapNoArguments(ComputeTS(categorizeBySector(list.SubIndustry))),
	},
	//ComputationStep{
	//	Type:          component.CrossEntityAggregation,
	//	Name:          "Histogram",
	//	DefaultString: "Histogram",
	//	ArgCheckFn:    verifyNoArguments("Histogram", "Histogram"),
	//	ComputeFn:     WrapNoArguments(CrossEntityAggregation(histogramAggregator)),
	//},
	ComputationStep{
		Type:          component.Classification,
		Name:          "Quartile",
		DefaultString: "Quartile",
		ArgCheckFn:    verifyNoArguments("Quartile", "Quartile"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(quantileAggregator(4))),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "Quintile",
		DefaultString: "Quintile",
		ArgCheckFn:    verifyNoArguments("Quintile", "Quintile"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(quantileAggregator(5))),
	},
	ComputationStep{
		Type:          component.Classification,
		Name:          "Decile",
		DefaultString: "Decile",
		ArgCheckFn:    verifyNoArguments("Decile", "Decile"),
		ComputeFn:     WrapNoArguments(CrossEntityAggregation(quantileAggregator(10))),
	},
	ComputationStep{
		Type:          component.GraphicalPreference,
		Name:          "Boxplot",
		DefaultString: "Boxplot",
		ArgCheckFn:    verifyNoArguments("Boxplot", "Boxplot"),
		ComputeFn:     WrapNoArguments(ComposeStepFn(CrossEntityAggregation(boxplotAggregator), GraphicalPreference("Boxplot"))),
	},
}

func InsertComputationTerms(terms *term.TermData) {
	for _, v := range ComputationsSteps {
		if v.Name != "" {
			terms.Insert(v.Name, component.QueryComponent{0, v.Name, v.Name, string(v.Type), v.Name, "", v.DefaultString, nil})
		}
	}
}

func getPriceNode(universe string) ExecutionNode {
	var c ExecutionNode = ExecutionNode{
		Type: component.GetData,
		Arguments: []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType:           component.ConceptSecurity,
				QueryComponentOriginalString: "Price",
			},
			component.QueryComponent{
				QueryComponentType:           component.TimeRange,
				QueryComponentOriginalString: "Last Data Point",
			},
		},
		Children: []ExecutionNode{
			ExecutionNode{
				Type: component.CustomQuandlCode,
				Arguments: []component.QueryComponent{
					component.QueryComponent{
						QueryComponentType:           component.UniverseExpandable,
						QueryComponentOriginalString: universe,
					},
				},
			},
		},
	}

	return c
}

func ZapierHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, query string, terms *term.TermData) {
	c1 := getPriceNode("S&P 500 Stocks")
	c2 := getPriceNode("Energy Stocks")
	c3 := getPriceNode("Financials Stocks")
	c4 := getPriceNode("Consumer Discretionary Stocks")
	c5 := getPriceNode("Health Care Stocks")
	c6 := getPriceNode("Industrials Stocks")
	c7 := getPriceNode("Telecommunication Services Stocks")
	c8 := getPriceNode("Consumer Staples Stocks")
	c9 := getPriceNode("Materials Stocks")
	c10 := getPriceNode("Industrials Stocks")
	c11 := getPriceNode("Utilities Stocks")
	c12 := getPriceNode("Information Technology Stocks")
	c13 := getPriceNode("Sector ETFs")
	c14 := getPriceNode("iShares Popular ETFs")
	c15 := getPriceNode("Commodities")
	c16 := getPriceNode("Currencies")

	id := runTree(ctx, c1, "", terms)
	runTree(ctx, c2, "", terms)
	runTree(ctx, c3, "", terms)
	runTree(ctx, c4, "", terms)
	runTree(ctx, c5, "", terms)
	runTree(ctx, c6, "", terms)
	runTree(ctx, c7, "", terms)
	runTree(ctx, c8, "", terms)
	runTree(ctx, c9, "", terms)
	runTree(ctx, c10, "", terms)
	runTree(ctx, c11, "", terms)
	runTree(ctx, c12, "", terms)
	runTree(ctx, c13, "", terms)
	runTree(ctx, c14, "", terms)
	runTree(ctx, c15, "", terms)
	runTree(ctx, c16, "", terms)

	http.Redirect(w, r, "/tree/"+id, http.StatusFound)
}

func RunHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, title string, terms *term.TermData) {
	decoder := json.NewDecoder(r.Body)
	var c ExecutionNode
	_ = decoder.Decode(&c)

	// m, _ := json.Marshal(c)
	// log.Infof(ctx, "body = %s", m)

	RunHandlerNoDecoder(ctx, w, r, title, terms, c)
}

func RunHandlerNoDecoder(ctx context.Context, w http.ResponseWriter, r *http.Request, title string, terms *term.TermData, c ExecutionNode) {
	id := runTree(ctx, c, title, terms)

	t := taskqueue.NewPOSTTask("/apiv1/worker", map[string][]string{"id": {id}})
	if t.RetryOptions == nil {
		t.RetryOptions = &taskqueue.RetryOptions{}
	}
	t.RetryOptions.RetryLimit = 2
	if _, err := taskqueue.Add(ctx, t, ""); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "{\"id\": \"%s\"}\n", id)
}

func ReRunHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, query string) {

	type Dummy struct {
		Tree ExecutionNode `bson:"tree"`
	}

	var m Dummy

	db.GetFromKey(ctx, query, &m)

	go func(id string) {
		// defer func() {
		// 	if r := recover(); r != nil {
		// 		track.Update(id, "Error", 0)
		// 		zlog.LogRecovery(r)
		// 	}
		// }()
		track.Update(ctx, id, "Running", 0)
		med := m.Tree.Execute(ctx, id, "")
		db.DatabaseInsert(ctx, db.RunData, &med, "")
		track.Update(ctx, id, "Completed", 1)
		//push.PushMessage("Completed: " + med.Title)
		//m, _ := json.Marshal(ConvertMultiEntityDataToHighcharts(med))
	}(query)

	http.Redirect(w, r, "/render/"+query, http.StatusTemporaryRedirect)
}

// var laterFunc = delay.Func("key")

func WorkerHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := r.FormValue("id")

	var m TreeDummy

	db.GetFromKey(ctx, id, &m)

	var t ExecutionNode

	json.Unmarshal(m.Tree, &t)

	log.Infof(ctx, "Running id = %s", id)
	track.Update(ctx, id, "Running", 0)
	med := t.Execute(ctx, id, "")

	var m2 DataDummy
	m2.RunId = id
	m2.Data, _ = json.Marshal(med)

	db.DatabaseInsert(ctx, db.RunData, &m2, "")
	track.Update(ctx, id, "Completed", 1)
	log.Infof(ctx, "Completed id = %s", id)
}

type TreeDummy struct {
	Test   string
	Field2 string
	Tree   []byte
}

type DataDummy struct {
	RunId string
	Data  []byte
}

func runTree(ctx context.Context, c ExecutionNode, title string, terms *term.TermData) string {
	c.ParseTree(ctx, terms)

	var m TreeDummy
	temp, _ := json.Marshal(c)
	m.Tree = temp
	m.Test = "Zain"
	m.Field2 = "Hello"

	id := db.DatabaseInsert(ctx, db.RunTree, &m, "")

	return id
}

func TreeHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get(":query")

	log.Infof(ctx, "query = %s", query)

	var m TreeDummy

	err := db.GetFromKey(ctx, query, &m)

	if err != nil {
		log.Infof(ctx, "tree retrieval error = %s", err)
	}

	fmt.Fprintf(w, "%s\n", m.Tree)
}

func DataHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get(":query")

	details := track.GetDetails(ctx, query)

	if query != "" && (details.Message == "" || details.PercentComplete >= 0.999) {
		var m DataDummy

		err := db.GetFromField(ctx, db.RunData, "RunId", query, &m)

		if err == nil && len(m.Data) > 0 {
			// chartOptions := GetChartOptions(ctx, r.FormValue("options"))
			chartOptions := ParseChartOptions(ctx, r.FormValue("options"))

			var med MultiEntityData

			json.Unmarshal(m.Data, &med)

			output, _ := json.Marshal(ConvertMultiEntityDataToHighcharts(med, chartOptions))

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, "%s\n", output)
		} else {
			returnJson(ctx, w, track.TrackData{
				Message:         "Finishing Up",
				PercentComplete: 0.9,
			})
		}
	} else {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s\n", track.GetData(ctx, query))
	}
}

func returnJson(ctx context.Context, w http.ResponseWriter, data interface{}) {
	m, err := json.Marshal(data)

	if logError(ctx, err) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s", m)
	}
}

func logError(ctx context.Context, err error) bool {
	if err == nil {
		return true
	}

	log.Warningf(ctx, "Error = %s", err)
	return false
}

func RawHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get(":query")

	details := track.GetDetails(ctx, query)

	if query != "" && (details.Message == "" || details.PercentComplete >= 0.999) {
		// type Dummy struct {
		// 	Data MultiEntityData `bson:"data"`
		// }

		var m DataDummy

		// db.GetFromKey(ctx, query, &m)
		db.GetFromField(ctx, db.RunData, "RunId", query, &m)

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s\n", m.Data)
	} else {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "%s\n", track.GetData(ctx, query))
	}
}

// func PreviewHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, query string) {
// 	details := track.GetDetails(ctx, query)
//
// 	if query != "" && (details.Message == "" || details.PercentComplete >= 0.999) {
// 		type Dummy struct {
// 			Data MultiEntityData `bson:"data"`
// 		}
//
// 		var m Dummy
//
// 		db.GetFromKey(ctx, query, &m)
//
// 		sheet := convertMultiEntityDataToSheet(m.Data, true)
//
// 		output, _ := json.Marshal(sheet)
//
// 		fmt.Fprintf(w, "%s\n", output)
// 	} else {
// 		fmt.Fprintf(w, "%s\n", track.GetData(ctx, query))
// 	}
// }

// func TableHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, query string) {
// 	details := track.GetDetails(ctx, query)
//
// 	if query != "" && (details.Message == "" || details.PercentComplete >= 0.999) {
// 		type Dummy struct {
// 			Data MultiEntityData `bson:"data"`
// 		}
//
// 		var m Dummy
//
// 		db.GetFromKey(ctx, query, &m)
//
// 		sheet := convertMultiEntityDataToSheet(m.Data, false)
//
// 		output, _ := json.Marshal(sheet)
//
// 		fmt.Fprintf(w, "%s\n", output)
// 	} else {
// 		fmt.Fprintf(w, "%s\n", track.GetData(ctx, query))
// 	}
// }

func ExcelHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	hex := r.URL.Query().Get(":query")

	details := track.GetDetails(ctx, hex)

	if hex != "" && (details.Message == "" || details.PercentComplete >= 0.999) {
		var m DataDummy

		err := db.GetFromField(ctx, db.RunData, "RunId", hex, &m)

		if err == nil && len(m.Data) > 0 {

			var med MultiEntityData

			json.Unmarshal(m.Data, &med)

			sheet := convertMultiEntityDataToSheet(med, false)

			w.Header().Set("Content-Disposition", "attachment; filename="+hex+".xlsx")
			GenerateExcelFile(ctx, hex, sheet, w)
			// http.ServeFile(w, r, "/root/dploy/i/"+hex+".xlsx")
		}
	} else {
		fmt.Fprintf(w, "%s\n", track.GetData(ctx, hex))
	}
}

// func ArgcheckHandler(w http.ResponseWriter, r *http.Request, query string, terms *term.TermData) {
// 	decoder := json.NewDecoder(r.Body)
// 	var c []component.QueryComponent
// 	_ = decoder.Decode(&c)
//
// 	c = parseArguments(c, terms)
//
// 	argcheck, err := findArgcheck(query, c)
//
// 	w.Header().Set("Content-Type", "application/json")
//
// 	if argcheck == nil {
// 		fmt.Fprintf(w, "{\"components\":null, \"error\":\"%s\"}", err)
// 	} else {
// 		components, err := argcheck(MultiEntityData{}, c)
// 		components = parseArguments(components, terms)
// 		m, _ := json.Marshal(components)
// 		fmt.Fprintf(w, "{\"components\":%s, \"error\":\"%s\"}", m, err)
// 	}
// }

func findArgcheck(majorType string, c []component.QueryComponent) (func(MultiEntityData, []component.QueryComponent) ([]component.QueryComponent, error), error) {
	for _, v := range ComputationsSteps {
		if majorType == string(v.Type) && len(c) > 0 && c[0].QueryComponentCanonicalName == string(v.Name) {
			return v.ArgCheckFn, nil
		}
	}

	for _, v := range ComputationsSteps {
		if majorType == string(v.Type) {
			return v.ArgCheckFn, errors.New("No matching computation step found")
		}
	}

	return nil, errors.New("No matching computation step or matching major type found")
}

func findComputationStep(majorType string, c []component.QueryComponent) func([]component.QueryComponent) StepFnType {
	for _, v := range ComputationsSteps {
		if majorType == string(v.Type) && len(c) > 0 && c[0].QueryComponentCanonicalName == string(v.Name) {
			return v.ComputeFn
		}
	}

	for _, v := range ComputationsSteps {
		if majorType == string(v.Type) {
			return v.ComputeFn
		}
	}

	return nil
}

func verifyNoArguments(computationName string, defaultString string) func(MultiEntityData, []component.QueryComponent) ([]component.QueryComponent, error) {
	return func(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
		if len(c) == 0 || c[0].QueryComponentCanonicalName != computationName {
			return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: defaultString}}, nil
		}

		if len(c) > 1 {
			return []component.QueryComponent{c[0]}, nil
		}

		return c, nil
	}

}

func verifyTimeRange(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	return verifyDateRange(c)
}

func verifyDateRange(c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 1 {
		for _, v := range c {
			if v.QueryComponentType == component.TimeRange {
				return c, nil
			}
		}
	}

	c = append(c, component.QueryComponent{QueryComponentOriginalString: "Last Twelve Months"})

	return c, errors.New("No date range specified")
}

func verifyUniverse(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 0 {
		//return verifyDateRange(c)
	}

	if len(c) > 0 && c[0].QueryComponentType == component.Universe || c[0].QueryComponentType == component.UniverseExpandable {
		return []component.QueryComponent{c[0]}, nil
	}

	return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: "Apple Inc."}}, errors.New("No universe specified")
}

func verifyPortfolio(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 0 {
		//return verifyDateRange(c)
	}

	if len(c) > 0 && c[0].QueryComponentType == component.UniverseExpandable {
		return []component.QueryComponent{c[0]}, nil
	}

	return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: "FANG (Custom)"}}, errors.New("No portfolio specified")
}

func verifyQuandlCode(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) < 1 || c[0].QueryComponentType != component.CustomQuandlCode {
		return []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType: component.CustomQuandlCode,
			},
		}, nil
	}

	return c, nil
}

func verifyFormula(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) < 2 || c[0].QueryComponentType != component.TimeSeriesFormula {
		return []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType: component.TimeSeriesFormula,
			},
			component.QueryComponent{
				QueryComponentType: component.FreeText,
			},
		}, nil
	}

	return c, nil
}

func verifyRemoveData(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) < 1 || c[0].QueryComponentType != component.RemoveData {
		return []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType: component.RemoveData,
			},
		}, nil
	}

	return c, nil
}

func verifyRenameEntity(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) < 1 || c[0].QueryComponentType != component.RenameEntity {
		return []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType: component.RenameEntity,
			},
		}, nil
	}

	return c, nil
}

func verifyFreeText(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) < 1 || c[0].QueryComponentType != component.FreeText {
		return []component.QueryComponent{
			component.QueryComponent{
				QueryComponentType: component.FreeText,
			},
		}, nil
	}

	return c, nil
}

func verifyDataField(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 0 {
		if c[0].QueryComponentType == component.ConceptSecurity {
			return verifyDateRange(c)
		} else {
			c[0] = component.QueryComponent{QueryComponentOriginalString: "Price"}
			return verifyDateRange(c)
		}
	}

	return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: "Price"}}, errors.New("No data field specified")
}

func verifyMacroEvent(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 0 && c[0].QueryComponentType == component.MacroEvent {
		return []component.QueryComponent{c[0]}, nil
	}

	return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: "QE announcements"}}, errors.New("No event specified")
}

func verifyStockEvent(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
	if len(c) > 0 && c[0].QueryComponentType == component.MacroEvent {
		return []component.QueryComponent{c[0]}, nil
	}

	return []component.QueryComponent{component.QueryComponent{QueryComponentOriginalString: "FDA Approval"}}, errors.New("FDA Approval")
}

func requireNumberOfFields(numFields int) func(MultiEntityData, []component.QueryComponent) ([]component.QueryComponent, error) {
	return func(m MultiEntityData, c []component.QueryComponent) ([]component.QueryComponent, error) {
		// Check number of fields matches
		if len(c) == 0 {
			return nil, errors.New("No step specified")
		}

		newComponents := make([]component.QueryComponent, 1)
		newComponents[0] = c[0]
		newComponents = append(newComponents, createFieldArray(numFields, m)...)

		if len(c) < (numFields + 1) {
			return newComponents, errors.New(fmt.Sprintf("Computation step %s needs %v arguments but only %v are supplied", c[0].QueryComponentCanonicalName, numFields, len(c)-1))
		}

		// Check for the existence of the necessary fields
		for i := 1; i < len(c); i++ {
			//if !checkField(m, c[i].QueryComponentCanonicalName) {
			if c[i].QueryComponentType != component.Field {
				return newComponents, errors.New(c[i].QueryComponentCanonicalName + " is not a valid field")
			}
		}

		return c, nil
	}
}

func createFieldArray(numFields int, m MultiEntityData) []component.QueryComponent {
	c := make([]component.QueryComponent, numFields, numFields)

	fields := getUniqueFields(m)
	var fieldName string

	for i := 0; i < numFields; i++ {
		if i < len(fields) {
			fieldName = fields[i]
		} else {
			if len(fields) > 0 {
				fieldName = fields[0]
			} else {
				fieldName = "Specify Field Name"
			}
		}
		c[i] = component.QueryComponent{QueryComponentCanonicalName: fieldName, QueryComponentName: fieldName, QueryComponentType: "Field", QueryComponentOriginalString: fieldName}
	}

	return c
}

func getUniqueFields(m MultiEntityData) []string {
	mp := make(map[string]bool)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			mp[v2.Meta.Label] = true
		}
	}

	fields := make([]string, 0)

	for k, _ := range mp {
		fields = append(fields, k)
	}

	return fields
}

func checkField(m MultiEntityData, field string) bool {
	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if v2.Meta.Label == field {
				return true
			}
		}
	}

	return false
}

func DynamicDispatchUnary(major component.MajorType, c []component.QueryComponent, m MultiEntityData) (StepFnType, error) {
	compute, err := retrieveComputationStep(major, c)

	if err != nil {
		return nil, err
	}

	_, err = compute.ArgCheckFn(m, c)

	if err != nil {
		return nil, err
	}

	return compute.ComputeFn(c), nil
}

func retrieveComputationStep(major component.MajorType, c []component.QueryComponent) (ComputationStep, error) {
	if len(c) < 1 {
		return ComputationStep{}, errors.New("No components specified for the step " + string(major))
	}

	for _, v := range ComputationsSteps {
		if v.Type == major && v.Name == c[0].QueryComponentCanonicalName {
			return v, nil
		}
	}

	return ComputationStep{}, errors.New("No computation step found for " + string(major) + " specifically " + c[0].QueryComponentCanonicalName)
}

func componentRequiresParameter(c []component.QueryComponent) (bool, int) {
	for i, v := range c {
		if strings.Contains(v.QueryComponentCanonicalName, "{") {
			return true, i
		}
	}

	return false, -1
}

func convertComponentsToStringArray(c []component.QueryComponent) []string {
	s := make([]string, len(c), len(c))

	for i, v := range c {
		s[i] = v.QueryComponentCanonicalName
	}

	return s
}

// Mathematical Functions
func periodReturn(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	var newData []DataPoint

	if len(d) > 0 {
		newData = make([]DataPoint, 0, len(d)-1)
	} else {
		return nil
	}
	for i := 1; i < len(d); i++ {
		if d[i-1].Data != 0 {
			newData = append(newData, DataPoint{d[i].Time, (d[i].Data / d[i-1].Data) - 1})
		}
	}

	return newData
}

func actualChange(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	var newData []DataPoint

	if len(d) > 0 {
		newData = make([]DataPoint, 0, len(d)-1)
	} else {
		return nil
	}
	for i := 1; i < len(d); i++ {
		if d[i-1].Data != 0 {
			newData = append(newData, DataPoint{d[i].Time, d[i].Data - d[i-1].Data})
		}
	}

	return newData
}

func cagr(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	if len(d) == 0 {
		return d
	}

	firstPoint := d[0]
	lastPoint := d[len(d)-1]

	years := lastPoint.Time.Sub(firstPoint.Time).Hours() / 24.0 / 365.0

	data := math.Pow(lastPoint.Data/firstPoint.Data, 1.0/years) - 1

	return []DataPoint{DataPoint{lastPoint.Time, data}}
}

func cumulativeChange(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	var newData []DataPoint
	var isEventStudy bool = false
	var zeroDayValue float64

	if len(d) > 0 {
		newData = make([]DataPoint, 0, len(d))
		if d[0].Time.Before(time.Date(1850, 01, 01, 0, 0, 0, 0, time.UTC)) {
			isEventStudy = true
		}
	} else {
		return nil
	}

	if isEventStudy {
		for _, v := range d {
			daysValue := v.Time.UTC().Sub(zeroDay()).Hours() / 24
			if daysValue < 0.01 {
				zeroDayValue = v.Data
			}
		}
		if zeroDayValue == 0 {
			zeroDayValue = d[0].Data
		}
	}

	for i, v := range d {
		if isEventStudy {
			newData = append(newData, DataPoint{d[i].Time, (v.Data / zeroDayValue) - 1})
		} else {
			newData = append(newData, DataPoint{d[i].Time, (v.Data / d[0].Data) - 1})
		}
	}

	return newData
}

func sum(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	s := 0.0

	for _, v := range d {
		s = s + v.Data
	}

	if len(d) > 0 {
		return []DataPoint{DataPoint{d[len(d)-1].Time, s}}
	}

	return nil
}

func lastValue(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	if len(d) > 0 {
		return []DataPoint{d[len(d)-1]}
	}

	return nil
}

func zeroValue(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	if len(d) > 0 {
		return []DataPoint{DataPoint{d[len(d)-1].Time, 0.0}}
	}

	return nil
}

func count(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	if len(d) > 0 {
		return []DataPoint{DataPoint{d[len(d)-1].Time, float64(len(d))}}
	}

	return nil
}

func average(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	s := sum(m, d, w)
	c := count(m, d, w)

	if len(s) > 0 {
		return []DataPoint{DataPoint{s[0].Time, s[0].Data / c[0].Data}}
	}

	return nil
}

func geometric(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	s := 1.0

	for _, v := range d {
		s = s * (1 + v.Data)
	}

	s = s - 1

	if len(d) > 0 {
		return []DataPoint{DataPoint{d[len(d)-1].Time, s}}
	}

	return nil
}

// Resampling Functions
func resampleOnDatesFast(newDates []time.Time, beginIncomplete bool, endIncomplete bool) func(SeriesMeta, []DataPoint, []DataPoint) []DataPoint {
	return func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
		if m.Downsample == ResampleArithmetic || m.Downsample == ResampleGeometric {
			return resampleOnDates(newDates, beginIncomplete, endIncomplete)(m, d, w)
		}
		// Otherwise, we do a much faster resampling

		newData := make([]DataPoint, len(newDates))

		j := 0
		leftTrim := 0
		for i, v := range newDates {
			newData[i].Time = v

			for ; j+1 < len(d) && (v.After(d[j+1].Time) || v.Equal(d[j+1].Time)); j++ {
			}

			if j < len(d) && v.Before(d[j].Time) {
				leftTrim = leftTrim + 1
			} else if j+1 < len(d) && v.Before(d[j+1].Time) || j == len(d)-1 {
				if m.Downsample == ResampleZero {
					newData[i].Data = 0
				}
				newData[i].Data = d[j].Data
			} else {
				if i > 0 {
					if m.Upsample == ResampleNone {
						leftTrim = leftTrim + 1
					} else if m.Upsample == ResampleZero {
						newData[i].Data = 0
					} else {
						newData[i].Data = newData[i-1].Data
					}
				} else {
					leftTrim = leftTrim + 1
				}
			}
		}

		return newData[:len(newData)-leftTrim]
	}
}

func resampleOnDates(newDates []time.Time, beginIncomplete bool, endIncomplete bool) func(SeriesMeta, []DataPoint, []DataPoint) []DataPoint {
	return func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
		newData := make([]DataPoint, 0)
		var newPoints []DataPoint
		var newWeights []DataPoint

		// fmt.Printf("oldPoints = %s\n", d)
		// fmt.Printf("newDates = %s\n", newDates)
		// fmt.Printf("zeroDay = %s\n", zeroDay())

		dropBegin := (m.Downsample == ResampleArithmetic || m.Downsample == ResampleGeometric) && beginIncomplete
		dropEnd := (m.Downsample == ResampleArithmetic || m.Downsample == ResampleGeometric) && endIncomplete

		for i, v := range newDates {
			if (dropBegin && i == 0) || (dropEnd && i == len(newDates)-1) {
				// Don't add this point because we're told to drop it
			} else {
				if i == 0 {
					newPoints = resampleChunk(time.Time{}, v, d, m)
					newWeights = resampleChunk(time.Time{}, v, w, weightStub(zeroDay()).Meta)
				} else {
					newPoints = resampleChunk(newDates[i-1], v, d, m)
					newWeights = resampleChunk(time.Time{}, v, w, weightStub(zeroDay()).Meta)
				}

				// fmt.Printf("v = %s, newPoints = %s\n", v, newPoints)
				if len(newPoints) > 0 && len(newWeights) > 0 && newWeights[len(newWeights)-1].Data != 0 {
					newData = append(newData, newPoints...)
				}
			}
		}

		return newData

	}
}

func resampleChunk(startDate time.Time, endDate time.Time, d []DataPoint, m SeriesMeta) []DataPoint {
	dataChunk := getDataBetween(startDate, endDate, d)
	// fmt.Printf("startDate=%s, endDate=%s, d=%s, dataChunk=%s\n", startDate, endDate, d, dataChunk)

	// This means that no data exists between the dates. We need to use the upsample
	if len(dataChunk) == 0 {
		previousData := getDataBetween(zeroDay(), startDate, d)

		if len(previousData) == 0 {
			return nil
		}

		switch m.Upsample {
		case ResampleNone:
			return nil
		case ResampleLastValue:
			lv := lastValue(m, previousData, nil)
			if len(lv) > 0 {
				lv[0].Time = endDate
			}
			return lv
		case ResampleZero:
			zv := zeroValue(m, previousData, nil)
			if len(zv) > 0 {
				zv[0].Time = endDate
			}
			return zv
		}
	} else {
		// Downsamples

		switch m.Downsample {
		case ResampleNone:
			return nil
		case ResampleGeometric:
			return geometric(m, dataChunk, nil)
		case ResampleArithmetic:
			return sum(m, dataChunk, nil)
		case ResampleLastValue:
			return lastValue(m, dataChunk, nil)
		case ResampleZero:
			// Doesn't make any sense in this context
			return nil
		}
	}

	return nil
}

func getDataBetweenInclusive(startDate time.Time, endDate time.Time, d []DataPoint) []DataPoint {
	newData := make([]DataPoint, 0)
	var latestPoint int = -1

	for i, v := range d {
		if v.Time.Before(startDate) || v.Time.Equal(startDate) {
			latestPoint = i
		}

		if startDate.Before(v.Time) && (v.Time.Before(endDate) || v.Time.Equal(endDate)) {
			newData = append(newData, v)
		}
	}

	if latestPoint > -1 {
		newData = append([]DataPoint{DataPoint{startDate, d[latestPoint].Data}}, newData...)
		//newData = append([]DataPoint{d[latestPoint]}, newData...)
	}

	return newData
}

func getDataBetween(startDate time.Time, endDate time.Time, d []DataPoint) []DataPoint {
	newData := make([]DataPoint, 0)

	for _, v := range d {
		if startDate.Before(v.Time) && (v.Time.Before(endDate) || v.Time.Equal(endDate)) {
			newData = append(newData, v)
		}
	}

	return newData
}

// Date Functions
func getDaysBetween(startDate time.Time, endDate time.Time) []time.Time {
	tArr := make([]time.Time, 0)

	for startDate.Before(endDate) || startDate.Equal(endDate) {
		// if startDate.Weekday() != time.Saturday && startDate.Weekday() != time.Sunday {
		tArr = append(tArr, startDate)
		// }
		startDate = startDate.AddDate(0, 0, 1)
	}

	return tArr
}

func getWeeklyDatesBetween(startDate time.Time, endDate time.Time) ([]time.Time, bool, bool) {
	var beginIncomplete = false
	var endIncomplete = false

	tArr := make([]time.Time, 0)

	if startDate.Weekday() != time.Monday && startDate.Weekday() != time.Saturday {
		// TODO: This will change depending on what our week boundary is
		beginIncomplete = true
	}

	for startDate.Before(endDate) {
		if startDate.Weekday() == time.Friday {
			tArr = append(tArr, startDate)
		}
		startDate = startDate.AddDate(0, 0, 1)
	}

	// Add the end date if it's not included
	if len(tArr) == 0 || (len(tArr) > 0 && tArr[len(tArr)-1].Before(endDate)) {
		tArr = append(tArr, endDate)
		endIncomplete = true
	}

	return tArr, beginIncomplete, endIncomplete
}

func getBOMDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

func getEOMDate(date time.Time) time.Time {
	// Get the first of the month
	date = getBOMDate(date)

	// Subtract a day to get the end of the last month
	// Then add a month to get the end of the current month
	date = date.AddDate(0, 1, 0).AddDate(0, 0, -1)

	return date
}

func getBOMDates(startDate time.Time, endDate time.Time) []time.Time {
	tArr := make([]time.Time, 0)

	for startDate.Before(endDate) || startDate.Equal(endDate) {
		startDate = getBOMDate(startDate)
		tArr = append(tArr, startDate)
		startDate = startDate.AddDate(0, 1, 0)
	}

	return tArr
}

func getMonthlyDates(startDate time.Time, endDate time.Time) ([]time.Time, bool, bool) {
	var beginIncomplete = false
	var endIncomplete = false

	tArr := getBOMDates(startDate, endDate)

	if len(tArr) > 0 && tArr[0].Before(startDate) {
		beginIncomplete = true
	}

	for i, _ := range tArr {
		tArr[i] = getEOMDate(tArr[i])
		if tArr[i].After(endDate) {
			tArr[i] = endDate
			endIncomplete = true
		}
	}

	return tArr, beginIncomplete, endIncomplete
}

func getQuarterlyDates(startDate time.Time, endDate time.Time) ([]time.Time, bool, bool) {
	tArr, beginIncompleteMonth, endIncompleteMonth := getMonthlyDates(startDate, endDate)
	newArr := make([]time.Time, 0)

	beginIncomplete := beginIncompleteMonth
	endIncomplete := endIncompleteMonth

	if len(tArr) > 0 && (tArr[0].Month() != 1 && tArr[0].Month() != 4 && tArr[0].Month() != 8 && tArr[0].Month() != 11) {
		beginIncomplete = true
	}

	// TODO: Check the begin and end logic
	for i, v := range tArr {
		if v.Month() == 12 || v.Month() == 3 || v.Month() == 6 || v.Month() == 9 {
			newArr = append(newArr, v)
		} else if i == (len(tArr) - 1) {
			endIncomplete = true
			newArr = append(newArr, v)
		}
	}

	return newArr, beginIncomplete, endIncomplete
}

func getYearlyDates(startDate time.Time, endDate time.Time) ([]time.Time, bool, bool) {
	tArr, beginIncompleteMonth, endIncompleteMonth := getMonthlyDates(startDate, endDate)
	newArr := make([]time.Time, 0)

	beginIncomplete := beginIncompleteMonth
	endIncomplete := endIncompleteMonth

	if len(tArr) > 0 && tArr[0].Month() != 1 {
		beginIncomplete = true
	}

	// TODO: Check the begin and end logic
	for i, v := range tArr {
		if v.Month() == 12 {
			newArr = append(newArr, v)
		} else if i == (len(tArr) - 1) {
			endIncomplete = true
			newArr = append(newArr, v)
		}
	}

	return newArr, beginIncomplete, endIncomplete
}

func getStartEndDatesForEntity(s SingleEntityData) (time.Time, time.Time) {
	var startDate, endDate time.Time
	startDate = time.Now()

	for _, v := range s.Data {
		if len(v.Data) > 0 {
			if v.Data[0].Time.Before(startDate) {
				startDate = v.Data[0].Time
			}

			if v.Data[len(v.Data)-1].Time.After(endDate) {
				endDate = v.Data[len(v.Data)-1].Time
			}
		}
	}

	return startDate, endDate
}

func getWeightStartEndDates(w Series) ([]time.Time, []time.Time) {
	startDates := make([]time.Time, 0)
	endDates := make([]time.Time, 0)
	now := time.Now()

	for i, v := range w.Data {
		if i == 0 {
			if v.Data != 0 {
				startDates = append(startDates, v.Time)
				endDates = append(endDates, now)
			}
		} else {
			if v.Data == 0 && w.Data[i-1].Data != 0 {
				// This means that this is a new endDate
				if len(endDates) > 0 {
					endDates[len(endDates)-1] = w.Data[i-1].Time
				} else {
					startDates = append(startDates, w.Data[i-1].Time)
					endDates = append(endDates, w.Data[i-1].Time)
				}
			} else if v.Data != 0 && w.Data[i-1].Data == 0 {
				// This means that this is a new startDate
				startDates = append(startDates, v.Time)
				endDates = append(endDates, now)
			}
		}
	}

	return startDates, endDates
}

func getAllStartDates(w Series) []time.Time {
	startDates := make([]time.Time, 0)

	for _, v := range w.Data {
		if v.Data != 0 {
			startDates = append(startDates, v.Time)
		}
	}

	return startDates
}

func getSeriesDataFromStartEndDate(s Series, startDate time.Time, endDate time.Time) (Series, string, bool) {
	var newSeries Series
	var timeRange string
	var rangeFound bool = false

	newSeries.Data = make([]DataPoint, 0)
	newSeries.Meta = s.Meta

	var minIndex int = -1
	var maxIndex int = -1

	// Use inclusive search
	// Search for the data that's between the startDate and endDate. If none, return false

	for i, v := range s.Data {
		if (startDate.Before(v.Time) || startDate.Equal(v.Time)) && (endDate.After(v.Time) || endDate.Equal(v.Time)) {
			if minIndex == -1 {
				minIndex = i
			}
			maxIndex = i
			rangeFound = true
			newSeries.Data = append(newSeries.Data, v)
		}
	}

	if rangeFound {
		timeFrom := s.Data[minIndex].Time.String()[0:10]
		timeTo := s.Data[maxIndex].Time.String()[0:10]

		if timeFrom == timeTo {
			timeRange = " on " + timeFrom
		} else {
			timeRange = " from " + timeFrom + " to " + s.Data[maxIndex].Time.String()[0:10]
		}
		newSeries.Meta.Label = newSeries.Meta.Label + timeRange
	}

	return newSeries, timeRange, rangeFound
}

func breakSeriesOnStartEndDates(s Series, startDates []time.Time, endDates []time.Time) []Series {
	var seriesArray []Series = make([]Series, 0)

	for i := 0; i < len(startDates); i++ {
		newSeries, _, dataIsBetween := getSeriesDataFromStartEndDate(s, startDates[i], endDates[i])

		if dataIsBetween {
			seriesArray = append(seriesArray, newSeries)
		}
	}

	return seriesArray
}

func breakSeriesOnStartEndDatesAlignData(s Series, startDates []time.Time, endDates []time.Time, pivotDates []time.Time, unadjustedDates []time.Time) ([]Series, []string, []time.Time, []time.Time, []time.Time) {
	var seriesArray []Series = make([]Series, 0)
	var timeRangeArray []string = make([]string, 0)
	var startDateArray []time.Time = make([]time.Time, 0)
	var alignedDateArray []time.Time = make([]time.Time, 0)
	var unadjustedArray []time.Time = make([]time.Time, 0)

	for i := 0; i < len(startDates); i++ {
		newSeries, timeRange, dataIsBetween := getSeriesDataFromStartEndDate(s, startDates[i], endDates[i])

		newSeries = alignTimePointsToZero(newSeries, pivotDates[i])

		if dataIsBetween {
			seriesArray = append(seriesArray, newSeries)
			timeRangeArray = append(timeRangeArray, timeRange)
			startDateArray = append(startDateArray, startDates[i])
			alignedDateArray = append(alignedDateArray, newSeries.GetDates()[0])
			unadjustedArray = append(unadjustedArray, unadjustedDates[i])
		}
	}

	return seriesArray, timeRangeArray, startDateArray, alignedDateArray, unadjustedArray
}

func filterSeriesOnStartEndDates(s Series, startDates []time.Time, endDates []time.Time) Series {
	var newSeries Series
	newSeries.IsWeight = s.IsWeight
	newSeries.Meta = s.Meta

	for i := 0; i < len(startDates); i++ {
		currentSeries, _, dataIsBetween := getSeriesDataFromStartEndDate(s, startDates[i], endDates[i])

		if dataIsBetween {
			newSeries.Data = append(newSeries.Data, currentSeries.Data...)
		}
	}

	return newSeries
}

func alignTimePointsToZero(s Series, pivotPoint time.Time) Series {
	if len(s.Data) == 0 {
		return s
	}

	for i, _ := range s.Data {
		s.Data[i].Time = zeroDay().Add(s.Data[i].Time.Sub(pivotPoint))
	}

	return s
}

// SingleEntityData functions
func AlignEvent(s SingleEntityData) []SingleEntityData {
	// Create a new entity
	var entityArray []SingleEntityData = make([]SingleEntityData, 0)

	weightSeries := getWeightSeries(s.Data)

	// Get all the nonzero periods
	startDates, endDates := getWeightStartEndDates(weightSeries)

	for _, v := range s.Data {
		if !v.IsWeight {
			// Run a function to break the current series into ones specified by the data points
			temp, dateRange, _, newAlignedDates, unadjustedDates := breakSeriesOnStartEndDatesAlignData(v, startDates, endDates, startDates, startDates)

			for i, v2 := range temp {
				var e SingleEntityData
				v2.Meta.Label = v.Meta.Label
				e.Data = []Series{v2}
				e.Meta = s.Meta
				e.Meta.Name = e.Meta.Name + dateRange[i]
				e.Meta.UniqueId = e.Meta.Name
				e.Category = e.Category.Insert(newAlignedDates[i], CategoryLabel{Label: s.Category.LookupCategory(unadjustedDates[i])})
				// if e.Category.Data == nil || len(e.Category.Data) == 0 {
				// 	e.Category = e.Category.Insert(zeroDay(), CategoryLabel{Label: s.Meta.Name})
				// }
				entityArray = updateEntityArray(entityArray, e)
			}
		}
	}

	return entityArray
}

func AlignEventBeforeAfterPreserveDates(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	singleSeries := func(s SingleEntityData) ([]SingleEntityData, []time.Time) {
		beforeInt := 0
		afterInt := 90

		// Create a new entity
		var entityArray []SingleEntityData = make([]SingleEntityData, 0)

		weightSeries := getWeightSeries(s.Data)

		// Get all the nonzero periods
		// startDatesOriginal, _ := getWeightStartEndDates(weightSeries)
		startDatesUnaltered := getAllStartDates(weightSeries)

		var startDatesOriginal []time.Time
		// fmt.Printf("startDates before market impact=%s\n", startDatesOriginal)

		startDatesOriginal = startDatesUnaltered
		// fmt.Printf("startDates after market impact=%s\n", startDatesOriginal)

		startDates := make([]time.Time, len(startDatesOriginal))
		endDates := make([]time.Time, len(startDatesOriginal))

		for i, _ := range startDatesOriginal {
			startDates[i] = startDatesOriginal[i].AddDate(0, 0, -beforeInt)
			endDates[i] = startDatesOriginal[i].AddDate(0, 0, afterInt)
		}

		for _, v := range s.Data {
			if !v.IsWeight {
				// Run a function to break the current series into ones specified by the data points
				temp, dateRange, _, newAlignedDates, unadjustedDates := breakSeriesOnStartEndDatesAlignData(v, startDates, endDates, startDatesOriginal, startDatesUnaltered)

				for i, v2 := range temp {
					var e SingleEntityData
					v2.Meta.Label = v.Meta.Label
					e.Data = []Series{v2}
					e.Meta = s.Meta
					e.Meta.Name = e.Meta.Name + dateRange[i]
					e.Meta.UniqueId = e.Meta.Name
					e.Category = e.Category.Insert(newAlignedDates[i], CategoryLabel{Label: s.Category.LookupCategory(unadjustedDates[i])})
					// if e.Category.Data == nil || len(e.Category.Data) == 0 {
					// 	e.Category = e.Category.Insert(zeroDay(), CategoryLabel{Label: s.Meta.Name})
					// }
					entityArray = updateEntityArray(entityArray, e)
				}
			}
		}

		return entityArray, startDatesOriginal
	}

	if len(mArr) > 0 {
		m := mArr[0]
		tempEntity := make([]SingleEntityData, 0)

		var latestStartDate time.Time

		log.Debugf(ctx, "latestStartDate = %s", latestStartDate)

		for _, v := range m.EntityData {
			temp, startDatesOriginal := singleSeries(v)

			if len(temp) > 0 {
				log.Debugf(ctx, "len(temp) > 0")

				tempEntity = append(tempEntity, temp...)

				for _, startDate := range startDatesOriginal {
					if startDate.After(latestStartDate) {
						latestStartDate = startDate
						log.Debugf(ctx, "latestStartDate = %s", latestStartDate)
					}
				}
			}
		}

		// Restore dates to the data

		delta := latestStartDate.Sub(zeroDay())

		for i, v := range tempEntity {
			for i2, v2 := range v.Data {
				for i3, v3 := range v2.Data {
					tempEntity[i].Data[i2].Data[i3].Time = v3.Time.Add(delta)
				}
			}
		}

		m.EntityData = tempEntity

		return m
	}

	return MultiEntityData{}
}

func AlignEventBeforeAfter(marketAlign bool) func(float64, float64) func(SingleEntityData) []SingleEntityData {
	return func(before, after float64) func(SingleEntityData) []SingleEntityData {
		return func(s SingleEntityData) []SingleEntityData {
			beforeInt := int(before)
			afterInt := int(after)

			// Create a new entity
			var entityArray []SingleEntityData = make([]SingleEntityData, 0)

			weightSeries := getWeightSeries(s.Data)

			// Get all the nonzero periods
			// startDatesOriginal, _ := getWeightStartEndDates(weightSeries)
			startDatesUnaltered := getAllStartDates(weightSeries)

			marketDates := unionSeriesDates(s.Data)

			var startDatesOriginal []time.Time
			// fmt.Printf("startDates before market impact=%s\n", startDatesOriginal)
			if marketAlign {
				startDatesOriginal = MarketImpactDates(startDatesUnaltered, marketDates)
			} else {
				startDatesOriginal = startDatesUnaltered
			}
			// fmt.Printf("startDates after market impact=%s\n", startDatesOriginal)

			startDates := make([]time.Time, len(startDatesOriginal))
			endDates := make([]time.Time, len(startDatesOriginal))

			for i, _ := range startDatesOriginal {
				startDates[i] = startDatesOriginal[i].AddDate(0, 0, -beforeInt)
				endDates[i] = startDatesOriginal[i].AddDate(0, 0, afterInt)
			}

			for _, v := range s.Data {
				if !v.IsWeight {
					// Run a function to break the current series into ones specified by the data points
					temp, dateRange, _, newAlignedDates, unadjustedDates := breakSeriesOnStartEndDatesAlignData(v, startDates, endDates, startDatesOriginal, startDatesUnaltered)

					for i, v2 := range temp {
						var e SingleEntityData
						v2.Meta.Label = v.Meta.Label
						e.Data = []Series{v2}
						e.Meta = s.Meta
						e.Meta.Name = e.Meta.Name + dateRange[i]
						e.Meta.UniqueId = e.Meta.Name
						e.Category = e.Category.Insert(newAlignedDates[i], CategoryLabel{Label: s.Category.LookupCategory(unadjustedDates[i])})
						// if e.Category.Data == nil || len(e.Category.Data) == 0 {
						// 	e.Category = e.Category.Insert(zeroDay(), CategoryLabel{Label: s.Meta.Name})
						// }
						entityArray = updateEntityArray(entityArray, e)
					}
				}
			}

			return entityArray
		}
	}
}

func MarketImpactDates(eventDates []time.Time, marketDates []time.Time) []time.Time {
	tempEventDates := make([]time.Time, len(eventDates))

	for i, _ := range eventDates {
		tempEventDates[i] = eventDates[i]
	}

	impactDates := make([]time.Time, len(eventDates))

	// Adjust date based on time
	// - If the UTC time is exactly midnight, then use the UTC date (i.e. pretend during market)
	// - If the UTC time is not midnight
	//	- Convert to Eastern Time Zone
	//	- If the time is before or equal to 4:00 pm, then set the date to the current date
	//	- If the time is after 4:00 pm, set the date to the current date + 1

	for i, v := range tempEventDates {
		if v.Hour() == 0 && v.Minute() == 0 {
			tempEventDates[i] = v
		} else {
			easternZone, err := time.LoadLocation("America/New_York")
			if err != nil {
				tempEventDates[i] = v
			} else {
				easternTime := v.In(easternZone)
				currentDateUtc := time.Date(easternTime.Year(), easternTime.Month(), easternTime.Day(), 0, 0, 0, 0, time.UTC)

				if easternTime.Hour() < 16 {
					tempEventDates[i] = currentDateUtc
				} else {
					tempEventDates[i] = currentDateUtc.AddDate(0, 0, 1)
				}
			}

		}
	}

	// For each of the adjusted dates:
	//	- Find the first greatest market date date that is equal to or less than the adjusted date
	//	- There is an exception such that if the event occurs before the first market date
	//	then we should keep the event date because we will later need to throw away this data point
	//	since we don't have market data for it
	j := 0
	for i, v := range tempEventDates {
		for ; j < len(marketDates) && marketDates[j].Before(v); j++ {

		}

		if j == 0 || j >= len(marketDates) {
			// We don't want all events that occur before the first
			// market date to cluster to the first market date,
			// so we set it to its original event date with the
			// idea that we're going to throw away the value in the
			// caller function.
			impactDates[i] = v
		} else {
			impactDates[i] = marketDates[j]
		}

	}

	return impactDates
}

func updateEntityArray(entityArray []SingleEntityData, s SingleEntityData) []SingleEntityData {
	var found bool = false

	for i, v := range entityArray {
		if v.Meta.Name == s.Meta.Name {
			found = true
			entityArray[i].Data = append(entityArray[i].Data, s.Data...)
		}
	}

	if !found {
		entityArray = append(entityArray, s)
	}

	return entityArray
}

func spread(s SingleEntityData) SingleEntityData {
	if len(s.Data) < 2 {
		return s
	}

	for i := 1; i < len(s.Data); i++ {
		if s.Data[i].IsWeight == true {
			continue
		}

		s.Data[i] = runBinaryFunctionOnSeries(s.Data[0], s.Data[i], getWeightSeries(s.Data),
			func(a float64, b float64) float64 {
				return a - b
			},
			metaTransformBinary(
				func(s1, s2 string) string { return "Spread between " + s1 + " and " + s2 },
				func(s1, s2 string) string {
					if s1 == s2 {
						return s1
					}
					return "Spread between " + s1 + " and " + s2
				},
			),
		)

	}

	s.Data = s.Data[1:]

	return s
}

func ratio(s SingleEntityData) SingleEntityData {
	if len(s.Data) < 2 {
		return s
	}

	for i := 1; i < len(s.Data); i++ {
		if s.Data[i].IsWeight == true {
			continue
		}

		s.Data[i] = runBinaryFunctionOnSeries(s.Data[0], s.Data[i], getWeightSeries(s.Data),
			func(a float64, b float64) float64 {
				if b != 0 {
					return a / b
				}
				return 0
			},
			metaTransformBinary(
				func(s1, s2 string) string { return "Ratio of " + s1 + " to " + s2 },
				func(s1, s2 string) string { return "Ratio" },
			),
		)

	}

	s.Data = s.Data[1:]

	return s
}

func categorizeBySector(level list.SectorLevel) func(SingleEntityData) SingleEntityData {
	// Get all the S&P 500 sector mappings
	members, sectors, _ := data.GetAllSectorMappings(level)

	return func(s SingleEntityData) SingleEntityData {
		s.Category = CategorySeries{}

		for i, v := range members {
			if v.QueryComponentProviderId == s.Meta.UniqueId {
				s.Category = s.Category.Insert(zeroDay(), CategoryLabel{Label: sectors[i]})
			}
		}

		if len(s.Category.Data) == 0 {
			fmt.Printf("No category for %s (%s)\n", s.Meta.Name, s.Meta.UniqueId)
		}

		return s
	}
}

func noClassification(s SingleEntityData) SingleEntityData {
	s.Category = CategorySeries{}

	return s
}

func categorizeByCompany(s SingleEntityData) SingleEntityData {
	s.Category = CategorySeries{}
	s.Category = s.Category.Insert(zeroDay(), CategoryLabel{Label: s.Meta.Name})

	return s
}

func tsTotalReturn() func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data, periodReturn, metaTransform(prependString("% Change in "), replaceString("%Δ"), changeResampleType(ResampleNone), changeResampleType(ResampleGeometric)))

		return s
	}
}

func tsDifference() func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data, actualChange, metaTransform(prependString("Actual Change in "), prependString("Δ in"), changeResampleType(ResampleNone), changeResampleType(ResampleArithmetic)))

		return s
	}
}

func tsCumulativeChange() func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data, cumulativeChange, metaTransform(prependString("% Cumulative Change in "), replaceString("%Δ"), changeResampleType(ResampleLastValue), changeResampleType(ResampleLastValue)))

		return s
	}
}

func tsRemoveData(s SingleEntityData) SingleEntityData {
	keep := make([]Series, 0)

	for _, v := range s.Data {
		if v.IsWeight {
			keep = append(keep, v)
		}
	}

	s.Data = keep

	return s
}

func keepNamedField(field string) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		keep := make([]Series, 0)

		for _, v := range s.Data {
			if v.Meta.Label == field {
				keep = append(keep, v)
			}
		}

		s.Data = keep

		return s
	}
}

func removeNamedField(field string) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		keep := make([]Series, 0)

		for _, v := range s.Data {
			if v.Meta.Label != field &&
				v.Meta.Label != strings.Replace(field, " (Custom)", "", 1) &&
				strings.Replace(v.Meta.Label, " ", "", -1) != strings.Replace(field, " ", "", -1) {
				keep = append(keep, v)
			}
		}

		s.Data = keep

		return s
	}
}

func renameEntity(newName string) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		if s.Meta.Name != "" {
			s.Meta.Name = s.Meta.Name + " " + newName
			s.Meta.UniqueId = s.Meta.UniqueId + " " + newName
		} else {
			s.Meta.Name = newName
			s.Meta.UniqueId = newName
		}

		return s
	}
}

func tsRemoveWeights(s SingleEntityData) SingleEntityData {
	keep := make([]Series, 0)

	for _, v := range s.Data {
		if !v.IsWeight {
			keep = append(keep, v)
		}
	}

	s.Data = keep

	return s
}

func tsRemoveFilteredData(s SingleEntityData) SingleEntityData {
	weightSeries := getWeightSeries(s.Data)

	// Get all the nonzero periods
	startDates, endDates := getWeightStartEndDates(weightSeries)

	for i, v := range s.Data {
		if !v.IsWeight {
			// Run a function to break the current series into ones specified by the data points
			s.Data[i] = filterSeriesOnStartEndDates(v, startDates, endDates)

		}
	}

	//s.Data = applyToSeries(
	//	s.Data,
	//	func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
	//		d2 := make([]DataPoint}, 0)

	//		for _, v := range d {
	//			if v.Time
	//		}

	//		return d2
	//	},
	//	func(smeta SeriesMeta) SeriesMeta {
	//		return smeta
	//	},
	//)

	return tsRemoveWeights(s)
}

func earliestDate(s SingleEntityData) time.Time {
	t := time.Now()

	for _, v := range s.Data {
		if len(v.Data) > 0 && v.Data[0].Time.Before(t) {
			t = v.Data[0].Time
		}
	}

	return t
}

func latestWeightsHistorical(s SingleEntityData) SingleEntityData {
	// Find the earliest date
	earliestDate := earliestDate(s)

	// Return a data point that takes the last data point in the series
	// and puts it as the only data point with the date as the first date
	dataTransform := func(_ SeriesMeta, d []DataPoint, _ []DataPoint) []DataPoint {
		newData := make([]DataPoint, 1)
		if len(d) > 0 {
			newData[0].Data = d[len(d)-1].Data
			newData[0].Time = earliestDate
		}
		return newData
	}

	s.Data = applyToWeights(s.Data, dataTransform, metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

	return s
}

func rebalanceWeekly(s SingleEntityData) SingleEntityData {
	startDate, endDate := getStartEndDatesForEntity(s)
	dates, beginIncomplete, endIncomplete := getWeeklyDatesBetween(startDate, endDate)

	s.Data = applyToWeights(s.Data, resampleOnDates(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

	return s
}

func rebalanceMonthly(s SingleEntityData) SingleEntityData {
	startDate, endDate := getStartEndDatesForEntity(s)
	dates, beginIncomplete, endIncomplete := getMonthlyDates(startDate, endDate)

	s.Data = applyToWeights(s.Data, resampleOnDates(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

	return s
}

func rebalanceQuarterly(s SingleEntityData) SingleEntityData {
	startDate, endDate := getStartEndDatesForEntity(s)
	dates, beginIncomplete, endIncomplete := getQuarterlyDates(startDate, endDate)

	s.Data = applyToWeights(s.Data, resampleOnDates(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

	return s
}

func rebalanceYearly(s SingleEntityData) SingleEntityData {
	startDate, endDate := getStartEndDatesForEntity(s)
	dates, beginIncomplete, endIncomplete := getYearlyDates(startDate, endDate)

	s.Data = applyToWeights(s.Data, resampleOnDates(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

	return s
}

func tsDaily(endDate time.Time) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		startDate, _ := getStartEndDatesForEntity(s)
		dates := getDaysBetween(startDate, endDate)

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, false, false), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsByWeek(endDate time.Time) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		startDate, _ := getStartEndDatesForEntity(s)
		dates, beginIncomplete, endIncomplete := getWeeklyDatesBetween(startDate, endDate)

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsByMonth(endDate time.Time) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		startDate, _ := getStartEndDatesForEntity(s)
		dates, beginIncomplete, endIncomplete := getMonthlyDates(startDate, endDate)

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsByQuarter(endDate time.Time) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		startDate, _ := getStartEndDatesForEntity(s)
		dates, beginIncomplete, endIncomplete := getQuarterlyDates(startDate, endDate)

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsByYear(endDate time.Time) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		startDate, _ := getStartEndDatesForEntity(s)
		dates, beginIncomplete, endIncomplete := getYearlyDates(startDate, endDate)

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, beginIncomplete, endIncomplete), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsAllTime() func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		_, endDate := getStartEndDatesForEntity(s)
		dates := []time.Time{endDate}

		s.Data = applyToSeries(s.Data, resampleOnDatesFast(dates, false, false), metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

		return s
	}
}

func tsCagr(s SingleEntityData) SingleEntityData {
	s.Data = applyToSeries(s.Data, cagr, metaTransform(appendString("CAGR"), replaceString("%"), changeResampleType(ResampleLastValue), changeResampleType(ResampleLastValue)))

	return s
}

//func tsLatestData(s SingleEntityData) SingleEntityData {
//	applyToSeries(s.Data, lastValue, metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))

//	return s
//}

func tsDividedBy(field1, field2 float64) func(SingleEntityData) SingleEntityData {
	ix1 := int(field1) - 1
	ix2 := int(field2) - 1

	return applyFunctionToFields(ix1, ix2,
		func(a, b float64) float64 { return a / b },
		metaTransformBinary(
			func(s1, s2 string) string { return s1 + " divided by " + s2 },
			func(s1, s2 string) string { return "Ratio" },
		))
}

func tsMultiply(field1, field2 float64) func(SingleEntityData) SingleEntityData {
	ix1 := int(field1) - 1
	ix2 := int(field2) - 1

	return applyFunctionToFields(ix1, ix2,
		func(a, b float64) float64 { return a * b },
		metaTransformBinary(
			func(s1, s2 string) string { return s1 + " multiplied by " + s2 },
			func(s1, s2 string) string { return s1 + " * " + s2 },
		))
}

func tsAdd(field1, field2 float64) func(SingleEntityData) SingleEntityData {
	ix1 := int(field1) - 1
	ix2 := int(field2) - 1

	return applyFunctionToFields(ix1, ix2,
		func(a, b float64) float64 { return a + b },
		metaTransformBinary(
			func(s1, s2 string) string { return s1 + " + " + s2 },
			mergeSource,
		))
}

//func addFields(s SingleEntityData) SingleEntityData {
//	return applyFunctionToAllFields(
//		func(d []float64) float64 {
//			sum := 0.0
//			for _, v := range d {
//				sum += v
//			}
//			return sum
//		},
//		metaTransformBinary(
//			func(s1, s2 string) string { return s1 + " + " + s2 },
//			mergeSource,
//		))(s)
//}

//func averageFields(s SingleEntityData) SingleEntityData {
//	return applyFunctionToAllFields(
//		func(d []float64) float64 {
//			sum := 0.0
//			for _, v := range d {
//				sum += v
//			}
//			return sum / float64(len(d))
//		},
//		metaTransformBinary(
//			func(s1, s2 string) string { return s1 + " + " + s2 },
//			mergeSource,
//		))(s)
//}

func tsSubtract(field1, field2 float64) func(SingleEntityData) SingleEntityData {
	ix1 := int(field1) - 1
	ix2 := int(field2) - 1

	return applyFunctionToFields(ix1, ix2,
		func(a, b float64) float64 { return b - a },
		metaTransformBinary(
			func(s1, s2 string) string { return s2 + " - " + s1 },
			mergeSource,
		))
}

func metaTransformBinary(labelTransform func(string, string) string, unitsTransform func(string, string) string) func(SeriesMeta, SeriesMeta) SeriesMeta {
	return func(s1 SeriesMeta, s2 SeriesMeta) SeriesMeta {
		newMeta := SeriesMeta{}
		newMeta.IsTransformed = true

		if s1.Downsample == s2.Downsample {
			newMeta.Downsample = s1.Downsample
		} else {
			newMeta.Downsample = ResampleNone
		}

		if s1.Upsample == s2.Upsample {
			newMeta.Upsample = s1.Upsample
		} else {
			newMeta.Upsample = ResampleNone
		}

		newMeta.Label = labelTransform(s1.Label, s2.Label)

		newMeta.Source = mergeSource(s1.Source, s2.Source)

		newMeta.Units = unitsTransform(s1.Units, s2.Units)

		return newMeta
	}
}

func mergeSource(source1, source2 string) string {
	if source1 == source2 {
		return source1
	}

	return source1 + ", " + source2
}

//func applyFunctionToAllFields(fn func([]float64) float64, seriesMetaTransform func([]SeriesMeta) SeriesMeta) func(SingleEntityData) SingleEntityData {
//	return func(s SingleEntityData) SingleEntityData {
//		weightSeries := getWeightSeries(s.Data)

//		var newDates []time.Time

//		if len(s.Data) > 0 {
//			newDates = getDates(s.Data[0].Data)
//		}
//		for i := 1; i < len(s.Data); i++ {
//			newDates = unionDates(newDates, getDates(s.Data[i].Data))
//		}

//		// For each date, get the resampled data. If the resample is anything other than LastValue, no data point for this date
//		resampleFn := resampleOnDates(newDates)

//		for i, _ := range s.Data {
//			if !s.Data[i].IsWeight {
//				s.Data[i].Data = resampleFn(s.Data[i].Meta, s.Data[i].Data, weightSeries.Data)
//			}
//		}

//		//newData := matchDatesAndRun(newSeries1, newSeries2, fn)
//		newMeta := seriesMetaTransform(series1.Meta, series2.Meta)

//		//oldSeriesList := s.Data

//		//// Add the new series
//		//newSeries := runBinaryFunctionOnSeries(s.Data[idx1], s.Data[idx2], weightSeries, fn, seriesMetaTransform)

//		//// Remove the old series
//		//s.Data = make([]Series, 0)

//		//for i, v := range oldSeriesList {
//		//	if i != idx1 && i != idx2 {
//		//		s.Data = append(s.Data, v)
//		//	}
//		//}

//		//s.Data = append(s.Data, newSeries)

//		return s
//	}
//}

func applyFunctionToFields(idx1, idx2 int, fn func(float64, float64) float64, seriesMetaTransform func(SeriesMeta, SeriesMeta) SeriesMeta) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		weightSeries := getWeightSeries(s.Data)

		if idx1 == -1 || idx2 == -1 {
			return s
		}

		oldSeriesList := s.Data

		// Add the new series
		newSeries := runBinaryFunctionOnSeries(s.Data[idx1], s.Data[idx2], weightSeries, fn, seriesMetaTransform)

		// Remove the old series
		s.Data = make([]Series, 0)

		for i, v := range oldSeriesList {
			if i != idx1 && i != idx2 {
				s.Data = append(s.Data, v)
			}
		}

		s.Data = append(s.Data, newSeries)

		return s
	}
}

func runBinaryFunctionOnSeries(series1 Series, series2 Series, weightSeries Series, fn func(float64, float64) float64, metaFn func(SeriesMeta, SeriesMeta) SeriesMeta) Series {
	// Union the dates
	dates1 := getDates(series1.Data)
	dates2 := getDates(series2.Data)
	newDates := unionDates(dates1, dates2)

	// For each date, get the resampled data. If the resample is anything other than LastValue, no data point for this date
	resampleFn := resampleOnDates(newDates, false, false)

	newSeries1 := resampleFn(series1.Meta, series1.Data, weightSeries.Data)
	newSeries2 := resampleFn(series2.Meta, series2.Data, weightSeries.Data)

	newData := matchDatesAndRun(newSeries1, newSeries2, fn)
	newMeta := metaFn(series1.Meta, series2.Meta)

	return Series{Data: newData, Meta: newMeta, IsWeight: series1.IsWeight || series2.IsWeight}
}

func forceSeriesAlignment(sArr []Series) []Series {
	var newDates []time.Time = []time.Time{}
	var newSeries []Series = make([]Series, len(sArr))

	for i, _ := range sArr {
		newDates = unionDates(newDates, getDates(sArr[i].Data))
	}

	resampleFn := resampleOnDates(newDates, false, false)

	for i, _ := range sArr {
		weightSeries := getWeightSeries(sArr)
		newSeries[i].Meta = sArr[i].Meta
		newSeries[i].IsWeight = sArr[i].IsWeight
		newSeries[i].Data = resampleFn(sArr[i].Meta, sArr[i].Data, weightSeries.Data)

		// NaN Fill when not enough data
		if len(newSeries[i].Data) < len(newDates) {
			// fmt.Println("Insufficient data")
			newData := make([]DataPoint, len(newDates))
			for x, y := 0, 0; x < len(newSeries[i].Data) && y < len(newDates); {
				date1 := newSeries[i].Data[x].Time
				date2 := newDates[y]

				if date1.Equal(date2) {
					// No problem. Dates match.
					// fmt.Println("Equal dates")
					newData[y] = newSeries[i].Data[x]
					x++
					y++
				} else if date1.Before(date2) {
					// This really shouldn't happen since newDates is technically the union of all the dates
					// but just in case, this prevents an infinite loop.
					// fmt.Println("Weird problem with union dates")
					x++
				} else {
					// Need to NaN pad the data
					newData[y] = DataPoint{date2, math.NaN()}
					// fmt.Println("newData[%v] = %s\n", y, newData[y])
					y++
				}
			}
			// fmt.Printf("newData=%s\n", newData)
			newSeries[i].Data = newData
		}
	}

	return newSeries
}

func removeNaNs(sArr []Series) []Series {
	// fmt.Printf("Attempting to remove NaNs from sArr=%s\n", sArr)
	for i, _ := range sArr {
		newData := make([]DataPoint, 0, len(sArr[i].Data))
		for _, v := range sArr[i].Data {
			if !math.IsNaN(v.Data) {
				newData = append(newData, v)
				// fmt.Printf("This isn't a NaN v.Data=%s\n", v.Data)
			} else {
				// fmt.Printf("This IS A NaN v.Data=%s\n", v.Data)
			}
		}
		sArr[i].Data = newData
	}

	return sArr
}

func matchDatesAndRun(data1 []DataPoint, data2 []DataPoint, fn func(float64, float64) float64) []DataPoint {
	newData := make([]DataPoint, 0, len(data1))

	for i, j := 0, 0; i < len(data1) && j < len(data2); {
		if data1[i].Time.Equal(data2[j].Time) {
			newData = append(newData, DataPoint{data1[i].Time, fn(data1[i].Data, data2[j].Data)})
			i++
			j++
		} else if data1[i].Time.Before(data2[j].Time) {
			i++
		} else {
			j++
		}
	}

	return newData
}

func getDates(d []DataPoint) []time.Time {
	t := make([]time.Time, len(d), len(d))

	for i, v := range d {
		t[i] = v.Time
	}

	return t
}

func getFieldNum(field int64, s SingleEntityData) int {
	if int(field-1) < len(s.Data) {
		return int(field - 1)
	}
	//for i, v := range s.Data {
	//	if v.Meta.Label == field {
	//		return i
	//	}
	//}

	return -1
}

func tsMultiplyFieldByNumber(field string, number float64) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				if m.Label == field {
					for i, _ := range d {
						d[i].Data = d[i].Data * number
					}
				}
				return d
			},
			func(m SeriesMeta) SeriesMeta {
				m.IsTransformed = true
				m.Label = fmt.Sprintf("%s Multiplied By %v", m.Label, number)

				return m
			})

		return s
	}
}

func tsSampleEveryNumPeriods(number float64) func(SingleEntityData) SingleEntityData {
	days := int(number)

	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				// Create new data to return
				newData := make([]DataPoint, 0, len(d))

				for i := 0; i < len(d); i++ {
					if (i+1)%days == 0 {
						newData = append(newData, d[i])
					}
				}

				return newData
			},
			func(m SeriesMeta) SeriesMeta {
				return m
			})

		return s
	}
}

func tsForwardReturn(number float64) func(SingleEntityData) SingleEntityData {
	days := int(number)

	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				// If the series is too short, we can't compute anything
				if len(d) < 2 || d[len(d)-1].Time.Sub(d[0].Time).Hours()/24 < float64(days) {
					return []DataPoint{}
				}

				// Create new data to return
				newData := make([]DataPoint, 0, len(d))

				for i := 0; i < len(d); i++ {
					newData = append(newData, DataPoint{
						Time: d[i].Time,
						Data: percentageChange(getDataBetween(d[i].Time, d[i].Time.AddDate(0, 0, days), d)),
					})
				}

				return newData
			},
			func(m SeriesMeta) SeriesMeta {
				m.Label = fmt.Sprintf("%v-Day Forward Return", days)
				m.Units = "%"

				return m
			})

		return s
	}
}

func tsStdDev(number float64) func(SingleEntityData) SingleEntityData {
	days := int(number)

	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				// If the series is too short, we can't compute anything
				if len(d) < 2 || d[len(d)-1].Time.Sub(d[0].Time).Hours()/24 < float64(days) {
					return []DataPoint{}
				}

				// Determine the first start and end dates within the period
				startDate := d[0].Time
				var i int
				for i = 1; i < len(d); i++ {
					if d[i].Time.Sub(startDate).Hours()/24 >= float64(days) {
						break
					}
				}

				// Create new data to return
				newData := make([]DataPoint, len(d)-i)

				for j := i; j < len(d); j++ {
					newData[j-i] = DataPoint{
						Time: d[j].Time,
						Data: stdDev(getDataBetween(d[j].Time.AddDate(0, 0, -days), d[j].Time, d)),
					}
				}

				return newData
			},
			func(m SeriesMeta) SeriesMeta {
				m.Label = fmt.Sprintf("%v-Day Standard Deviation of %s", days, m.Label)

				return m
			})

		return s
	}
}

func percentageChange(d []DataPoint) float64 {
	if len(d) < 2 {
		return 0
	}

	return (d[len(d)-1].Data / d[0].Data) - 1
}

func stdDev(d []DataPoint) float64 {
	if len(d) < 2 {
		return 0
	}

	var sum float64 = 0
	var average float64 = 0
	var dayDistanceSum float64 = 0
	var dayDistanceAverage float64 = 0
	var annualizationFactor float64

	for i, v := range d {
		sum = sum + v.Data
		if i > 0 {
			// This is here to help determine the average distance between the
			// time periods that we're averaging for proper annualization
			dayDistanceSum = dayDistanceSum + (v.Time.Sub(d[i].Time).Hours() / 24)
		}
	}
	average = sum / float64(len(d))
	dayDistanceAverage = math.Floor(dayDistanceSum / float64(len(d)))
	if dayDistanceAverage < 2 {
		annualizationFactor = math.Sqrt(260)
	} else if dayDistanceAverage < 15 {
		annualizationFactor = math.Sqrt(52 / 2)
	} else if dayDistanceAverage < 35 {
		annualizationFactor = math.Sqrt(12)
	} else {
		annualizationFactor = 1
	}

	var sse float64 = 0

	for _, v := range d {
		sse = sse + ((v.Data - average) * (v.Data - average))
	}

	return annualizationFactor * math.Sqrt(sse/float64(len(d)-1))
}

//func tsGreater(number float64) func(SingleEntityData) SingleEntityData {
//	return func(s SingleEntityData) SingleEntityData {
//		s.Data = applyToSeries(s.Data,
//			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
//				for i := len(d) - 1; i >= lagPeriods; i-- {
//					d[i].Data = d[i-lagPeriods].Data
//				}
//				return d[lagPeriods:]
//			},
//			func(m SeriesMeta) SeriesMeta {
//				m.Label = fmt.Sprintf("%s > %v", m.Label, number)
//				return m
//			})

//		return s
//	}
//}

func tsLag(number float64) func(SingleEntityData) SingleEntityData {
	lagPeriods := int(number)

	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				if len(d) < lagPeriods || lagPeriods < 0 {
					return []DataPoint{}
				}

				for i := len(d) - 1; i >= lagPeriods; i-- {
					d[i].Data = d[i-lagPeriods].Data
				}
				return d[lagPeriods:]
			},
			func(m SeriesMeta) SeriesMeta {
				if lagPeriods > 1 {
					m.Label = fmt.Sprintf("%s lagged by %v periods", m.Label, lagPeriods)
				} else {
					m.Label = fmt.Sprintf("%s lagged by %v period", m.Label, lagPeriods)
				}
				return m
			})

		return s
	}
}

func tsDaysSinceDrop(number float64) func(SingleEntityData) SingleEntityData {
	return func(s SingleEntityData) SingleEntityData {
		s.Data = applyToSeries(s.Data,
			func(m SeriesMeta, d []DataPoint, w []DataPoint) []DataPoint {
				if len(d) == 0 {
					return d
				}
				var lastEncountered time.Time = d[0].Time

				for i, _ := range d {
					if d[i].Data < number {
						lastEncountered = d[i].Time
					}
					d[i].Data = math.Floor(0.49 + (d[i].Time.Sub(lastEncountered).Hours() / 24))
				}
				return d
			},
			func(m SeriesMeta) SeriesMeta {
				m.IsTransformed = true
				m.Label = fmt.Sprintf("Days since %s was less than %v", m.Label, number)
				m.Units = "Days"

				return m
			})

		return s
	}
}

// []Series functions
func applyToWeights(s []Series, dataTransform func(SeriesMeta, []DataPoint, []DataPoint) []DataPoint, metaTransform func(SeriesMeta) SeriesMeta) []Series {
	weightSeries := getWeightSeries(s)

	for i, _ := range s {
		if s[i].IsWeight {
			s[i].Data = dataTransform(s[i].Meta, s[i].Data, weightSeries.Data)
			s[i].Meta = metaTransform(s[i].Meta)
		}
	}

	return s
}

func applyToSeries(s []Series, dataTransform func(SeriesMeta, []DataPoint, []DataPoint) []DataPoint, metaTransform func(SeriesMeta) SeriesMeta) []Series {
	weightSeries := getWeightSeries(s)

	for i, _ := range s {
		if !s[i].IsWeight {
			s[i].Data = dataTransform(s[i].Meta, s[i].Data, weightSeries.Data)
			s[i].Meta = metaTransform(s[i].Meta)
		}
	}

	return s
}

// Metadata Transformation
func metaTransform(labelFunc func(string) string, unitsFunc func(string) string, upsampleFunc func(ResampleType) ResampleType, downsampleFunc func(ResampleType) ResampleType) func(SeriesMeta) SeriesMeta {
	return func(m SeriesMeta) SeriesMeta {
		return SeriesMeta{m.VendorCode, labelFunc(m.Label), unitsFunc(m.Units), m.Source, upsampleFunc(m.Upsample), downsampleFunc(m.Downsample), true}
	}
}

func noResampleChange(rs ResampleType) ResampleType {
	return rs
}

func changeResampleType(rs ResampleType) func(ResampleType) ResampleType {
	return func(blah ResampleType) ResampleType {
		return rs
	}
}

func prependString(s string) func(string) string {
	return func(s2 string) string {
		return s + " " + s2
	}
}

func appendString(s string) func(string) string {
	return func(s2 string) string {
		return s2 + " " + s
	}
}

func replaceString(s string) func(string) string {
	return func(s2 string) string {
		return s
	}
}

func noStringChange(s string) string {
	return s
}

// Universe functions
func dummyUniverse(s string) []EntityMeta {
	m := make([]EntityMeta, 3)

	m[0].Name = "Apple Inc."
	m[0].UniqueId = "WIKI/AAPL"
	m[1].Name = "Google Inc."
	m[1].UniqueId = "WIKI/GOOGL"
	m[2].Name = "Microsoft Corp"
	m[2].UniqueId = "WIKI/MSFT"

	return m
}

func getUniverse(ctx context.Context, c component.QueryComponent) []EntityMeta {
	// TODO: Make this time-varying

	var newEntityMeta []EntityMeta

	var tempUniverse []component.QueryComponent

	if c.QueryComponentType == "Universe: Expandable" {
		// TODO: For now send every date. Eventually, this should take into account the rebalance points
		if c.QueryComponentCanonicalName == "S&P 500 Stocks" {
			tempUniverse, _ = data.GetSP500Constituents()
		} else if c.QueryComponentCanonicalName == "iShares Popular ETFs" {
			tempUniverse, _ = data.GetISharesPopular()
		} else if c.QueryComponentCanonicalName == "Sector ETFs" {
			tempUniverse, _ = data.GetSectorETFs()
		} else if c.QueryComponentCanonicalName == "Currencies" {
			tempUniverse, _ = data.GetCurrencies(ctx)
		} else if c.QueryComponentCanonicalName == "Commodities" {
			tempUniverse, _ = data.GetCommodities(ctx)
			// } else if c.QueryComponentSource == component.Custom {
			// 	return customUniverse(c.QueryComponentProviderId)
		} else {
			tempUniverse, _ = data.GetSP500SectorConstituents(c.QueryComponentCanonicalName, list.Sector)
			if len(tempUniverse) == 0 {
				tempUniverse, _ = data.GetSP500SectorConstituents(c.QueryComponentCanonicalName, list.IndustryGroup)
			}
			if len(tempUniverse) == 0 {
				tempUniverse, _ = data.GetSP500SectorConstituents(c.QueryComponentCanonicalName, list.Industry)
			}
			if len(tempUniverse) == 0 {
				tempUniverse, _ = data.GetSP500SectorConstituents(c.QueryComponentCanonicalName, list.SubIndustry)
			}
		}
	} else {
		// For everything else, just use the individual component
		tempUniverse = []component.QueryComponent{c}
	}

	newEntityMeta = convertComponentToEntityMeta(tempUniverse)

	return newEntityMeta
}

func convertComponentToEntityMeta(universe []component.QueryComponent) []EntityMeta {
	var entityMeta []EntityMeta = make([]EntityMeta, len(universe), len(universe))

	for i, v := range universe {
		entityMeta[i].Name = v.QueryComponentCanonicalName
		entityMeta[i].UniqueId = v.QueryComponentProviderId
		if v.QueryComponentType == component.CustomQuandlCode {
			entityMeta[i].IsCustom = true
		}
	}

	return entityMeta
}

// Data functions
func generateDays(dateStart time.Time, numDays int) []time.Time {
	var newDates = make([]time.Time, numDays+1, numDays+1)

	for i := 0; i <= numDays; i++ {
		newDates[i] = dateStart.AddDate(0, 0, i)
	}

	return newDates
}

//func dummyData(s string) func(EntityMeta) Series {
//	return func(e EntityMeta) Series {
//		sampleDate, _ := time.Parse("2006-01-02", "2014-12-31")

//		sampleDays1 := generateDays(sampleDate, 9)
//		priceSeries1 := []float64{1, 1.5, 2.0, 1.5, 2.0, 2.5, 2.3, 2.5, 2.2, 2.3}

//		data1 := createSeriesArray(sampleDays1, priceSeries1)

//		series1 := Series{data1, SeriesMeta{"WIKI/AAPL", "Price", "$", "Quandl", ResampleLastValue, ResampleLastValue, false}, false}

//		return series1
//	}
//}

func TimeSlicer(c []component.QueryComponent) StepFnType {
	if len(c) > 0 {
		timeRange := c[0]

		return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
			if len(mArr) > 0 {
				m := mArr[0]

				startDate, endDate := extractStartEndDate(timeRange)
				lastDataPointOnly := timeRange.QueryComponentCanonicalName == "Last Data Point"
				allAvailable := timeRange.QueryComponentCanonicalName == "All Available"

				for i, v := range m.EntityData {
					for i2, v2 := range v.Data {
						if lastDataPointOnly && m.EntityData[i].Data[i2].Len() > 0 {
							m.EntityData[i].Data[i2].Data = []DataPoint{m.EntityData[i].Data[i2].Data[m.EntityData[i].Data[i2].Len()-1]}
						} else if !allAvailable {
							m.EntityData[i].Data[i2].Data = getDataBetween(startDate, endDate, v2.Data)
						}
					}
				}

				return m
			}

			return MultiEntityData{Error: "No data to compute"}
		}
	}

	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		return MultiEntityData{Error: "No time range specified"}
	}
}

func getData(dataField component.QueryComponent, timeRange component.QueryComponent) func(context.Context, EntityMeta) Series {
	return func(ctx context.Context, e EntityMeta) Series {
		var s Series

		startDate, endDate := extractStartEndDate(timeRange)
		lastDataPointOnly := timeRange.QueryComponentCanonicalName == "Last Data Point"
		allAvailable := timeRange.QueryComponentCanonicalName == "All Available"

		// Handle custom macro data
		// if strings.Contains(e.Name, "Custom") {
		// 	return customMacroData(dataField.QueryComponentProviderId, e, startDate, endDate, lastDataPointOnly, allAvailable)
		// } else if dataField.QueryComponentSource == component.Custom {
		// 	return customStockData(dataField.QueryComponentProviderId, e, startDate, endDate, lastDataPointOnly, allAvailable)
		// }

		universeMember := component.QueryComponent{QueryComponentCanonicalName: e.Name, QueryComponentName: e.Name, QueryComponentType: "Universe", QueryComponentProviderId: e.UniqueId, QueryComponentSource: "Quandl Open Data", QueryComponentOriginalString: e.Name}

		ts, _ := data.GetDataFull(ctx, universeMember, dataField)

		s = convertTsToSeries(ts, e.IsCustom, dataField.QueryComponentOriginalString, lastDataPointOnly, allAvailable, startDate, endDate)

		return s
	}
}

func convertTsToSeries(ts *timeseries.TimeSeries, isCustom bool, fieldName string, lastDataPointOnly bool, allAvailableData bool, startDate time.Time, endDate time.Time) Series {
	s := Series{}

	if ts == nil {
		s.Data = make([]DataPoint, 0)
		return s
	}

	s.Data = make([]DataPoint, ts.Len())

	for i, _ := range ts.Data {
		s.Data[i] = DataPoint{Time: ts.Date[i].UTC(), Data: ts.Data[i]}
	}

	if lastDataPointOnly {
		// If we only want the last data point
		if len(s.Data) > 0 {
			s.Data = []DataPoint{s.Data[len(s.Data)-1]}
		} else {
			s.Data = []DataPoint{}
		}
	} else if allAvailableData {
		// Do nothing (i.e. s.Data = s.Data)
	} else {
		// Chop the series down to only the date range provided
		s.Data = getDataBetweenInclusive(startDate, endDate, s.Data)
	}

	s.Meta.Downsample = ResampleLastValue
	s.Meta.Upsample = ResampleLastValue
	s.Meta.IsTransformed = false
	s.Meta.Units = ts.Units
	s.Meta.VendorCode = ts.Name
	if isCustom {
		// If this is a custom quandl code, we'll just grab the name
		// from the quandl response. We don't ordinarily do this because
		// the name for a series of WIKI/AAPL is "Apple Inc."
		// That's not the name of the series, it's normally the name of the
		// universe member. However, since we don't actually have the proper
		// name of the universe member when users input just the custom
		// Quandl code, we're extracting the name and sticking it here.
		s.Meta.Label = ts.DisplayName
	} else {
		s.Meta.Label = fieldName
	}
	s.Meta.Source = ts.Source
	return s
}

// Functions that operate on MultiEntityData
func ExcludeData(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	if len(mArr) == 0 {
		return MultiEntityData{}
	} else if len(mArr) == 1 {
		return MultiEntityData{}
	}

	m := MultiEntityData{}
	m.EntityData = make([]SingleEntityData, 0)

	m.Title = mArr[0].Title + " excluding " + mArr[1].Title
	m.Error = mArr[0].Error + ", " + mArr[1].Error

	for _, v := range mArr[0].EntityData {
		ix := indexOfEntity(v.Meta, mArr[1])

		//marshalOutput("ix", ix)
		if ix < 0 {
			// Entity doesn't exist, we add
			m.EntityData = append(m.EntityData, v)
			//marshalOutput("entity doesn't exist. Adding", m.EntityData)
		} else {
			// Entity exists, add it to the intersection
			m.EntityData = append(m.EntityData, mArr[1].EntityData[ix])
			//marshalOutput("entity exists. Adding", m.EntityData)

			for _, v2 := range v.Data {
				if !v2.IsWeight {
					// This entity already exists. We add each series that isn't a weight
					m.EntityData[len(m.EntityData)-1].Data = append(m.EntityData[len(m.EntityData)-1].Data, v2)
				}
			}

			m.EntityData[len(m.EntityData)-1] = excludeWeights(m.EntityData[len(m.EntityData)-1], v)

		}
	}

	return m.RemoveSuperflousDataAndWeights(true)
}

func IntersectData(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	if len(mArr) == 0 {
		return MultiEntityData{}
	} else if len(mArr) == 1 {
		return MultiEntityData{}
	}

	m := MultiEntityData{}
	m.EntityData = make([]SingleEntityData, 0)

	m.Title = "Intersection of " + mArr[0].Title + " and " + mArr[1].Title
	m.Error = mArr[0].Error + ", " + mArr[1].Error

	for _, v := range mArr[1].EntityData {
		ix := indexOfEntity(v.Meta, mArr[0])

		if ix < 0 {
			// Entity doesn't exist, we don't add
		} else {
			// Entity exists, add it to the intersection
			m.EntityData = append(m.EntityData, mArr[0].EntityData[ix])

			for _, v2 := range v.Data {
				if !v2.IsWeight {
					// This entity already exists. We add each series that isn't a weight
					m.EntityData[len(m.EntityData)-1].Data = append(m.EntityData[len(m.EntityData)-1].Data, v2)
				}
			}

			m.EntityData[len(m.EntityData)-1] = intersectWeights(m.EntityData[len(m.EntityData)-1], v)

		}
	}

	return m
}

func DifferenceData(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	if len(mArr) == 0 {
		return MultiEntityData{}
	} else if len(mArr) == 1 || len(mArr[1].EntityData) == 0 || len(mArr[1].EntityData[0].Data) == 0 {
		return mArr[0]
	}

	indexData := mArr[1].EntityData[0].Data[0]

	for j, s := range mArr[0].EntityData {
		for i, _ := range s.Data {
			if s.Data[i].IsWeight == true {
				continue
			}

			s.Data[i] = runBinaryFunctionOnSeries(s.Data[0], indexData, getWeightSeries(s.Data),
				func(a float64, b float64) float64 {
					return a - b
				},
				metaTransformBinary(
					func(s1, s2 string) string { return "Spread between " + s1 + " and " + s2 },
					func(s1, s2 string) string {
						if s1 == s2 {
							return s1
						}
						return "Spread between " + s1 + " and " + s2
					},
				),
			)
		}

		mArr[0].EntityData[j].Data = s.Data
	}

	return mArr[0]
}

func UnionData(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	if len(mArr) == 0 {
		return MultiEntityData{}
	} else if len(mArr) == 1 {
		return mArr[0]
	}

	m := mArr[0]

	m.Title = mArr[0].Title + ", " + mArr[1].Title
	m.Error = mArr[0].Error + ", " + mArr[1].Error

	for _, v := range mArr[1].EntityData {
		ix := indexOfEntity(v.Meta, m)

		if ix < 0 {
			m.EntityData = append(m.EntityData, v)
		} else {
			for _, v2 := range v.Data {
				if !v2.IsWeight {
					// This entity already exists. We add each series that isn't a weight
					m.EntityData[ix].Data = append(m.EntityData[ix].Data, v2)
				}
			}
			m.EntityData[ix] = unionWeights(m.EntityData[ix], v)
		}
	}

	return m
}

func findWeightIndices(e1 SingleEntityData, e2 SingleEntityData) (int, int) {
	var w1Ix int = -1
	var w2Ix int = -1

	for i, v := range e1.Data {
		if v.IsWeight {
			w1Ix = i
		}
	}

	for i, v := range e2.Data {
		if v.IsWeight {
			w2Ix = i
		}
	}

	return w1Ix, w2Ix
}

func excludeWeights(e1 SingleEntityData, e2 SingleEntityData) SingleEntityData {
	//marshalOutput("e1", e1)
	//marshalOutput("e2", e2)

	w1Ix, w2Ix := findWeightIndices(e1, e2)

	if w1Ix < 0 || w2Ix < 0 {
		if e1.Meta.UniqueId == e2.Meta.UniqueId {
			s := SingleEntityData{}
			s.Data = make([]Series, 0)
			return s
		}
		return e1
	}

	w1 := e1.Data[w1Ix]
	w2 := e2.Data[w2Ix]

	e1.Data[w1Ix] = runBinaryFunctionOnSeries(
		w1,
		w2,
		weightStub(zeroDay()),
		func(a float64, b float64) float64 {
			if a != 0 && b == 0 {
				return a
			}
			return 0
		},
		func(a SeriesMeta, b SeriesMeta) SeriesMeta {
			return a
		},
	)

	return e1
}

func intersectWeights(e1 SingleEntityData, e2 SingleEntityData) SingleEntityData {
	w1Ix, w2Ix := findWeightIndices(e1, e2)

	if w1Ix < 0 || w2Ix < 0 {
		return e1
	}

	w1 := e1.Data[w1Ix]
	w2 := e2.Data[w2Ix]

	e1.Data[w1Ix] = runBinaryFunctionOnSeries(
		w1,
		w2,
		weightStub(zeroDay()),
		func(a float64, b float64) float64 {
			if a != 0 && b != 0 {
				return a
			}
			return 0
		},
		func(a SeriesMeta, b SeriesMeta) SeriesMeta {
			return a
		},
	)

	return e1
}

func unionWeights(e1 SingleEntityData, e2 SingleEntityData) SingleEntityData {
	w1Ix, w2Ix := findWeightIndices(e1, e2)

	if w1Ix < 0 || w2Ix < 0 {
		return e1
	}

	e1.Data[w1Ix] = intersperseWeights(e1.Data[w1Ix], e2.Data[w2Ix])

	return e1
}

func intersperseWeights(w1 Series, w2 Series) Series {
	return runBinaryFunctionOnSeries(
		w1,
		w2,
		weightStub(zeroDay()),
		func(a float64, b float64) float64 {
			if a != 0 {
				return a
			}
			if b != 0 {
				return b
			}
			return 0
		},
		func(a SeriesMeta, b SeriesMeta) SeriesMeta {
			return a
		},
	)
}

func indexOfEntity(e EntityMeta, m MultiEntityData) int {
	for i, v := range m.EntityData {
		if e.UniqueId == v.Meta.UniqueId {
			return i
		}
	}

	return -1
}

func Alpha(numMonths float64) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		dependent := mArr[0]
		independent := mArr[1]

		var m MultiEntityData
		m.EntityData = make([]SingleEntityData, 0)

		for i, v := range dependent.EntityData {
			m.EntityData = append(m.EntityData, SingleEntityData{Meta: v.Meta})
			m.EntityData[i].Data = make([]Series, 0)

			for _, v2 := range v.Data {
				s := rollingRegression(v2, independent, int(numMonths))

				m.EntityData[i].Data = append(m.EntityData[i].Data, s)

			}
		}

		return m
	}
}

func Regression(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	dependent := mArr[0]
	independent := mArr[1]

	var m MultiEntityData

	//fmt.Printf("Regression start\n")

	for _, v := range dependent.EntityData {
		for _, v2 := range v.Data {
			r := regressSeries(v2, independent)

			//fmt.Printf("r=%s\n", r)
			if containsInfOrNan(r) {
				continue
			}

			newEntityMeta := EntityMeta{
				UniqueId: v.Meta.UniqueId + " " + v2.Meta.Label,
				Name:     v.Meta.Name + " " + v2.Meta.Label,
			}

			//m = m.Insert(
			//	newEntityMeta,
			//	SeriesMeta{
			//		Label: "Alpha",
			//	},
			//	DataPoint{zeroDay(), r.GetRegCoeff(0)},
			//	CategoryLabel{},
			//)

			for j := 1; j < len(r.RegCoeff); j++ {
				m = m.Insert(
					newEntityMeta,
					SeriesMeta{
						Label:         "Beta to " + r.GetVarName(j-1),
						Units:         "Beta",
						Source:        strings.Join(m.GetSources(), ", "),
						Upsample:      ResampleLastValue,
						Downsample:    ResampleLastValue,
						IsTransformed: true,
					},
					DataPoint{independent.LastDay(), r.GetRegCoeff(j)},
					CategoryLabel{},
					false,
				)
			}

			m = m.Insert(
				newEntityMeta,
				SeriesMeta{
					Label:         "R Squared",
					Units:         "R²",
					Source:        strings.Join(m.GetSources(), ", "),
					Upsample:      ResampleLastValue,
					Downsample:    ResampleLastValue,
					IsTransformed: true,
				},
				DataPoint{independent.LastDay(), r.Rsquared},
				CategoryLabel{},
				false,
			)

		}
	}

	//fmt.Printf("Regression complete\n")

	return m
}

func containsInfOrNan(r regression.Regression) bool {
	for _, v := range r.RegCoeff {
		if math.IsNaN(v) || math.IsInf(v, -1) || math.IsInf(v, 1) {
			return true
		}
	}

	return false
}

func rollingRegression(s Series, independent MultiEntityData, numMonths int) Series {
	var r regression.Regression
	var alpha Series

	datesForResample := getDates(s.Data)

	independent = alignDataPoints(datesForResample, independent.Duplicate())

	r.SetObservedName(s.Meta.Label)

	numIndependent := independent.NumSeries()

	var varNames []string
	var temp []string
	var variables []float64
	var v DataPoint
	var i int

	// Add numMonths worth of data
	for i, v = range s.Data {
		monthsAhead := s.Data[0].Time.AddDate(0, numMonths, 0)
		if monthsAhead.Equal(v.Time) || monthsAhead.Before(v.Time) {
			// We now have enough data to compute the beta
			break
		}
		variables, temp = getDataArrayExact(v.Time, independent)

		if len(variables) == numIndependent {
			varNames = temp
			r.AddDataPoint(regression.DataPoint{Observed: v.Data, Variables: variables})
		}
	}

	for i, v := range varNames {
		r.SetVarName(i, v)
	}

	alpha.Data = make([]DataPoint, 0)

	r.RunLinearRegression()

	//alpha.Data = append(alpha.Data, DataPoint{v.Time, r.GetRegCoeff(0)})

	// Add additional data points, but remove the first data point
	for j := i; j < len(s.Data); j++ {
		v = s.Data[j]
		variables, temp = getDataArrayExact(v.Time, independent)

		if len(variables) == numIndependent {
			// We should compute an alpha based on the previous betas
			alpha.Data = append(alpha.Data, DataPoint{v.Time, computeAlpha(v.Data, variables, r)})

			varNames = temp
			r.AddDataPoint(regression.DataPoint{Observed: v.Data, Variables: variables})
			r.Data = r.Data[1:]

			if numIndependent == 0 {
				fmt.Printf("No independent variables right now!!")
				continue
			}
			//fmt.Printf("Before=%s\n", r.RegCoeff)

			if !r.Initialised {
				fmt.Printf("Not initialized!!!\n")
				continue
			}

			if len(r.Data) == 0 {
				fmt.Printf("No data\n")
				//fmt.Printf("No data!!j=%v, i=%v, variables=%s, v=%s, len(alpha.Data)=%v, len(r.Data)=%v\n", j, i, variables, v, len(alpha.Data), len(r.Data))
				//f, _ := os.Create("/Users/zain/data.txt")
				//w := bufio.NewWriter(f)
				//m, _ := json.MarshalIndent(s, "", "\t")
				//w.Write(m)
				//m, _ = json.MarshalIndent(independent, "", "\t")
				//w.Write(m)
				//m, _ = json.Marshal(numPeriods)
				//w.Write(m)

				//w.Flush()
				//os.Exit(0)
				continue
			}

			r.RunLinearRegression()
			//fmt.Printf("After =%s\n", r.RegCoeff)
		}
	}

	// Metadata
	alpha.IsWeight = false
	alpha.Meta = SeriesMeta{
		Label:         "Alpha",
		Units:         "%",
		Source:        s.Meta.Source,
		Upsample:      ResampleZero,
		Downsample:    ResampleArithmetic,
		IsTransformed: true,
	}

	return alpha
}

func computeAlpha(ret float64, variables []float64, r regression.Regression) float64 {
	var alpha float64 = ret

	alpha = alpha - r.GetRegCoeff(0)

	for i, _ := range variables {
		//fmt.Printf("i=%v, r.GetRegCoeff(i+1)=%v, variables[i]=%v, ret=%v\n", i, r.GetRegCoeff(i+1), variables[i], ret)
		//fmt.Printf("Formula=%s\n", r.Formula)
		//r.Dump(false)
		alpha = alpha - (r.GetRegCoeff(i+1) * variables[i])
	}

	return alpha
}

func regressSeries(s Series, independent MultiEntityData) regression.Regression {
	var r regression.Regression

	datesForResample := getDates(s.Data)

	//fmt.Printf("datesForResample=%s\n", datesForResample)

	independent = alignDataPoints(datesForResample, independent)

	r.SetObservedName(s.Meta.Label)

	numIndependent := independent.NumSeries()

	var varNames []string
	var temp []string
	var variables []float64

	for _, v := range s.Data {
		variables, temp = getDataArrayExact(v.Time, independent)

		if len(variables) == numIndependent {
			varNames = temp
			r.AddDataPoint(regression.DataPoint{Observed: v.Data, Variables: variables})
		}
	}

	//fmt.Printf("varNames final=%s\n", varNames)

	for i, v := range varNames {
		r.SetVarName(i, v)
	}

	r.RunLinearRegression()

	//rd.Output.Alpha = r.GetRegCoeff(0)

	//rd.Output.Coefficients = make([]float64, 0)

	//for i := 0; i < len(rd.IndependentData); i++ {
	//	rd.Output.Coefficients = append(rd.Output.Coefficients, r.GetRegCoeff(i+1))
	//}

	return r
}

func getDataArrayExact(date time.Time, m MultiEntityData) ([]float64, []string) {
	data := make([]float64, 0)
	varNames := make([]string, 0)

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			for _, v3 := range v2.Data {
				if v3.Time.Equal(date) {
					data = append(data, v3.Data)
					varNames = append(varNames, v.Meta.Name+" "+v2.Meta.Label)
				}
			}
		}
	}

	//fmt.Printf("date = %s. data = %v\n", date, data)

	return data, varNames
}

func alignDataPoints(dates []time.Time, m MultiEntityData) MultiEntityData {
	resampleFn := resampleOnDates(dates, false, false)

	for i, v := range m.EntityData {
		m.EntityData[i].Data = applyToSeries(v.Data, resampleFn, metaTransform(noStringChange, noStringChange, noResampleChange, noResampleChange))
	}

	return m
}

func GetUniverseData(fn func(context.Context, EntityMeta) Series) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		for i, v := range m.EntityData {
			newData := fn(ctx, v.Meta)

			if m.EntityData[i].Meta.IsCustom {
				m.EntityData[i].Meta.Name = newData.Meta.Label
				newData.Meta.Label = "Value"
			}
			m.EntityData[i].Data = append(m.EntityData[i].Data, newData)

		}

		return m
	}
}

func SetWeights(fn func(context.Context, EntityMeta) Series) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		for i, v := range m.EntityData {
			newData := fn(ctx, v.Meta)
			newData.IsWeight = true

			m.EntityData[i].Data = append(m.EntityData[i].Data, newData)
		}

		return m
	}
}

func SetWeightsAndCategory(fn func(EntityMeta) (Series, CategorySeries)) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		for i, v := range m.EntityData {
			newData, newCategory := fn(v.Meta)
			newData.IsWeight = true

			m.EntityData[i].Data = append(m.EntityData[i].Data, newData)
			m.EntityData[i].Category = newCategory
		}

		return m
	}
}

func GetBulkData(c []component.QueryComponent) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		queryName := c[0]
		timeRange := c[1]

		var paramMap map[string][]string = make(map[string][]string)

		if len(c) > 2 {
			for i := 2; i < len(c); i++ {
				paramMap[c[i].GetLabel()] = c[i].QueryComponentParams
			}
		}

		startDate, endDate := extractStartEndDate(timeRange)
		// lastDataPointOnly := timeRange.QueryComponentCanonicalName == "Last Data Point"
		// allAvailable := timeRange.QueryComponentCanonicalName == "All Available"

		return GetSQLData(ctx, m.GetEntities(), queryName.GetLabel(), paramMap, startDate, endDate)
	}
}

func WrapUniverseData(majorFn func(fn func(context.Context, EntityMeta) Series) StepFnType, fn func(component.QueryComponent, component.QueryComponent) func(context.Context, EntityMeta) Series) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		dataField := c[0]

		dataFn := fn(dataField, c[1])

		return majorFn(dataFn)
	}
}

func WrapEvent(majorFn func(fn func(EntityMeta) (Series, CategorySeries)) StepFnType, fn func(component.QueryComponent) func(EntityMeta) (Series, CategorySeries)) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		eventName := c[0]

		dataFn := fn(eventName)

		return majorFn(dataFn)
	}
}

func extractStartEndDate(c component.QueryComponent) (time.Time, time.Time) {
	//m, _ := json.Marshal(c)
	//fmt.Printf("c=%s\n", m)

	c = build.SetTimeRangeParameters(c)

	if len(c.QueryComponentParams) == 2 {
		startDate, _ := time.Parse("2006-01-02", c.QueryComponentParams[0])

		endDate, _ := time.Parse("2006-01-02", c.QueryComponentParams[1])

		return startDate, endDate
	}

	return time.Time{}, time.Time{}
}

func GetUniverse(c component.QueryComponent, fn func(context.Context, component.QueryComponent) []EntityMeta) StepFnType {
	return func(ctx context.Context, _ []MultiEntityData) MultiEntityData {
		m := MultiEntityData{}

		m.Title = c.QueryComponentCanonicalName
		e := fn(ctx, c)
		m.EntityData = make([]SingleEntityData, len(e))

		for i, v := range e {
			m.EntityData[i].Meta = v
		}

		return m
	}
}

func ComposeStepFn(fn1 StepFnType, fn2 StepFnType) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		return fn2(ctx, []MultiEntityData{fn1(ctx, mArr)})
	}
}

func GraphicalPreference(s string) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		m.GraphicalPreference = s

		return m
	}
}

func WrapUniverseArgument(fn func(context.Context, component.QueryComponent) []EntityMeta) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		return GetUniverse(c[0], fn)
	}
}

func WrapPortfolioArgument(membersFn func(context.Context, component.QueryComponent) []EntityMeta, weightsFn func(component.QueryComponent) func(context.Context, EntityMeta) Series) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		return ComposeStepFn(
			GetUniverse(c[0], membersFn),
			SetWeights(weightsFn(c[0])),
		)
	}
}

func WrapLatestData() StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		lastDay := m.LastDay()

		resampleFn := resampleOnDates([]time.Time{lastDay}, false, false)

		// Run the resampling on all the series

		for i, _ := range m.EntityData {
			m.EntityData[i].Data = resampleSeriesArray(m.EntityData[i].Data, resampleFn)
		}

		m = m.RemoveSuperflousDataAndWeights(false)

		return m
	}
}

func AlignLastDay() StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		lastDay := m.LastDay()

		// Run the resampling on all the series

		for i, _ := range m.EntityData {
			for j, _ := range m.EntityData[i].Data {
				length := len(m.EntityData[i].Data[j].Data)
				if length > 0 {
					m.EntityData[i].Data[j].Data[length-1].Time = lastDay
				}
			}

		}

		return m
	}
}

func AlignFirstDay() StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		lastDay := m.LastDay()

		var startDate time.Time

		// Find the series that has the latest starting point
		for i, _ := range m.EntityData {
			for j, _ := range m.EntityData[i].Data {
				length := len(m.EntityData[i].Data[j].Data)
				if length > 0 {
					temp := m.EntityData[i].Data[j].Data[0].Time

					if temp.After(startDate) {
						startDate = temp
					}
				}
			}

		}

		// Run the resampling on all the series

		for i, _ := range m.EntityData {
			for j, _ := range m.EntityData[i].Data {
				m.EntityData[i].Data[j].Data = getDataBetweenInclusive(startDate, lastDay, m.EntityData[i].Data[j].Data)
			}
		}

		return m
	}
}

func WeightsToMacro(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	var m MultiEntityData
	if len(mArr) > 0 {
		m = mArr[0]
	}

	var w Series

	for _, v := range m.EntityData {
		for _, v2 := range v.Data {
			if v2.IsWeight {
				w = v2
				break
			}
		}
	}

	for i, _ := range m.EntityData {
		var hasWeight bool = false
		for j, _ := range m.EntityData[i].Data {

			if m.EntityData[i].Data[j].IsWeight {
				m.EntityData[i].Data[j] = w
				hasWeight = true
			}
		}
		if !hasWeight {
			m.EntityData[i].Data = append(m.EntityData[i].Data, w)
		}
	}

	return m
}

func AlignCalendar(tsFunc func(time.Time) func(SingleEntityData) SingleEntityData) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		lastDay := m.LastDay()

		alignFn := tsFunc(lastDay)

		for i, v := range m.EntityData {
			m.EntityData[i] = alignFn(v)
		}

		return m
	}
}

func ComputeTS(tsFunc func(SingleEntityData) SingleEntityData) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		for i, v := range m.EntityData {
			m.EntityData[i] = tsFunc(v)
		}

		return m
	}
}

func flatten(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	m := mArr[0]

	m2 := m.Duplicate()
	m2.EntityData = make([]SingleEntityData, 1)

	for i, _ := range m.EntityData {
		for j, _ := range m.EntityData[i].Data {
			if m.EntityData[i].Meta.Name != "" {
				m.EntityData[i].Data[j].Meta.Label = m.EntityData[i].Meta.Name + ": " + m.EntityData[i].Data[j].Meta.Label
			}
		}
		m2.EntityData[0].Data = append(m2.EntityData[0].Data, m.EntityData[i].Data...)
	}

	return m2
}

func ComputeTSMulti(tsFunc func(SingleEntityData) []SingleEntityData) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]
		temp := make([]SingleEntityData, 0)

		for _, v := range m.EntityData {
			temp = append(temp, tsFunc(v)...)
		}

		m.EntityData = temp

		return m
	}
}

func greaterXaggregator(numX float64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {

		for i, _ := range d.Data {
			if d.Data[i].Data.Data <= numX {
				d.Data[i].Weight.Data = 0
			}
		}

		return d
	}
}

func lessXaggregator(numX float64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {

		for i, _ := range d.Data {
			if d.Data[i].Data.Data >= numX {
				d.Data[i].Weight.Data = 0
			}
		}

		return d
	}
}

func equalXaggregator(numX float64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {

		for i, _ := range d.Data {
			if d.Data[i].Data.Data != numX {
				d.Data[i].Weight.Data = 0
			}
		}

		return d
	}
}

func topXaggregator(numX float64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {
		//fmt.Printf("topX d before sort = %s\n", d)

		sort.Sort(ByWeight{d})

		//fmt.Printf("topX d after sort = %s\n", d)

		d = d.Chop(int(numX))

		//fmt.Printf("topX d after chop = %s\n", d)

		return d
	}
}

func bottomXaggregator(numX float64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {
		// marshalOutput("bottomX d before sort", d)

		sort.Sort(sort.Reverse(ByWeight{d}))

		// marshalOutput("bottomX d after sort", d)

		d = d.Chop(int(numX))

		// marshalOutput("bottomX d after chop", d)

		return d
	}
}

func averageAggregator(category string, d DataForAggregation) DataForAggregation {
	var sum float64 = 0
	var sumWeights float64 = 0
	var t time.Time

	for _, v := range d.Data {
		sum = sum + (v.Data.Data * v.Weight.Data)
		sumWeights = sumWeights + v.Weight.Data
		t = v.Data.Time
	}

	var average float64

	if sumWeights > 0 {
		average = sum / sumWeights
	}
	// m, _ := json.Marshal(d)
	// fmt.Printf("In averageAggregator: %s\n", m)

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         d.SeriesM.Units,
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category,
					Name:     category,
				},
				Data: DataPoint{
					Data: average,
					Time: t,
				},
				Weight: DataPoint{
					Data: sumWeights,
					Time: t,
				},
			},
		},
	}
}

func getMaxMin(d DataForAggregation) (float64, float64) {
	var max float64 = math.Inf(-1)
	var min float64 = math.Inf(1)

	for _, v := range d.Data {
		if v.Data.Data > max {
			max = v.Data.Data
		}

		if v.Data.Data < min {
			min = v.Data.Data
		}
	}

	return max, min
}

//func getBinBounds(numBins int64, d DataForAggregation) ([]float64, []float64) {
//	var binLowerBound []float64 = make([]float64, numBins, numBins)
//	var binUpperBound []float64 = make([]float64, numBins, numBins)

//	// Get the max and min
//	max, min := getMaxMin(d)

//	for i, _ := range binLowerBound {
//		binLowerBound[i] = min + (float64(i) * (max - min) / float64(numBins))
//		binUpperBound[i] = min + (float64(i+1) * (max - min) / float64(numBins))
//	}

//	return binLowerBound, binUpperBound
//}

//func countBinData(d DataForAggregation, binLowerBound []float64, binUpperBound []float64) []float64 {
//	var binCount []float64 = make([]float64, len(binLowerBound))

//	for i, _ := range binCount {
//		for _, v := range d.Data {
//			if binLowerBound[i] < v.Data.Data && binUpperBound[i] <= v.Data.Data {
//				binCount[i]++
//			}
//		}
//	}

//	return binCount
//}

//func histogramAggregator(category string, d DataForAggregation) DataForAggregation {
//	var numBins int64 = int64(math.Sqrt(float64(d.Len())))

//	// Set the bin bounds
//	binLowerBound, binUpperBound := getBinBounds(numBins, d)

//	binCount := countBinData(d, binLowerBound, binUpperBound)

//	var data []EntityPlusDataPoint = make([]EntityPlusDataPoint, numBins, numBins)

//	for i, _ := range data {
//		data[i].Data.Data = binCount[i]
//		data[i].Data.Time = d.Data[0].Data.Time
//		data[i].Weight.Time = d.Data[0].Data.Time
//		data[i].Weight.Data = 1
//		data[i].EntityM = EntityMeta{
//			UniqueId: category + fmt.Sprintf("%v-%v", binLowerBound[i], binUpperBound[i]),
//			Name:     category + fmt.Sprintf("%v-%v", binLowerBound[i], binUpperBound[i]),
//		}
//	}

//	return DataForAggregation{
//		SeriesM: SeriesMeta{
//			Label:         d.SeriesM.Label,
//			Source:        d.SeriesM.Source,
//			Upsample:      d.SeriesM.Upsample,
//			Units:         d.SeriesM.Units,
//			Downsample:    d.SeriesM.Downsample,
//			IsTransformed: true,
//			VendorCode:    d.SeriesM.VendorCode,
//		},
//		Data: data,
//	}
//}

func convertDataForAggregationToArray(d DataForAggregation) []float64 {
	var arr []float64 = make([]float64, d.Len(), d.Len())

	for i, _ := range arr {
		arr[i] = d.Data[i].Weight.Data
	}

	sort.Float64s(arr)

	return arr
}

func quantileAggregator(numQuantiles int64) func(string, DataForAggregation) DataForAggregation {
	return func(category string, d DataForAggregation) DataForAggregation {

		boundaries := zstats.GetQuantileBoundaries(convertDataForAggregationToArray(d), numQuantiles)

		//fmt.Printf("boundaries=%v\n", boundaries)

		for i, v := range d.Data {
			quantileNum := zstats.GetQuantileNumber(v.Weight.Data, boundaries)
			quantileLabel := zstats.QuantileLabel(numQuantiles, quantileNum)
			d.Data[i].Category.Id = quantileNum
			d.Data[i].Category.Label = quantileLabel
		}

		//fmt.Printf("d=%v\n", d)

		return d
	}
}

func boxplotAggregator(category string, d DataForAggregation) DataForAggregation {
	max, min := getMaxMin(d)
	boundaries := zstats.GetQuantileBoundaries(convertDataForAggregationToArray(d), 4)
	t := d.Data[0].Data.Time

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         d.SeriesM.Units,
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category + " Boxplot Min of " + d.SeriesM.Label,
					Name:     category + " Boxplot Min of " + d.SeriesM.Label,
				},
				Data: DataPoint{
					Data: min,
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category + " Boxplot Q1 of " + d.SeriesM.Label,
					Name:     category + " Boxplot Q1 of " + d.SeriesM.Label,
				},
				Data: DataPoint{
					Data: boundaries[0],
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category + " Boxplot Q2 of " + d.SeriesM.Label,
					Name:     category + " Boxplot Q2 of " + d.SeriesM.Label,
				},
				Data: DataPoint{
					Data: boundaries[1],
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category + " Boxplot Q3 of " + d.SeriesM.Label,
					Name:     category + " Boxplot Q3 of " + d.SeriesM.Label,
				},
				Data: DataPoint{
					Data: boundaries[2],
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category + " Boxplot Max of " + d.SeriesM.Label,
					Name:     category + " Boxplot Max of " + d.SeriesM.Label,
				},
				Data: DataPoint{
					Data: max,
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
		},
	}
}

func medianAggregator(category string, d DataForAggregation) DataForAggregation {
	sort.Sort(d)
	var median float64
	var t time.Time

	if d.Len() >= 2 && d.Len()%2 == 0 {
		// Even, so take the average of the middle 2
		median = 0.5*d.Data[int(d.Len()/2)].Data.Data + 0.5*d.Data[int(d.Len()/2)-1].Data.Data
		t = d.Data[int(d.Len()/2)].Data.Time
	} else {
		idx := int((d.Len() - 1) / 2)
		if 0 <= idx && idx < d.Len() {
			median = d.Data[idx].Data.Data
			t = d.Data[idx].Data.Time
		}
	}

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         d.SeriesM.Units,
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category,
					Name:     category,
				},
				Data: DataPoint{
					Data: median,
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
		},
	}
}

func sumAggregator(category string, d DataForAggregation) DataForAggregation {
	var sum float64 = 0
	var sumWeights float64 = 0
	var t time.Time

	for _, v := range d.Data {
		sum = sum + v.Data.Data
		sumWeights = sumWeights + v.Weight.Data
		t = v.Data.Time
	}

	// m, _ := json.Marshal(d)
	// fmt.Printf("In sumAggregator: %s\n", m)

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         d.SeriesM.Units,
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category,
					Name:     category,
				},
				Data: DataPoint{
					Data: sum,
					Time: t,
				},
				Weight: DataPoint{
					Data: sumWeights,
					Time: t,
				},
			},
		},
	}
}

func percentAggregator(category string, d DataForAggregation) DataForAggregation {
	var positiveCount float64 = 0
	var totalCount float64 = 0
	var t time.Time

	for _, v := range d.Data {
		if v.Weight.Data != 0 {
			totalCount = totalCount + 1
			if v.Data.Data > 0 {
				positiveCount = positiveCount + 1
			}
		}
		t = v.Data.Time
	}

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         "%",
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category,
					Name:     category,
				},
				Data: DataPoint{
					Data: positiveCount / totalCount,
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
		},
	}
}

func countAggregator(category string, d DataForAggregation) DataForAggregation {
	var count float64 = 0
	var t time.Time

	for _, v := range d.Data {
		if v.Weight.Data != 0 {
			count = count + 1
		}
		t = v.Data.Time
	}

	return DataForAggregation{
		SeriesM: SeriesMeta{
			Label:         d.SeriesM.Label,
			Source:        d.SeriesM.Source,
			Upsample:      d.SeriesM.Upsample,
			Units:         "#",
			Downsample:    d.SeriesM.Downsample,
			IsTransformed: true,
			VendorCode:    d.SeriesM.VendorCode,
		},
		Data: []EntityPlusDataPoint{
			EntityPlusDataPoint{
				EntityM: EntityMeta{
					UniqueId: category,
					Name:     category,
				},
				Data: DataPoint{
					Data: count,
					Time: t,
				},
				Weight: DataPoint{
					Data: 1,
					Time: t,
				},
			},
		},
	}
}

func CrossEntityAggregation(aggregatorFn func(string, DataForAggregation) DataForAggregation) StepFnType {
	return func(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
		m := mArr[0]

		newData := MultiEntityData{}

		// Get Fields
		fields := m.GetFields()

		// Get Categories
		categories := m.GetCategories()

		// marshalOutput("m", m)

		for _, f := range fields {
			for _, c := range categories {
				// fmt.Printf("f=%s,c=%s\n", f, c)
				tempData := runFunctionOnSeriesArray(m, f, c, aggregatorFn)

				// marshalOutput("tempData", tempData)
				newData = UnionData(ctx, []MultiEntityData{newData, tempData})
				// marshalOutput("newData", newData)
			}
		}

		newData.Title = m.Title
		newData.Error = m.Error

		return newData
	}
}

func SummaryStats(ctx context.Context, mArr []MultiEntityData) MultiEntityData {
	m := mArr[0]

	newData := MultiEntityData{}

	// Get Fields
	fields := m.GetFields()

	// Get Categories
	categories := m.GetCategories()

	aggregatorFnArray := []func(string, DataForAggregation) DataForAggregation{averageAggregator, medianAggregator, countAggregator, percentAggregator}

	for _, f := range fields {
		for _, c := range categories {
			// fmt.Printf("f=%s,c=%s\n", f, c)
			for i, aggregatorFn := range aggregatorFnArray {
				tempData := runFunctionOnSeriesArray(m, f, c, aggregatorFn)
				switch i {
				case 0:
					tempData = relabelSeries(tempData, "Average ", "", "")
				case 1:
					tempData = relabelSeries(tempData, "Median ", "", "")
				case 2:
					tempData = relabelSeries(tempData, "", "", "Count")
				case 3:
					tempData = relabelSeries(tempData, "", "", "% Positive")
				}
				// marshalOutput("tempData", tempData)

				newData = UnionData(ctx, []MultiEntityData{newData, tempData})
				// marshalOutput("newData", newData)
			}
		}
	}

	newData.Title = m.Title
	newData.Error = m.Error

	return newData
}

func relabelSeries(m MultiEntityData, prefix string, postfix string, replace string) MultiEntityData {
	for i, _ := range m.EntityData {
		for j, _ := range m.EntityData[i].Data {
			if replace != "" {
				m.EntityData[i].Data[j].Meta.Label = replace
			} else {
				m.EntityData[i].Data[j].Meta.Label = prefix + m.EntityData[i].Data[j].Meta.Label + postfix
			}
		}
	}

	return m
}

func runFunctionOnSeriesArray(m MultiEntityData, fieldName string, categoryName string, fn func(string, DataForAggregation) DataForAggregation) MultiEntityData {
	// Loop through and extract all the right series and associated entities
	entityMeta, series, categorySeries, weightsSeries := getSeriesAndEntities(m, fieldName)

	// Union the dates
	newDates := unionSeriesDates(series)

	// Get the resample function
	// fmt.Printf("newDates = %s\n", newDates)
	resampleFn := resampleOnDates(newDates, false, false)

	// marshalOutput("series before resample", series)
	// fmt.Printf("series before resample=%s\n", series)
	//fmt.Printf("weights before resample=%s\n", weightsSeries)

	// Run the resampling on all the series
	series = resampleSeriesArray(series, resampleFn)
	weightsSeries = resampleSeriesArray(weightsSeries, resampleFn)

	// marshalOutput("series after resample", series)
	// marshalOutput("weights after resample", weightsSeries)

	// For each day, for this category, run the function
	m2 := runAggregationOnDates(newDates, entityMeta, series, categorySeries, weightsSeries, categoryName, fn)

	// marshalOutput("m2", m2)

	return m2.RemoveSuperflousDataAndWeights(false)
}

func runAggregationOnDates(dates []time.Time, entityMeta []EntityMeta, series []Series, categorySeries []CategorySeries, weightsSeries []Series, categoryName string, fn func(string, DataForAggregation) DataForAggregation) MultiEntityData {
	var m MultiEntityData

	for _, v := range dates {
		m = accumulateAggregatedDataIntoMultiEntity(m, fn(categoryName, getDataForAggregation(v, entityMeta, series, categorySeries, weightsSeries, categoryName)))
	}

	return m
}

func accumulateAggregatedDataIntoMultiEntity(m MultiEntityData, d DataForAggregation) MultiEntityData {
	// fmt.Printf("data for aggregation=%s\n", d)
	for _, v := range d.Data {
		if !v.Data.Time.IsZero() {
			// marshalOutput("before insert", m)
			// marshalOutput("data to be inserted", v)
			m = m.Insert(v.EntityM, d.SeriesM, v.Data, v.Category, false)
			// marshalOutput("after insert", m)

			// fmt.Printf("Weight before inserting=%q\n", v.Weight)
			m = m.Insert(v.EntityM, d.SeriesM, v.Weight, v.Category, true)
			// marshalOutput("after weights", m)
		}
	}

	return m
}

func getDataForAggregation(date time.Time, entityMeta []EntityMeta, series []Series, categorySeries []CategorySeries, weightsSeries []Series, categoryName string) DataForAggregation {
	// fmt.Printf("date = %s\n", date)

	var d DataForAggregation
	d.Data = make([]EntityPlusDataPoint, 0)

	// Assumes data is already resampled

	// fmt.Printf("series=%s\n", series)
	//fmt.Printf("weightsSeries=%s\n", weightsSeries)

	for i, v := range series {
		isInCategory, categoryLabel := categorySeries[i].CheckCategory(date, categoryName)
		// marshalOutput("isInCategory", isInCategory)
		// marshalOutput("categoryLabel", categoryLabel)
		// marshalOutput("v", v)
		// for _, v2 := range v.Data {
		v2, found := v.Find(date)
		if found && isInCategory {
			d.SeriesM = v.Meta
			d.Data = append(d.Data, EntityPlusDataPoint{entityMeta[i], v2, categoryLabel, getWeightData(date, weightsSeries[i])})
		}
		// }
	}

	// marshalOutput("data for aggregation", d)

	return d
}

func getWeightSeries(s []Series) Series {
	uniqueDates := make([]time.Time, 0)

	for _, v := range s {
		uniqueDates = unionDates(uniqueDates, v.GetDates())
	}

	var w Series

	if len(uniqueDates) > 0 && uniqueDates[0].Before(zeroDay()) {
		w = weightStub(uniqueDates[0])
	} else {
		w = weightStub(zeroDay())
	}

	for _, v := range s {
		if v.IsWeight {
			return v
		}
	}

	return w
}

func getWeightData(date time.Time, weightSeries Series) DataPoint {
	for _, v := range weightSeries.Data {
		if v.Time.Equal(date) {
			return v
		}
	}

	return DataPoint{date, 1}
}

func resampleSeriesArray(s []Series, resampleFn func(SeriesMeta, []DataPoint, []DataPoint) []DataPoint) []Series {
	weightSeries := getWeightSeries(s)

	for i, _ := range s {
		// marshalOutput("before resample", s[i].Data)
		s[i].Data = resampleFn(s[i].Meta, s[i].Data, weightSeries.Data)
		// marshalOutput("after resample", s[i].Data)
	}

	return s
}

func unionSeriesDates(s []Series) []time.Time {
	uniqueDates := make([]time.Time, 0)

	for _, v := range s {
		if !v.IsWeight {
			newDates := getDates(v.Data)
			uniqueDates = unionDates(uniqueDates, newDates)
		}
	}

	return uniqueDates
}

func weightStub(t time.Time) Series {
	return Series{
		Data: []DataPoint{DataPoint{t, 1}},
		Meta: SeriesMeta{
			VendorCode:    "Weights",
			Label:         "Weights",
			Units:         "Weight",
			Source:        "",
			Upsample:      ResampleLastValue,
			Downsample:    ResampleLastValue,
			IsTransformed: true,
		},
		IsWeight: true,
	}
}

func getSeriesAndEntities(m MultiEntityData, fieldName string) ([]EntityMeta, []Series, []CategorySeries, []Series) {
	eMeta := make([]EntityMeta, 0)
	sArr := make([]Series, 0)
	cMeta := make([]CategorySeries, 0)
	wArr := make([]Series, 0)

	for _, e := range m.EntityData {
		var weightSeries Series = getWeightSeries(e.Data)

		for _, v := range e.Data {
			if v.IsWeight {
				weightSeries = v
			}
		}

		for _, s := range e.Data {
			if s.Meta.Label == fieldName {
				eMeta = append(eMeta, e.Meta)
				sArr = append(sArr, s)
				cMeta = append(cMeta, e.Category)
				wArr = append(wArr, weightSeries)
			}
		}
	}

	return eMeta, sArr, cMeta, wArr
}

func WrapNoArguments(fn StepFnType) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		return fn
	}
}

func WrapNumericalArgument(fn func(float64) func(string, DataForAggregation) DataForAggregation) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		var numericalArgument float64
		// var err error

		if len(c) > 0 {
			numericalArgument, _ = strconv.ParseFloat(c[0].QueryComponentParams[0], 64)
		}

		return CrossEntityAggregation(fn(numericalArgument))
	}
}

func WrapNumericalArgumentCombine(fn func(float64) StepFnType) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		var numericalArgument float64
		// var err error

		if len(c) > 0 {
			numericalArgument, _ = strconv.ParseFloat(c[0].QueryComponentParams[0], 64)
		}

		return fn(numericalArgument)
	}
}

//func WrapTwoFields(fn func(int64, int64) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
//	return func(c []component.QueryComponent) StepFnType {
//		// Get field1
//		field1 := getField(0, c)

//		// Get field2
//		field2 := getField(1, c)

//		return ComputeTS(fn(field1, field2))
//	}
//}

//func WrapFieldAndNumber(fn func(int64, float64) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
//	return func(c []component.QueryComponent) StepFnType {
//		// Get the field
//		field := getField(0, c)

//		// Convert Parameter
//		number := convertParameterToFloat(c[0])

//		return ComputeTS(fn(field, number))
//	}
//}

func WrapStringArgumentTS(fn func(string) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		// Convert Parameter
		formula := c[0].QueryComponentOriginalString

		return ComputeTS(fn(formula))
	}
}

func WrapStringArgumentTS2(fn func(string, string) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		// Convert Parameter
		formula := c[0].QueryComponentOriginalString
		label := c[1].QueryComponentOriginalString

		return ComputeTS(fn(formula, label))
	}
}

func WrapNumericalArgumentTS(fn func(float64) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		// Convert Parameter
		number := convertParameterToFloat(c[0])

		return ComputeTS(fn(number))
	}
}

func WrapNumericalArgumentTS2(fn func(float64, float64) func(SingleEntityData) SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		// Convert Parameter
		before, _ := strconv.ParseFloat(c[0].QueryComponentParams[0], 64)
		after, _ := strconv.ParseFloat(c[0].QueryComponentParams[1], 64)

		return ComputeTS(fn(before, after))
	}
}

func WrapNumericalArgumentTS2Multi(fn func(float64, float64) func(SingleEntityData) []SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		// Convert Parameter
		before, _ := strconv.ParseFloat(c[0].QueryComponentParams[0], 64)
		after, _ := strconv.ParseFloat(c[0].QueryComponentParams[1], 64)

		return ComputeTSMulti(fn(before, after))
	}
}

func WrapNoArgumentTSMulti(fn func(SingleEntityData) []SingleEntityData) func([]component.QueryComponent) StepFnType {
	return func(c []component.QueryComponent) StepFnType {
		return ComputeTSMulti(fn)
	}
}

func getField(fieldNum int, c []component.QueryComponent) int64 {
	str := c[1+fieldNum].QueryComponentParams[0]

	i, _ := strconv.ParseInt(str, 10, 64)

	return i
}

func convertParameterToFloat(c component.QueryComponent) float64 {
	p, _ := strconv.ParseFloat(c.QueryComponentParams[0], 64)

	return p
}

func marshalOutput(key string, d interface{}) {
	js, err := json.Marshal(d)
	if err == nil {
		fmt.Printf("%s=%s\n", key, js)
	} else {
		fmt.Printf("%s err=%s\n", key, err)
	}
}
