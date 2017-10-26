package zstats

import (
	"fmt"
	"sort"
)

func GetBoxplotValues(data []float64) []float64 {
	var boxplot []float64 = make([]float64, 5, 5)

	boundaries := GetQuantileBoundaries(data, 4)

	// Data is sorted, so min is at the beginning and max is at the end
	boxplot[0] = data[0]
	boxplot[1] = boundaries[0]
	boxplot[2] = boundaries[1]
	boxplot[3] = boundaries[2]
	boxplot[4] = data[len(data)-1]

	return boxplot
}

// Requires sorted data
func GetQuantileBoundaries(data []float64, numBuckets int64) []float64 {
	if !sort.Float64sAreSorted(data) {
		panic(fmt.Sprintf("Data not sorted data=%s\n", data))
	}

	boundaries := make([]float64, numBuckets-1, numBuckets-1)

	for i, _ := range boundaries {
		dataIndex := int(float64(i+1)*float64(len(data))/float64(numBuckets)) - 1
		if dataIndex < 0 {
			dataIndex = 0
		}
		if dataIndex < len(data) {
			//fmt.Printf("i=%v\nboundaries=%s\ndata=%s\ndataIndex=%v\n", i, boundaries, data, dataIndex)
			boundaries[i] = data[dataIndex]
		} else if len(data) > 1 {
			distance := (data[len(data)-1] - data[0]) / float64(numBuckets)
			if i > 0 {
				boundaries[i] = boundaries[i-1] + distance
			} else {
				boundaries[i] = data[0]
			}
		} else if len(data) == 1 {
			boundaries[i] = data[0]
		} else {
			boundaries[i] = 0
		}
	}

	return boundaries
}

func GetQuantileNumber(data float64, boundaries []float64) int64 {
	return int64(sort.SearchFloat64s(boundaries, data)) + 1
}

func QuantileLabel(numBucket int64, quantileNum int64) string {
	var beginning, postFix string
	if numBucket == 4 {
		postFix = " Quartile"
	} else if numBucket == 5 {
		postFix = " Quintile"
	} else if numBucket == 10 {
		postFix = " Decile"
	} else {
		postFix = " Quantile"
	}

	if numBucket < 10 {
		switch quantileNum {
		case 1:
			beginning = "1st"
		case 2:
			beginning = "2nd"
		case 3:
			beginning = "3rd"
		default:
			beginning = fmt.Sprintf("%vth", quantileNum)
		}
	} else {
		switch quantileNum {
		case 1:
			beginning = " 1st"
		case 2:
			beginning = " 2nd"
		case 3:
			beginning = " 3rd"
		default:
			if quantileNum < 10 {
				beginning = fmt.Sprintf(" %vth", quantileNum)
			} else {
				beginning = fmt.Sprintf("%vth", quantileNum)
			}
		}
	}

	return beginning + postFix
}
