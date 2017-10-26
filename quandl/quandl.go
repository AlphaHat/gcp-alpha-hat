package quandl

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"google.golang.org/appengine/log"
	"google.golang.org/appengine/urlfetch"

	"github.com/AlphaHat/gcp-alpha-hat/cache"
)

const (
	quandlApiRoot    = "https://www.quandl.com/api/v1/datasets/"
	quandlSearchRoot = "https://www.quandl.com/api/v1/datasets.json"

	sharadarList = "http://www.sharadar.com/meta/indicators.txt"

	quandlStockList     = "https://s3.amazonaws.com/quandl-static-content/quandl-stock-code-list.csv"
	quandlStockWikiList = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/WIKI_tickers.csv"
	sectorList          = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Stock+Exchanges/stockinfo.csv"
	etfList             = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/ETFs.csv"
	stockIndexList      = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Stock+Exchanges/Indicies.csv"
	mutualFundList      = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Stock+Exchanges/funds.csv"

	// Code already contains the source
	commoditiesList = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/commodities.csv"

	// Source is in the file and needs to be pre-pended
	currencyList = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/currencies.csv"

	spxConstituents             = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Indicies/SP500.csv"
	dowConstituents             = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Indicies/dowjonesIA.csv"
	nasdaqCompositeConstituents = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Indicies/NASDAQComposite.csv"
	nasdaq100Constituents       = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Indicies/nasdaq100.csv"
	ftse100Constituents         = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/Indicies/FTSE100.csv"

	// Note that this data is pipe-delimited rather than comma delimited
	economicData = "https://s3.amazonaws.com/quandl-static-content/Ticker+CSV%27s/FRED/fred_allcodes.csv"

	fredPopularJson = "https://api.stlouisfed.org/fred/tags/series?tag_names=usa&api_key=bdd6684ed70330ecb9e3fcb8d2dc4781&order_by=popularity&limit=1000&sort_order=desc&file_type=json"

	format = ".json"
)

var authToken string
var csvCache *cache.GenericCache = cache.NewGenericCache(time.Hour*6, "csvCache", func(ctx context.Context, fileName string) (interface{}, bool) {
	val, err := loadCSV(ctx, fileName)
	var found bool = false

	if err == nil {
		found = true
	}

	return val, found
})
var macCsvCachce *cache.GenericCache = cache.NewGenericCache(time.Hour*6, "macCsvCachce", func(ctx context.Context, fileName string) (interface{}, bool) {
	val, err := loadCSVMac(ctx, fileName)
	var found bool = false

	if err == nil {
		found = true
	}

	return val, found
})

type QuandlResponse struct {
	SourceCode  string      `json:"source_code" bson:"source_code"`
	SourceName  string      `json:"source_name" bson:"source_name"`
	QuandlName  string      `json:"name"`
	Code        string      `json:"code" bson:"code"`
	Frequency   string      `json:"frequency" bson:"frequency"`
	FromDate    string      `json:"from_date" bson:"from_date"`
	ToDate      string      `json:"to_date" bson:"to_date"`
	Description string      `json:"description" bson:"description"`
	Columns     []string    `json:"column_names" bson:"column_names"`
	Data        interface{} `json:"data" bson:"data"`
}

/*type TimeSeriesDataPoint struct {
	Date string
	Data float64
}

type TimeSeries struct {
	Data []TimeSeriesDataPoint
}*/

// SetAuthToken sets the auth token globally so that all subsequent calls that
// retrieve data from the Quandl API will use the auth token.
func SetAuthToken(token string) {
	authToken = token
}

func assembleQueryURL(query string) string {
	var url string

	if authToken == "" {
		fmt.Printf("No auth token set. API calls are limited.\n")
		url = fmt.Sprintf("%s?query=%s", quandlSearchRoot, query)
	} else {
		url = fmt.Sprintf("%s?query=%s&auth_token=%s", quandlSearchRoot, query, authToken)
	}
	return url
}

func assembleURLwithDates(identifier string, startDate string, endDate string) string {
	var url string

	if authToken == "" {
		fmt.Printf("No auth token set. API calls are limited.\n")
		url = fmt.Sprintf("%s%s%s?trim_start=%s&trim_end=%s", quandlApiRoot, identifier, format, startDate, endDate)
	} else {
		url = fmt.Sprintf("%s%s%s?trim_start=%s&trim_end=%s&auth_token=%s", quandlApiRoot, identifier, format, startDate, endDate, authToken)
	}
	return url
}

func assembleURLwithoutDates(identifier string) string {
	var url string
	if authToken == "" {
		fmt.Printf("No auth token set. API calls are limited.\n")
		url = fmt.Sprintf("%s%s%s", quandlApiRoot, identifier, format)
	} else {
		url = fmt.Sprintf("%s%s%s?auth_token=%s", quandlApiRoot, identifier, format, authToken)
	}
	return url
}

func readBytesFromUrl(ctx context.Context, url string) ([]byte, error) {
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Errorf(ctx, "err=%s\n", err)
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}

func getDataFromURL(ctx context.Context, url string) (*QuandlResponse, error) {
	//fmt.Printf("%s\n", url)
	log.Infof(ctx, "Url = %s", url)

	body, err := readBytesFromUrl(ctx, url)
	if err != nil {
		log.Errorf(ctx, "err=%s\n", err)
		return nil, err
	}

	var quandlResponse *QuandlResponse

	err = json.Unmarshal(body, &quandlResponse)
	if err != nil {
		log.Errorf(ctx, "err=%s\n", err)
		return nil, err
	}

	if quandlResponse.SourceCode == "RAYMOND" && quandlResponse.SourceName == "" {
		quandlResponse.SourceName = "U.S. Securities and Exchange Commission"
	}

	//fmt.Printf("%s\n", quandlResponse)
	return quandlResponse, nil
}

