package run

import (
	"fmt"
	"sort"
	"strings"
)

type CellType int
type CellStyleType int

const (
	CellString CellType = iota
	CellFloat
	CellDate
)

const (
	CellNone CellStyleType = iota
	CellBold
	CellIndent
)

const MAX_DATES_TO_DISPLAY = 10

const MAX_COMPANIES_TO_DISPLAY = 10

type SheetCell struct {
	Value      string
	Type       CellType
	Style      CellStyleType
	Identifier string
	FieldName  string
}

type Sheet [][]SheetCell

func newSheet() Sheet {
	var s Sheet

	s = make([][]SheetCell, 0)

	return s
}

func (s Sheet) addDates(m MultiEntityData, isPreview bool) Sheet {
	row := make([]SheetCell, 1)

	uniqueDates := m.UniqueDates()
	length := len(uniqueDates)

	for i, v := range uniqueDates {

		if isPreview {
			if i == int(MAX_DATES_TO_DISPLAY/2) && length > MAX_DATES_TO_DISPLAY {
				row = append(row, SheetCell{
					Value: "...",
					Type:  CellString,
					Style: CellBold,
				})
			} else if length <= MAX_DATES_TO_DISPLAY || i < (MAX_DATES_TO_DISPLAY/2) || i >= length-(MAX_DATES_TO_DISPLAY/2) {
				row = append(row, SheetCell{
					Value: v.UTC().String()[0:10],
					Type:  CellDate,
					Style: CellBold,
				})
			}
		} else {
			row = append(row, SheetCell{
				Value: v.UTC().String()[0:10],
				Type:  CellDate,
				Style: CellBold,
			})
		}

	}

	s = append(s, row)

	return s
}

func (s Sheet) addRows(m MultiEntityData, isPreview bool) Sheet {
	// Loop through each entity and series
	// If it's the entity name, just add blank row with leftmost column as the entity name
	// For all the other stuff, display the actual data (whether it's the weight, data, or category)

	for i, v := range m.EntityData {
		length := len(m.EntityData)
		if !isPreview || (len(m.EntityData) <= MAX_COMPANIES_TO_DISPLAY || i < (MAX_COMPANIES_TO_DISPLAY/2) || i >= length-(MAX_COMPANIES_TO_DISPLAY/2)) {
			// s = append(s, s.rowHeader(v.Meta.Name, CellBold))

			for _, v2 := range v.Data {
				row := s.rowHeader(v.Meta.Name+" - "+v2.Meta.Label, CellIndent)
				row = matchDates(s[0], v.Meta, v2, row)
				s = append(s, row)
			}
			// if len(v.Category.Labels) > 0 {
			// 	row := s.rowHeader("Classification", CellIndent)
			// 	row = matchDatesForClassification(s[0], v.Category, row)
			// 	s = append(s, row)
			// }
		} else if i == int(MAX_COMPANIES_TO_DISPLAY/2) {
			s = append(s, s.rowHeader("...", CellBold))
		}
	}

	return s
}

func matchDatesForClassification(dateRow []SheetCell, s CategorySeries, row []SheetCell) []SheetCell {
	if len(s.Data) > 0 && s.Data[0].Time.Equal(zeroDay()) {
		row[1].Value = s.GetCategoryLabel(s.Data[0].Data)
		row[1].Type = CellString
	}

	for i, v := range dateRow {
		if v.Value == "..." {
			row[i].Value = "..."
			row[i].Type = CellString
		} else {
			found := sort.Search(len(s.Data), func(i int) bool { return s.Data[i].Time.UTC().String()[0:10] >= v.Value })

			if found < len(s.Data) && s.Data[found].Time.UTC().String()[0:10] == v.Value {
				// date is present
				row[i].Value = s.GetCategoryLabel(s.Data[found].Data)
				row[i].Type = CellString
			} else {
				// date is not present
			}
		}
	}

	return row
}

func matchDates(dateRow []SheetCell, e EntityMeta, s Series, row []SheetCell) []SheetCell {
	for i, v := range dateRow {
		if v.Value == "..." {
			row[i].Value = "..."
			row[i].Type = CellString
		} else {
			found := sort.Search(s.Len(), func(i int) bool { return s.Data[i].Time.UTC().String()[0:10] >= v.Value })

			if found < s.Len() && s.Data[found].Time.UTC().String()[0:10] == v.Value {
				// date is present
				row[i].Value = fmt.Sprintf("%v", s.Data[found].Data)
				row[i].Type = CellFloat
			} else {
				// date is not present
			}
			row[i].Identifier = stripOff(e.UniqueId)
			row[i].FieldName = removeCustom(s.Meta.Label)
		}
	}

	return row
}

func stripOff(identifier string) string {
	result := strings.Split(identifier, "/")

	if len(result) > 0 {
		return result[len(result)-1]
	}

	return identifier
}

func removeCustom(s string) string {
	s = strings.Replace(s, "(Custom)", "", -1)
	s = strings.Trim(s, " ")

	return s
}

func (s Sheet) rowHeader(header string, style CellStyleType) []SheetCell {
	c := make([]SheetCell, len(s[0]))

	if len(s) > 0 {
		c[0] = SheetCell{
			Value: header,
			Type:  CellString,
			Style: style,
		}
	}

	return c
}

func convertMultiEntityDataToSheet(m MultiEntityData, isPreview bool) Sheet {
	s := newSheet()

	s = s.addDates(m, isPreview)
	s = s.addRows(m, isPreview)

	transposed := make([][]SheetCell, len(s[0]))
	for x, _ := range transposed {
		transposed[x] = make([]SheetCell, len(s))
	}
	for y, v := range s {
		for x, e := range v {
			transposed[x][y] = e
		}
	}

	return transposed
}
