package build

import (
	"context"
	"encoding/json"
	"os"
	"regexp"
	"strings"

	"github.com/AlphaHat/gcp-alpha-hat/component"

	"github.com/AlphaHat/gcp-alpha-hat/list"
	"github.com/AlphaHat/gcp-alpha-hat/timeseries"

	"github.com/AlphaHat/gcp-alpha-hat/quandl"
	"github.com/AlphaHat/gcp-alpha-hat/term"
)

func SetTimeRangeParameters(tr component.QueryComponent) component.QueryComponent {

	switch tr.QueryComponentCanonicalName {
	case "from {Date} to {Date}":
		tr.QueryComponentParams[0] = timeseries.ParseToFirstCD(tr.QueryComponentParams[0])
		tr.QueryComponentParams[1] = timeseries.ParseToLastCD(tr.QueryComponentParams[1])
	case "since {Date}":
		tr.QueryComponentParams[0] = timeseries.ParseToLastCD(tr.QueryComponentParams[0])
		tr.QueryComponentParams = append(tr.QueryComponentParams, timeseries.GetLastBD())
	case "on {Date}":
		tr.QueryComponentParams[0] = timeseries.ParseToFirstCD(tr.QueryComponentParams[0])
		tr.QueryComponentParams = append(tr.QueryComponentParams, timeseries.ParseToLastCD(tr.QueryComponentParams[0]))
	case "YTD":
		date1, date2 := timeseries.GetYTD()
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "LTM":
		date1, date2 := timeseries.GetLTM()
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Month":
		date1, date2 := timeseries.GetLastMonths(1)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Three Months":
		date1, date2 := timeseries.GetLastMonths(3)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Six Months":
		date1, date2 := timeseries.GetLastMonths(6)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Two Years":
		date1, date2 := timeseries.GetLastMonths(24)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Three Years":
		date1, date2 := timeseries.GetLastMonths(36)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Five Years":
		date1, date2 := timeseries.GetLastMonths(60)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	case "Last Ten Years":
		date1, date2 := timeseries.GetLastMonths(120)
		tr.QueryComponentParams = make([]string, 2, 2)
		tr.QueryComponentParams[0] = date1
		tr.QueryComponentParams[1] = date2
	}

	return tr
}

func ExtractQueryComponentExact(query string, terms *term.TermData, userTerms *term.TermData) component.QueryComponent {
	if userTerms != nil {
		userTermLookup := userTerms.Lookup(query)

		if userTermLookup != nil {
			v := userTermLookup.(component.QueryComponent)
			v.QueryComponentOriginalString = query
			return v
		}
	}

	noParams := terms.Lookup(query)

	if noParams != nil {
		v := noParams.(component.QueryComponent)
		v.QueryComponentOriginalString = query
		return v
	}

	withReplacedParameters, parameters := ParseParameters(query)
	withParams := terms.Lookup(withReplacedParameters)

	if withParams != nil {
		v := withParams.(component.QueryComponent)
		v.QueryComponentOriginalString = query
		//fmt.Printf("v=%s\nparameters=%q\n", v, parameters)
		componentArray := matchParametersToComponentsExact(parameters, []component.QueryComponent{v})
		if len(componentArray) > 0 {
			return componentArray[0]
		}
	}

	return component.QueryComponent{QueryComponentCanonicalName: "No Results", QueryComponentType: "No Results", QueryComponentName: "No Results", QueryComponentOriginalString: "No Results"}
}

