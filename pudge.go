package pudge

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var (
	dbs struct {
		sync.RWMutex
		dbs map[string]*Db
	}
	// ErrKeyNotFound - key not found
	ErrKeyNotFound = errors.New("Error: key not found")
	mutex          = &sync.RWMutex{}
)

// Db represent database
type Db struct {
	sync.RWMutex
	name          string
	fk            *os.File
	fv            *os.File
	keys          [][]byte
	vals          map[string]*Cmd
	cancelSyncer  context.CancelFunc
	orderedInsert bool
}

// Cmd represent keys and vals addresses
type Cmd struct {
	Seek    uint32
	Size    uint32
	KeySeek uint32
	//CounterVal int64
}

// Config fo db
// Default FileMode = 0666
// Default DirMode = 0777
// Default SyncInterval = 1 sec, 0 - disable sync (os will sync, typically 30 sec or so)
type Config struct {
	FileMode      int  // 0666
	DirMode       int  // 0777
	SyncInterval  int  // in seconds
	OrderedInsert bool // keep keys sorted on insert
}

func init() {
	dbs.dbs = make(map[string]*Db)
}

// Open return db object if it opened.
// Create new db if not exist.
// Read db to obj if exist.
// Or error if any.
// Default Config (if nil): &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 1}
func Open(f string, cfg *Config) (*Db, error) {
	if cfg == nil {
		cfg = &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 1, OrderedInsert: false}
	}
	dbs.RLock()
	db, ok := dbs.dbs[f]
	if ok {
		dbs.RUnlock()
		return db, nil
	}
	dbs.RUnlock()
	dbs.Lock()
	db, err := newDb(f, cfg)
	if err == nil {
		dbs.dbs[f] = db
	}
	dbs.Unlock()
	return db, err
}

