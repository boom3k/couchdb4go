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
	ServerAddress    string
	Username         string
	Password         string
	IsSecureServer   bool
	ActiveConnection bool
	Request          *http.Request
}

// Initialize Creates a new running couchdb object
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
	database.ActiveConnection = database.VerifyConnection()
	return database
}

func (c *CouchDB) SetRequest(method, ask string, body []byte) *CouchDB {
	request, err := http.NewRequest(strings.ToUpper(method), c.ServerAddress+ask, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	c.Request = request
	return c
}

func (c *CouchDB) VerifyConnection() bool {
	c.SetRequest("GET", "_all_dbs", nil)
	response, err := c.Do()
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	log.Printf("Connection to <%s> [%s]\n", c.ServerAddress, response.Status)
	if response.StatusCode == 200 {
		return true
	}
	return false
}

// Do Request Handler
func (c *CouchDB) Do() (*http.Response, error) {
	c.Request.SetBasicAuth(c.Username, c.Password)
	c.Request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	reader := new(bytes.Buffer)
	reader.ReadFrom(c.Request.Body)
	log.Println(c.Request.Method + ": " + reader.String() + " -> " + fmt.Sprint(c.Request.URL))
	httpClient := &http.Client{}
	response, err := httpClient.Do(c.Request)
	c.Request = nil
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err.Error())
		return response, err
	}

	response.Proto = string(responseBody)
	return response, nil
}

// ExecuteURL For testing db calls
func ExecuteURL(method, username, password, url string, body []byte) (*http.Response, error) {
	var httpClient http.Client
	request, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	request.SetBasicAuth(username, password)
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err.Error())
		return response, err
	}
	response.Proto = string(responseBody)
	return response, nil
}

/*Insert Methods*/
func (c *CouchDB) CreateDatabase(databaseName string) (*http.Response, error) {
	c.SetRequest("PUT", databaseName, nil)
	return c.Do()
}

func (c *CouchDB) CreateDocument(database string, jsonData []byte) (*http.Response, error) {

	c.SetRequest("POST", database, jsonData)
	return c.Do()
}
