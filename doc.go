// Package pudge implements a low-level key/value store in pure Go.
// Keys stored in memory, Value stored on disk
//
// Usage
//
//	package main
//
//	import (
//		"log"
//
//		"github.com/recoilme/pudge"
//	)
//
//	func main() {
//		cfg := pudge.DefaultConfig()
//		cfg.SyncInterval = 0 //disable every second fsync
//		db, err := pudge.Open("../test/db", cfg)
//		if err != nil {
//			log.Panic(err)
//		}
//		defer db.DeleteFile()
//		type Point struct {
//			X int
//			Y int
//		}
//		for i := 100; i >= 0; i-- {
//			p := &Point{X: i, Y: i}
//			db.Set(i, p)
//		}
//		var point Point
//		db.Get(8, &point)
//		log.Println(point)
//		// Output: {8 8}
//	}
package pudge
