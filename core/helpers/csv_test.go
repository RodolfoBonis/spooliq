package helpers

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadCSV_DetectsSeparatorAndStripsBOM(t *testing.T) {
	header, rows, err := ReadCSV(strings.NewReader("\xEF\xBB\xBFnome;email\nAna;ana@x.com\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"nome", "email"}, header)
	assert.Equal(t, [][]string{{"Ana", "ana@x.com"}}, rows)

	header, rows, err = ReadCSV(strings.NewReader("name,email\n\"Souza, Ana\",a@b.c\n"))
	require.NoError(t, err)
	assert.Equal(t, "name", header[0])
	assert.Equal(t, "Souza, Ana", rows[0][0])
}

func TestWriteCSV_RoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)

	WriteCSV(c, "x.csv", []string{"Nome", "Total"}, [][]string{{"Vaso; grande", CSVCents(123456)}})

	assert.Contains(t, rec.Header().Get("Content-Disposition"), `filename="x.csv"`)
	header, rows, err := ReadCSV(rec.Body)
	require.NoError(t, err)
	assert.Equal(t, []string{"Nome", "Total"}, header)
	assert.Equal(t, []string{"Vaso; grande", "1234,56"}, rows[0])
}

func TestCSVFormatting(t *testing.T) {
	assert.Equal(t, "0,05", CSVCents(5))
	assert.Equal(t, "-10,00", CSVCents(-1000))
	d := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, "08/10/2026", CSVDate(&d))
	assert.Equal(t, "", CSVDate(nil))
}