// GetData gets Quandl data for a particular identifier and a date range.
// You can optionally set the auth token before running this function so that you
// can make unlimited API calls instead of being limited to 500/day.
func GetData(ctx context.Context, identifier string, startDate string, endDate string) (*QuandlResponse, error) {
	url := assembleURLwithDates(identifier, startDate, endDate)

	return getDataFromURL(ctx, url)
}

// GetAllHistory is similar to GetData except that it does not restrict a date range
func GetAllHistory(ctx context.Context, identifier string) (*QuandlResponse, error) {
	url := assembleURLwithoutDates(identifier)

	return getDataFromURL(ctx, url)
}

func (q *QuandlResponse) GetUnits() string {
	r := regexp.MustCompile(`Units[A-Za-z<>\/]*:([^<]+)`)

	v := r.FindString(q.Description)

	r = regexp.MustCompile("<.*>")
	v = r.ReplaceAllString(v, "")

	if v == "Units: currency" {
		v = "$"
	}

	return v
}

func (q *QuandlResponse) GetMultiplier() float64 {
	// fmt.Printf("units=%s, description=%s\n", q.GetUnits(), q.Description)
	if strings.Contains(q.GetUnits(), "million") {
		return 1e6
	} else if strings.Contains(q.GetUnits(), "billion") {
		return 1e9
	} else if strings.Contains(q.GetUnits(), "thousand") {
		return 1000
	}

	return 1.0
}

// GetTimeSeriesColumn returns the data from the Quandl response for a particular column.
// For some series, particularly stock data, multiple columns are returned. Using this
// method you can specify the specific column to extract.
func (q *QuandlResponse) GetTimeSeriesColumn(ctx context.Context, column string) []float64 {
	_, data := q.GetTimeSeries(ctx, column)

	return data
}

// GetTimeSeriesData returns the most relevant data column from the Quandl response.
// In many cases you will not necessarily know beforehand what type of data is being
// requested and therefore cannot determine if it's stock data vs. economic data. In
// such cases, you can use this function to grab the column that is most likely relevant.
// The method also returns the most relevant
func (q *QuandlResponse) GetTimeSeriesData(ctx context.Context) ([]float64, string) {
	column := q.getLikelyDataColumnName()

	_, data := q.GetTimeSeries(ctx, column)

	return data, column
}

// GetTimeSeriesDate returns the series of dates in the time series
func (q *QuandlResponse) GetTimeSeriesDate(ctx context.Context) []string {
	column := q.getLikelyDataColumnName()

	date, _ := q.GetTimeSeries(ctx, column)

	return date
}

// GetTimeSeries returns a date vector and the value vector for a particular
// column in the QuandlResponse
func (q *QuandlResponse) GetTimeSeries(ctx context.Context, column string) ([]string, []float64) {
	if q == nil || q.Data == nil {
		return nil, nil
	}

	dataArray := q.Data.([]interface{})

	dateVector := make([]string, 0, len(dataArray))
	dataVector := make([]float64, 0, len(dataArray))
	dateColumnNum := q.getColumnNum("Date")
	// If the date column isn't called "Date", try "Settlement Date"
	if dateColumnNum == -1 {
		dateColumnNum = q.getColumnNum("Settlement Date")
	}
	// If the date column isn't settlement date or date, just take the first column
	if dateColumnNum == -1 {
		dateColumnNum = 0
	}
	dataColumnNum := q.getColumnNum(column)

	if dateColumnNum == -1 || dataColumnNum == -1 {
		return nil, nil
	}

	for k, v := range dataArray {
		switch vv := v.(type) {
		case []interface{}:
			// Check that 0 is the date
			switch vv[dateColumnNum].(type) {
			case string:
				dateVector = append(dateVector, vv[dateColumnNum].(string))
			default:
				log.Infof(ctx, "error: Problem reading %q as a string.\n", vv[dateColumnNum])
				return nil, nil
			}

			// Match the right column with the requested column
			switch vv[dataColumnNum].(type) {
			case float64:
				dataVector = append(dataVector, vv[dataColumnNum].(float64)*q.GetMultiplier())
			case int:
				dataVector = append(dataVector, float64(vv[dataColumnNum].(int))*q.GetMultiplier())
			default:
				log.Infof(ctx, "error: Problem reading %q (%v) as a float64. Data type is %s. Column is %s. Code is %s.\n", vv[dataColumnNum], vv[dataColumnNum], reflect.TypeOf(vv[dataColumnNum]), column, q.Code)
				dataVector = append(dataVector, 0)
				//return nil, nil
			}
		default:
			log.Infof(ctx, fmt.Sprintf("%q", k)+" is of a type I don't know how to handle")
			return nil, nil
		}
	}

	return dateVector, dataVector
}

// getLikelyDataColumnName finds the column most likely to be the "data"
// column. It either uses adjusted close or just takes the last column in the series
func (q *QuandlResponse) getLikelyDataColumnName() string {
	// Get the column called Adj. Close
	adjustedCloseColumn := q.getColumnNum("Adj. Close")

	if adjustedCloseColumn == -1 {
		adjustedCloseColumn = q.getColumnNum("Adjusted Close")
	}

	if len(q.Columns) < 1 {
		return "N/A"
	} else if adjustedCloseColumn == -1 {
		// If there's no Adj. Close column, get the Close column
		adjustedCloseColumn = q.getColumnNum("Close")
		if adjustedCloseColumn == -1 {
			rateColumn := q.getColumnNum("Rate")
			if rateColumn == -1 {
				// Return the last column
				settleColumn := q.getColumnNum("Settle")
				if settleColumn == -1 {
					indexValueColumn := q.getColumnNum("Index Value")

					if indexValueColumn == -1 {
						valueColumn := q.getColumnNum("Value")

						if valueColumn == -1 {
							return q.Columns[len(q.Columns)-1]
						}

						return q.Columns[valueColumn]
					}
					return q.Columns[indexValueColumn]
				}
				return q.Columns[settleColumn]
			}
			return q.Columns[rateColumn]
		}
	}

	return q.Columns[adjustedCloseColumn]
}

