package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"uuid"
)

func TestAssignRequestIDReplacesClientIDAndKeepsUUIDInContext(t *testing.T) {
	clientID := uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")
	var seenID uuid.UUID
	handler := assignRequestID(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		seenID = requestID(request.Context())
		if seenID == uuid.Nil() || seenID == clientID {
			t.Fatalf("expected server-generated request ID, got %v", seenID)
		}
	}))
	request := httptest.NewRequest("GET", "/", nil)
	request.Header.Set("X-Request-ID", clientID.String())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Header().Get("X-Request-ID") != seenID.String() {
		t.Fatal("response header does not match the context UUID")
	}
}
