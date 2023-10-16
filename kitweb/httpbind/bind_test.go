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

func TestBinder_Bind(t *testing.T) {
	m := mux.NewRouter()

	type testStruct struct {
		FromPath                     string   `path:"fromPath"`
		FromQuery                    string   `query:"fromQuery"`
		FromHeader                   string   `header:"fromHeader"`
		FromContext                  string   `ctx:"fromContext"`
		FromJson                     string   `json:"fromJson"`
		FromDefaultValue             string   `query:"123" default:"value"`
		FromDefaultValueWithExploder []string `query:"123" default:"value1,value2,value3" exploder:","`
	}

	binder := NewBinder(
		StringsParamExtractors,
		ValuesParamExtractors,
	)

	m.HandleFunc("/{fromPath}", func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), "fromContext", "value"))

		var test testStruct
		err := binder.Bind(r, &test)
		require.Len(t, err, 0)

		require.Equal(t, "value", test.FromPath)
		require.Equal(t, "value", test.FromQuery)
		require.Equal(t, "value", test.FromHeader)
		require.Equal(t, "value", test.FromContext)
		require.Equal(t, "value", test.FromJson)
		require.Equal(t, "value", test.FromDefaultValue)
		require.Equal(t, []string{"value1", "value2", "value3"}, test.FromDefaultValueWithExploder)
	})

	server := httptest.NewServer(m)
	t.Cleanup(server.Close)

	e := httpexpect.Default(t, server.URL)
	e.GET("/{fromPath}").
		WithJSON(map[string]string{
			"fromJson": "value",
		}).
		WithHeader("fromHeader", "value").
		WithQuery("fromQuery", "value").
		WithHeader("fromContext", "value").
		WithPath("fromPath", "value").
		Expect()

	t.Run("test with form", func(t *testing.T) {
		m := mux.NewRouter()

		type testStruct struct {
			FromForm    string `form:"fromForm"`
			FromFormInt int    `form:"fromFormInt"`
		}

		binder := NewBinder(StringsParamExtractors, ValuesParamExtractors)

		m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var test testStruct
			err := binder.Bind(r, &test)
			require.Len(t, err, 0)

			require.Equal(t, "value", test.FromForm)
			require.Equal(t, 1, test.FromFormInt)
		})

		server := httptest.NewServer(m)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.POST("/").
			WithForm(map[string]string{
				"fromForm":    "value",
				"fromFormInt": "1",
			}).
			Expect()
	})

	t.Run("test with invalid field setter", func(t *testing.T) {

		m := mux.NewRouter()

		type testStruct struct {
			FormFormIntError    int  `form:"fromFormIntError"`
			FormFormIntNilError *int `form:"fromFormIntNilError"`
		}

		binder := NewBinder(
			StringsParamExtractors,
			ValuesParamExtractors,
		)

		m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			var test testStruct
			err := binder.Bind(r, &test)
			require.Len(t, err.(Error).Errors, 1)

			require.IsType(t, &FieldSetterError{}, err.(Error).Errors[0])

			require.Equal(t, 0, test.FormFormIntError)
			require.Nil(t, test.FormFormIntNilError)
		})

		server := httptest.NewServer(m)
		t.Cleanup(server.Close)

		e := httpexpect.Default(t, server.URL)
		e.POST("/").
			WithForm(map[string]string{
				"fromFormIntError": "not int",
			}).
			Expect()
	})
}