// getColumnNum returns the column number associated with a particular column name.
// It returns -1 if the column is not found.
func (q *QuandlResponse) getColumnNum(column string) int {
	for i, v := range q.Columns {
		if v == column {
			return i
		}
	}

	return -1
}

// Search executes a query against the Quandl API and returns the JSON object
// as a byte stream. In future releases of this Go (golang) Quandl package
// this will return a native object instead of the json
func Search(ctx context.Context, query string) ([]byte, error) {
	query = strings.Replace(query, " ", "+", -1)

	url := assembleQueryURL(query)

	body, err := readBytesFromUrl(ctx, url)

	return body, err
}

func loadJson(ctx context.Context, url string) []byte {
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil
	}

	return body
}

func loadPipeDelimited(ctx context.Context, url string) [][]string {
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil
	}

	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	reader.Comma = '|'

	records, _ := reader.ReadAll()

	return records
}

func loadTabDelimited(ctx context.Context, url string) [][]string {
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil
	}
	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	reader.Comma = '\t'

	records, err := reader.ReadAll()
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil
	}

	return records
}

type linefeedConverter struct {
	r *bufio.Reader
}

func newLinefeedConverter(r io.Reader) io.Reader {
	return linefeedConverter{bufio.NewReader(r)}
}

func (r linefeedConverter) Read(b []byte) (int, error) {
	n, err := r.r.Read(b)
	if err != nil && n == 0 {
		return n, err
	}
	b = b[:n]
	for i := range b {
		if b[i] == '\r' {
			var next byte
			if j := i + 1; j < len(b) {
				next = b[j]
			} else {
				next, err = r.r.ReadByte()
				if err == nil {
					r.r.UnreadByte()
				}
			}
			if next != '\n' {
				b[i] = '\n'
			}
		}
	}

	//fmt.Printf("b=%s\n", b)

	return n, err
}

func loadCSVMac(ctx context.Context, url string) ([][]string, error) {
	log.Infof(ctx, "Loading csv %s", url)
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil, err
	}

	defer resp.Body.Close()

	reader := csv.NewReader(newLinefeedConverter(resp.Body))

	records, err := reader.ReadAll()

	// if err != nil {
	// 	fmt.Printf("err=%s\n", err)
	// }
	return records, err
}

func loadCSV(ctx context.Context, url string) ([][]string, error) {
	log.Infof(ctx, "Loading csv %s", url)

	//file, err := os.Open(fileName)
	resp, err := urlfetch.Client(ctx).Get(url)
	if err != nil {
		log.Infof(ctx, "err=%s\n", err)
		return nil, err
	}

	defer resp.Body.Close()

	reader := csv.NewReader(resp.Body)

	// for i := 0; true; i++ {
	// 	record, err := reader.Read()

	// 	if err == io.EOF {
	// 		break
	// 	} else if err != nil {
	// 		log.Fatal(err)
	// 		return nil
	// 	} else {
	// 		if i <= 1 {
	// 			fmt.Printf("%q\n", record)
	// 		}
	// 	}
	// }

	records, err := reader.ReadAll()

	if len(records) <= 1 {
		records, err = loadCSVMac(ctx, url)
	}

	return records, err
}

func getCsvColumnNumber(csvArray [][]string, column string) int {
	for i, v := range csvArray[0][:] {
		if v == column {
			return i
		}
	}

	return -1
}

// Get various lists
func extractColumns(csvArray [][]string, column1 int, column2 int, skipHeader bool) ([]string, []string) {
	identifier := make([]string, 0, len(csvArray))
	description := make([]string, 0, len(csvArray))

	for i, v := range csvArray[:] {
		for j, v2 := range v {
			// If the header is included then we use the first row
			// otherwise we skip the first row
			if (!skipHeader && i == 0) || i > 0 {
				if j == column1 {
					identifier = append(identifier, replaceTickers(v2))
				} else if j == column2 {
					description = append(description, replaceDescriptions(v2))
				}
			}
		}
	}

	return identifier, description
}

func replaceTickers(ticker string) string {
	if ticker == "GOOG" {
		return "GOOGL"
	}

	return ticker
}

func replaceDescriptions(desc string) string {
	if desc == "Google'C'" {
		return "Google"
	}

	return desc
}

// GetAllSecurityList gets all the security identifiers and descriptions
func GetAllSecurityList(ctx context.Context) ([]string, []string) {

	identifier, description := GetStockList(ctx)

	// tempIdentifier, tempDescription := GetStockTickerList()
	// identifier = append(identifier, tempIdentifier...)
	// description = append(description, tempDescription...)

	tempIdentifier, tempDescription := GetETFList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	// tempIdentifier, tempDescription = GetETFTickerList()
	// identifier = append(identifier, tempIdentifier...)
	// description = append(description, tempDescription...)

	tempIdentifier, tempDescription = GetStockIndexList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	tempIdentifier, tempDescription = GetCommoditiesList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	tempIdentifier, tempDescription = GetBitcoinList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	return identifier, description
}

func GetPriceOnlySeries(ctx context.Context) ([]string, []string) {
	// tempIdentifier, tempDescription := GetStockTickerList()
	// identifier = append(identifier, tempIdentifier...)
	// description = append(description, tempDescription...)

	identifier, description := GetETFList(ctx)

	// tempIdentifier, tempDescription = GetETFTickerList()
	// identifier = append(identifier, tempIdentifier...)
	// description = append(description, tempDescription...)

	tempIdentifier, tempDescription := GetStockIndexList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	tempIdentifier, tempDescription = GetCommoditiesList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	tempIdentifier, tempDescription = GetBitcoinList(ctx)
	identifier = append(identifier, tempIdentifier...)
	description = append(description, tempDescription...)

	return identifier, description
}