func newDb(f string, cfg *Config) (*Db, error) {
	log.Println("newdb1:", f)
	var err error
	// create
	db := new(Db)
	db.Lock()
	defer db.Unlock()
	// init
	db.name = f
	db.orderedInsert = cfg.OrderedInsert
	db.keys = make([][]byte, 0)
	db.vals = make(map[string]*Cmd)

	_, err = os.Stat(f)
	if err != nil {
		// file not exists - create dirs if any
		if os.IsNotExist(err) {
			if filepath.Dir(f) != "." {
				err = os.MkdirAll(filepath.Dir(f), os.FileMode(cfg.DirMode))
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	db.fv, err = os.OpenFile(f, os.O_CREATE|os.O_RDWR, os.FileMode(cfg.FileMode))
	if err != nil {
		return nil, err
	}
	db.fk, err = os.OpenFile(f+".idx", os.O_CREATE|os.O_RDWR, os.FileMode(cfg.FileMode))
	if err != nil {
		return nil, err
	}
	//read keys
	buf := new(bytes.Buffer)
	b, err := ioutil.ReadAll(db.fk)
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	var readSeek uint32
	for buf.Len() > 0 {
		_ = uint8(buf.Next(1)[0]) //format version
		t := uint8(buf.Next(1)[0])
		seek := binary.BigEndian.Uint32(buf.Next(4))
		size := binary.BigEndian.Uint32(buf.Next(4))
		_ = buf.Next(4) //time
		sizeKey := int(binary.BigEndian.Uint16(buf.Next(2)))
		key := buf.Next(sizeKey)
		strkey := string(key)
		cmd := &Cmd{
			Seek:    seek,
			Size:    size,
			KeySeek: readSeek,
		}
		readSeek += uint32(16 + sizeKey)
		switch t {
		case 0:
			if _, exists := db.vals[strkey]; !exists {
				//write new key at keys store
				db.appendKey(key, cfg.OrderedInsert)
			}
			db.vals[strkey] = cmd
		case 1:
			delete(db.vals, strkey)
			db.deleteFromKeys(key)
		}
	}

	if cfg.SyncInterval > 0 {
		db.backgroundManager(cfg.SyncInterval)
	}
	return db, err
}

// backgroundManager runs continuously in the background and performs various
// operations such as syncing to disk.
func (db *Db) backgroundManager(interval int) {
	ctx, cancel := context.WithCancel(context.Background())
	db.cancelSyncer = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				db.fk.Sync()
				db.fv.Sync()
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}
	}()
}

//appendAsc insert key in slice in ascending order
func (db *Db) appendKey(b []byte, ordered bool) {
	if !ordered {
		db.keys = append(db.keys, b)
		return
	}
	keysLen := len(db.keys)
	found := db.found(b)
	if found == 0 {
		//prepend
		db.keys = append([][]byte{b}, db.keys...)

	} else {
		if found >= keysLen {
			//not found - postpend ;)
			db.keys = append(db.keys, b)
		} else {
			//found
			//https://blog.golang.org/go-slices-usage-and-internals
			db.keys = append(db.keys, nil)           //grow origin slice capacity if needed
			copy(db.keys[found+1:], db.keys[found:]) //ha-ha, lol, 20x faster
			db.keys[found] = b
		}
	}
}

// deleteFromKeys delete key from slice keys
func (db *Db) deleteFromKeys(b []byte) {
	found := db.found(b)
	if found < len(db.keys) {
		if bytes.Equal(db.keys[found], b) {
			db.keys = append(db.keys[:found], db.keys[found+1:]...)
		}
	}
}

func (db *Db) sort() {
	if !db.orderedInsert {
		log.Println("sort")
		sort.Slice(db.keys, func(i, j int) bool {
			return bytes.Compare(db.keys[i], db.keys[j]) <= 0
		})
	}
}

//found return binary search result
func (db *Db) found(b []byte) int {
	db.sort()
	found := sort.Search(len(db.keys), func(i int) bool {
		return bytes.Compare(db.keys[i], b) >= 0
	})
	return found
}

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

func valToBinary(v interface{}) ([]byte, error) {
	var err error
	buf := new(bytes.Buffer)
	switch v.(type) {
	case []byte:
		return v.([]byte), nil
	default:
		err = gob.NewEncoder(buf).Encode(v)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), err
}

func writeKeyVal(fk, fv *os.File, readKey, writeVal []byte, exists bool, oldCmd *Cmd) (cmd *Cmd, err error) {

	var seek, newSeek int64
	cmd = &Cmd{Size: uint32(len(writeVal))}
	if exists {
		// key exists
		cmd.Seek = oldCmd.Seek
		cmd.KeySeek = oldCmd.KeySeek
		if oldCmd.Size >= uint32(len(writeVal)) {
			//write at old seek new value
			_, _, err = writeAtPos(fv, writeVal, int64(oldCmd.Seek))
		} else {
			//write at new seek (at the end of file)
			seek, _, err = writeAtPos(fv, writeVal, int64(-1))
			cmd.Seek = uint32(seek)
		}
		if err == nil {
			// if no error - store key at KeySeek
			newSeek, err = writeKey(fk, 0, cmd.Seek, cmd.Size, []byte(readKey), int64(cmd.KeySeek))
			cmd.KeySeek = uint32(newSeek)
		}
	} else {
		// new key
		// write value at the end of file
		seek, _, err = writeAtPos(fv, writeVal, int64(-1))
		cmd.Seek = uint32(seek)
		if err == nil {
			newSeek, err = writeKey(fk, 0, cmd.Seek, cmd.Size, []byte(readKey), -1)
			cmd.KeySeek = uint32(newSeek)
		}
	}
	return cmd, err
}

// if pos<0 store at the end of file
func writeAtPos(f *os.File, b []byte, pos int64) (seek int64, n int, err error) {
	seek = pos
	if pos < 0 {
		seek, err = f.Seek(0, 2)
		if err != nil {
			return seek, 0, err
		}
	}
	n, err = f.WriteAt(b, seek)
	if err != nil {
		return seek, n, err
	}
	return seek, n, err
}

// writeKey create buffer and store key with val address and size
func writeKey(fk *os.File, t uint8, seek, size uint32, key []byte, keySeek int64) (newSeek int64, err error) {
	//get buf from pool
	buf := new(bytes.Buffer)
	buf.Reset()
	buf.Grow(16 + len(key))

	//encode
	binary.Write(buf, binary.BigEndian, uint8(0))                  //1byte version
	binary.Write(buf, binary.BigEndian, t)                         //1byte command code(0-set,1-delete)
	binary.Write(buf, binary.BigEndian, seek)                      //4byte seek
	binary.Write(buf, binary.BigEndian, size)                      //4byte size
	binary.Write(buf, binary.BigEndian, uint32(time.Now().Unix())) //4byte timestamp
	binary.Write(buf, binary.BigEndian, uint16(len(key)))          //2byte key size
	buf.Write(key)                                                 //key

	if keySeek < 0 {
		newSeek, _, err = writeAtPos(fk, buf.Bytes(), int64(-1))
	} else {
		newSeek, _, err = writeAtPos(fk, buf.Bytes(), int64(keySeek))
	}

	return newSeek, err
}

// Set store any key value to db
func (db *Db) Set(key, value interface{}) error {
	db.Lock()
	defer db.Unlock()
	k, err := keyToBinary(key)
	if err != nil {
		return err
	}
	v, err := valToBinary(value)
	if err != nil {
		return err
	}
	//log.Println("Set:", k, v)
	oldCmd, exists := db.vals[string(k)]
	cmd, err := writeKeyVal(db.fk, db.fv, k, v, exists, oldCmd)
	if err != nil {
		return err
	}
	db.vals[string(k)] = cmd
	//switch value.(type) {
	//case int64:
	//	cmd.CounterVal = value.(int64)
	//}
	if !exists {
		db.appendKey(k, db.orderedInsert)
	}

	return err
}

// Close - sync & close files.
// Return error if any.
func (db *Db) Close() error {
	if db.cancelSyncer != nil {
		db.cancelSyncer()
	}
	db.Lock()
	defer db.Unlock()
	err := db.fk.Sync()
	if err != nil {
		return err
	}
	err = db.fv.Sync()
	if err != nil {
		return err
	}
	err = db.fk.Close()
	if err != nil {
		return err
	}
	err = db.fv.Close()
	if err != nil {
		return err
	}
	dbs.Lock()
	delete(dbs.dbs, db.name)
	dbs.Unlock()
	return nil
}

// CloseAll - close all opened Db
func CloseAll() (err error) {
	dbs.Lock()
	stores := dbs.dbs
	dbs.Unlock()
	for _, db := range stores {
		err = db.Close()
		if err != nil {
			break
		}
	}

	return err
}

// DeleteFile close and delete file
func (db *Db) DeleteFile() error {
	return DeleteFile(db.name)
}

func DeleteFile(file string) error {
	dbs.Lock()
	db, ok := dbs.dbs[file]
	if ok {
		dbs.Unlock()
		err := db.Close()
		if err != nil {
			return err
		}
	} else {
		dbs.Unlock()
	}

	err := os.Remove(file)
	if err != nil {
		return err
	}
	err = os.Remove(file + ".idx")
	return err
}

// Get return value by key
// Return error if any.
func (db *Db) Get(key, value interface{}) error {
	db.RLock()
	defer db.RUnlock()
	k, err := keyToBinary(key)
	if err != nil {
		return err
	}
	if val, ok := db.vals[string(k)]; ok {
		switch value.(type) {
		//case *int64:
		//	*value.(*int64) = val.CounterVal
		case *[]byte:
			b := make([]byte, val.Size)
			_, err := db.fv.ReadAt(b, int64(val.Seek))
			if err != nil {
				return err
			}
			*value.(*[]byte) = b
			return nil
		default:

			buf := new(bytes.Buffer)
			b := make([]byte, val.Size)
			_, err := db.fv.ReadAt(b, int64(val.Seek))
			if err != nil {
				return err
			}
			buf.Write(b)
			err = gob.NewDecoder(buf).Decode(value)
			return err
		}
	}
	return ErrKeyNotFound
}

// Has return true if key exists.
// Return error if any.
func (db *Db) Has(key interface{}) (bool, error) {
	db.RLock()
	defer db.RUnlock()
	k, err := keyToBinary(key)
	if err != nil {
		return false, err
	}
	_, has := db.vals[string(k)]
	return has, nil
}

// FileSize returns the total size of the disk storage used by the DB.
func (db *Db) FileSize() (int64, error) {
	db.RLock()
	defer db.RUnlock()
	var err error
	is, err := db.fk.Stat()
	if err != nil {
		return -1, err
	}
	ds, err := db.fv.Stat()
	if err != nil {
		return -1, err
	}
	return is.Size() + ds.Size(), nil
}

// Count returns the number of items in the Db.
func (db *Db) Count() int {
	db.RLock()
	defer db.RUnlock()
	return len(db.keys)
}

// Delete remove key
// Returns error if key not found
func (db *Db) Delete(key interface{}) error {
	db.Lock()
	defer db.Unlock()
	k, err := keyToBinary(key)
	if err != nil {
		return err
	}
	if _, ok := db.vals[string(k)]; ok {
		delete(db.vals, string(k))
		db.deleteFromKeys(k)
		writeKey(db.fk, 1, 0, 0, k, -1)
		return nil
	}
	return ErrKeyNotFound
}

// Keys return keys in ascending  or descending order (false - descending,true - ascending)
// if limit == 0 return all keys
// if offset > 0 - skip offset records
// If from not nil - return keys after from (from not included)
func (db *Db) Keys(from interface{}, limit, offset int, asc bool) ([][]byte, error) {
	db.RLock()
	defer db.RUnlock()
	end := 0
	start, err := db.FindKey(from, asc)
	if err != nil {
		return nil, err
	}
	excludeFrom := 0
	if from != nil {
		excludeFrom = 1
	}
	if asc {
		start += (offset + excludeFrom)
		if limit == 0 {
			end = len(db.keys) - excludeFrom
		} else {
			end = (start + limit - 1)
		}
	} else {
		start -= (offset + excludeFrom)
		if limit == 0 {
			end = 0
		} else {
			end = start - limit + 1
		}
	}
	if end < 0 {
		end = 0
	}
	if end >= len(db.keys) {
		end = len(db.keys) - 1
	}
	// resulting array
	arr := make([][]byte, 0, 0)
	if start < 0 || start >= len(db.keys) {
		return arr, nil
	}
	if asc {
		for i := start; i <= end; i++ {
			arr = append(arr, db.keys[i])
		}
	} else {
		for i := start; i >= end; i-- {
			arr = append(arr, db.keys[i])
		}
	}
	return arr, nil
}

// FindKey return index of first key in ascending mode
// FindKey return index of last key in descending mode
func (db *Db) FindKey(key interface{}, asc bool) (int, error) {
	if key == nil {
		db.sort()
		if asc {
			return 0, nil
		}
		return len(db.keys) - 1, nil
	}
	k, err := keyToBinary(key)
	if err != nil {
		return -1, err
	}
	found := db.found(k)
	// check found
	if found >= len(db.keys) {
		return -1, ErrKeyNotFound
	}
	if !bytes.Equal(db.keys[found], k) {
		return -1, ErrKeyNotFound
	}
	return found, nil
}

// Counter return int64 incremented on incr
func (db *Db) Counter(key interface{}, incr int) (int64, error) {
	var counter int64
	err := db.Get(key, &counter)
	if err != nil && err != ErrKeyNotFound {
		return -1, err
	}
	mutex.Lock()
	counter = counter + int64(incr)
	mutex.Unlock()
	err = db.Set(key, counter)
	return counter, nil
}

// Set store any key value to db with opening if needed
func Set(f string, key, value interface{}) error {
	db, err := Open(f, nil)
	if err != nil {
		return err
	}
	return db.Set(key, value)
}

// Get return value by key with opening if needed
// Return error if any.
func Get(f string, key, value interface{}) error {
	db, err := Open(f, nil)
	if err != nil {
		return err
	}
	return db.Get(key, value)
}

// Counter return int64 incremented on incr with lazy open
func Counter(f string, key interface{}, incr int) (int64, error) {
	db, err := Open(f, nil)
	if err != nil {
		return 0, err
	}
	return db.Counter(key, incr)
}
