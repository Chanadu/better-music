package handlers

import (
	"net/http/httptest"
	"testing"
)

func TestWriteJSON(test *testing.T) {
	responseRecorder := httptest.NewRecorder()
	writeJSON(responseRecorder, 201, apiMessage("created"))
	if responseRecorder.Code != 201 || responseRecorder.Body.String() != `{"message":"created"}`+"\n" {
		test.Fatal(responseRecorder)
	}

	responseRecorder = httptest.NewRecorder()
	writeJSON(responseRecorder, 200, make(chan int))
	if responseRecorder.Code != 500 {
		test.Fatal(responseRecorder)
	}
}
