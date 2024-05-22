package main

import (
	"testing"

	"github.com/dop251/goja"
	"github.com/expr-lang/expr"
	"github.com/jaswdr/faker/v2"
)

type Item struct {
	Name    string
	Address string
}

const SIZE = 1_000

func setupItems(b *testing.B) []Item {
	b.Helper()

	items := make([]Item, 0, SIZE)

	fake := faker.New()
	for range SIZE {
		items = append(items, Item{
			Name:    fake.Person().Name(),
			Address: fake.Address().Address(),
		})
	}

	return items
}

func BenchmarkEvaluation(b *testing.B) {
	items := setupItems(b)

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

			var filterItems []Item
			err = vm.ExportTo(vm.Get("filteredItems"), &filterItems)
			if err != nil {
				b.Fatalf("could not export: %s", err)
			}
		}
	})
}

const javascript = `
	var filteredItems = []
	
	for (let i = 0; i < items.length; i++) {
		if (items[i].Name.toLowerCase().includes("a")) {
			filteredItems.push(items[i])
		}
	}
`
