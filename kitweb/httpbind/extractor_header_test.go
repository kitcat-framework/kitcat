package httpbind

import (
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHeader_Extract(t *testing.T) {
	t.Run("retrieve a header", func(t *testing.T) {
		handler := headerHandler(t, headerTestCase{
			valueOfTag:    "choco",
			expectedValue: []string{"123"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").WithHeader("choco", "123").Expect()
	})

	t.Run("retrieve no value", func(t *testing.T) {
		handler := headerHandler(t, headerTestCase{
			valueOfTag:    "choco",
			expectedValue: nil,
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").Expect()
	})

	t.Run("retrieve multiple values", func(t *testing.T) {
		handler := headerHandler(t, headerTestCase{
			valueOfTag:    "choco",
			expectedValue: []string{"123", "456"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").WithHeader("choco", "123").WithHeader("choco", "456").Expect()
	})
}

type headerTestCase struct {
	valueOfTag string

	expectedValue []string
}

func headerHandler(t *testing.T, ctxHeaderTestCase headerTestCase) http.Handler {
	m := mux.NewRouter()
	c := HeaderExtractor{}

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		extract, _ := c.Extract(r, ctxHeaderTestCase.valueOfTag)

		require.Equal(t, ctxHeaderTestCase.expectedValue, extract)
	})

	return m
}
