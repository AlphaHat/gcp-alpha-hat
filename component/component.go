// Package component provides definition for a QueryComponent
package component

import (
	"math"
	"time"

	"github.com/AlphaHat/gcp-alpha-hat/quandl"
	"github.com/AlphaHat/gcp-alpha-hat/timeseries"
)

// Component Types
const (
	Universe                 = "Universe"
	TimeRange                = "Time Range"
	UniverseExpandable       = "Universe: Expandable"
	RebalanceMethodology     = "Rebalance Methodology"
	RebalanceFrequency       = "Rebalance Frequency"
	ReportType               = "Report Type"
	ConceptSecurity          = "Concept: Security"
	Classification           = "Classification"
	TimeAggregationType      = "Time Aggregation Type"
	TimeAggregationFrequency = "Time Aggregation Frequency"
	ContextClue              = "Context Clue"
	Filter                   = "Filter"
	Aggregation              = "Univ. Aggregation"
	TimeHorizon              = "Time Horizon"
	Event                    = "Event"
	EventAggregationType     = "Event Aggregation Type"
	AggregationConcept       = "Aggregation Concept"
	AggregationWeightConcept = "Aggregation Weight Concept"
	MacroEvent               = "Macro Event"
	StockEvent               = "Stock Event"
	MacroData                = "Macro Data"
	StockData                = "Stock Data"
	OverrideData             = "Stock Data Estimates and Overrides"
	FreeText                 = "Free Text"
	RenameEntity             = "Rename Entity"
)

// Sources
const (
	Damodaran        = "Damodaran"
	QuandlOpenData   = "Quandl Open Data"
	Sharadar         = "Sharadar"
	SECHarmonized    = "SEC Harmonized"
	EarningsEstimate = "Zacks Earnings Estimates"
	EarningsSurprise = "Zacks Earnings Surprises"
	Custom           = "Custom"
	Thomson          = "Thomson"
	CapIQ            = "CapIQ"
	Factset          = "Factset"
)

const (
	Weekly    = "Weekly"
	Daily     = "Daily"
	Monthly   = "Monthly"
	Quarterly = "Quarterly"
	Yearly    = "Yearly"
	AllTime   = "All Time"
)

const (
	EventAggregationStock = "By Stock"
	EventAggregationDate  = "By Start Date"
)

type MajorType string

const (
	TimeSeriesTransformation MajorType = "Time Series Transformation"
	TimeSeriesFormula                  = "Time Series Formula"
	TimeSlice                          = "Time Slice"
	RemoveData                         = "Remove Data"
	KeepData                           = "Keep Data"
	SameEntityAggregation              = "Same Entity Aggregation"
	CrossEntityAggregation             = "Cross Entity Aggregation"
	GetUniverse                        = "Get Universe"
	GetData                            = "Get Data"
	GetBulkData                        = "Get Bulk Data"
	CombineData                        = "Combine Data"
	SetWeights                         = "Set Weights"
	TransformWeights                   = "Transform Weights"
	GraphicalPreference                = "Graphical Preference"
	CustomQuandlCode                   = "Custom Quandl Code"
	Field                              = "Field"
	GetPortfolio                       = "Get Portfolio"
)

type QuandlCache struct {
	Ticker string                 `json:"quandl_ticker" bson:"quandl_ticker"`
	QR     *quandl.QuandlResponse `json:"quandl_response" bson:"quandl_response"`
}

type CapiqCache struct {
	Ticker string                 `json:"capiq_ticker" bson:"capiq_ticker"`
	Data   *timeseries.TimeSeries `json:"capiq_data" bson:"capiq_data"`
}

type FactsetCache struct {
	Ticker string                 `json:"factset_ticker" bson:"factset_ticker"`
	Data   *timeseries.TimeSeries `json:"factset_data" bson:"factset_data"`
}

type ComponentGroup struct {
	Name     string           `json:"canonical_name"`
	Children []QueryComponent `json:"children"`
}

type UniverseMember interface {
	GetLabel() string
	NewWithLabel(string) UniverseMember
	GetID() string
}

type QueryComponent struct {
	QueryComponentId             int      `json:"id" bson:"id"`
	QueryComponentCanonicalName  string   `json:"canonical_name" bson:"canonical_name"`
	QueryComponentName           string   `json:"name" bson:"name"`
	QueryComponentType           string   `json:"type" bson:"type"`
	QueryComponentProviderId     string   `json:"provider_id"`
	QueryComponentSource         string   `json:"source"`
	QueryComponentOriginalString string   `json:"original_string"`
	QueryComponentParams         []string `json:"parameters"`
}

func (q QueryComponent) GetLabel() string {
	return q.QueryComponentCanonicalName
}

