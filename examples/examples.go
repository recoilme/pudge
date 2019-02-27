package main

import (
	"log"

	"github.com/recoilme/pudge"
)

func main() {
	ExampleSet()
	ExampleGet()
	ExampleDelete()
	ExampleDeleteFile()
	ExampleOpen()
	ExampleInMemoryWithoutPersist()
}

//ExampleSet lazy
func ExampleSet() {
	pudge.Set("../test/test", "Hello", "World")
	defer pudge.CloseAll()
}

//ExampleGet lazy
func ExampleGet() {
	output := ""
	pudge.Get("../test/test", "Hello", &output)
	log.Println("Output:", output)
	// Output: World
	defer pudge.CloseAll()
}

//ExampleDelete lazy
func ExampleDelete() {
	err := pudge.Delete("../test/test", "Hello")
	if err == pudge.ErrKeyNotFound {
		log.Println(err)
	}
}

//ExampleDeleteFile lazy
func ExampleDeleteFile() {
	err := pudge.DeleteFile("../test/test")
	if err != nil {
		log.Panic(err)
	}
}

//ExampleOpen complex example
func ExampleOpen() {
	cfg := &pudge.Config{
		SyncInterval: 0} //disable every second fsync
	db, err := pudge.Open("../test/db", cfg)
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

// ExampleInMemoryWithoutPersist -if file is empty in storemode 2 - without persist
func ExampleInMemoryWithoutPersist() {
	cfg := &pudge.Config{StoreMode: 2} //in memory
	db, err := pudge.Open("", cfg)     // if file is empty in storemode 2 - without persist
	if err != nil {
		log.Panic(err)
	}
	defer db.Close() //remove from memory
	type Point struct {
		X int
		Y int
	}
	for i := 100; i >= 0; i-- {
		p := &Point{X: i, Y: i}
		db.Set(i, p)
	}
	var point Point
	db.Get(89, &point)
	log.Println(point)
	// Output: {89 89}
	keys, _ := db.Keys(77, 2, 0, true)
	for _, key := range keys {
		var p Point
		db.Get(key, &p)
		log.Println(p)
	}
	// Output: {78 78}
	// Output: {79 79}
}