// ParseParameters replaces things that looks like dates, numbers or percentages
// with {Date}, {Number}, or {Percentage}
func ParseParameters(input string) (string, []string) {
	params := make([]string, 0, len(input))
	locations := make([][]int, 0, len(input))

	// Well-formed dates
	re := regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)[\t\n\f\r ]*[0-9]{0,2}[\t\n\f\r ]*[,]?[ ]?[0-9]{4}")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Date}")

	// Abbreviated dates
	re = regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)[\t\n\f\r ]*[0-9]{0,2}[\t\n\f\r ]*[,]?[ ]?[0-9]{4}")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Date}")

	// Internet date standard
	re = regexp.MustCompile("(19|20)[0-9]{2}-[0-9]{1,2}-[0-9]{1,2}")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Date}")

	// Just years
	re = regexp.MustCompile("(19|20)[0-9]{2}")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Date}")

	// Percentages
	re = regexp.MustCompile("-?[0-9]+(.[0-9]+)?%")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Percentage}")

	// Other numbers
	re = regexp.MustCompile("-?[0-9]+(.[0-9]+)?")
	params = append(params, re.FindAllString(input, -1)...)
	locations = append(locations, re.FindAllStringIndex(input, -1)...)
	input = re.ReplaceAllString(input, "{Number}")

	newParams := make([]string, len(input), len(input))

	// Here we are reodering the parameters based on their locations. A parameter earlier should
	// show up earlier in the parameter list
	for i, v := range locations {
		newParams[v[0]] = params[i]
	}

	i := 0
	for _, v := range newParams {
		if v != "" {
			params[i] = v
			i++
		}
	}

	//fmt.Printf("%s\n%s\n", params, locations)

	return input, params
}

func matchParametersToComponentsExact(parameters []string, components []component.QueryComponent) []component.QueryComponent {
	var valence int
	k := 0 // This is the position of the parameter
	for i, v := range components {
		valence = strings.Count(v.QueryComponentName, "{")
		components[i].QueryComponentParams = make([]string, valence, valence)

		for j := 0; j < valence; j++ {
			//startIndex := strings.Index(components[i].QueryComponentName, "{")
			//endIndex := strings.Index(components[i].QueryComponentName, "}")
			// Some of the input parameters come in as blank so we find only the populated ones and match
			// them up into the correct parameter
			components[i].QueryComponentParams[j] = findNonBlank(parameters, k)

			k++
		}

	}

	return components
}

func findNonBlank(parameters []string, z int) string {
	j := 0
	for _, v2 := range parameters {
		if v2 != "" {
			if j == z {
				return v2
			}
			j++
		}
	}

	return ""
}