func (q QueryComponent) GetID() string {
	return q.QueryComponentProviderId
}

func (q QueryComponent) NewWithLabel(label string) UniverseMember {
	q.QueryComponentCanonicalName = label
	return q
}

type LogEntry struct {
	LogType               string `json:"log_type"`
	FunctionBeingExecuted string `json:"function"`
	Details               string `json:"details"`
}

type QueryData struct {
	X string  `json:"x"`
	Y float64 `json:"y"`
}

type ColumnRangeData struct {
	X    int64  `json:"x"`
	Low  string `json:"low"`
	High string `json:"high"`
}

type QueryResponse struct {
	ResponseSeries []QueryResponseSeries `json:"series"`
	LogData        []LogEntry            `json:"log_entries"`
}

type QueryResponseSeries struct {
	ResponseType    string      `json:"type"`
	XaxisType       string      `json:"xAxisType"`
	XaxisLabel      string      `json:"xAxisLabel"`
	XaxisCategories []string    `json:"xAxisCategories"`
	YaxisType       string      `json:"yAxisType"`
	YaxisLabel      string      `json:"yAxisLabel"`
	YaxisCategories []string    `json:"yAxisCategories"`
	DataSource      string      `json:"data_source"`
	SeriesName      string      `json:"series_name"`
	DataEnd         string      `json:"data_end"`
	Data            interface{} `json:"data"`
}

type SavedSearch struct {
	Query    string        `json:"Query"`
	Response QueryResponse `json:"Response"`
}

type BrowseListItem struct {
	Title       string  `json:"title"`
	Keywords    string  `json:"keywords"`
	Recency     string  `json:"recency"`
	Score       float64 `json:"score"`
	Impressions int     `json:"impressions"`
	Votes       int     `json:"votes"`
	Flags       int     `json:"flags"`
}

type LoginSession struct {
	Time  string `json:"time" bson:"time"`
	Token string `json:"token" bson:"token"`
}

type UserDetails struct {
	Id              string `json:"user_id" bson:"user_id"`
	FirstName       string `json:"first_name" bson:"first_name"`
	LastName        string `json:"last_name" bson:"last_name"`
	Email           string `json:"email" bson:"email"`
	Login           string `json:"login" bson:"login"`
	LastLoginAtUnix int64  `json:"last_login_at" bson:"last_login_at"`
	LastLoginAt     time.Time
	PictureUrl      string         `json:"picture_url" bson:"picture_url"`
	Bio             string         `json:"bio" bson:"bio"`
	Session         []LoginSession `json:"session" bson:"session"`
}

type SearchCount struct {
	Query      string        `json:"Query" bson:"Query"`
	Impression []UserAction  `json:"impression" bson:"impression"`
	Votes      []UserAction  `json:"votes" bson:"votes"`
	Flags      []UserAction  `json:"flags" bson:"flags"`
	Scratch    []UserAction  `json:"scratch" bson:"scratch"`
	Summaries  []UserSummary `json:"summaries" bson:"summaries"`
	Views      []UserAction  `json:"views" bson:"views"`
}

type UserSummary struct {
	User    string `json:"user"`
	Summary string `json:"summary"`
}

type UserAction struct {
	User string `json:"user"`
	Time string `json:"time"`
}

func (sc SearchCount) VoteCount() int {
	return len(sc.Votes)
}

func (sc SearchCount) FlagCount() int {
	return len(sc.Flags)
}

func (sc SearchCount) ImpressionCount() int {
	if len(sc.Impression) > 0 {
		return len(sc.Impression)
	}

	return 1
}

func (sc SearchCount) ScratchCount() int {
	return len(sc.Scratch)
}

func (sc SearchCount) DatePosted() string {
	minDate := ""

	for i, v := range sc.Views {
		if i == 0 {
			minDate = v.Time
		}
		if v.Time < minDate {
			minDate = v.Time
		}
	}

	return minDate
}

func (sc SearchCount) Score() float64 {
	recency := GetRecency(sc.DatePosted())

	//m, _ := json.Marshal(sc)
	score := (float64(sc.VoteCount()-sc.FlagCount()) / float64(sc.ImpressionCount())) + math.Exp(-(recency+240*float64(sc.FlagCount()))/15)
	//score := float64(sc.VoteCount-sc.FlagCount) * math.Exp(-(recency)/(4*7*24*60))
	score = math.Floor(1000 * score)
	// fmt.Printf("%d = %s\n", score, m)

	return score
}

func GetRecency(whenPosted string) float64 {
	timePosted, err := time.Parse("2006-01-02 15:04:05.999999999 -0700 MST", whenPosted)

	if err == nil {
		return time.Since(timePosted).Minutes()
	}

	return 10000
}
