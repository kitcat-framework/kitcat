package httpbind

import (
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFormExtractor_Extract(t *testing.T) {
	t.Run("retrieve a form", func(t *testing.T) {
		handler := formHandler(t, formTestCase{
			valueOfTag:    "choco",
			expectedValue: []string{"123"},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.POST("/").WithFormField("choco", "123").Expect()
	})

	t.Run("retrieve no value", func(t *testing.T) {
		handler := formHandler(t, formTestCase{
			valueOfTag:    "choco",
			expectedValue: nil,
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.POST("/").Expect()
	})
}

type formTestCase struct {
	valueOfTag string

	expectedValue []string
}

func formHandler(t *testing.T, ctxHeaderTestCase formTestCase) http.Handler {
	m := mux.NewRouter()
	c := FormExtractor{}

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		extract, _ := c.Extract(r, ctxHeaderTestCase.valueOfTag)

		require.Equal(t, ctxHeaderTestCase.expectedValue, extract)
	})

	return m
}
