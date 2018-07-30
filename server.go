package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	goji "goji.io"
	"goji.io/pat"
)

func (s *service) StartServer(Host, Port string) {
	mux := goji.NewMux()
	mux.HandleFunc(pat.Put("/entries/:key"), s.handleSet)
	mux.HandleFunc(pat.Get("/entries"), s.handleList)
	mux.HandleFunc(pat.Get("/entries/:key"), s.handleGet)
	mux.HandleFunc(pat.Delete("/entries/:key"), s.handleDel)

	http.ListenAndServe(Host+":"+Port, mux)
}

func (s *service) handleSet(w http.ResponseWriter, r *http.Request) {
	cas := r.URL.Query().Get("cas")
	key := pat.Param(r, "key")

	var keyValue keyData
	errMsg := s.GetBodyContent(r, &keyValue)
	if errMsg != nil {
		w.WriteHeader(http.StatusInternalServerError)
		byteErrMsg, _ := json.Marshal(errMsg)
		w.Write(byteErrMsg)
		return
	}
	defer r.Body.Close()

	errMsg = s.AddValue(key, cas, keyValue)
	if errMsg != nil {
		w.WriteHeader(http.StatusInternalServerError)
		byteErrMsg, _ := json.Marshal(errMsg)
		w.Write(byteErrMsg)
		return
	}

	w.WriteHeader(http.StatusOK)
	jsonByte, _ := json.Marshal(keyValue)
	w.Write(jsonByte)
}

func (s *service) handleList(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}
	// ToDo
	s.ListPage(page)
	fmt.Println("list: ", page)
}

func (s *service) handleGet(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	fmt.Println("get: ", key)
	var val []byte
	var ok bool
	if val, ok = s.GetValueByKey(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(val)
}

func (s *service) handleDel(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	fmt.Println("delete: ", key)
	var val []byte
	var ok bool
	if val, ok = s.DeleteValue(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(val)
}

func (s *service) GetBodyContent(r *http.Request, keyValue *keyData) *errorMsg {
	var container interface{}
	rawJSON, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return &errorMsg{
			Name:    "Read Body",
			Message: "Could not read body",
		}
	}
	err = json.Unmarshal(rawJSON, &container)
	if err != nil {
		return &errorMsg{
			Name:    "Unmarshal",
			Message: "Could not unmarshal body",
		}
	}
	mapContainer := container.(map[string]interface{})
	for key, value := range mapContainer {
		switch key {
		case "key":
			keyValue.Key = value.(string)
		case "value":
			keyValue.Value = value.(string)
		case "revision":
			keyValue.Revision = int(value.(float64))
		case "timestamp":
			keyValue.Timestamp = time.Unix(int64(value.(float64)), 0)
		default:
			return &errorMsg{
				Name:    "Field error",
				Message: "Filed doesn't exists",
			}
		}
	}

	if keyValue.Revision == 0 {
		keyValue.Revision = 1
	}
	if keyValue.Timestamp.IsZero() {
		keyValue.Timestamp = time.Now()
	}

	return nil
}
