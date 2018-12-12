package pudge

import (
	"log"
)

func ExampleOpen() {
	cfg := DefaultConfig()
	cfg.SyncInterval = 0 //disable every second fsync
	db, err := Open("test/db", cfg)
	if err != nil {
		log.Panic(err)
	}
	defer db.DeleteFile()
	type Point struct {
		X int
		Y int
	}
	for i := 100; i >= 0; i-- {
		p := &Point{X: i, Y: i}
		db.Set(i, p)
	}
	var point Point
	db.Get(8, &point)
	log.Println(point)
	// Output: {8 8}
	// Select 2 keys, from 7 in ascending order
	keys, _ := db.Keys(7, 2, 0, true)
	for _, key := range keys {
		var p Point
		db.Get(key, &p)
		log.Println(p)
	}
	// Output: {8 8}
	// Output: {9 9}
}
func ExampleSet() {
	Set("test/test", "Hello", "World")
	defer CloseAll()
}

func ExampleGet() {
	Set("test/test", "Hello", "World")
	output := ""
	Get("test/test", "Hello", &output)
	log.Println("output:", output)
	defer CloseAll()
}
