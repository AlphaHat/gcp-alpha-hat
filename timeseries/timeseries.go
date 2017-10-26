package timeseries

import (
	//"fmt"
	// "log"
	"log"
	"regexp"
	"sort"
	"time"
)

type TimeSeries struct {
	Date        []time.Time
	DateString  []string
	Data        []float64
	Name        string
	DisplayName string
	Source      string
	Units       string
	DataEnd     string
}

func (t *TimeSeries) Len() int {
	return len(t.Date)
}

func (t *TimeSeries) Swap(i, j int) {
	t.Data[i], t.Data[j] = t.Data[j], t.Data[i]
	t.Date[i], t.Date[j] = t.Date[j], t.Date[i]
	t.DateString[i], t.DateString[j] = t.DateString[j], t.DateString[i]
}

func (t *TimeSeries) Less(i, j int) bool {
	return t.Date[i].Before(t.Date[j])
}

func NewTimeSeries(dates []string, data []float64, name string, displayName string) *TimeSeries {
	t := new(TimeSeries)

	t.Date = make([]time.Time, len(dates), len(dates))

	for i, v := range dates {
		t.Date[i], _ = ParseDate(v)
	}
	t.DateString = dates
	t.Data = data
	t.Name = name
	t.DisplayName = displayName

	sort.Sort(t)

	return t
}

func ParseDate(date string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", date)

	if err == nil {
		return t, err
	}

	t, err = time.Parse("01/02/2006", date)

	if err == nil {
		return t, err
	}

	t, err = time.Parse("1/2/2006", date)

	if err == nil {
		return t, err
	}

	t, err = time.Parse("02-Jan-2006", date)

	return t, err
}

func (t *TimeSeries) GetData(date string, fillMethod string) float64 {
	timeToFind, _ := ParseDate(date)

	i := sort.Search(t.Len(), func(i int) bool {
		if t.Date[i].Equal(timeToFind) || t.Date[i].After(timeToFind) {
			return true
		}

		if fillMethod == "Previous" && i < t.Len()-1 && t.Date[i].Before(timeToFind) && t.Date[i+1].After(timeToFind) {
			return true
		}

		return false
	})

	// If i is the length, then that means this was not in there at all
	// If fill method is not previous, make sure that there is an exact date match (just a stupid quirk of sort.Search)
	// If fill method is previous, then return the closest match found value
	if i != t.Len() && (t.Date[i].Equal(timeToFind) || (fillMethod == "Previous" && t.Date[i].Before(timeToFind))) {
		return t.Data[i]
	}

	// for i, v := range t.Date {
	// 	if v == timeToFind {
	// 		return t.Data[i]
	// 	}

	// 	if fillMethod == "Previous" && v.After(timeToFind) {
	// 		if i >= 1 {
	// 			return t.Data[i-1]
	// 		}
	// 		return 0.0
	// 	}

	// }

	//fmt.Printf("Date %q not found\n", date)
	// This should eventually return the correct value depending on how the upsample settings are set
	// I.e. how should this series be filled? 0's, nan's previous value, etc
	if fillMethod == "Previous" && len(t.Data) >= 1 && t.Date[len(t.Date)-1].Before(timeToFind) {
		return t.Data[len(t.Data)-1]
	}
	return 0.0
}

func GetWeekdaysBetween(dateStart string, dateEnd string) []string {
	dateStartTime, dateEndTime := convertDateStringToDate(dateStart, dateEnd)

	// If the start date and end date are the same, return the date
	if dateStart == dateEnd {
		return []string{dateStart}
	}

	days := dateEndTime.Sub(dateStartTime) / (time.Hour * 24)
	// Prepare number of days
	timePoints := make([]string, 0, days)

	for currentTime := dateStartTime; currentTime.Before(dateEndTime); currentTime = currentTime.AddDate(0, 0, 1) {
		if currentTime.Weekday() == time.Sunday || currentTime.Weekday() == time.Saturday {
			continue
		}

		timePoints = append(timePoints, currentTime.String()[0:10])
	}

	return timePoints
}

func GetWeeklyDatesBetween(dateStart string, dateEnd string) []string {
	dateStartTime, dateEndTime := convertDateStringToDate(dateStart, dateEnd)

	dateArray := make([]string, 0)

	for currentDate := dateStartTime; currentDate.Before(dateEndTime); currentDate = currentDate.AddDate(0, 0, 1) {
		if currentDate.Weekday() == time.Friday {
			dateArray = append(dateArray, currentDate.String()[0:10])
		}
	}

	return dateArray
}

func GetMonthlyDatesBetween(dateStart string, dateEnd string) []string {
	dateStartTime, dateEndTime := convertDateStringToDate(dateStart, dateEnd)

	dateArray := make([]string, 0)

	// Start with the beggining date of the next month and then subtract 1 day
	for currentDate := time.Date(dateStartTime.Year(), dateStartTime.Month()+1, 0, 0, 0, 0, 0, time.Local); currentDate.Before(dateEndTime); currentDate = currentDate.AddDate(0, 1, 0) {
		eomDate := currentDate.AddDate(0, 0, -1)
		dateArray = append(dateArray, eomDate.String()[0:10])
	}

	return dateArray
}

