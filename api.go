package pudge

import (
	"bytes"
	"encoding/gob"
	"os"
)

// DefaultConfig return default config
func DefaultConfig() *Config {
	return &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 1}
}

// Open return db object if it opened.
// Create new db if not exist.
// Read db to obj if exist.
// Or error if any.
// Default Config (if nil): &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 1}
func Open(f string, cfg *Config) (*Db, error) {
	if cfg == nil {
		cfg = &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 1}
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
		db.appendKey(k)
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
	start, _ := db.findKey(from, asc)
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