func GetBaseComponents() []component.QueryComponent {
	components := []component.QueryComponent{
		component.QueryComponent{1, "from {Date} to {Date}", "from {Date} to {Date}", "Time Range", "", "", "from 2013 to 2014", nil},
		component.QueryComponent{2, "since {Date}", "since {Date}", "Time Range", "", "", "since 2013", nil},
		component.QueryComponent{3, "on {Date}", "on {Date}", "Time Range", "", "", "on 2014-12-31", nil},
		component.QueryComponent{4, "YTD", "YTD", "Time Range", "", "", "", nil},
		component.QueryComponent{4, "YTD", "Year-to-Date", "Time Range", "", "", "", nil},
		component.QueryComponent{5, "LTM", "LTM", "Time Range", "", "", "", nil},
		component.QueryComponent{5, "LTM", "Last Twelve Months", "Time Range", "", "", "", nil},
		component.QueryComponent{6, "S&P 500 Stocks", "Stocks", "Universe: Expandable", "", "", "", nil},
		component.QueryComponent{6, "S&P 500 Stocks", "S&P 500 Stocks", "Universe: Expandable", "", "", "", nil},
		//component.QueryComponent{7, "Equal Weight", "Equal", "Rebalance Methodology", "", "", "", nil},
		//component.QueryComponent{8, "Value Weight", "Value", "Rebalance Methodology", "", "", "", nil},
		//component.QueryComponent{8, "Value Weight", "Cap", "Rebalance Methodology", "", "", "", nil},
		//component.QueryComponent{9, "Yearly", "Yearly", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{9, "Yearly", "Annually", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{10, "Quarterly", "Quarterly", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{11, "Monthly", "Monthly", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{12, "Weekly", "Weekly", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{13, "Daily", "Daily", "Rebalance Frequency", "", "", "", nil},
		//component.QueryComponent{14, "Regression", "Regress", "Report Type", "", "", "", nil},
		//component.QueryComponent{14, "Regression", "Regression", "Report Type", "", "", "", nil},
		//component.QueryComponent{15, "Performance", "Performance", "Report Type", "", "", "", nil},
		//component.QueryComponent{16, "Histogram", "Histogram", "Report Type", "", "", "", nil},
		//component.QueryComponent{17, "Total Return", "Total Return", "Concept: Security", "Total Return", "Computed", "", nil},
		//component.QueryComponent{17, "Total Return", "Return", "Concept: Security", "Total Return", "Computed", "", nil},
		component.QueryComponent{18, "Price", "Price", "Concept: Security", "", "", "", nil},
		component.QueryComponent{18, "Price", "Stock Price", "Concept: Security", "", "", "", nil},
		component.QueryComponent{18, "Price", "Value", "Concept: Security", "", "", "", nil},
		component.QueryComponent{19, "Volume", "Volume", "Concept: Security", "", "", "", nil},
		component.QueryComponent{20, "Short Interest", "Short Interest", "Concept: Security", "", "", "", nil},
		component.QueryComponent{21, "Days to Cover", "Days to Cover", "Concept: Security", "", "", "", nil},
		//component.QueryComponent{22, "By Industry", "By Industry", "Classification", "", "", "", nil},
		//component.QueryComponent{22, "with", "with", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{23, "By Sector", "By Sector", "Classification", "", "", "", nil},
		//component.QueryComponent{24, component.Yearly, "By Year", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{24, component.Yearly, "Yearly", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{25, component.Quarterly, "By Quarter", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{25, component.Quarterly, "Quarterly", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{26, component.Monthly, "By Month", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{26, component.Monthly, "Monthly", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{27, "By Decade", "By Decade", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{27, component.Daily, "Daily", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{28, component.Weekly, "Weekly", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{29, "Percentage Change", "Percentage Change", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{30, "Cumulative Change", "Cumulative Change", component.TimeAggregationFrequency, "", "", "", nil},
		component.QueryComponent{31, "Align Last Day", "Align Last Day", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{28, component.Weekly, "By Week", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{29, "Historical", "Historical", "Context Clue", "", "", "", nil},
		//component.QueryComponent{30, "Event Study", "Event Study:", component.ReportType, "", "", "", nil},
		//component.QueryComponent{31, "Top {Number}", "Top {Number}", "Filter", "", "", "Top 5", nil},
		//component.QueryComponent{32, "Bottom {Number}", "Bottom {Number}", "Filter", "", "", "Bottom 5", nil},
		//component.QueryComponent{33, "Sum", "Sum", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{33, "Sum", "Total", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{34, "Average", "Average", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{34, "Average", "Mean", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{35, "Count", "Count", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{35, "Count", "Number Of", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{36, "Min", "Minimum", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{36, "Min", "Min", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{37, "Max", "Maximum", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{37, "Max", "Max", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{38, "Median", "Median", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{39, "By", "By", "Context Clue", "", "", "", nil},
		//component.QueryComponent{40, "My", "My", "Universe", "", "", "", nil},
		//component.QueryComponent{41, "All Time", "All Time", component.TimeAggregationFrequency, "", "", "", nil},
		//component.QueryComponent{42, "Geometric", "Geometric", component.TimeAggregationType, "", "", "", nil},
		//component.QueryComponent{43, "and", "and", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{43, "of", "of", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{45, "Cumulative Arithmetic Sum", "Cumulative Arithmetic Sum", component.TimeAggregationType, "", "", "", nil},
		//component.QueryComponent{46, "against", "against", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{47, "versus", "vs.", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{48, "versus", "versus", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{49, "Spread", "Spread", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{50, "between", "between", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{51, "Cumulative Geometric", "Cumulative Geometric", component.TimeAggregationType, "", "", "", nil},
		component.QueryComponent{52, "iShares Popular ETFs", "iShares Popular ETFs", component.UniverseExpandable, "", "", "", nil},
		component.QueryComponent{53, "Sector ETFs", "Sector ETFs", component.UniverseExpandable, "", "", "", nil},
		//component.QueryComponent{53, "Quintile", "Quintile", component.Classification, "", "", "", nil},
		//component.QueryComponent{54, "Quartile", "Quartile", component.Classification, "", "", "", nil},
		//component.QueryComponent{55, "Segmented Into", "Segmented Into", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{56, "Interquartile Range", "Interquartile Range", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{57, "Boxplot", "Boxplot", component.Aggregation, "", "", "", nil},
		//component.QueryComponent{58, "Greater Than {Number}", "Greater Than {Number}", component.Filter, "", "", "Greater Than 30", nil},
		//component.QueryComponent{59, "Less Than {Number}", "Less Than {Number}", component.Filter, "", "", "Less Than 30", nil},
		component.QueryComponent{60, "Last Month", "Last Month", component.TimeRange, "", "", "", nil},
		component.QueryComponent{61, "Last Three Months", "Last Three Months", component.TimeRange, "", "", "", nil},
		component.QueryComponent{62, "Last Six Months", "Last Six Months", component.TimeRange, "", "", "", nil},
		component.QueryComponent{81, "Last Two Years", "Last Two Years", component.TimeRange, "", "", "", nil},
		component.QueryComponent{82, "Last Three Years", "Last Three Years", component.TimeRange, "", "", "", nil},
		component.QueryComponent{63, "Last Five Years", "Last Five Years", component.TimeRange, "", "", "", nil},
		component.QueryComponent{64, "Last Ten Years", "Last Ten Years", component.TimeRange, "", "", "", nil},
		component.QueryComponent{65, "Last Data Point", "Last Data Point", component.TimeRange, "", "", "", nil},
		component.QueryComponent{66, "All Available", "All Available", component.TimeRange, "", "", "", nil},
		//component.QueryComponent{65, "{Number}-days after", "{Number}-days after", component.TimeHorizon, "", "", "90-days after", nil},
		//component.QueryComponent{66, "During", "During", component.TimeHorizon, "", "", "", nil},
		//component.QueryComponent{67, ",", ",", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{68, "{Number}-Month Forward Return", "{Number}-Month Forward Return", component.ConceptSecurity, "", "Computed", "12-Month Forward Return", nil},
		//component.QueryComponent{49, "Divided By", "Divided By", component.Aggregation, "", "", "", nil},
		component.QueryComponent{69, component.EventAggregationDate, component.EventAggregationDate, component.EventAggregationType, "", "", "", nil},
		component.QueryComponent{70, component.EventAggregationStock, component.EventAggregationStock, component.EventAggregationType, "", "", "", nil},
		component.QueryComponent{71, "Currencies", "Currencies", component.UniverseExpandable, "", "", "", nil},
		component.QueryComponent{72, "Commodities", "Commodities", component.UniverseExpandable, "", "", "", nil},
		//component.QueryComponent{73, "in", "in", component.ContextClue, "", "", "", nil},
		//component.QueryComponent{74, "Simple Alpha", "Simple Alpha", component.ConceptSecurity, "Simple Alpha", "Computed", "", nil},
		//component.QueryComponent{75, "Market-Neutral Beta", "Market-Neutral Beta", component.TimeAggregationType, "Market-Neutral Beta", "Computed", "", nil},
		//component.QueryComponent{76, "Market Beta", "Market Beta", component.TimeAggregationType, "Market Beta", "Computed", "", nil},
		//component.QueryComponent{77, "Regression Beta", "Regression Beta", component.TimeAggregationType, "Regression Beta", "Computed", "", nil},
		//component.QueryComponent{78, "Days Since {Number} Percent Drop", "Days Since {Number} Percent Drop", component.ConceptSecurity, "Days Since {Number} Percent Drop", "Computed", "Days Since 10 Percent Drop", nil},
		component.QueryComponent{79, "Union", "Union", "Combine Data", "Union", "Computed", "Union", nil},
		component.QueryComponent{80, "Field {Number}", "Field {Number}", component.Field, "", "", "Field 1", nil},
		// component.QueryComponent{81, "Revenues (Thomson)", "Revenues (Thomson)", component.ConceptSecurity, "SREV", component.Thomson, "Revenues (Thomson)", nil},
	}

	return components
}

func ReadQuandlConcepts(ctx context.Context) (*term.TermData, error) {
	//newTrie := patricia.NewTrie()
	newTermData := term.NewTermData()

	// Base Components
	baseComponents := GetBaseComponents()

	for _, v := range baseComponents {
		//newTrie.Insert(patricia.Prefix(strings.Trim(strings.ToLower(v.QueryComponentName), " ")), v)
		newTermData.Insert(v.QueryComponentName, v)
	}

	// Quandl Concepts
	quandl.SetAuthToken(os.Getenv("QUANDL_KEY"))

	offset := len(baseComponents)

	identifier, description := quandl.GetPriceOnlySeries(ctx)

	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe: Macro", component.QuandlOpenData, "")

	// identifier, description = quandl.GetStockList()
	identifier, description = list.GetConstituents(list.MAX_RANK)

	offset = insertIdentifiers(offset, newTermData, identifier, description, "Universe", component.QuandlOpenData, "")
	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe", component.QuandlOpenData, "")

	// Others (foreign stocks where we're not using a ticker)
	identifier, description = list.GetOthers()
	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe", component.QuandlOpenData, "")

	// sector := quandl.GetSP500SectorNames()
	sector := list.GetSectorNames(list.Sector)

	offset = insertIdentifierAndDescription(offset, newTermData, sector, sector, "Universe: Expandable", component.QuandlOpenData, "")

	industryGroup := list.GetSectorNames(list.IndustryGroup)

	offset = insertIdentifierAndDescription(offset, newTermData, industryGroup, industryGroup, "Universe: Expandable", component.QuandlOpenData, "")

	industry := list.GetSectorNames(list.Industry)

	offset = insertIdentifierAndDescription(offset, newTermData, industry, industry, "Universe: Expandable", component.QuandlOpenData, "")

	subIndustry := list.GetSectorNames(list.SubIndustry)

	offset = insertIdentifierAndDescription(offset, newTermData, subIndustry, subIndustry, "Universe: Expandable", component.QuandlOpenData, "")

	//identifier, description = quandl.GetFinancialRatiosList()

	//offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Damodaran, "Annual ")

	identifier, description = quandl.GetSharadarList(ctx)

	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Sharadar, "")

	// td := thomson.GetData("AAPL")
	// identifier, description = td.GetCoaMap()
	//
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Thomson, " (Thomson)")

	// description, identifier = capiq.SplitIntoTwoLists(capiq.GetCapIQCodes)
	//
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.CapIQ, " (Quarterly, S&P)")
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.CapIQ, " (Annual, S&P)")
	//
	// description, identifier = capiq.SplitIntoTwoLists(capiq.FilterTTM(capiq.GetCapIQCodes))
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.CapIQ, " (TTM, S&P)")
	//
	// description, identifier = factset.SplitIntoTwoLists(factset.GetFactsetCodes)
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Factset, " (Quarterly, Factset)")
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Factset, " (Annual, Factset)")
	//
	// description, identifier = factset.SplitIntoTwoLists(factset.FilterTTM(factset.GetFactsetCodes))
	// offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.Factset, " (TTM, Factset)")

	//identifier, description = quandl.GetSECHarmonizedFields()

	//offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Concept: Security", component.SECHarmonized, " (Quarterly, XBRL)")

	identifier, description = quandl.GetEconomicDataList(ctx)

	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe: Macro", component.QuandlOpenData, "")

	identifier, description = quandl.GetCurrenciesList(ctx)

	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe: Macro", component.QuandlOpenData, "")

	identifier, description = quandl.GetSP500Multiples(ctx)

	offset = insertIdentifierAndDescription(offset, newTermData, identifier, description, "Universe: Macro", component.QuandlOpenData, "")

	// // Events
	// events := event.GetEventList()

	// offset = insertIdentifierAndDescription(offset, newTermData, events, events, component.Event, "Alpha Hat Event Database", "")

	// // News Classifications
	// newsClassifications := classifier.GetUniques()
	// offset = insertIdentifierAndDescription(offset, newTermData, newsClassifications, newsClassifications, "News Classification", "Alpha Hat", "")

	return newTermData, nil
}

