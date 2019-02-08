[![Documentation](https://godoc.org/github.com/recoilme/pudge?status.svg)](https://godoc.org/github.com/recoilme/pudge)
[![Go Report Card](https://goreportcard.com/badge/github.com/recoilme/pudge)](https://goreportcard.com/report/github.com/recoilme/pudge)
[![Build Status](https://travis-ci.org/recoilme/pudge.svg?branch=master)](https://travis-ci.org/recoilme/pudge)
[![Mentioned in Awesome Go](https://awesome.re/mentioned-badge-flat.svg)](https://github.com/avelino/awesome-go)

Table of Contents
=================

* [Description](#description)
* [Usage](#usage)
* [Cookbook](#cookbook)
* [Disadvantages](#disadvantages)
* [Motivation](#motivation)
* [Benchmarks](#benchmarks)
	* [Test 1](#test-1)
	* [Test 4](#test-4)

## Description

Package pudge is a fast and simple key/value store written using Go's standard library.

It presents the following:
* Supporting very efficient lookup, insertions and deletions
* Performance is comparable to hash tables
* Ability to get the data in sorted order, which enables additional operations like range scan
* Select with limit/offset/from key, with ordering or by prefix
* Safe for use in goroutines
* Space efficient
* Very short and simple codebase
* Well tested, used in production

![pudge](https://avatars3.githubusercontent.com/u/417177?s=460&v=4)

## Usage


```golang
package main

import (
	"log"

	"github.com/recoilme/pudge"
)

func main() {
	// Close all database on exit
	defer pudge.CloseAll()

	// Set (directories will be created)
	pudge.Set("../test/test", "Hello", "World")

	// Get (lazy open db if needed)
	output := ""
	pudge.Get("../test/test", "Hello", &output)
	log.Println("Output:", output)

	ExampleSelect()
}


//ExampleSelect
func ExampleSelect() {
	cfg := &pudge.Config{
		SyncInterval: 1} // every second fsync
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

```

## Cookbook

 - Store data of any type. Pudge uses Gob encoder/decoder internally. No limits on keys/values size.

```golang
pudge.Set("strings", "Hello", "World")
pudge.Set("numbers", 1, 42)

type User struct {
	Id int
	Name string
}
u := &User{Id: 1, Name: "name"}
pudge.Set("users", u.Id, u)

```
 - Pudge is stateless and safe for use in goroutines. You don't need to create/open files before use. Just write data to pudge, don't worry about state. [web server example](https://github.com/recoilme/pixel)

 - Pudge is parallel. Readers don't block readers, but a writer - does, but by the stateless nature of pudge it's safe to use multiples files for storages.

 ![Illustration from slowpoke (based on pudge)](https://camo.githubusercontent.com/a1b406485fa8cd52a98d820de706e3fd255941e9/68747470733a2f2f686162726173746f726167652e6f72672f776562742f79702f6f6b2f63332f79706f6b63333377702d70316a63657771346132323164693168752e706e67)


 - Default store system: like memcache + file storage. Pudge uses in-memory hashmap for keys, and writes values to files (no value data stored in memory). But you may use inmemory mode for values, with custom config:
```golang
cfg = pudge.DefaultConfig()
cfg.StoreMode = 2
db, err := pudge.Open(dbPrefix+"/"+group, cfg)
...
db.Counter(key, val)
```
In that case, all data is stored in memory and will be stored on disk only on Close. 

[Example server for highload, with http api](https://github.com/recoilme/bandit-server)

 - You may use pudge as an engine for creating databases. 
 
 [Example database](https://github.com/recoilme/slowpoke)

 - Don't forget to close all opened databases on shutdown/kill.
```golang
 	// Wait for interrupt signal to gracefully shutdown the server 
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit
	log.Println("Shutdown Server ...")
	if err := pudge.CloseAll(); err != nil {
		log.Println("Pudge Shutdown err:", err)
	}
 ```
 [example recovery function for gin framework](https://github.com/recoilme/bandit-server/blob/02e6eb9f89913bd68952ec35f6c37fc203d71fc2/bandit-server.go#L89)

 - Pudge has a primitive select/query engine.
 ```golang
 // Select 2 keys, from 7 in ascending order
	keys, _ := db.Keys(7, 2, 0, true)
// select keys from db where key>7 order by keys asc limit 2 offset 0
 ```

 - Pudge will work well on SSD or spined disks. Pudge doesn't eat memory or storage or your sandwich. No hidden compaction/rebalancing/resizing and so on tasks. No LSM Tree. No MMap. It's a very simple database with less than 500 LOC. It's good for [simple social network](https://github.com/recoilme/tgram) or highload system 


## Disadvantages

 - No transaction system. All operations are isolated, but you don't may batching them with automatic rollback.
 - [Keys](https://godoc.org/github.com/recoilme/pudge#Keys) function (select/query engine) may be slow. Speed of query may vary from 10ms to 1sec per million keys. Pudge don't use BTree/Skiplist or Adaptive radix tree for store keys in ordered way on every insert. Ordering operation is "lazy" and run only if needed.
 - No fsync on every insert. Most of database fsync data by the timer too
 - Deleted data don't remove from physically (but upsert will try to reuse space). You may shrink database only with backup right now
```golang
pudge.BackupAll("backup")
```
 - Keys automatically convert to binary and ordered with binary comparator. It's simple for use, but ordering will not work correctly for negative numbers for example
 - Author of project don't work at Google or Facebook and his name not Howard Chu or Brad Fitzpatrick. But I'm open for issue or contributions.


## Motivation

Some databases very well for writing. Some of the databases very well for reading. But [pudge is well balanced for both types of operations](https://github.com/recoilme/pogreb-bench). It has small [cute api](https://godoc.org/github.com/recoilme/pudge), and don't have hidden graveyards. It's just hashmap where values written in files. And you may use one database for in-memory/persistent storage in a stateless stressfree way


## Benchmarks

[All tests here](https://github.com/recoilme/pogreb-bench)

***Some tests, MacBook Pro (Retina, 13-inch, Early 2015)***



### Test 1
Number of keys: 1000000
Minimum key size: 16, maximum key size: 64
Minimum value size: 128, maximum value size: 512
Concurrency: 2


|                       | pogreb  | goleveldb | bolt   | badgerdb | pudge  | slowpoke | pudge(mem) |
|-----------------------|---------|-----------|--------|----------|--------|----------|------------|
| 1M (Put+Get), seconds | 187     | 38        | 126    | 34       | 23     | 23       | 2          |
| 1M Put, ops/sec       | 5336    | 34743     | 8054   | 33539    | 47298  | 46789    | 439581     |
| 1M Get, ops/sec       | 1782423 | 98406     | 499871 | 220597   | 499172 | 445783   | 1652069    |
| FileSize,Mb           | 568     | 357       | 552    | 487      | 358    | 358      | 358        |


### Test 4
Number of keys: 10000000
Key size: 8
Value size: 16
Concurrency: 100


|                       | goleveldb | badgerdb | pudge  |
|-----------------------|-----------|----------|--------|
| 10M (Put+Get), seconds| 165       | 120      | 243    |
| 10M Put, ops/sec      | 122933    | 135709   | 43843  |
| 10M Get, ops/sec      | 118722    | 214981   | 666067 |
| FileSize,Mb           | 312       | 1370     | 381    |