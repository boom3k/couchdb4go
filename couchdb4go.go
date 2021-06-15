package couchdb4go

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type CouchDB struct {
	ServerAddress  string
	Username       string
	Password       string
	IsSecureServer bool
	Request        *http.Request
}

func Initialize(username, password, serverAddress string, secureSever bool) *CouchDB {
	database := &CouchDB{}
	port := 5984
	httpType := "http://"
	if secureSever {
		httpType = "https://"
	}
	database.ServerAddress = httpType + serverAddress + ":" + fmt.Sprint(port) + "/"
	database.Username = username
	database.Password = password
	database.IsSecureServer = secureSever
	return database
}

func (couchDB *CouchDB) CreateDatabase(databaseName string) (*CouchDB, error) {
	url := couchDB.ServerAddress + strings.ToLower(databaseName)
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return nil, err
	}
	couchDB.Request = request
	return couchDB, nil
}

func (couchDB *CouchDB) CreateDocument(database string, documentData []byte) (*CouchDB, error) {
	url := couchDB.ServerAddress + database
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(documentData))
	if err != nil {
		return nil, err
	}
	couchDB.Request = request
	return couchDB, nil
}

// Do Request Handler
func (couchDB *CouchDB) Do() (*http.Response, error) {
	log.Println(couchDB.Request.Method + " -> " + fmt.Sprint(couchDB.Request.URL) + " ")
	couchDB.Request.SetBasicAuth(couchDB.Username, couchDB.Password)
	couchDB.Request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	httpClient := &http.Client{}
	response, err := httpClient.Do(couchDB.Request)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err.Error())
		return response, err
	}
	couchDB.Request = nil
	response.Proto = string(responseBody)
	return response, nil
}
