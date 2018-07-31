package store

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	goji "goji.io"
	"goji.io/pat"
)

type Server struct {
	service *Service
	logger  *log.Logger
	mux     *goji.Mux
}

func NewServer(service *Service, options ...func(*Server)) *Server {
	s := &Server{
		service: service,
		mux:     goji.NewMux(),
	}

	for _, f := range options {
		f(s)
	}

	if s.logger == nil {
		s.logger = log.New(os.Stdout, "", 0)
	}

	s.mux.HandleFunc(pat.Put("/entries/:key"), s.handleSet)
	s.mux.HandleFunc(pat.Get("/entries"), s.handleList)
	s.mux.HandleFunc(pat.Get("/entries/:key"), s.handleGet)
	s.mux.HandleFunc(pat.Delete("/entries/:key"), s.handleDel)

	return s
}

func Logger(logger *log.Logger) func(*Server) {
	return func(s *Server) {
		s.logger = logger
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) handleSet(w http.ResponseWriter, r *http.Request) {
	cas := r.URL.Query().Get("cas")
	key := pat.Param(r, "key")

	var keyValue KeyData
	errMsg := s.GetBodyContent(r, &keyValue)
	if errMsg != nil {
		w.WriteHeader(http.StatusInternalServerError)
		byteErrMsg, _ := json.Marshal(errMsg)
		w.Write(byteErrMsg)
		return
	}
	defer r.Body.Close()

	if cas == "" || cas == "0" {
		fmt.Println("cas:", cas)
		errMsg = s.service.AddUpdateWithoutCas(key, cas, keyValue)
		if errMsg != nil {
			w.WriteHeader(http.StatusInternalServerError)
			byteErrMsg, _ := json.Marshal(errMsg)
			w.Write(byteErrMsg)
			return
		}
	} else {
		errMsg = s.service.AddUpdateWithCas(key, cas, keyValue)
		if errMsg != nil {
			w.WriteHeader(http.StatusPreconditionFailed)
			byteErrMsg, _ := json.Marshal(errMsg)
			w.Write(byteErrMsg)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	jsonByte, _ := json.Marshal(
		StoreKvs{
			Kvs:      []KeyData{keyValue},
			Revision: s.service.GetRevision(),
		})
	w.Write(jsonByte)
}

func (s *Server) handleList(w http.ResponseWriter, r *http.Request) {
	page := r.URL.Query().Get("page")
	if page == "" {
		page = "1"
	}

	var respKvs StoreKvs
	respKvs, ok := s.service.ListPage(page)
	if ok == false {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	jsonByte, _ := json.Marshal(respKvs)
	w.Write(jsonByte)
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	fmt.Println("get: ", key)
	var val []byte
	var ok bool
	if val, ok = s.service.GetValueByKey(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(val)
}

func (s *Server) handleDel(w http.ResponseWriter, r *http.Request) {
	key := pat.Param(r, "key")
	fmt.Println("delete: ", key)
	var val []byte
	var ok bool
	if val, ok = s.service.DeleteValue(key); !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(val)
}

func (s *Server) GetBodyContent(r *http.Request, keyValue *KeyData) *errorMsg {
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
			keyValue.Revision = s.service.GetRevision()
		case "timestamp":
			keyValue.Timestamp = time.Unix(int64(value.(float64)), 0)
		default:
			return &errorMsg{
				Name:    "Field error",
				Message: "Field does NOT exists",
			}
		}
	}

	if keyValue.Timestamp.IsZero() {
		keyValue.Timestamp = time.Now()
	}

	return nil
}
