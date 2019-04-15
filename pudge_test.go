package pudge

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	f = "test/1"
)

func nrandbin(n int) [][]byte {
	i := make([][]byte, n)
	for ind := range i {
		bin, _ := KeyToBinary(rand.Int())
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
		db.Set(k, v)
	}
	for i := 22; i >= 1; i-- {
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
	if s != "01020304050607080910111213141516171819202122" {
		t.Error("not asc", s)
	}

	//descending
	resdesc, _ := db.Keys(nil, 0, 0, false)
	s = ""
	for _, r := range resdesc {
		s += string(r)
	}
	if s != "22212019181716151413121110090807060504030201" {
		t.Error("not desc", s)
	}

	//offset limit asc
	reslimit, _ := db.Keys(nil, 2, 2, true)

	s = ""
	for _, r := range reslimit {
		s += string(r)
	}
	if s != "0304" {
		t.Error("not off", s)
	}

	//offset limit desc
	reslimitdesc, _ := db.Keys(nil, 2, 2, false)

	s = ""
	for _, r := range reslimitdesc {
		s += string(r)
	}
	if s != "2019" {
		t.Error("not off desc", s)
	}

	//from byte asc
	resfromasc, _ := db.Keys([]byte("10"), 2, 2, true)
	s = ""
	for _, r := range resfromasc {
		s += string(r)
	}
	if s != "1314" {
		t.Error("not off asc", s)
	}

	//from byte desc
	resfromdesc, _ := db.Keys([]byte("10"), 2, 2, false)
	s = ""
	for _, r := range resfromdesc {
		s += string(r)
	}
	if s != "0706" {
		t.Error("not off desc", s)
	}

	//from byte desc
	resnotfound, _ := db.Keys([]byte("100"), 2, 2, false)
	s = ""
	for _, r := range resnotfound {
		s += string(r)
	}
	if s != "" {
		t.Error("resnotfound", s)
	}

	//from byte not eq
	resnoteq, _ := db.Keys([]byte("33"), 2, 2, false)
	s = ""
	for _, r := range resnoteq {
		s += string(r)
	}
	if s != "" {
		t.Error("resnoteq ", s)
	}

	//by prefix
	respref, _ := db.Keys([]byte("2*"), 4, 0, false)
	s = ""
	for _, r := range respref {
		s += string(r)
	}
	if s != "222120" {
		t.Error("respref", s)
	}

	//by prefix2
	respref2, _ := db.Keys([]byte("1*"), 2, 0, false)
	s = ""
	for _, r := range respref2 {
		s += string(r)
	}
	if s != "1918" {
		t.Error("respref2", s)
	}

	//by prefixasc
	resprefasc, err := db.Keys([]byte("1*"), 2, 0, true)
	s = ""
	for _, r := range resprefasc {
		s += string(r)
	}
	if s != "1011" {
		t.Error("resprefasc", s, err)
	}

	//by prefixasc2
	resprefasc2, err := db.Keys([]byte("1*"), 0, 0, true)
	s = ""
	for _, r := range resprefasc2 {
		s += string(r)
	}
	if s != "10111213141516171819" {
		t.Error("resprefasc2", s, err)
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
		if err != nil {
			t.Error(err)
		}
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
		counter, _ = db.Counter(key2, 1)
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

func TestStoreMode(t *testing.T) {
	cfg := &Config{StoreMode: 2}
	db, err := Open("test/sm", cfg)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 2)
	if err != nil {
		t.Error(err)
	}
	var v int
	err = db.Get(1, &v)
	if err != nil {
		t.Error(err)
	}
	if v != 2 {
		t.Error("not 2")
	}
	db.Set(1, 42)
	db.Close()
	db, err = Open("test/sm", nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Get(1, &v)
	if err != nil {
		t.Error(err)
	}
	if v != 42 {
		t.Error("not 42")
	}
	DeleteFile("test/sm")
	//log.Println(v)
	//CloseAll()
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

func TestBackup(t *testing.T) {
	Set("test/1", 1, 2)
	Set("test/4", "4", "4")
	BackupAll("")
	DeleteFile("test/1")
	DeleteFile("test/4")
	var v1 int
	Get("backup/test/1", 1, &v1)
	if v1 != 2 {
		t.Error("not 2")
	}
	var v2 = ""
	Get("backup/test/4", "4", &v2)
	if v2 != "4" {
		t.Error("not 4")
	}
	DeleteFile("backup/test/1")
	DeleteFile("backup/test/4")
	CloseAll()
}

func TestMultipleOpen(t *testing.T) {
	for i := 1; i < 100000; i++ {
		Set("test/m", i, i)
	}
	Close("test/m")
	for i := 1; i < 100; i++ {
		go Open("test/m", nil)
	}
	time.Sleep(1 * time.Millisecond)
	DeleteFile("test/m")
}

func TestInMemory(t *testing.T) {
	DefaultConfig.StoreMode = 2

	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("test/inmemory%d", i)
		err := Set(fileName, i, i)
		if err != nil {
			t.Error(err)
		}
	}

	err := CloseAll()
	if err != nil {
		t.Error(err)
	}
	for i := 0; i < 10; i++ {
		fileName := fmt.Sprintf("test/inmemory%d", i)
		c, e := Count(fileName)
		if c == 0 || e != nil {
			t.Error("no persist")
			break
		}
		DeleteFile(fileName)
	}
}

func TestInMemoryWithoutPersist(t *testing.T) {
	DefaultConfig.StoreMode = 2

	for i := 0; i < 10000; i++ {
		err := Set("", i, i)
		if err != nil {
			t.Error(err)
		}
	}
	j := 0
	Get("", 8, &j)
	if j != 8 {
		t.Error("j must be 8", j)
	}
	cnt, e := Count("")
	if cnt != 10000 {
		t.Error("count must be 10000", cnt, e)
	}
	for i := 0; i < 10; i++ {
		c, e := Count("")
		if c != 10000 || e != nil {
			t.Error("no persist", c, e)
			break
		}
	}
	noerr := DeleteFile("")
	if noerr != nil {
		t.Error("Delete empty file", noerr)
	}
	noerr = Close("")
	if noerr != nil {
		t.Error("Close empty file", noerr)
	}
	jj := 0
	notpresent := Get("", 8, &jj)
	if jj == 8 {
		t.Error("jj  must be 0", j)
	}
	if notpresent != ErrKeyNotFound {
		t.Error("Must be Error: key not found error", notpresent)
	}

}

func Test42(t *testing.T) {
	DefaultConfig.StoreMode = 0
	f := "test/int64"
	for i := 1; i < 64; i++ {
		Set(f, int64(i), int64(i))
	}
	keys, err := Keys(f, int64(42), 100, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(keys) != 22 {
		t.Error("not 21", len(keys))
	}
	DeleteFile(f)
}
