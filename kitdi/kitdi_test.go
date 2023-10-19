package kitdi

import (
	"fmt"
	"github.com/expectedsh/dig"
	"github.com/stretchr/testify/require"
	"testing"
)

type Test struct {
}

func NewTest() *Test {
	return &Test{}
}

func (t Test) String() string {
	return "test"
}

func TestAnnotate(t *testing.T) {
	t.Run("annotate with nothing", func(t *testing.T) {
		container := dig.New()

		err := Annotate(NewTest()).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		err = container.Invoke(func(test *Test) {})
		require.NoError(t, err)
	})

	t.Run("annotate with group", func(t *testing.T) {
		container := dig.New()

		err := Annotate(NewTest(), Group("test")).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		type tests struct {
			dig.In
			Tests []*Test `group:"test"`
		}
		err = container.Invoke(func(tests tests) {
			require.Len(t, tests.Tests, 1)
		})

		require.NoError(t, err)
	})

	t.Run("annotate with name", func(t *testing.T) {
		container := dig.New()

		err := Annotate(NewTest(), Name("test")).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		type testIn struct {
			dig.In
			Test *Test `name:"test"`
		}

		err = container.Invoke(func(test testIn) {
			require.NotNil(t, test.Test)
		})

		require.NoError(t, err)
	})

	t.Run("annotate with as", func(t *testing.T) {
		container := dig.New()

		err := Annotate(NewTest(), As(new(fmt.Stringer))).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		type testIn struct {
			dig.In
			Test *Test `name:"test"`
		}

		err = container.Invoke(func(test fmt.Stringer) {
			require.NotNil(t, test)
		})

		require.NoError(t, err)
	})
}

func TestSupply(t *testing.T) {
	t.Run("supply a struct directly", func(t *testing.T) {
		container := dig.New()

		err := Supply(NewTest()).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		err = container.Invoke(func(test *Test) {})
		require.NoError(t, err)

	})

	t.Run("supply in combination with annotate should work", func(t *testing.T) {
		container := dig.New()

		err := Annotate(NewTest()).Apply(container)

		if err != nil {
			require.NoError(t, err)
		}

		err = container.Invoke(func(test *Test) {})
		require.NoError(t, err)

		t.Run("with option", func(t *testing.T) {
			err = Annotate(NewTest(), Group("test")).Apply(container)
			require.NoError(t, err)

			type tests struct {
				dig.In
				Tests []*Test `group:"test"`
			}

			err = container.Invoke(func(tests tests) {
				require.Len(t, tests.Tests, 1)
			})
		})
	})
}

func TestProvidableInvoke(t *testing.T) {
	container := dig.New()

	ok := false
	err := ProvidableInvoke(func() {
		ok = true
	}).Apply(container)

	require.NoError(t, err)
	require.True(t, ok)

	t.Run("invokable that register annotation", func(t *testing.T) {
		container := dig.New()
		require.NoError(t, Supply(container).Apply(container))

		invokable := func(c *dig.Container) {
			require.NoError(t, Annotate(NewTest, Group("test")).Apply(c))
		}

		require.NoError(t, ProvidableInvoke(invokable).Apply(container))

		type tests struct {
			dig.In
			Tests []*Test `group:"test"`
		}

		err := container.Invoke(func(tests tests) {
			require.Len(t, tests.Tests, 1)
		})

		require.NoError(t, err)
	})
}
