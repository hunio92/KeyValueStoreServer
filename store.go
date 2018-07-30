package store

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type kvs struct {
	data     []keyData `json:kvs`
	revision int       `json:revision`
}

type keyData struct {
	Key       string    `json:key`
	Value     string    `json:value`
	Revision  int       `json:revision`
	Timestamp time.Time `json:timestamp`
}

type errorMsg struct {
	Name    string
	Message string
}

type Service struct {
	db           *database
	maxKeyValues int
}

func NewService(db *database, maxKeyValues int) *Service {
	return &Service{
		db:           db,
		maxKeyValues: maxKeyValues,
	}
}

func (s *Service) AddValue(key, cas string, keyValue keyData) *errorMsg {
	s.db.m.Lock()
	defer s.db.m.Unlock()

	if len(s.db.keyValues) < s.maxKeyValues {
		if _, ok := s.db.keyValues[key]; !ok {
			/*
				 keyValue.Revision++: INTERNAL revision incerement what does it mean ?
				also increment the global revision or replace with keyValue.Revision++ ?
				Set is also Update ?
			*/
			s.db.Add(key, keyValue)
		}
		/*
			else {
			if has already the key then what ?
			}
		*/
	} else {
		return &errorMsg{
			Name:    "Max keys reached",
			Message: "Could not add key: max key number reached",
		}
	}

	fmt.Println("Add: ", keyValue)

	return nil
}

func (s *Service) GetValueByKey(key string) ([]byte, bool) {
	var jsonBytes []byte
	if val, ok := s.db.Get(key); ok {
		jsonBytes, err := json.Marshal(val)
		if err != nil {
			return jsonBytes, false
		}
		return jsonBytes, true
	}
	return jsonBytes, false
}

func (s *Service) DeleteValue(key string) ([]byte, bool) {
	var jsonBytes []byte
	if jsonBytes, ok := s.GetValueByKey(key); ok {
		s.db.Del(key)
		return jsonBytes, true
	}
	return jsonBytes, false
}

func (s *Service) ListPage(pageStr string) (kvs, *errorMsg) {
	var respKvs kvs
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return respKvs, &errorMsg{
			Name:    "Convert",
			Message: "Could not convert page number",
		}
	}

	var keysToAdd []string
	if page < 2 && len(s.db.sortedKeys) < 11 {
		keysToAdd = s.db.sortedKeys[:len(s.db.sortedKeys)]
	} else if page > 1 {
		limit := 10 * page
		start := limit - 9
		if len(s.db.sortedKeys) >= limit {
			keysToAdd = s.db.sortedKeys[start:limit]
		} else {
			keysToAdd = s.db.sortedKeys[start:len(s.db.sortedKeys)]
		}
	}

	for _, key := range keysToAdd {
		respKvs.data = append(respKvs.data, s.db.keyValues[key])
	}

	// listKvs.Revision = ???
	return respKvs, nil
}