func insertIdentifierAndDescription(offset int, t *term.TermData, identifier []string, description []string, componentType string, source string, appendString string) int {
	for i, v := range description {
		// Replace &amp; with &

		// Only insert continued series
		if !strings.Contains(v, "DISCONTINUED") {
			v = strings.Replace(v, "&amp;", "&", -1)
			description[i] = strings.Replace(description[i], "&amp;", "&", -1)
			//if description[i] > 3 {
			name := description[i] + appendString
			t.Insert(name, component.QueryComponent{i + offset, name, name, componentType, identifier[i], source, "", nil})
			//}
		}
	}

	return offset + len(description)
}

// insertIdentifiers sends the ticker as the name but the description as the canonical name
func insertIdentifiers(offset int, t *term.TermData, identifier []string, description []string, componentType string, source string, prepend string) int {
	for i, v := range description {
		// Replace &amp; with &
		v = strings.Replace(v, "&amp;", "&", -1)
		description[i] = strings.Replace(description[i], "&amp;", "&", -1)
		if len(v) > 4 {
			name := strings.Join([]string{prepend, description[i]}, "")
			re, err := regexp.Compile("/([A-Za-z]+)")

			// fmt.Printf("identifier = %s\n", identifier[i])
			if err == nil {
				friendlyId := re.FindString(identifier[i])
				friendlyId = friendlyId[1:]
				// fmt.Printf("friendlyId=%s\n", friendlyId)
				if friendlyId != "SUM" {
					t.Insert(friendlyId, component.QueryComponent{i + offset, name, friendlyId, componentType, identifier[i], source, "", nil})
				}
			}
		}
	}

	return offset + len(description)
}

