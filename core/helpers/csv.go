package helpers

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// utf8BOM makes Excel open UTF-8 CSVs with the right encoding.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// WriteCSV streams a pt-BR friendly CSV (UTF-8 BOM, ';' separator) as a
// download named filename.
func WriteCSV(c *gin.Context, filename string, header []string, rows [][]string) {
	var buf bytes.Buffer
	buf.Write(utf8BOM)
	w := csv.NewWriter(&buf)
	w.Comma = ';'
	_ = w.Write(header)
	for _, row := range rows {
		_ = w.Write(row)
	}
	w.Flush()

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", buf.Bytes())
}

// ReadCSV parses a CSV exported by spreadsheets: strips the UTF-8 BOM and
// detects ';' or ',' from the header line. Returns the header and data rows.
func ReadCSV(r io.Reader) ([]string, [][]string, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	data = bytes.TrimPrefix(data, utf8BOM)

	firstLine, _, _ := bufio.NewReader(bytes.NewReader(data)).ReadLine()
	comma := ','
	if strings.Count(string(firstLine), ";") > strings.Count(string(firstLine), ",") {
		comma = ';'
	}

	reader := csv.NewReader(bytes.NewReader(data))
	reader.Comma = comma
	reader.FieldsPerRecord = -1 // tolerate ragged rows; callers validate
	reader.TrimLeadingSpace = true
	records, err := reader.ReadAll()
	if err != nil {
		return nil, nil, err
	}
	if len(records) == 0 {
		return nil, nil, fmt.Errorf("arquivo vazio")
	}
	return records[0], records[1:], nil
}

// CSVCents formats cents as "1234,56" (decimal comma, no thousands separator,
// so spreadsheets in pt-BR read it as a number).
func CSVCents(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d,%02d", sign, cents/100, cents%100)
}

// CSVDate formats a date as dd/mm/yyyy in Brazil's time zone ("" for nil).
func CSVDate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("02/01/2006")
}