// GetStockList gets all the Quandl stock codes and descriptions
func GetStockList(ctx context.Context) ([]string, []string) {
	//list := loadCSV(quandlStockWikiList)
	temp, _ := csvCache.Retrieve(ctx, quandlStockWikiList)

	list := temp.([][]string)

	return extractColumns(list, 0, 1, true)
}

// GetStockTickerList gets all the Quandl codes and tickers
func GetStockTickerList(ctx context.Context) ([]string, []string) {
	temp, _ := csvCache.Retrieve(ctx, quandlStockList)

	list := temp.([][]string)

	return extractColumns(list, 2, 0, true)
}

// GetETFList gets all the Quandl codes and ETF descriptions
func GetETFList(ctx context.Context) ([]string, []string) {
	temp, _ := csvCache.Retrieve(ctx, etfList)

	list := temp.([][]string)

	return extractColumns(list, 1, 2, true)
}

// GetETFTickerList gets all the Quandl codes and ETF tickers
func GetETFTickerList(ctx context.Context) ([]string, []string) {
	temp, _ := csvCache.Retrieve(ctx, etfList)

	list := temp.([][]string)

	return extractColumns(list, 1, 0, true)
}

// GetStockIndexList gets all the Quandl codes and stock index descriptions
func GetStockIndexList(ctx context.Context) ([]string, []string) {
	// list := loadCSV(stockIndexList)
	temp, _ := csvCache.Retrieve(ctx, stockIndexList)

	list := temp.([][]string)

	return extractColumns(list, 1, 2, true)
}

// GetCommoditiesList gets all the Quandl codes and commodity descriptions
func GetCommoditiesList(ctx context.Context) ([]string, []string) {
	// list := loadCSV(commoditiesList)
	temp, _ := csvCache.Retrieve(ctx, commoditiesList)

	list := temp.([][]string)

	return extractColumns(list, 1, 0, true)
}

// GetBitcoinList just returns http://www.quandl.com/api/v1/datasets/BITSTAMP/USD
func GetBitcoinList(ctx context.Context) ([]string, []string) {
	identifier := make([]string, 1, 1)
	description := make([]string, 1, 1)

	identifier[0] = "BITSTAMP/USD"
	description[0] = "Bitcoin Exchange Rate BTC per USD"
	return identifier, description
}

// Data item identifier
// GetAllDataItemList

// GetFinancialRatiosList returns the list of Damordoran financial ratios. Currently
// this list is hard-coded into the file because Quandl does not provide a file from
// where to read. A caveat about using these is that you need to append the ticker
// in a particular way to use these ratios
func GetFinancialRatiosList(ctx context.Context) ([]string, []string) {
	test := [][]string{{"FLOAT", "Number of Shares Outstanding"},
		{"INSIDER", "Insider Holdings"},
		{"CAPEX", "Capital Expenditures"},
		{"NET_MARG", "Net Margin"},
		{"INV_CAP", "Invested Capital"},
		{"P_S", "Price to Sales Ratio"},
		{"ROC", "Return on Capital"},
		{"STOCK_PX", "Stock Price"},
		{"MKT_DE", "Market Debt to Equity Ratio"},
		{"CORREL", "Correlation with the Market"},
		{"PE_FWD", "Forward PE Ratio"},
		{"REV_GRO", "Previous Year Growth in Revenues"},
		{"EBIT_1T", "EBIT for Previous Period"},
		{"DIV", "Dividends"},
		{"EPS_FWD", "Forward Earnings Per Share"},
		{"CHG_NCWC", "Change in Non-Cash Working Capital"},
		{"CASH_FV", "Cash as Percentage of Firm Value"},
		{"INST_HOLD", "Institutional Holdings"},
		{"EFF_TAX", "Effective Tax Rate"},
		{"CASH_ASSETS", "Cash as Percentage of Total Assets"},
		{"FIXED_TOT", "Ratio of Fixed Assets to Total Assets"},
		{"BETA_VL", "Value Line Beta"},
		{"BV_ASSETS", "Book Value of Assets"},
		{"BV_EQTY", "Book Value of Equity"},
		{"FCFF", "Free Cash Flow to Firm"},
		{"CASH_REV", "Cash as Percentage of Revenues"},
		{"MKT_CAP", "Market Capitalization"},
		{"EFF_TAX_INC", "Effective Tax Rate on Income"},
		{"EV_SALES", "EV To Sales Ratio"},
		{"TOT_DEBT", "Total Debt"},
		{"INTANG_TOT", "Ratio of Intangible Assets to Total Assets"},
		{"PE_G", "PE to Growth Ratio"},
		{"REINV_RATE", "Reinvestment Rate"},
		{"BOOK_DC", "Book Debt to Capital Ratio"},
		{"EPS_GRO_EXP", "Expected Growth in Earnings Per Share"},
		{"EV_EBIT", "EV to EBIT Ratio"},
		{"PE_CURR", "Current PE Ratio"},
		{"MKT_DC", "Market Debt to Capital Ratio"},
		{"NCWC_REV", "Non-Cash Working Capital as Percentage of Revenues"},
		{"REV_12M", "Trailing 12-month Revenues"},
		{"REV_GRO_EXP", "Expected Growth in Revenues"},
		{"REV_TRAIL", "Trailing Revenues"},
		{"ROE", "Return on Equity"},
		{"EV_EBITDA", "EV to EBITDA Ratio"},
		{"EBITDA", "Earnings Before Interest Taxes Depreciation and Amortization"},
		{"BETA", "3-Year Regression Beta"},
		{"DEPREC", "Depreciation"},
		{"EV_SALESTR", "EV to Trailing Sales Ratio"},
		{"EPS_GRO", "Growth in Earnings Per Share"},
		{"P_BV", "Price to Book Value Ratio"},
		{"NET_INC_TRAIL", "Trailing Net Income"},
		{"PE_TRAIL", "Trailing PE Ratio"},
		{"OP_MARG", "Pre-Tax Operating Margin"},
		{"FIRM_VAL", "Firm Value"},
		{"STDEV", "3-year Standard Deviation of Stock Price"},
		{"TRAD_VOL", "Trading Volume"},
		{"CASH", "Cash"},
		{"DIV_YLD", "Dividend Yield"},
		{"REV_LAST", "Revenues"},
		{"NET_INC", "Net Income"},
		{"EV_BV", "EV to Book Value Ratio"},
		{"REINV", "Reinvestment Amount"},
		{"EBIT", "Earnings Before Interest and Taxes"},
		{"EV_CAP", "EV to Invested Capital Ratio"},
		{"PAYOUT", "Payout Ratio"},
		{"HILO", "Hi-Lo Risk"},
		{"ALLFINANCIALRATIOS", "All Financial Ratios"},
		{"SGA", "Sales General and Administration Expenses"},
		{"EV", "Enterprise Value"},
		{"NCWC", "Non-Cash Working Capital"},
	}

	return extractColumns(test, 0, 1, false)
}

