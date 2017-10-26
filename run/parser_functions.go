package run

import (
	"math"
	"sort"
)

func parseSum(d []DataPoint) float64 {
	var sum float64

	for _, v := range d {
		sum = sum + v.Data
	}

	return sum
}

func parseCount(d []DataPoint) float64 {
	return float64(len(d))
}

func parseAverage(d []DataPoint) float64 {
	sum := parseSum(d)
	count := parseCount(d)

	if count > 0 {
		return sum / count
	}
	return 0
}

func parseVariance(d []DataPoint) float64 {
	mean := parseAverage(d)
	count := parseCount(d)

	var sumOfSquares float64

	for _, v := range d {
		diffFromMean := (v.Data - mean)
		sumOfSquares = sumOfSquares + (diffFromMean * diffFromMean)
	}

	if count > 0 {
		return sumOfSquares / count
	}

	return 0
}

func parseStdDev(d []DataPoint) float64 {
	return math.Sqrt(parseVariance(d))
}

func parseSumProduct(d1 []DataPoint, d2 []DataPoint) float64 {
	var sum float64

	for i := 0; i < len(d1) && i < len(d2); i++ {
		sum = sum + (d1[i].Data * d2[i].Data)
	}

	return sum
}

func parseMedian(d []DataPoint) float64 {
	floatArr := make([]float64, len(d), len(d))

	for i, _ := range floatArr {
		floatArr[i] = d[i].Data
	}
	sort.Float64s(floatArr)

	if len(floatArr)%2 == 0 {
		idx1 := int(len(floatArr) / 2)
		idx2 := idx1 - 1
		return (floatArr[idx1] + floatArr[idx2]) / 2.0
	}
	idx := int(len(floatArr) / 2)
	return floatArr[idx]
}

func getIf(d1 []DataPoint, d2 []DataPoint) []DataPoint {
	d := make([]DataPoint, 0, len(d1))
	for i := 0; i < len(d1) && i < len(d2); i++ {
		if d2[i].Data > 0 {
			d = append(d, d1[i])
		}
	}

	return d
}

func parseMedianIf(d1 []DataPoint, d2 []DataPoint) float64 {
	return parseMedian(getIf(d1, d2))
}

func parseAverageIf(d1 []DataPoint, d2 []DataPoint) float64 {
	return parseAverage(getIf(d1, d2))
}

func parseSumIf(d1 []DataPoint, d2 []DataPoint) float64 {
	return parseSum(getIf(d1, d2))
}

func parseMax(d []DataPoint) float64 {
	var max = math.Inf(-1)
	for _, v := range d {
		if v.Data > max {
			max = v.Data
		}
	}

	return max
}

func parseMin(d []DataPoint) float64 {
	var min = math.Inf(1)
	for _, v := range d {
		if v.Data < min {
			min = v.Data
		}
	}

	return min
}

func parseCompound(d []DataPoint) float64 {
	var compounded float64 = 1

	for _, v := range d {
		compounded = compounded * (1 + v.Data)
	}

	return compounded - 1
}

func parseCAGR(d []DataPoint) float64 {
	firstPoint := d[0]
	lastPoint := d[len(d)-1]

	years := lastPoint.Time.Sub(firstPoint.Time).Hours() / 24.0 / 365.0

	data := math.Pow(lastPoint.Data/firstPoint.Data, 1.0/years) - 1

	return data
}

func parseProduct(d []DataPoint) float64 {
	var product float64 = 1

	for _, v := range d {
		product = product * v.Data
	}

	return product
}

func getValence(functionName string) int {
	switch functionName {
	case "sumproduct", "medianif", "sumif", "averageif":
		return 2
	}

	return 1
}

func runFunction1(functionName string, d []DataPoint) float64 {
	switch functionName {
	case "sum":
		return parseSum(d)
	case "count":
		return parseCount(d)
	case "average", "mean", "avg":
		return parseAverage(d)
	case "variance", "var":
		return parseVariance(d)
	case "stddev", "stdev":
		return parseStdDev(d)
	case "median", "med":
		return parseMedian(d)
	case "maximum", "max":
		return parseMax(d)
	case "minimum", "min":
		return parseMin(d)
	case "compound":
		return parseCompound(d)
	case "cagr":
		return parseCAGR(d)
	case "product":
		return parseProduct(d)
	}

	return 0
}

func runFunction2(functionName string, d1 []DataPoint, d2 []DataPoint) float64 {
	switch functionName {
	case "sumproduct":
		return parseSumProduct(d1, d2)
	case "medianif":
		return parseMedianIf(d1, d2)
	case "sumif":
		return parseSumIf(d1, d2)
	case "averageif":
		return parseAverageIf(d1, d2)
	}

	return 0
}

func functionUnits1(functionName string, units string) string {
	switch functionName {
	case "cagr":
		return "%"
	case "count":
		return "#"
	}

	return units
}

func functionUnits2(functionName string, units1 string, units2 string) string {
	switch functionName {
	case "sumproduct":
		return units1
	}

	return units1
}
