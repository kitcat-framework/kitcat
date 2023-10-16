package httpbind

import (
	"context"
	"github.com/gavv/httpexpect/v2"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

type chocolate struct {
	ID   int
	Name string
}

func TestContext_Extract(t *testing.T) {
	t.Run("retrieve a type", func(t *testing.T) {
		handler := ctxHandler(t, ctxHandlerTestCase[chocolate]{
			valueOfTag:   "choco",
			contextKey:   "choco",
			contextValue: &chocolate{ID: 123, Name: "ccccc"},
			expectedValue: &chocolate{
				ID:   123,
				Name: "ccccc",
			},

			expectedType: &chocolate{},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").Expect()
	})

	t.Run("retrieve no value", func(t *testing.T) {
		handler := ctxHandler(t, ctxHandlerTestCase[chocolate]{
			valueOfTag:    "choco",
			contextKey:    "choco",
			contextValue:  nil,
			expectedValue: nil,
			expectedType:  &chocolate{},
		})

		server := httptest.NewServer(handler)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.GET("/").Expect()
	})

}

type ctxHandlerTestCase[T any] struct {
	valueOfTag   string
	contextKey   string
	contextValue *T

	expectedValue *T
	expectedType  *T
	expectedError bool
}

func ctxHandler[T any](t *testing.T, testCase ctxHandlerTestCase[T]) http.Handler {
	m := mux.NewRouter()
	c := ContextExtractor{}

	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), testCase.contextKey, testCase.contextValue))

		extract, err := c.Extract(r, testCase.valueOfTag)

		if testCase.expectedError {
			require.Error(t, err)
			return
		} else {
			require.NoError(t, err)
		}

		require.IsType(t, testCase.expectedType, extract)
		require.Equal(t, testCase.expectedValue, extract)
	})

	return m
}
