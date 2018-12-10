package ss

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"log"
	"math/rand"
	"sort"
	"strconv"
	"testing"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
	"github.com/emirpasic/gods/utils"
	"github.com/huandu/skiplist"
	"github.com/plar/go-adaptive-radix-tree"
)

func keyToBinary(v interface{}) ([]byte, error) {
	var err error

	buf := new(bytes.Buffer)
	switch v.(type) {
	case []byte:
		return v.([]byte), nil
	case bool, float32, float64, complex64, complex128, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		err = binary.Write(buf, binary.BigEndian, v)
	case int:
		err = binary.Write(buf, binary.BigEndian, int64(v.(int)))
	case string:
		_, err = buf.Write([]byte((v.(string))))
	default:
		err = gob.NewEncoder(buf).Encode(v)
	}
	return buf.Bytes(), err
}
func TestArt(t *testing.T) {
	tree := art.New()
	items := make([][]byte, 0)
	for i := 11; i >= 0; i-- {
		bin, _ := keyToBinary(i)
		items = append(items, bin)
	}

	for _, v := range items {
		tree.Insert(v, v)
	}

	tree.ForEach(func(node art.Node) bool {

		buf := new(bytes.Buffer)
		buf.Write(node.Value().([]byte))
		var j int64
		binary.Read(buf, binary.BigEndian, &j)
		//binary.Read(bytes.Reader(b.([]byte)), binary.BigEndian, &j)
		//fmt.Printf("Callback value=%v\n", j)
		return true
	})

	for it := tree.Iterator(); it.HasNext(); {
		value, _ := it.Next()
		_ = value
		//fmt.Printf("Iterator value=%v\n", value.Value())
	}
	//fmt.Println(tree)
}

func nrandbin(n int) [][]byte {
	bins := make([][]byte, n)
	for i := 0; i < n; i++ {
		bin, _ := keyToBinary(i)
		bins[i] = bin
	}
	return bins
}

// Keys generator
func randKeys(totalKeys int) [][]byte {
	keys := make([][]byte, totalKeys)
	for i := range keys {
		keys[i] = make([]byte, 8)
		rand.Read(keys[i])
	}
	return keys
}

//BenchmarkArtSetRand-4            2000000               888 ns/op           9.01 MB/s         129 B/op          3 allocs/op
func BenchmarkArtSetRand(b *testing.B) {
	tree := art.New()
	bins := nrandbin(b.N)
	b.ResetTimer()
	b.SetBytes(8)
	for _, v := range bins {
		tree.Insert(v, v)
	}
}

//BenchmarkArtSetOrder-4           3000000               408 ns/op          19.57 MB/s         221 B/op          7 allocs/op
func BenchmarkArtSetOrder(b *testing.B) {
	tree := art.New()

	b.ResetTimer()
	b.SetBytes(8)
	for i := 0; i < b.N; i++ {
		bin, _ := keyToBinary(i)
		tree.Insert(bin, nil)
	}
}

//BenchmarkArtSetOrderDesc-4       3000000               504 ns/op          15.87 MB/s         221 B/op          7 allocs/op
func BenchmarkArtSetOrderDesc(b *testing.B) {
	tree := art.New()

	b.ResetTimer()
	b.SetBytes(8)
	for i := b.N; i >= 0; i-- {
		bin, _ := keyToBinary(i)
		tree.Insert(bin, nil)
	}
}

//BenchmarkArtGetRand-4            5000000               323 ns/op          24.70 MB/s           0 B/op          0 allocs/op
func BenchmarkArtGetRand(b *testing.B) {
	tree := art.New()
	bins := nrandbin(b.N)
	for _, v := range bins {
		tree.Insert(v, nil)
	}
	b.ResetTimer()
	b.SetBytes(8)
	for _, v := range bins {
		_, f := tree.Search(v)
		if !f {
			log.Fatal("not found")
		}
	}
}

//BenchmarkHash-4         10000000               364 ns/op          21.94 MB/s           7 B/op          0 allocs/op
func BenchmarkHash(b *testing.B) {
	var m = make(map[string]string)
	b.SetBytes(8)
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		m[s] = s
	}
	b.ResetTimer()
	//log.Println(len(m))
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		v, ok := m[s]
		if !ok {
			log.Fatal("not found BenchmarkHash", s, v)
		}
	}
}

//Load
//BenchmarkMa-4            2000000               658 ns/op          12.14 MB/s          87 B/op          2 allocs/op
//Read
//BenchmarkMa-4            5000000               422 ns/op          18.93 MB/s          23 B/op          1 allocs/op
func BenchmarkRbtGet(b *testing.B) {
	tree := rbt.NewWithStringComparator()
	b.SetBytes(8)
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		tree.Put(s, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s := strconv.Itoa(i)
		tree.Get(s)
	}
}

func BenchmarkStoreGodsBtree(b *testing.B) {
	//b.N = 100000
	tree := rbt.NewWith(utils.BinComparator)
	b.SetBytes(8)
	nums := nrand(b.N) //100000)
	for _, v := range nums {
		bin, _ := keyToBinary(v)
		//s := strconv.Itoa(v)
		tree.Put(bin, bin)
	}
}

func nrand(n int) []int {
	i := make([]int, n)
	for ind := range i {
		i[ind] = rand.Int()
	}
	return i
}

func BenchmarkLoadGodsbtree(b *testing.B) {
	tree := rbt.NewWithStringComparator()
	b.SetBytes(8)
	nums := nrand(b.N)
	for _, v := range nums {
		s := strconv.Itoa(v)
		tree.Put(s, nil)
	}
	b.ResetTimer()
	for _, v := range nums {
		s := strconv.Itoa(v)
		tree.Get(s)
	}
}

func BenchmarkSLSet(b *testing.B) {
	list := skiplist.New(skiplist.String)
	b.ResetTimer()
	b.SetBytes(8)
	for i := 0; i < b.N; i++ {
		bin, _ := keyToBinary(i)
		list.Set(bin, nil)
	}
}

func TestSSNew(t *testing.T) {
	ss := New()
	ss.Set([]byte("1"))
	log.Println(ss)
}

//BenchmarkSlice-4        20000000               309 ns/op          25.82 MB/s
func BenchmarkSlice(b *testing.B) {
	b.StopTimer()
	s := make([][]byte, 0)
	bins := randKeys(b.N)
	//b.ResetTimer()
	//b.SetBytes(8)
	for _, v := range bins {
		s = append(s, v)

	}
	b.StartTimer()
	if !sort.SliceIsSorted(s, func(i, j int) bool {
		return bytes.Compare(s[i], s[j]) > 0
	}) {
		s = sortSlice(s)
	}

	b.StopTimer()
	for i, j := range s {
		log.Println(binary.BigEndian.Uint64(j), j)
		if i > 4 {
			log.Println("=====")
			break
		}
	}

}

func bubleSort(x [][]byte) [][]byte {
	end := len(x) - 1

	for {

		if end == 0 {
			break
		}

		for i := 0; i < len(x)-1; i++ {

			if bytes.Compare(x[i], x[i+1]) <= 0 {
				x[i], x[i+1] = x[i+1], x[i]
			}

		}

		end -= 1

	}
	return x
}

func sortSlice(s [][]byte) [][]byte {
	sort.Slice(s, func(i, j int) bool {
		return bytes.Compare(s[i], s[j]) > 0
	})
	return s
}
