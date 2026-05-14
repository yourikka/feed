package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newQueryContext(rawURL string) (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, rawURL, nil)
	return ctx, recorder
}

func TestParsePositiveUintQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, recorder := newQueryContext("/demo?video_id=12")
	value, ok := parsePositiveUintQuery(ctx, "video_id")
	if !ok || value != 12 {
		t.Fatalf("expected valid value 12, got ok=%v, value=%d", ok, value)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body for valid query")
	}
}

func TestParsePositiveUintQueryInvalid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, recorder := newQueryContext("/demo?video_id=0")
	_, ok := parsePositiveUintQuery(ctx, "video_id")
	if ok {
		t.Fatalf("expected invalid query")
	}

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if body["status_code"] != float64(1) {
		t.Fatalf("expected status_code=1, got %v", body["status_code"])
	}
}

func TestParseOptionalPositiveIntQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx, recorder := newQueryContext("/demo?limit=15")
	value, ok := parseOptionalPositiveIntQuery(ctx, "limit", 10)
	if !ok || value != 15 {
		t.Fatalf("expected valid limit 15, got ok=%v value=%d", ok, value)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body for valid query")
	}

	ctx, recorder = newQueryContext("/demo")
	value, ok = parseOptionalPositiveIntQuery(ctx, "limit", 10)
	if !ok || value != 10 {
		t.Fatalf("expected default limit 10, got ok=%v value=%d", ok, value)
	}
	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty response body for missing query")
	}
}