func GetSECHarmonizedFields(ctx context.Context) ([]string, []string) {
	test := [][]string{
		{"REVENUE", "Revenue"},
		{"TOTAL_REVENUE", "Total Revenue"},
		{"COST_OF_REVENUE_TOTAL", "Cost of Revenue, Total"},
		{"GROSS_PROFIT", "Gross Profit"},
		{"SELLING_GENERAL_ADMIN_EXPENSES_TOTAL", "Selling/General/Admin. Expenses, Total"},
		{"GAIN_LOSS_ON_SALE_OF_ASSETS", "Gain (Loss) on Sale of Assets"},
		{"OTHER_NET", "Other, Net"},
		{"INCOME_BEFORE_TAX", "Income Before Tax"},
		{"INCOME_AFTER_TAX", "Income After Tax"},
		{"NET_INCOME_BEFORE_EXTRA_ITEMS", "Net Income Before Extra. Items"},
		{"NET_INCOME", "Net Income"},
		{"INCOME_AVAILABLE_TO_COMMON_EXCL_EXTRA_ITEMS", "Income Available to Common Excl. Extra Items"},
		{"INCOME_AVAILABLE_TO_COMMON_INCL_EXTRA_ITEMS", "Income Available to Common Incl. Extra Items"},
		{"DILUTED_WEIGHTED_AVERAGE_SHARES", "Diluted Weighted Average Shares"},
		{"DILUTED_EPS_EXCLUDING_EXTRAORDINARY_ITEMS", "Diluted EPS Excluding Extraordinary Items"},
		{"RESEARCH_DEVELOPMENT", "Research & Development"},
		{"UNUSUAL_EXPENSE_INCOME", "Unusual Expense (Income)"},
		{"OPERATING_INCOME", "Operating Income"},
		{"MINORITY_INTEREST", "Minority Interest"},
		{"DIVIDENDS_PER_SHARE_COMMON_STOCK_PRIMARY_ISSUE", "Dividends per Share - Common Stock Primary Issue"},
		{"DEPRECIATION_AMORTIZATION", "Depreciation/Amortization"},
		{"EQUITY_IN_AFFILIATES", "Equity In Affiliates"},
		{"TOTAL_OPERATING_EXPENSE", "Total Operating Expense"},
		{"DILUTED_NORMALIZED_EPS", "Diluted Normalized EPS"},
		{"OTHER_OPERATING_EXPENSES_TOTAL", "Other Operating Expenses, Total"},
		{"DILUTION_ADJUSTMENT", "Dilution Adjustment"},
		{"OTHER_REVENUE_TOTAL", "Other Revenue, Total"},
		{"INTEREST_INCOME_EXPENSE_NET_NON_OPERATING", "Interest Income(Expense), Net Non-Operatig"},
		{"ACCOUNTS_RECEIVABLE_TRADE_NET", "Accounts Receivable - Trade, Net"},
		{"TOTAL_INVENTORY", "Total Inventory"},
		{"PREPAID_EXPENSES", "Prepaid Expenses"},
		{"OTHER_CURRENT_ASSETS_TOTAL", "Other Current Assets, Total"},
		{"TOTAL_CURRENT_ASSETS", "Total Current Assets"},
		{"GOODWILL_NET", "Goodwill, Net"},
		{"TOTAL_ASSETS", "Total Assets"},
		{"ACCOUNTS_PAYABLE", "Accounts Payable"},
		{"CURRENT_PORT_OF_LT_DEBT_CAPITAL_LEASES", "Current Port. of LT Debt/Capital Leases"},
		{"TOTAL_CURRENT_LIABILITIES", "Total Current Liabilities"},
		{"DEFERRED_INCOME_TAX", "Deferred Income Tax"},
		{"TOTAL_LIABILITIES", "Total Liabilities"},
		{"COMMON_STOCK_TOTAL", "Common Stock, Total"},
		{"ADDITIONAL_PAID_IN_CAPITAL", "Additional Paid-In Capital"},
		{"RETAINED_EARNINGS_ACCUMULATED_DEFICIT", "Retained Earnings (Accumulated Deficit)"},
		{"TOTAL_EQUITY", "Total Equity"},
		{"TOTAL_LIABILITIES_SHAREHOLDERS_EQUITY", "Total Liabilities & Shareholders' Equity"},
		{"TOTAL_COMMON_SHARES_OUTSTANDING", "Total Common Shares Outstanding"},
		{"CASH_EQUIVALENTS", "Cash & Equivalents"},
		{"CASH_AND_SHORT_TERM_INVESTMENTS", "Cash and Short Term Investments"},
		{"INTANGIBLES_NET", "Intangibles, Net"},
		{"OTHER_LONG_TERM_ASSETS_TOTAL", "Other Long Term Assets, Total"},
		{"LONG_TERM_DEBT", "Long Term Debt"},
		{"TOTAL_LONG_TERM_DEBT", "Total Long Term Debt"},
		{"TOTAL_DEBT", "Total Debt"},
		{"MINORITY_INTEREST", "Minority Interest"},
		{"OTHER_EQUITY_TOTAL", "Other Equity, Total"},
		{"PROPERTY_PLANT_EQUIPMENT_TOTAL_GROSS", "Property/Plant/Equipment, Total - Gross"},
		{"OTHER_CURRENT_LIABILITIES_TOTAL", "Other Current liabilities, Total"},
		{"OTHER_LIABILITIES_TOTAL", "Other Liabilities, Total"},
		{"LONG_TERM_INVESTMENTS", "Long Term Investments"},
		{"ACCRUED_EXPENSES", "Accrued Expenses"},
		{"NOTES_PAYABLE_SHORT_TERM_DEBT", "Notes Payable/Short Term Debt"},
		{"TOTAL_RECEIVABLES_NET", "Total Receivables, Net"},
		{"PREFERRED_STOCK_NON_REDEEMABLE_NET", "Preferred Stock - Non Redeemable, Net"},
		{"SHORT_TERM_INVESTMENTS", "Short Term Investments"},
		{"CAPITAL_LEASE_OBLIGATIONS", "Capital Lease Obligations"},
		{"ACCUMULATED_DEPRECIATION_TOTAL", "Accumulated Depreciation, Total"},
		{"REDEEMABLE_PREFERRED_STOCK_TOTAL", "Redeemable Preferred Stock, Total"},
		{"TREASURY_STOCK_COMMON", "Treasury Stock - Common"},
		{"NET_INCOME_STARTING_LINE", "Net Income/Starting Line"},
		{"DEPRECIATION_DEPLETION", "Depreciation/Depletion"},
		{"AMORTIZATION", "Amortization"},
		{"CASH_FROM_OPERATING_ACTIVITIES", "Cash from Operating Activities"},
		{"ISSUANCE_RETIREMENT_OF_DEBT_NET", "Issuance (Retirement) of Debt, Net"},
		{"CASH_FROM_FINANCING_ACTIVITIES", "Cash from Financing Activities"},
		{"NET_CHANGE_IN_CASH", "Net Change in Cash"},
		{"CASH_INTEREST_PAID_SUPPLEMENTAL", "Cash Interest Paid, Supplemental"},
		{"CASH_TAXES_PAID_SUPPLEMENTAL", "Cash Taxes Paid, Supplemental"},
		{"DEFERRED_TAXES", "Deferred Taxes"},
		{"CHANGES_IN_WORKING_CAPITAL", "Changes in Working Capital"},
		{"CASH_FROM_INVESTING_ACTIVITIES", "Cash from Investing Activities"},
		{"FOREIGN_EXCHANGE_EFFECTS", "Foreign Exchange Effects"},
		{"NON_CASH_ITEMS", "Non-Cash Items"},
		{"OTHER_INVESTING_CASH_FLOW_ITEMS_TOTAL", "Other Investing Cash Flow Items, Total"},
		{"FINANCING_CASH_FLOW_ITEMS", "Financing Cash Flow Items"},
		{"TOTAL_CASH_DIVIDENDS_PAID", "Total Cash Dividends Paid"},
		{"ISSUANCE_RETIREMENT_OF_STOCK_NET", "Issuance (Retirement) of Stock, Net"},
		{"CAPITAL_EXPENDITURES", "Capital Expenditures"},
	}

	return extractColumns(test, 0, 1, false)
}

