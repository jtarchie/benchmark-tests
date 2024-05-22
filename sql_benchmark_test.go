package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func randomBoundingBox() (float64, float64, float64, float64) {
	minX := rand.Float64() * 100
	maxX := minX + rand.Float64()*10
	minY := rand.Float64() * 100
	maxY := minY + rand.Float64()*10
	return minX, maxX, minY, maxY
}

func setupDB(b *testing.B) *sql.DB {
	b.Helper()

	db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`CREATE VIRTUAL TABLE demo_rtree USING rtree(
        id INTEGER PRIMARY KEY,
        minX REAL,
        maxX REAL,
        minY REAL,
        maxY REAL
    );`)
	if err != nil {
		panic(err)
	}
	return db
}

func BenchmarkRtreeInsert(b *testing.B) {
	db := setupDB(b)
	defer db.Close()

	// Pure Insert Statement
	b.Run("Pure Insert in parallel", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				minX, maxX, minY, maxY := randomBoundingBox()
				query := fmt.Sprintf("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (%f, %f, %f, %f)", minX, maxX, minY, maxY)
				_, err := db.Exec(query)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("Pure Insert in parallel", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			minX, maxX, minY, maxY := randomBoundingBox()
			query := fmt.Sprintf("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (%f, %f, %f, %f)", minX, maxX, minY, maxY)
			_, err := db.Exec(query)
			if err != nil {
				b.Error(err)
			}
		}
	})

	// Prepared Statement
	b.Run("Prepared Statement in parallel", func(b *testing.B) {
		stmt, err := db.Prepare("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (?, ?, ?, ?)")
		if err != nil {
			b.Fatal(err)
		}
		defer stmt.Close()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				minX, maxX, minY, maxY := randomBoundingBox()
				_, err := stmt.Exec(minX, maxX, minY, maxY)
				if err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("Prepared Statement", func(b *testing.B) {
		stmt, err := db.Prepare("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (?, ?, ?, ?)")
		if err != nil {
			b.Fatal(err)
		}
		defer stmt.Close()

		for i := 0; i < b.N; i++ {
				minX, maxX, minY, maxY := randomBoundingBox()
				_, err := stmt.Exec(minX, maxX, minY, maxY)
				if err != nil {
					b.Error(err)
				}
			}
	})

	// Transaction
	b.Run("Transaction", func(b *testing.B) {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		defer tx.Rollback()

		for i := 0; i < b.N; i++ {
			minX, maxX, minY, maxY := randomBoundingBox()
			_, err := tx.Exec("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (?, ?, ?, ?)", minX, maxX, minY, maxY)
			if err != nil {
				b.Error(err)
			}
		}

		tx.Commit()
	})

	// Transaction with Prepared Statement
	b.Run("Transaction with Prepared Statement in parallel", func(b *testing.B) {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (?, ?, ?, ?)")
		if err != nil {
			b.Fatal(err)
		}
		defer stmt.Close()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				minX, maxX, minY, maxY := randomBoundingBox()
				_, err := stmt.Exec(minX, maxX, minY, maxY)
				if err != nil {
					b.Error(err)
				}
			}
		})

		tx.Commit()
	})

	b.Run("Transaction with Prepared Statement", func(b *testing.B) {
		tx, err := db.Begin()
		if err != nil {
			b.Fatal(err)
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare("INSERT INTO demo_rtree (minX, maxX, minY, maxY) VALUES (?, ?, ?, ?)")
		if err != nil {
			b.Fatal(err)
		}
		defer stmt.Close()

		for i := 0; i < b.N; i++ {
			minX, maxX, minY, maxY := randomBoundingBox()
			_, err := stmt.Exec(minX, maxX, minY, maxY)
			if err != nil {
				b.Error(err)
			}
		}

		tx.Commit()
	})
}
