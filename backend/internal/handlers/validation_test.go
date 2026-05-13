package handlers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestParsePaginationAcceptsDefaults(t *testing.T) {
	c := testContext("/medicines")
	page, perPage, ok := parsePagination(c)
	if !ok {
		t.Fatal("parsePagination rejected default query")
	}
	if page != 1 || perPage != defaultPageSize {
		t.Fatalf("unexpected defaults: page=%d perPage=%d", page, perPage)
	}
}

func TestParsePaginationRejectsInvalidValues(t *testing.T) {
	for _, target := range []string{
		"/medicines?page=0",
		"/medicines?page=abc",
		"/medicines?per_page=0",
		"/medicines?per_page=101",
	} {
		c := testContext(target)
		if _, _, ok := parsePagination(c); ok {
			t.Fatalf("parsePagination accepted invalid query %q", target)
		}
	}
}

func TestParseDateQueryRejectsInvalidDate(t *testing.T) {
	c := testContext("/confirmations?date=2026-02-30")
	if _, ok := parseDateQuery(c, "date", time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)); ok {
		t.Fatal("parseDateQuery accepted invalid calendar date")
	}
}

func testContext(target string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", target, nil)
	return c
}