func GetAllTerms(terms *term.TermData, incoming string) []byte {
	filterParts := strings.Split(incoming, "|")
	var query, componentFilter, sourceFilter string
	if len(filterParts) > 0 {
		query = filterParts[0]
	}
	if len(filterParts) > 1 {
		componentFilter = filterParts[1]
	}
	if len(filterParts) > 2 {
		sourceFilter = filterParts[2]
	}
	var numResults int = 20
	if query == "" {
		if componentFilter == component.ConceptSecurity {
			numResults = 100
		} else {
			numResults = 40
		}
	}

	if componentFilter == component.CustomQuandlCode {
		cg := []component.ComponentGroup{
			component.ComponentGroup{
				Name: component.CustomQuandlCode,
				Children: []component.QueryComponent{
					component.QueryComponent{
						0,
						query,
						query,
						component.CustomQuandlCode,
						query,
						component.QuandlOpenData,
						query,
						nil,
					},
				},
			},
		}

		m, _ := json.Marshal(cg)

		return m
	} else if componentFilter == component.TimeSeriesFormula {
		cg := []component.ComponentGroup{
			component.ComponentGroup{
				Name: component.TimeSeriesFormula,
				Children: []component.QueryComponent{
					component.QueryComponent{
						0,
						query,
						query,
						component.TimeSeriesFormula,
						query,
						component.QuandlOpenData,
						query,
						nil,
					},
				},
			},
		}

		m, _ := json.Marshal(cg)

		return m
	} else if componentFilter == component.RemoveData {
		cg := []component.ComponentGroup{
			component.ComponentGroup{
				Name: component.RemoveData,
				Children: []component.QueryComponent{
					component.QueryComponent{
						0,
						query,
						query,
						component.RemoveData,
						query,
						component.QuandlOpenData,
						query,
						nil,
					},
				},
			},
		}

		m, _ := json.Marshal(cg)

		return m
	} else if componentFilter == component.FreeText {
		cg := []component.ComponentGroup{
			component.ComponentGroup{
				Name: component.FreeText,
				Children: []component.QueryComponent{
					component.QueryComponent{
						0,
						query,
						query,
						component.FreeText,
						query,
						component.QuandlOpenData,
						query,
						nil,
					},
				},
			},
		}

		m, _ := json.Marshal(cg)

		return m
		// } else if sourceFilter == "Stock-Moving Events" {
		// 	cg := []component.ComponentGroup{
		// 		component.ComponentGroup{
		// 			Name:     component.Universe,
		// 			Children: make([]component.QueryComponent, 0),
		// 		},
		// 	}
		//
		// 	tickerList := news.GetStockMovingEventTickers()
		//
		// 	addItem := func(prefix term.Prefix, item term.Item) error {
		// 		currentComponent := item.(component.QueryComponent)
		//
		// 		tickerComponents := strings.Split(currentComponent.QueryComponentProviderId, "/")
		// 		var ticker string
		//
		// 		if len(tickerComponents) > 0 {
		// 			ticker = tickerComponents[len(tickerComponents)-1]
		// 		} else {
		// 			ticker = currentComponent.QueryComponentProviderId
		// 		}
		//
		// 		tickerIdx := sort.SearchStrings(tickerList, ticker)
		// 		if ticker != "" && tickerIdx < len(tickerList) && tickerList[tickerIdx] == ticker {
		// 			currentComponent.QueryComponentOriginalString = currentComponent.QueryComponentName
		// 			cg[0].Children = append(cg[0].Children, currentComponent)
		// 		}
		//
		// 		return nil
		// 	}
		//
		// 	terms.VisitSubtreeExact(strings.ToLower(query), addItem)
		// 	terms.VisitSubtreeSubstring(strings.ToLower(query), addItem)
		//
		// 	m, _ := json.Marshal(cg)
		//
		// 	return m
	}

	componentGroup := make([]component.ComponentGroup, 0, 10)
	replaced, parameters := ParseParameters(query)

	addItem := func(prefix term.Prefix, item term.Item) error {
		currentComponent := item.(component.QueryComponent)

		valence := strings.Count(currentComponent.QueryComponentName, "{")

		if len(parameters) == valence && valence > 0 {
			// Correct Number of parameters
			currentComponent.QueryComponentOriginalString = query
			currentComponent.QueryComponentParams = parameters
		} else if len(parameters) < valence {
			// Need to fill in the extra parameters

		} else {
			currentComponent.QueryComponentOriginalString = currentComponent.QueryComponentName
		}

		// If there's a component filter set, the current component should match the component filter
		// before proceeding.
		if componentFilter != "" && currentComponent.QueryComponentType != componentFilter {
			return nil
		}

		// If there's a source filter set, the current component should match the source filter
		// before proceeding.
		if sourceFilter != "" && currentComponent.QueryComponentSource != sourceFilter {
			return nil
		}

		// If there's no source filter, don't include Thomson data or CapIQ data
		if sourceFilter == "" && (currentComponent.QueryComponentSource == component.Thomson || currentComponent.QueryComponentSource == component.CapIQ || currentComponent.QueryComponentSource == component.Factset || currentComponent.QueryComponentSource == component.Custom) {
			if componentFilter == component.ConceptSecurity {
				return nil
			}
		}

		indexFound := findInComponentGroup(currentComponent.QueryComponentType, componentGroup)
		if indexFound == -1 {
			componentGroup = append(componentGroup, component.ComponentGroup{currentComponent.QueryComponentType, make([]component.QueryComponent, 0, numResults)})
			indexFound = len(componentGroup) - 1
		}
		if len(componentGroup[indexFound].Children) < cap(componentGroup[indexFound].Children) {
			componentGroup[indexFound].Children = append(componentGroup[indexFound].Children, currentComponent)
		}
		return nil
	}

	// Try with parameters parsed
	//fmt.Printf("parameters=%s, len(parameters)=%v\n", parameters, len(parameters))
	// Need this to make sure that we don't get double results when trying something that has no parameters
	if len(parameters) > 0 {
		terms.VisitSubtreeExact(strings.ToLower(replaced), addItem)
		terms.VisitSubtreeSubstring(strings.ToLower(replaced), addItem)
	}
	// Try with parameters as original
	terms.VisitSubtreeExact(strings.ToLower(query), addItem)
	terms.VisitSubtreeSubstring(strings.ToLower(query), addItem)

	m, _ := json.Marshal(componentGroup)

	return m
}

func findInComponentGroup(groupName string, group []component.ComponentGroup) int {
	for i, v := range group {
		if groupName == v.Name {
			return i
		}
	}

	return -1
}
