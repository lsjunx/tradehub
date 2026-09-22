package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOKEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, map[string]string{"x": "1"})
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	var resp Body
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 || resp.Message != "ok" || resp.Timestamp == 0 || resp.Data == nil {
		t.Fatalf("%+v", resp)
	}
}

func TestFailEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, http.StatusNotFound, 404, "device not found")
	var resp Body
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != 404 || resp.Message != "device not found" || resp.Data != nil || resp.Timestamp == 0 {
		t.Fatalf("%+v", resp)
	}
}