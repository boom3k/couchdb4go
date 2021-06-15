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

// Initialize Creates a new running couchdb object
func Initialize(username, password, serverAddress string, secureSever bool) (*CouchDB, error) {
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
	return database.VerifyConnection()
}

func (c *CouchDB) VerifyConnection() (*CouchDB, error) {
	url := c.ServerAddress + "_all_dbs"
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err.Error())
		return c, err
	}
	var httpClient *http.Client
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err.Error())
		return c, err
	}

	if response.StatusCode == 200 {
		log.Printf("Connection to <%s> [SUCCESS]\n", c.ServerAddress)
	} else {
		log.Printf("Connection to <%s> [FAILED] - Error code: (%d)\n", c.ServerAddress, response.StatusCode)
	}
	return c, nil
}

func (c *CouchDB) CreateDatabase(databaseName string) (*CouchDB, error) {
	url := c.ServerAddress + strings.ToLower(databaseName)
	request, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return nil, err
	}
	c.Request = request
	return c, nil
}

func (c *CouchDB) CreateDocument(database string, documentData []byte) (*CouchDB, error) {
	url := c.ServerAddress + database
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(documentData))
	if err != nil {
		return nil, err
	}
	c.Request = request
	return c, nil
}

// Do Request Handler
func (c *CouchDB) Do() (*http.Response, error) {
	log.Println(c.Request.Method + " -> " + fmt.Sprint(c.Request.URL) + " ")
	c.Request.SetBasicAuth(c.Username, c.Password)
	c.Request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	httpClient := &http.Client{}
	response, err := httpClient.Do(c.Request)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Println(err.Error())
		return response, err
	}
	c.Request = nil
	response.Proto = string(responseBody)
	return response, nil
}

// ExecuteURL For testing db calls
func ExecuteURL(method, username, password, url string, body []byte) (*http.Response, error) {
	var httpClient *http.Client
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
