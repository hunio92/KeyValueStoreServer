package store

import (
	"sort"
	"sync"
)

type database struct {
	keyValues  map[string]keyData
	sortedKeys []string
	m          sync.Mutex
}

func NewDatabase() *database {
	return &database{
		keyValues:  make(map[string]keyData),
		sortedKeys: make([]string, 0),
	}
}

func (db *database) Add(key string, keyValue keyData) {
	// db.m.Lock()
	// defer db.m.Unlock()

	db.keyValues[key] = keyValue
	db.sortedKeys = db.insertToSortedKeys(db.sortedKeys, key)
}

func (db *database) Get(key string) (keyData, bool) {
	db.m.Lock()
	defer db.m.Unlock()
	if val, ok := db.keyValues[key]; ok {
		return val, ok
	}

	return keyData{}, false
}

func (db *database) Del(key string) {
	db.m.Lock()
	defer db.m.Unlock()

	delete(db.keyValues, key)
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
