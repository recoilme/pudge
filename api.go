package pudge

import (
	"bytes"
	"encoding/gob"
	"os"
)

// DefaultConfig is default config
var DefaultConfig = &Config{
	FileMode:     0666,
	DirMode:      0777,
	SyncInterval: 0,
	StoreMode:    0}

// Open return db object if it opened.
// Create new db if not exist.
// Read db to obj if exist.
// Or error if any.
// Default Config (if nil): &Config{FileMode: 0666, DirMode: 0777, SyncInterval: 0}
func Open(f string, cfg *Config) (*Db, error) {
	if cfg == nil {
		cfg = DefaultConfig
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
	//log.Println("n", db.name, db.config.StoreMode)
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
	k, err := KeyToBinary(key)
	if err != nil {
		return err
	}
	v, err := ValToBinary(value)
	if err != nil {
		return err
	}
	//log.Println("Set:", k, v)
	oldCmd, exists := db.vals[string(k)]
	//fmt.Println("StoreMode", db.config.StoreMode)
	if db.storemode == 2 {
		cmd := &Cmd{}
		cmd.Size = uint32(len(v))
		cmd.Val = make([]byte, len(v))
		copy(cmd.Val, v)
		db.vals[string(k)] = cmd
	} else {
		cmd, err := writeKeyVal(db.fk, db.fv, k, v, exists, oldCmd)
		if err != nil {
			return err
		}
		db.vals[string(k)] = cmd
	}
	if !exists {
		db.appendKey(k)
	}

	return err
}

// Get return value by key
// Return error if any.
func (db *Db) Get(key, value interface{}) error {
	db.RLock()
	defer db.RUnlock()
	k, err := KeyToBinary(key)
	if err != nil {
		return err
	}
	if val, ok := db.vals[string(k)]; ok {
		switch value.(type) {
		case *[]byte:
			b := make([]byte, val.Size)
			if db.storemode == 2 {
				copy(b, val.Val)
			} else {
				_, err := db.fv.ReadAt(b, int64(val.Seek))
				if err != nil {
					return err
				}
			}
			*value.(*[]byte) = b
			return nil
		default:

			buf := new(bytes.Buffer)
			b := make([]byte, val.Size)
			if db.storemode == 2 {
				//fmt.Println(val)
				copy(b, val.Val)
			} else {
				_, err := db.fv.ReadAt(b, int64(val.Seek))
				if err != nil {
					return err
				}
			}
			buf.Write(b)
			err = gob.NewDecoder(buf).Decode(value)
			return err
		}
	}
	return ErrKeyNotFound
}

// Close - sync & close files.
// Return error if any.
func (db *Db) Close() error {
	if db.cancelSyncer != nil {
		db.cancelSyncer()
	}
	db.Lock()
	defer db.Unlock()

	if db.storemode == 2 && db.name != "" {
		db.sort()
		keys := make([][]byte, len(db.keys))

		copy(keys, db.keys)

		db.storemode = 0
		for _, k := range keys {
			if val, ok := db.vals[string(k)]; ok {
				writeKeyVal(db.fk, db.fv, k, val.Val, false, nil)
			}
		}
	}
	if db.fk != nil {
		err := db.fk.Sync()
		if err != nil {
			return err
		}
		err = db.fk.Close()
		if err != nil {
			return err
		}
	}
	if db.fv != nil {
		err := db.fv.Sync()
		if err != nil {
			return err
		}

		err = db.fv.Close()
		if err != nil {
			return err
		}
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

// DeleteFile close db and delete file
func DeleteFile(file string) error {
	if file == "" {
		return nil
	}
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

// Has return true if key exists.
// Return error if any.
func (db *Db) Has(key interface{}) (bool, error) {
	db.RLock()
	defer db.RUnlock()
	k, err := KeyToBinary(key)
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
func (db *Db) Count() (int, error) {
	db.RLock()
	defer db.RUnlock()
	return len(db.keys), nil
}

// Delete remove key
// Returns error if key not found
func (db *Db) Delete(key interface{}) error {
	db.Lock()
	defer db.Unlock()
	k, err := KeyToBinary(key)
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

// KeysByPrefix return keys with prefix
// in ascending  or descending order (false - descending,true - ascending)
// if limit == 0 return all keys
// if offset > 0 - skip offset records
// If from not nil - return keys after from (from not included)
func (db *Db) KeysByPrefix(prefix []byte, limit, offset int, asc bool) ([][]byte, error) {
	//log.Println("KeysByPrefix")
	db.RLock()
	defer db.RUnlock()
	// resulting array
	arr := make([][]byte, 0, 0)
	found := db.foundPref(prefix, asc)
	if found >= len(db.keys) || !startFrom(db.keys[found], prefix) {
		//not found
		return arr, ErrKeyNotFound
	}

	start, end := checkInterval(found, limit, offset, 0, len(db.keys), asc)

	if start < 0 || start >= len(db.keys) {
		return arr, nil
	}

	if asc {
		for i := start; i <= end; i++ {
			if !startFrom(db.keys[i], prefix) {
				break
			}
			arr = append(arr, db.keys[i])
		}
	} else {
		for i := start; i >= end; i-- {
			if !startFrom(db.keys[i], prefix) {
				break
			}
			arr = append(arr, db.keys[i])
		}
	}
	return arr, nil
}

// Keys return keys in ascending  or descending order (false - descending,true - ascending)
// if limit == 0 return all keys
// if offset > 0 - skip offset records
// If from not nil - return keys after from (from not included)
func (db *Db) Keys(from interface{}, limit, offset int, asc bool) ([][]byte, error) {
	// resulting array
	//log.Println("pudge", from, from == nil)
	arr := make([][]byte, 0, 0)
	excludeFrom := 0
	if from != nil {
		excludeFrom = 1

		k, err := KeyToBinary(from)
		//log.Println(bytes.Equal(k[len(k)-1:], []byte("*")))
		if err != nil {
			return arr, err
		}
		if len(k) > 1 && bytes.Equal(k[len(k)-1:], []byte("*")) {
			byteOrStr := false
			switch from.(type) {
			case []byte:
				byteOrStr = true
			case string:
				byteOrStr = true
			}
			if byteOrStr {
				prefix := make([]byte, len(k)-1)
				copy(prefix, k)
				return db.KeysByPrefix(prefix, limit, offset, asc)
			}
		}
	}
	db.RLock()
	defer db.RUnlock()
	find, _ := db.findKey(from, asc)
	start, end := checkInterval(find, limit, offset, excludeFrom, len(db.keys), asc)
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
	mutex.Lock()
	var counter int64
	err := db.Get(key, &counter)
	if err != nil && err != ErrKeyNotFound {
		return -1, err
	}
	//mutex.Lock()
	counter = counter + int64(incr)
	//mutex.Unlock()
	err = db.Set(key, counter)
	mutex.Unlock()
	return counter, err
}

// Set store any key value to db with opening if needed
func Set(f string, key, value interface{}) error {
	db, err := Open(f, nil)
	if err != nil {
		return err
	}
	return db.Set(key, value)
}

// Sets store vals and keys
// Use it for mass insertion
// every pair must contain key and value
func Sets(file string, pairs []interface{}) (err error) {
	db, err := Open(file, nil)
	if err != nil {
		return err
	}
	for i := range pairs {
		if i%2 != 0 {
			// on odd - append val and store key
			if pairs[i] == nil || pairs[i-1] == nil {
				break
			}
			err = db.Set(pairs[i-1], pairs[i])
			if err != nil {
				break
			}
		}
	}
	return err
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

// Gets return key/value pairs in random order
// result contains key and value
// Gets not return error if key not found
// If no keys found return empty result
func Gets(file string, keys []interface{}) (result [][]byte) {
	db, err := Open(file, nil)
	if err != nil {
		return nil
	}
	for _, key := range keys {
		var v []byte
		err := db.Get(key, &v)
		if err == nil {
			k, err := KeyToBinary(key)
			if err == nil {
				val, err := ValToBinary(v)
				if err == nil {
					result = append(result, k)
					result = append(result, val)
				}
			}
		}
	}
	return result
}

// Counter return int64 incremented on incr with lazy open
func Counter(f string, key interface{}, incr int) (int64, error) {
	db, err := Open(f, nil)
	if err != nil {
		return 0, err
	}
	return db.Counter(key, incr)
}

// Delete remove key
// Returns error if key not found
func Delete(f string, key interface{}) error {
	db, err := Open(f, nil)
	if err != nil {
		return err
	}
	return db.Delete(key)
}

// Keys return keys in ascending  or descending order (false - descending,true - ascending)
// if limit == 0 return all keys
// if offset > 0 - skip offset records
// If from not nil - return keys after from (from not included)
func Keys(f string, from interface{}, limit, offset int, asc bool) ([][]byte, error) {
	db, err := Open(f, nil)
	if err != nil {
		return nil, err
	}
	return db.Keys(from, limit, offset, asc)
}

// Has return true if key exists.
// Return error if any.
func Has(f string, key interface{}) (bool, error) {
	db, err := Open(f, nil)
	if err != nil {
		return false, err
	}
	return db.Has(key)
}

// Count returns the number of items in the Db.
func Count(f string) (int, error) {
	db, err := Open(f, nil)
	if err != nil {
		return -1, err
	}
	return db.Count()
}

// Close - sync & close files.
// Return error if any.
func Close(f string) error {
	db, err := Open(f, nil)
	if err != nil {
		return err
	}
	return db.Close()
}

// BackupAll - backup all opened Db
// if dir not set it will be backup
// delete old backup file before run
// ignore all errors
func BackupAll(dir string) (err error) {
	if dir == "" {
		dir = "backup"
	}
	dbs.Lock()
	stores := dbs.dbs
	dbs.Unlock()
	//tmp := make(map[string]string)
	for _, db := range stores {
		backup := dir + "/" + db.name
		DeleteFile(backup)
		keys, err := db.Keys(nil, 0, 0, true)
		if err == nil {
			for _, k := range keys {
				var b []byte
				db.Get(k, &b)
				Set(backup, k, b)
			}
		}
		Close(backup)
	}

	return err
}
