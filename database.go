package store

import (
	"sort"
	"sync"
)

type database struct {
	keyValues  map[string]KeyData
	sortedKeys []string
	m          sync.Mutex
}

func NewDatabase() *database {
	return &database{
		keyValues:  make(map[string]KeyData),
		sortedKeys: make([]string, 0),
	}
}

func (db *database) Add(key string, keyValue KeyData) int {
	db.m.Lock()
	defer db.m.Unlock()

	keyValue.Revision++
	db.keyValues[key] = keyValue
	db.sortedKeys = db.insertToSortedKeys(db.sortedKeys, key)

	return keyValue.Revision
}

func (db *database) Get(key string) (KeyData, bool) {
	db.m.Lock()
	defer db.m.Unlock()
	if val, ok := db.keyValues[key]; ok {
		return val, ok
	}

	return KeyData{}, false
}

func (db *database) Del(key string) {
	db.m.Lock()
	defer db.m.Unlock()

	delete(db.keyValues, key)
	idx := sort.Search(len(db.sortedKeys), func(i int) bool { return db.sortedKeys[i] >= key })
	db.sortedKeys = append(db.sortedKeys[:idx], db.sortedKeys[idx+1:]...)
}

func (db *database) insertToSortedKeys(keys []string, newKey string) []string {
	index := sort.Search(len(keys), func(i int) bool { return keys[i] >= newKey })
	if index < len(keys) && keys[index] != newKey {
		keys = append(keys, "")
		copy(keys[index+1:], keys[index:])
		keys[index] = newKey
	} else if index == len(keys) {
		keys = append(keys, newKey)
	}

	return keys
}

func (db *database) IsMaxKeyReached(maxKeys int) bool {
	if len(db.keyValues) < maxKeys {
		return false
	}
	return true
}
