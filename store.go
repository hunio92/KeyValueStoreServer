package store

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type StoreKvs struct {
	Kvs      []KeyData `json:kvs`
	Revision int       `json:revision`
}

type KeyData struct {
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
	db            *database
	storeRevision int
	maxKeyValues  int
}

func NewService(db *database, maxKeyValues int) *Service {
	return &Service{
		db:            db,
		storeRevision: 1,
		maxKeyValues:  maxKeyValues,
	}
}

func (s *Service) AddUpdateWithoutCas(key string, casStr string, keyValue KeyData) *errorMsg {
	if _, ok := s.GetValueByKey(key); !ok {
		errMsg := s.addNewKeyIfNotFull(key, keyValue)
		if errMsg != nil {
			return errMsg
		}
	} else {
		if casStr != "0" {
			internalRevision := s.db.Add(key, keyValue)
			s.storeRevision = internalRevision
		} else {
			return &errorMsg{
				Name:    "Option cas equal 0",
				Message: "The cas equal 0 and the key already exists in the database",
			} // Didn't saw to be mentioned the return status code in case of 0 or it's 412 too ?!
		}
	}

	fmt.Println("without cas key values: ", s.db.keyValues)
	fmt.Println("without cas sorted values: ", s.db.sortedKeys)

	return nil
}

func (s *Service) AddUpdateWithCas(key string, casStr string, keyValue KeyData) *errorMsg {
	if !s.db.IsMaxKeyReached(s.maxKeyValues) {
		cas, err := strconv.Atoi(casStr)
		if err != nil {
			return &errorMsg{
				Name:    "Convert",
				Message: "Could not convert page number",
			}
		}

		if cas == s.storeRevision {
			if _, ok := s.GetValueByKey(key); !ok {
				errMsg := s.addNewKeyIfNotFull(key, keyValue)
				if errMsg != nil {
					return errMsg
				}
			} else {
				internalRevision := s.db.Add(key, keyValue)
				s.storeRevision = internalRevision
				fmt.Println("with cas", cas, casStr)
			}
		} else {
			return &errorMsg{
				Name:    "Bad Revision",
				Message: fmt.Sprintf("The internal revision do not match with current store revision: %d", s.storeRevision),
			}
		}
	} else {
		return &errorMsg{
			Name:    "Max keys reached",
			Message: "Could not add key: max key number reached",
		}
	}

	fmt.Println("with cas key values: ", s.db.keyValues)
	fmt.Println("with cas sorted values: ", s.db.sortedKeys)

	return nil
}

func (s *Service) GetValueByKey(key string) ([]byte, bool) {
	var jsonBytes []byte
	if val, ok := s.db.Get(key); ok {
		jsonBytes, err := json.Marshal(
			StoreKvs{
				Kvs:      []KeyData{val},
				Revision: s.storeRevision,
			})
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

func (s *Service) ListPage(pageStr string) (StoreKvs, bool) {
	var respKvs StoreKvs
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return respKvs, false
	}

	lenOfKeyValue := s.db.SizeOfKeyValues()
	start, limit := s.getPageContentIntervallum(page, lenOfKeyValue)

	var keysToAdd []string
	if start < limit {
		keysToAdd = s.db.sortedKeys[start:limit]
	} else {
		fmt.Println("WTF !?!")
		return respKvs, false
	}
	fmt.Println("keysToAdd: ", keysToAdd)

	for _, key := range keysToAdd {
		respKvs.Kvs = append(respKvs.Kvs, s.db.keyValues[key])
	}

	fmt.Println("respKvs: ", respKvs)

	respKvs.Revision = s.storeRevision
	return respKvs, true
}

func (s *Service) getPageContentIntervallum(page, lenOfKeyValue int) (int, int) {
	var start, limit int
	if page <= 1 || lenOfKeyValue < 11 {
		start = 0
		if lenOfKeyValue >= 10 {
			limit = page * 10
		} else {
			limit = lenOfKeyValue - 1
		}

		return start, limit
	}

	if page > 1 && page*10 < lenOfKeyValue {
		limit = lenOfKeyValue - (lenOfKeyValue % (page * 10))
		start = limit - 10

		return start, limit
	}
	if page*10 > lenOfKeyValue {
		limit = lenOfKeyValue
		start = page*10 - 10

		return start, limit
	}

	return 0, 0
}

func (s *Service) addNewKeyIfNotFull(key string, keyValue KeyData) *errorMsg {
	if !s.db.IsMaxKeyReached(s.maxKeyValues) {
		internalRevision := s.db.Add(key, keyValue)
		s.storeRevision = internalRevision
	} else {
		return &errorMsg{
			Name:    "Max keys reached",
			Message: "Could not add key: max key number reached",
		}
	}

	return nil
}

func (s *Service) GetRevision() int {
	return s.storeRevision
}
