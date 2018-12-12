package pudge

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"testing"
)

const (
	f = "test/1"
)

func nrandbin(n int) [][]byte {
	i := make([][]byte, n)
	for ind := range i {
		bin, _ := keyToBinary(rand.Int())
		i[ind] = bin
	}
	return i
}

func TestConfig(t *testing.T) {
	_, err := Open("", nil)
	if err == nil {
		t.Error("Open empty must error")
	}
	db, err := Open(f, &Config{FileMode: 0777, DirMode: 0777})
	if err != nil {
		t.Error(err)
	}
	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestOpen(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Error(err)
	}
	db.Close()
	db, err = Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	v := 1
	err = db.Get(1, &v)
	if err != nil {
		t.Error(err)
	}
	if v != 1 {
		t.Error("not 1")
	}
	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestSet(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Error(err)
	}
	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestGet(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Error(err)
	}
	var val int
	err = db.Get(1, &val)
	if err != nil {
		t.Error(err)
		return
	}

	if val != 1 {
		t.Error("val != 1", val)
		return
	}
	db.Close()

	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestKeys(t *testing.T) {

	f := "test/keys.db"

	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	append := func(i int) {
		k := []byte(fmt.Sprintf("%02d", i))
		v := []byte("Val:" + strconv.Itoa(i))
		err := db.Set(k, v)
		if err != nil {
			t.Error(err)
		}
	}
	for i := 20; i >= 1; i-- {
		append(i)
	}

	//ascending
	res, err := db.Keys(nil, 0, 0, true)
	if err != nil {
		t.Error(err)
	}
	var s = ""
	for _, r := range res {
		s += string(r)
	}
	if s != "0102030405060708091011121314151617181920" {
		t.Error("not asc", s)
	}

	//descending
	resdesc, err := db.Keys(nil, 0, 0, false)
	if err != nil {
		t.Error(err)
	}
	s = ""
	for _, r := range resdesc {
		s += string(r)
	}
	if s != "2019181716151413121110090807060504030201" {
		t.Error("not desc", s)
	}

	//offset limit asc
	reslimit, err := db.Keys(nil, 2, 2, true)
	if err != nil {
		t.Error(err)
	}
	s = ""
	for _, r := range reslimit {
		s += string(r)
	}
	if s != "0304" {
		t.Error("not off", s)
	}

	//offset limit desc
	reslimitdesc, err := db.Keys(nil, 2, 2, false)
	if err != nil {
		t.Error(err)
	}
	s = ""
	for _, r := range reslimitdesc {
		s += string(r)
	}
	if s != "1817" {
		t.Error("not off desc", s)
	}
	DeleteFile(f)
}

func TestCounter(t *testing.T) {
	f := "test/TestCnt.db"
	var counter int64
	var err error
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	key := []byte("postcounter")
	for i := 0; i < 10; i++ {
		counter, err = db.Counter(key, 1)
		//log.Println(counter, err)
	}
	//return
	for i := 0; i < 10; i++ {
		counter, err = db.Counter(key, 1)
		if err != nil {
			t.Error(err)
		}
	}
	if counter != 20 {
		t.Error("counter!=20")
	}
	key2 := []byte("counter2")
	for i := 0; i < 5; i++ {
		counter, err = db.Counter(key2, 1)
	}

	for i := 0; i < 5; i++ {
		counter, err = db.Counter(key2, 1)
		if err != nil {
			t.Error(err)
		}
	}
	if counter != 10 {
		t.Error("counter!=10")
	}
	db.DeleteFile()
}

func TestLazyOpen(t *testing.T) {
	Set(f, 2, 42)
	var val int
	CloseAll()
	Get(f, 2, &val)
	if val != 42 {
		t.Error("not 42")
	}
	DeleteFile(f)
}

func TestAsync(t *testing.T) {
	len := 5000
	file := "test/async.db"
	DeleteFile(file)
	defer CloseAll()

	messages := make(chan int)
	readmessages := make(chan string)
	var wg sync.WaitGroup

	append := func(i int) {
		defer wg.Done()
		k := ("Key:" + strconv.Itoa(i))
		v := ("Val:" + strconv.Itoa(i))
		err := Set(file, []byte(k), []byte(v))
		if err != nil {
			t.Error(err)
		}
		messages <- i
	}

	read := func(i int) {
		defer wg.Done()
		k := ("Key:" + strconv.Itoa(i))
		v := ("Val:" + strconv.Itoa(i))
		var b []byte
		Get(file, []byte(k), &b)

		if string(b) != string(v) {
			t.Error("not mutch", string(b), string(v))
		}
		readmessages <- fmt.Sprintf("read N:%d  content:%s", i, string(b))
	}

	for i := 1; i <= len; i++ {
		wg.Add(1)
		go append(i)

	}

	go func() {
		for i := range messages {
			_ = i
			//fmt.Println(i)
		}
	}()

	go func() {
		for i := range readmessages {
			_ = i
			//fmt.Println(i)
		}
	}()

	wg.Wait()

	for i := 1; i <= len; i++ {

		wg.Add(1)
		go read(i)
	}
	wg.Wait()
	DeleteFile(file)
}

// run go test -bench=Store -benchmem
func BenchmarkStore(b *testing.B) {
	b.StopTimer()
	nums := nrandbin(b.N)

	DeleteFile(f)

	rm, err := Open(f, nil)
	if err != nil {
		b.Error("Open", err)
	}
	b.SetBytes(8)
	b.StartTimer()
	for _, v := range nums {
		err = rm.Set(v, v)
		if err != nil {
			b.Error("Set", err)
		}
	}
	b.StopTimer()
	err = DeleteFile(f)
	if err != nil {
		b.Error("DeleteFile", err)
	}
}

func BenchmarkLoad(b *testing.B) {
	b.StopTimer()
	nums := nrandbin(b.N)
	DeleteFile(f)
	rm, err := Open(f, nil)
	if err != nil {
		b.Error("Open", err)
	}
	for _, v := range nums {
		err = rm.Set(v, v)
		if err != nil {
			b.Error("Set", err)
		}
	}
	var wg sync.WaitGroup
	read := func(db *Db, key []byte) {
		defer wg.Done()
		var b []byte
		db.Get(key, &b)
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		go read(rm, nums[i])
		//var v []byte
		//err := rm.Get(nums[i], &v)
		//if err != nil {
		//	log.Println("Get", err, nums[i], &v)
		//	break
		//}
	}
	wg.Wait()
	b.StopTimer()
	log.Println(rm.Count())
	DeleteFile(f)
}