func GetCurrenciesList(ctx context.Context) ([]string, []string) {
	//test := [][]string{
	//	{"Australian Dollar (AUD)", "DEXUSAL"},
	//	{"Brazilian Real (BRL)", "DEXBZUS"},
	//	{"British Pound (GBP)", "DEXUSUK"},
	//	{"Canadaian Dollar (CAD)", "DEXCAUS"},
	//	{"Chinese Yuan (CNY))", "DEXCHUS"},
	//	{"Denish Krone (DKK)", "DEXDNUS"},
	//	{"Euro (EUR)", "DEXUSEU"},
	//	{"Hong Kong Dollar (HKD)", "DEXHKUS"},
	//	{"Indian Rupee (INR)", "DEXINUS"},
	//	{"Japanese Yen (JPY)", "DEXJPUS"},
	//	{"Malaysian Ringgit (MYR)", "DEXMAUS"},
	//	{"Mexican Peso (MXN)", "DEXMXUS"},
	//	{"New Taiwan Dollar (TWD)", "DEXTAUS"},
	//	{"New Zealand Dollar (NZD)", "DEXUSNZ"},
	//	{"Norwegian Krone(NOK)", "DEXNOUS"},
	//	{"Singapore Dollar (SGD)", "DEXSIUS"},
	//	{"South African Rand(ZAR)", "DEXSFUS"},
	//	{"South Korean Won (KRW)", "DEXKOUS"},
	//	{"Sri Lankan Rupee(LKR)", "DEXSLUS"},
	//	{"Swedish Krona (SEK)", "DEXSDUS"},
	//	{"Swiss Franc (CHF)", "DEXSZUS"},
	//	{"Thai Baht (THB)", "DEXTHUS"},
	//	{"Venezuelan Bolivar (VEF)", "DEXVZUS"},
	//}

	test := [][]string{
		{"Australian Dollar AUD per USD", "USDAUD"},
		{"Canadian Dollar CAD per USD", "USDCAD"},
		{"Chinese Yuan CNY per USD", "USDCNY"},
		{"Czech Koruna CZK per USD", "USDCZK"},
		{"Danish Krone DKK per USD", "USDDKK"},
		{"Hong Kong Dollar HKD per USD", "USDHKD"},
		{"Hungarian Forint HUF per USD", "USDHUF"},
		{"Indian Rupee INR per USD", "USDINR"},
		{"Israeli Shekel NIS per USD", "USDNIS"},
		{"Japanese Yen JPY per USD", "USDJPY"},
		{"Lithuanian Litas LTL per USD", "USDLTL"},
		{"Malaysian Ringgit MYR per USD ", "USDMYR"},
		{"New Zealand Dollar NZD per USD", "USDNZD"},
		{"Norwegian Krone NOK per USD", "USDNOK"},
		{"Polish Zloty PLN per USD", "USDPLN"},
		{"Pound Sterling USD per GBP", "GBPUSD"},
		{"Russian Ruble RUB per USD", "USDRUB"},
		{"Saudi Riyal SAR per USD", "USDSAR"},
		{"Singapore Dollar SGD per USD", "USDSGD"},
		{"South African Rand ZAR per USD", "USDZAR"},
		{"South Korean Won KRW per USD", "USDKRW"},
		{"Swedish Krona SEK per USD", "USDSEK"},
		{"Swiss Franc CHF per USD", "USDCHF"},
		{"New Taiwan Dollar TWD per USD", "USDTWD"},
		{"Thai Baht THB per USD", "USDTHB"},
		{"Turkish Lira TRY per USD", "USDTRY"},
		{"Euro USD per EUR", "EURUSD"},
		{"Philippine Peso PHP per USD", "USDPHP"},
	}

	desc, id := extractColumns(test, 0, 1, false)

	return prependList("CURRFX/", id), desc
}

