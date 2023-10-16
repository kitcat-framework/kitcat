package httpbind

import (
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQuery_Extract(t *testing.T) {
	t.Run("retrieve a query", func(t *testing.T) {
		handler := queryHandler(t, queryTestCase{
			queryKey:      "choco",
			expectedValue: []string{"123"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").WithQuery("choco", "123").
			Expect()
	})

	t.Run("retrieve multiple values", func(t *testing.T) {
		handler := queryHandler(t, queryTestCase{
			queryKey:      "choco",
			expectedValue: []string{"123", "456"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").WithQuery("choco", "123").WithQuery("choco", "456").
			Expect()
	})

	t.Run("retrieve no value", func(t *testing.T) {
		handler := queryHandler(t, queryTestCase{
			queryKey:      "choco",
			expectedValue: nil,
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").Expect()
	})
}

type queryTestCase struct {
	queryKey string

	expectedValue []string
}

func queryHandler(t *testing.T, queryTestCase queryTestCase) http.Handler {
	m := mux.NewRouter()
	c := QueryExtractor{}

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		extract, _ := c.Extract(r, queryTestCase.queryKey)

		require.Equal(t, queryTestCase.expectedValue, extract)
	})

	return m
}