func convertDateStringToDate(dateStart string, dateEnd string) (time.Time, time.Time) {
	dateStartTime, error1 := ParseDate(dateStart)
	dateEndTime, error2 := ParseDate(dateEnd)

	if error1 != nil {
		// log.Printf("error parsing %s\n as date", dateStart)
		return time.Now(), time.Now()
	}

	if error2 != nil {
		// log.Printf("error parsing %s\n as date", dateEnd)
		return time.Now(), time.Now()
	}

	return dateStartTime, dateEndTime
}

func GetYearlyDatesBetween(dateStart string, dateEnd string) []string {
	dateStartTime, dateEndTime := convertDateStringToDate(dateStart, dateEnd)

	years := dateEndTime.Year() - dateStartTime.Year() + 1

	timePoints := make([]string, years, years)

	currentYear := dateStartTime.Year()
	for i, _ := range timePoints {
		timePoints[i] = time.Date(currentYear, 12, 31, 0, 0, 0, 0, time.UTC).String()[0:10]
		//timePoints[i] = time.Date(currentYear, 01, 01, 0, 0, 0, 0, time.UTC).String()[0:10]
		currentYear++
	}

	// If the range includes this year, don't go to the end of the year,
	// go to the last business day
	if dateEndTime.Year() == time.Now().Year() {
		timePoints[len(timePoints)-1] = GetLastBD()
	}

	return timePoints
}

func GetQuarterlyDatesBetween(dateStart string, dateEnd string) []string {
	dateStartTime, error1 := ParseDate(dateStart)
	dateEndTime, error2 := ParseDate(dateEnd)

	if error1 != nil {
		// log.Printf("error parsing %s\n as date", dateStart)
		return nil
	}

	if error2 != nil {
		// log.Printf("error parsing %s\n as date", dateEnd)
		return nil
	}

	var quarters int64 = int64(((dateEndTime.Sub(dateStartTime).Hours() / 24) / (7 * 13))) + 1

	timePoints := make([]string, quarters, quarters+1)

	var months []int = []int{12, 3, 6, 9}
	var days []int = []int{31, 31, 30, 30}

	currentDate := dateStartTime
	for i, _ := range timePoints {
		quarterNum := int64((currentDate.Month() - 1) / 3)
		var yearNum int
		if quarterNum == 0 {
			yearNum = currentDate.Year() - 1
		} else {
			yearNum = currentDate.Year()
		}
		timePoints[i] = time.Date(yearNum, time.Month(months[quarterNum]), days[quarterNum], 0, 0, 0, 0, time.UTC).String()[0:10]
		currentDate = currentDate.AddDate(0, 3, 0)
	}

	// If the range is beyond today's date, set it to the last Business Day
	//if dateEndTime.After(time.Now()) {
	//	timePoints[len(timePoints)-1] = GetLastBD()
	//}

	if dateStartTime != dateEndTime {
		timePoints = append(timePoints, GetLastBD())
	}

	return timePoints
}

func GetPreviousWeekday(currentDate string) string {
	currentDateTime, _ := ParseDate(currentDate)

	if currentDateTime.Weekday() == time.Monday {
		return currentDateTime.AddDate(0, 0, -3).String()[0:10]
	} else {
		return currentDateTime.AddDate(0, 0, -1).String()[0:10]
	}
}

