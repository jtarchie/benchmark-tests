package main

import (
	"fmt"
	"testing"

	"github.com/dop251/goja"
	"github.com/expr-lang/expr"
	"github.com/jaswdr/faker/v2"
	v8 "github.com/tommie/v8go"
)

type Item struct {
	Name    string
	Address string
}

const SIZE = 10_000

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
}

const javascript = `
	var filteredItems = []
	
	for (let i = 0; i < items.length; i++) {
		if (items[i].Name.toLowerCase().includes("a")) {
			filteredItems.push(items[i])
		}
	}
	filteredItems
`