func GetSharadarList(ctx context.Context) ([]string, []string) {
	list := loadTabDelimited(ctx, sharadarList)

	identifier, description := extractColumns(list, 0, 1, true)
	dimensions, _ := extractColumns(list, 2, 2, true)

	identifier, description = constructSharadarTickers(identifier, description, dimensions)

	return identifier, description
}

func GetSP500Multiples(ctx context.Context) ([]string, []string) {
	test := [][]string{
		{"MULTPL/SP500_EARNINGS_MONTH", "S&P 500 Earnings"},
		{"MULTPL/SP500_DIV_MONTH", "S&P 500 Dividend"},
		{"MULTPL/SP500_EARNINGS_YIELD_MONTH", "S&P 500 Earnings Yield"},
		{"MULTPL/SP500_DIV_YIELD_MONTH", "S&P 500 Dividend Yield"},
		{"MULTPL/SP500_PE_RATIO_MONTH", "S&P 500 PE Ratio"},
		{"MULTPL/SP500_REAL_PRICE_MONTH", "S&P 500 Real Price"},
		{"MULTPL/SHILLER_PE_RATIO_MONTH", "Shiller PE Ratio"},
	}

	id, desc := extractColumns(test, 0, 1, false)

	return id, desc
}

func constructSharadarTickers(identifier []string, description []string, dimensions []string) ([]string, []string) {
	newIdentifiers := make([]string, 0)
	newDescriptions := make([]string, 0)

	for i, _ := range identifier {
		// We don't need a duplicate price concept
		if identifier[i] == "PRICE" {
			continue
		}

		if dimensions[i] == "" {
			// Applies to things like shares and market capitalization
			newIdentifiers = append(newIdentifiers, identifier[i])
			newDescriptions = append(newDescriptions, description[i])
		}

		if strings.Contains(dimensions[i], "MRT") {
			newIdentifiers = append(newIdentifiers, identifier[i]+"_MRT")
			newDescriptions = append(newDescriptions, description[i]+" (TTM)")
		}

		if strings.Contains(dimensions[i], "MRQ") {
			newIdentifiers = append(newIdentifiers, identifier[i]+"_MRQ")
			newDescriptions = append(newDescriptions, description[i]+" (Quarterly)")
		}

		if strings.Contains(dimensions[i], "MRY") {
			newIdentifiers = append(newIdentifiers, identifier[i]+"_MRY")
			newDescriptions = append(newDescriptions, description[i]+" (Annual)")
		}

	}

	return newIdentifiers, newDescriptions
}

// Economic data (doesn't pertain to a particular security)
// GetEconomicDataList
func GetEconomicDataList_Old(ctx context.Context) ([]string, []string) {
	list := loadPipeDelimited(ctx, economicData)

	identifier, description := extractColumns(list, 0, 1, true)

	return prependList("FRED/", identifier), description
}

func GetEconomicDataList(ctx context.Context) ([]string, []string) {
	id := make([]string, 0)
	desc := make([]string, 0)

	data := loadJson(ctx, fredPopularJson)

	type FredRow struct {
		Id    string `json:"id"`
		Title string `json:"title"`
	}

	type Fred struct {
		Series []FredRow `json:"seriess"`
	}

	var unmarshalled Fred

	err := json.Unmarshal(data, &unmarshalled)

	if err == nil {
		for _, v := range unmarshalled.Series {
			id = append(id, v.Id)
			desc = append(desc, v.Title)
		}
	}

	return prependList("FRED/", id), desc
}

// Index membership
// GetSP500Constituents
func GetSP500Constituents(ctx context.Context) ([]string, []string) {
	//fmt.Printf("%s\n", spxConstituents)
	//list := loadCSVMac(spxConstituents)
	//list := loadCSV(spxConstituents)
	/*need to prepend col 0 ticker with WIKI*/

	temp, _ := csvCache.Retrieve(ctx, spxConstituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 0, 2, true)

	return prependList("WIKI/", identifier), description
}

