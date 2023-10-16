package httpbind

import (
	"fmt"
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPath_Extract(t *testing.T) {
	t.Run("retrieve a path", func(t *testing.T) {
		handler := pathHandler(t, pathTestCase{
			pathKey:       "choco",
			expectedValue: []string{"123"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/123").Expect()
	})

	t.Run("retrieve no value", func(t *testing.T) {
		handler := pathHandler(t, pathTestCase{
			pathKey:       "choco",
			expectedValue: nil,
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").Expect()
	})
}

type pathTestCase struct {
	pathKey string

	expectedValue []string
}

func pathHandler(t *testing.T, pathTestCase pathTestCase) http.Handler {
	m := mux.NewRouter()
	c := PathExtractor{}

	m.HandleFunc(fmt.Sprintf("/{%s}", pathTestCase.pathKey), func(w http.ResponseWriter, r *http.Request) {

		extract, _ := c.Extract(r, pathTestCase.pathKey)

		require.Equal(t, pathTestCase.expectedValue, extract)
	})

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		extract, _ := c.Extract(r, pathTestCase.pathKey)

		require.Equal(t, pathTestCase.expectedValue, extract)
	})

	return m
}
