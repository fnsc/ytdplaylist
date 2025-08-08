package excel

import (
	"strings"

	"github.com/xuri/excelize/v2"
	"ytpd/utils"
)

func ExtractLinks(path string) ([]string, error) {
	file, err := excelize.OpenFile(path)
	if err != nil {
		return nil, err
	}

	sheet := file.GetSheetName(0)
	rows, err := file.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	return utils.FilterMap(rows, func(row []string) (string, bool) {
		if len(row) == 0 || strings.TrimSpace(row[0]) == "" {
			return "", false
		}
		return strings.TrimSpace(row[0]), true
	}), nil
}