func GetISharesPopular() ([]string, []string) {
	s := [][]string{{"GOOG/NYSEARCA_IVV", "iShares S&P 500 Index"},
		{"GOOG/NYSEARCA_EFA", "iShares MSCI EAFE Index Fund"},
		{"GOOG/NYSEARCA_EEM", "iShares MSCI Emerging Markets Indx"},
		{"GOOG/NYSEARCA_IWM", "iShares Russell 2000 Index"},
		{"GOOG/NYSEARCA_IJH", "iShares Core S&P Mid Cap ETF"},
		{"GOOG/NYSEARCA_HYG", "iShares iBoxx $ High Yid Corp Bond"},
		{"GOOG/NYSEARCA_LQD", "iShares IBoxx $ Invest Grade Corp Bd Fd"},
		{"GOOG/NYSEARCA_AGG", "iShares Barclays Aggregate Bond Fund"},
		{"GOOG/NYSEARCA_IJR", "iShares S&P SmallCap 600 Index"},
		{"GOOG/NYSEARCA_TIP", "iShares Barclays TIPS Bond Fund"},
		{"GOOG/NYSEARCA_IVW", "iShares S&P 500 Growth Index"},
		{"GOOG/NYSEARCA_IAU", "iShares Gold Trust"},
		{"GOOG/NYSEARCA_SLV", "iShares Silver Trust"},
		{"GOOG/NYSEARCA_IVE", "iShares S&P 500 Value Index"},
		{"GOOG/NYSEARCA_MBB", "iShares Barclays MBS Bond Fund"},
		{"GOOG/NASDAQ_ACWI", "iShares MSCI ACWI Index Fund"},
		{"GOOG/NYSEARCA_IYR", "iShares Dow Jones US Real Estate"},
		{"GOOG/NYSEARCA_SHV", "iShares Barclays Short Treasury Bond Fnd"},
		{"GOOG/NYSEARCA_ICF", "iShares Cohen & Steers Realty Maj."},
		{"GOOG/NASDAQ_ACWX", "iShares MSCI ACWI ex US Index Fund"},
		{"GOOG/NYSEARCA_GSG", "iShares S&P GSCI Commodity-Indexed"},
		{"GOOG/NYSEARCA_AGZ", "iShares Barclays Agency Bond Fund"},
		{"GOOG/NYSEARCA_GBF", "iShares Barclays Govnment/Cdit Bond Fd"},
		{"GOOG/AMEX_GHYG", "iShares Global High Yield Corporate Bond ETF"},
		{"GOOG/AMEX_CEMB", "iShares Emerging Markets Corporate Bond ETF"},
	}

	id, desc := extractColumns(s, 0, 1, false)

	return id, desc
}

func GetSectorETFs() ([]string, []string) {
	s := [][]string{
		{"GOOG/NYSEARCA_XLE", "Energy"},
		{"GOOG/NYSEARCA_XLU", "Utilities"},
		{"GOOG/NYSEARCA_XLK", "Technology"},
		{"GOOG/NYSEARCA_XLB", "Materials"},
		{"GOOG/NYSEARCA_XLP", "Consumer Staples"},
		{"GOOG/NYSEARCA_XLY", "Consumer Discretionary"},
		{"GOOG/NYSEARCA_XLI", "Industrials"},
		{"GOOG/NYSEARCA_XLV", "Health Care"},
		{"GOOG/NYSEARCA_XLF", "Financials"},
	}

	id, desc := extractColumns(s, 0, 1, false)

	return id, desc
}

// GetDowConstituents
func GetDowConstituents(ctx context.Context) ([]string, []string) {
	// list := loadCSV(dowConstituents)
	/*need to prepend col 0 ticker with WIKI*/
	temp, _ := csvCache.Retrieve(ctx, dowConstituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 0, 2, true)

	return prependList("WIKI/", identifier), description
}

// GetNasdaqCompositeConstituents
func GetNasdaqCompositeConstituents(ctx context.Context) ([]string, []string) {
	// list := loadCSV(nasdaqCompositeConstituents)
	/*need to prepend col 0 ticker with WIKI*/
	temp, _ := csvCache.Retrieve(ctx, nasdaqCompositeConstituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 0, 2, true)

	return prependList("WIKI/", identifier), description
}

// GetNasdaq100Constituents
func GetNasdaq100Constituents(ctx context.Context) ([]string, []string) {
	// list := loadCSV(nasdaq100Constituents)
	/*need to prepend col 0 ticker with WIKI*/
	temp, _ := csvCache.Retrieve(ctx, nasdaq100Constituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 0, 2, true)

	return prependList("WIKI/", identifier), description
}

// GetFTSE100Constituents
func GetFTSE100Constituents(ctx context.Context) ([]string, []string) {
	// list := loadCSV(ftse100Constituents)
	temp, _ := csvCache.Retrieve(ctx, ftse100Constituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 1, 2, true)

	return identifier, description
}

// Sector mappings
// GetSP500SectorMappings
func GetSP500SectorMappings(ctx context.Context) ([]string, []string, []string) {
	//list := loadCSVMac(spxConstituents)
	// list := loadCSV(spxConstituents)
	temp, _ := csvCache.Retrieve(ctx, spxConstituents)

	list := temp.([][]string)

	identifier, description := extractColumns(list, 0, 3, true)
	_, stockName := extractColumns(list, 0, 2, true)

	return prependList("WIKI/", identifier), appendList(description, " Stocks"), stockName
}

func GetSP500SectorNames(ctx context.Context) []string {
	_, description, _ := GetSP500SectorMappings(ctx)

	return unique(description)
}

func unique(input []string) []string {
	u := make(map[string]int)
	s := make([]string, 0, len(input))

	// Put the strings into the map
	for _, v := range input {
		u[v] = u[v] + 1
	}

	// Append all the keys into the map
	for v, _ := range u {
		s = append(s, v)
	}

	sort.Strings(s)

	return s
}

func prependList(prefix string, list []string) []string {
	newList := make([]string, len(list), len(list))

	for i, v := range list {
		newList[i] = fmt.Sprintf("%s%s", prefix, v)
	}

	return newList
}

func appendList(list []string, suffix string) []string {
	newList := make([]string, len(list), len(list))

	for i, v := range list {
		newList[i] = fmt.Sprintf("%s%s", v, suffix)
	}

	return newList
}
