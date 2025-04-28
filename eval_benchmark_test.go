package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/expr-lang/expr"
	"github.com/jaswdr/faker/v2"
	v8 "github.com/tommie/v8go"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const SIZE = 10_000

func setupItems(b *testing.B) []struct {
	Name    string
	Address string
} {
	b.Helper()

	items := make([]struct {
		Name    string
		Address string
	}, 0, SIZE)

	fake := faker.New()
	for range SIZE {
		items = append(items, struct {
			Name    string
			Address string
		}{
			Name:    fake.Person().Name(),
			Address: fake.Address().Address(),
		})
	}

	return items
}

func BenchmarkEvaluation(b *testing.B) {
	items := setupItems(b)

	b.Run("Pure Go", func(b *testing.B) {
		filtered := make([]struct {
			Name    string
			Address string
		}, 0)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			for _, item := range items {
				if strings.Contains(strings.ToLower(item.Name), "a") {
					filtered = append(filtered, item)
				}
			}

			if len(filtered) == 0 {
				b.Fatalf("unexpected empty result set")
			}
			filtered = filtered[:0]
		}
	})

	b.Run("Expr", func(b *testing.B) {
		env := map[string]interface{}{
			"items": items,
		}

		code := `filter(items, {lower(.Name) contains "a"})`

		program, err := expr.Compile(code, expr.Env(env))
		if err != nil {
			b.Fatalf("could not compile: %s", err)
		}

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := expr.Run(program, env)
			if err != nil {
				b.Fatalf("could not run: %s", err)
			}
		}
	})

	b.Run("Goja", func(b *testing.B) {
		vm := goja.New()
		vm.Set("items", items)

		program := goja.MustCompile("file.js", javascript, true)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_, err := vm.RunProgram(program)
			if err != nil {
				b.Fatalf("could not run: %s", err)
			}

			var filterItems []struct {
				Name    string
				Address string
			}
			err = vm.ExportTo(vm.Get("filteredItems"), &filterItems)
			if err != nil {
				b.Fatalf("could not export: %s", err)
			}
		}
	})

	b.Run("v8", func(b *testing.B) {
		iso := v8.NewIsolate()
		ctx := v8.NewContext(iso)
		script, err := iso.CompileUnboundScript(javascript, "file.js", v8.CompileOptions{}) //
		if err != nil {
			b.Fatalf("could not compile: %s", err)
		}

		array, err := ctx.RunScript(fmt.Sprintf("new Array(%d)", len(items)), "")
		if err != nil {
			b.Fatalf("could not set: %s", err)
		}
		arrayObj := array.Object()

		for index, item := range items {
			hash, err := ctx.RunScript("new Map()", "")
			if err != nil {
				b.Fatalf("could not set: %s", err)
			}
			mapObject := hash.Object()
			mapObject.Set("Name", item.Name)
			mapObject.Set("Address", item.Address)

			arrayObj.SetIdx(uint32(index), mapObject)
		}

		ctx.Global().Set("items", arrayObj)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			value, err := script.Run(ctx)
			if err != nil {
				b.Fatalf("could not run: %s", err)
			}
			length, err := value.Object().Get("length")
			if err != nil {
				b.Fatalf("could not gte length: %s", err)
			}
			if length.Uint32() == 0 {
				b.Fatalf("could not run: %s", err)
			}
		}
	})

	b.Run("yaegi", func(b *testing.B) {
		i := interp.New(interp.Options{})
		i.Use(stdlib.Symbols)

		_, err := i.Eval(`
			package main

			import "strings"

			type Item struct {
				Name    string
				Address string
			}

			func Filter(items []Item) []Item {
				filtered := []Item
				for _, item := range items {
					if strings.Contains(strings.ToLower(item.Name), "a") {
						filtered = append(filtered, item)
					}
				}

				return filtered
			}
		`)
		if err != nil {
			b.Fatalf("could not eval: %s", err)
		}

		value, err := i.Eval("main.Filter")
		if err != nil {
			b.Fatalf("could not get filter: %s", err)
		}

		fun := value.Interface().(func([]struct {
			Name    string
			Address string
		}) []struct {
			Name    string
			Address string
		})

		for i := 0; i < b.N; i++ {
			filtered := fun(items)

			if len(filtered) == 0 {
				b.Fatalf("could not run: %s", err)
			}
		}
	})
}

const javascript = `
	var filteredItems = items.filter(item => 
		item.Name.toLowerCase().includes("a")
	);
	filteredItems
`
