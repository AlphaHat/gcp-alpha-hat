package run

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/tealeg/xlsx"
)

func GenerateExcelFile(ctx context.Context, hex string, sheetCells [][]SheetCell, w io.Writer) {
	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	file = xlsx.NewFile()
	sheet, err = file.AddSheet("Sheet1")
	if !logError(ctx, err) {
		return
	}
	sheet.SheetFormat.DefaultColWidth = 11.25

	bold := xlsx.NewStyle()
	bold.Font.Bold = true

	indent := xlsx.NewStyle()
	indent.Alignment.Indent = 1

	for _, sheetRows := range sheetCells {
		row = sheet.AddRow()

		for _, sheetCell := range sheetRows {
			cell = row.AddCell()
			cell.Value = sheetCell.Value
			switch sheetCell.Type {
			case CellString:
				cell.Value = sheetCell.Value
			case CellDate:
				date, err := time.Parse("2006-01-02", sheetCell.Value)
				if err == nil {
					cell.SetDate(date)
				}
			case CellFloat:
				n, _ := strconv.ParseFloat(sheetCell.Value, 64)
				cell.SetFloat(n)
				// cell.SetFloatWithFormat(n, `_($* #,##0,,_) "MM";_($* (#,##0,,) "MM";_($* "-"??_);_(@_)`)
			}

			switch sheetCell.Style {
			case CellBold:
				cell.SetStyle(bold)
			case CellIndent:
				cell.SetStyle(indent)
			}
		}
	}

	// err = file.Save("/root/dploy/i/" + hex + ".xlsx")
	// if zlog.LogError("GenerateExcelFile", err) {
	// 	return
	// }

	file.Write(w)
}