func parseDateAndSnap(date string, snapMethod string) string {
	var t time.Time
	var err error

	if regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)[\t\n\f\r ]*[0-9]{1,2}[\t\n\f\r ]*[0-9]{4}").MatchString(date) {
		// Day specified
		t, err = time.Parse("January 2 2006", date)
		if err != nil {
			return date
		}
	} else if regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)[\t\n\f\r ]*[0-9]{1,2}[\t\n\f\r ]*[0-9]{4}").MatchString(date) {
		// Day specified
		t, err = time.Parse("Jan 2 2006", date)
		if err != nil {
			return date
		}
	} else if regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)[\t\n\f\r ]*[0-9]{1,2}[\t\n\f\r ]*[,][ ][0-9]{4}").MatchString(date) {
		// Day specified
		t, err = time.Parse("January 2, 2006", date)
		if err != nil {
			return date
		}
	} else if regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)[\t\n\f\r ]*[0-9]{1,2}[\t\n\f\r ]*[,][ ][0-9]{4}").MatchString(date) {
		// Day specified
		t, err = time.Parse("Jan 2, 2006", date)
		if err != nil {
			return date
		}
	} else if regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)[\t\n\f\r ]*[0-9]{4}").MatchString(date) {
		// Day not specified
		t, err = time.Parse("January 2006", date)
		if err != nil {
			return date
		}
		switch snapMethod {
		case "Last CD":
			t = t.AddDate(0, 1, -1)
		}
	} else if regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)[\t\n\f\r ]*[0-9]{4}").MatchString(date) {
		// Day not specified
		t, err = time.Parse("Jan 2006", date)
		if err != nil {
			return date
		}
		switch snapMethod {
		case "Last CD":
			t = t.AddDate(0, 1, -1)
		}
	} else if regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)[\t\n\f\r ]*[0-9]{1,2}").MatchString(date) {
		// Year not specified
		t, err = time.Parse("January 2", date)
		if err != nil {
			return date
		}

		// Add the current year
		t = t.AddDate(time.Now().Year(), 0, 0)
		if t.After(time.Now()) {
			// If the current year makes this a future date, set the year back 1
			t = t.AddDate(-1, 0, 0)
		}

	} else if regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)[\t\n\f\r ]*[0-9]{1,2}").MatchString(date) {
		// Year not specified
		t, err = time.Parse("Jan 2", date)
		if err != nil {
			return date
		}

		// Add the current year
		t = t.AddDate(time.Now().Year(), 0, 0)
		if t.After(time.Now()) {
			// If the current year makes this a future date, set the year back 1
			t = t.AddDate(-1, 0, 0)
		}
	} else if regexp.MustCompile("((j|J)anuary|(f|F)ebruary|(m|M)arch|(a|A)pril|(m|M)ay|(j|J)une|(j|J)uly|(a|A)ugust|(s|S)eptember|(o|O)ctober|(n|N)ovember|(d|D)ecember)").MatchString(date) {
		// Only month specified
		t, err = time.Parse("January", date)
		if err != nil {
			return date
		}

		// Add the current year
		t = t.AddDate(time.Now().Year(), 0, 0)
		switch snapMethod {
		case "Last CD":
			t = t.AddDate(0, 1, -1)
		}
		if t.After(time.Now()) {
			// If the current year makes this a future date, set the year back 1
			t = t.AddDate(-1, 0, 0)
		}
	} else if regexp.MustCompile("((j|J)an|(f|F)eb|(m|M)ar|(a|A)pr|(m|M)ay|(j|J)un|(j|J)ul|(a|A)ug|(s|S)ept|(s|S)ep|(o|O)ct|(n|N)ov|Dec)").MatchString(date) {
		// Only month specified
		t, err = time.Parse("Jan", date)
		if err != nil {
			return date
		}
		// Add the current year
		t = t.AddDate(time.Now().Year(), 0, 0)
		switch snapMethod {
		case "Last CD":
			t = t.AddDate(0, 1, -1)
		}
		if t.After(time.Now()) {
			// If the current year makes this a future date, set the year back 1
			t = t.AddDate(-1, 0, 0)
		}
	} else if regexp.MustCompile("(19|20)[0-9]{2}-[0-9]{1,2}-[0-9]{1,2}").MatchString(date) {
		//fmt.Printf("Internet standard\n")
		t, err = time.Parse("2006-1-2", date)
		if err != nil {
			return date
		}
	} else if regexp.MustCompile("(19|20)[0-9]{2}").MatchString(date) {
		//fmt.Printf("Year only\n")
		t, err = time.Parse("2006", date)
		if err != nil {
			return date
		}
		switch snapMethod {
		case "Last CD":
			t = t.AddDate(1, 0, -1)
		}
	} else {
		log.Printf("Badly formatted date=%q\n", date)
		return date
	}
	return t.String()[0:10]
}

func ParseToFirstCD(date string) string {
	return parseDateAndSnap(date, "First CD")
}

func ParseToLastCD(date string) string {
	return parseDateAndSnap(date, "Last CD")
}

func GetLastMonths(n int) (string, string) {
	lastBDstring := GetLastBD()
	lastBD, _ := time.Parse("2006-01-02", lastBDstring)

	return lastBD.AddDate(0, n*-1, 0).String()[0:10], lastBDstring
}

func GetLTM() (string, string) {
	lastBDstring := GetLastBD()
	lastBD, _ := time.Parse("2006-01-02", lastBDstring)

	return lastBD.AddDate(-1, 0, 0).String()[0:10], lastBDstring
}

func GetYTD() (string, string) {
	lastBDstring := GetLastBD()
	lastBD, _ := time.Parse("2006-01-02", lastBDstring)

	return time.Date(lastBD.Year()-1, 12, 31, 0, 0, 0, 0, lastBD.Location()).String()[0:10], lastBDstring
}

func GetLastBD() string {
	t := time.Now()

	if t.Weekday() == time.Monday {
		t = t.AddDate(0, 0, -3)
	} else {
		t = t.AddDate(0, 0, -1)
	}

	return t.String()[0:10]
}
