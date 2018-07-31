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

func (s *Service) ListPage(pageStr string) (StoreKvs, *errorMsg) {
	var respKvs StoreKvs
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return respKvs, &errorMsg{
			Name:    "Convert",
			Message: "Could not convert page number",
		}
	}

	fmt.Println("LIST key values: ", s.db.keyValues)
	fmt.Println("LIST sorted values: ", s.db.sortedKeys)

	var keysToAdd []string
	if page <= 1 && len(s.db.sortedKeys) < 11 {
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
		respKvs.Kvs = append(respKvs.Kvs, s.db.keyValues[key])
	}

	respKvs.Revision = s.storeRevision
	return respKvs, nil
}

func (s *Service) GetRevision() int {
	return s.storeRevision
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
