package couchdb4go

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

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

type Session struct {
	ServerAddress    string
	Username         string
	Password         string
	IsSecureServer   bool
	ActiveConnection bool
	Request          *http.Request
}

func NewSession(username, password, serverAddress string, secureSever bool) *Session {
	session := &Session{}
	port := 5984
	httpType := "http://"
	if secureSever {
		httpType = "https://"
	}
	session.ServerAddress = httpType + serverAddress + ":" + fmt.Sprint(port) + "/"
	session.Username = username
	session.Password = password
	session.IsSecureServer = secureSever
	session.ActiveConnection = false
	response, err := ExecuteURL("GET", session.Username, session.Password, session.ServerAddress, nil)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	log.Printf("Connection to <%s> [%s]\n", session.ServerAddress, response.Status)
	if response.StatusCode < 205 {
		session.ActiveConnection = true
	}
	return session
}

type Database struct {
	Name    string
	Session *Session
}

func (session *Session) Get(databaseName string) (*Database, error) {
	database := &Database{Name: databaseName, Session: session}
	response, err := ExecuteURL("GET", session.Username, session.Password, session.ServerAddress+database.Name, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if response.StatusCode > 205 {
		e := errors.New(response.Status + " - " + response.Request.URL.String())
		return nil, e
	}
	return database, nil
}

func (session *Session) Delete(database *Database) error {
	response, err := ExecuteURL("DELETE", session.Username, session.Password, session.ServerAddress+database.Name, nil)
	if err != nil {
		log.Println(err)
		return err
	}
	if response.StatusCode > 205 {
		e := errors.New(response.Status + " - " + response.Request.URL.String())
		return e
	}
	return nil
}

type Res struct {
	Status  int
	JsonMap map[string]interface{}
	Body    string
}

func GetResponse(response *http.Response) *Res {
	var cr = &Res{}
	cr.Status = response.StatusCode
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(response.Body)
	cr.Body = buffer.String()
	err := json.Unmarshal([]byte(cr.Body), &cr.JsonMap)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	if cr.JsonMap["error"] != nil {
		err = errors.New(cr.JsonMap["reason"].(string))
	}
	return cr
}

func (session *Session) Do() (*Res, error) {
	if session.Request == nil {
		e := errors.New("No session request set @ " + session.ServerAddress)
		return nil, e
	}
	session.Request.SetBasicAuth(session.Username, session.Password)
	session.Request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(session.Request.Body)
	httpClient := http.Client{}
	httpResponse, err := httpClient.Do(session.Request)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	resultMsg := session.Request.Method + ": " + buffer.String() + " {" + fmt.Sprint(session.Request.URL) + "} || Status: " + fmt.Sprint(httpResponse.StatusCode)
	response := GetResponse(httpResponse)
	if response.JsonMap["error"] != nil {
		resultMsg += " -- " + (response.JsonMap["reason"]).(string)
	}
	log.Println(resultMsg)
	session.Request = nil
	return response, nil
}

func (session *Session) SetRequest(method, ask string, body []byte) *Session {
	request, err := http.NewRequest(strings.ToUpper(method), session.ServerAddress+ask, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	session.Request = request
	return session
}

func (session *Session) CreateDatabase(databaseName string, isPartitioned bool) (*Database, error) {
	_, err := session.SetRequest("PUT", databaseName+"?partitioned="+fmt.Sprint(isPartitioned), nil).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	db := &Database{}
	db.Name = databaseName
	db.Session = session
	return db, nil
}

func (database *Database) Insert(jsonData []byte) (map[string]interface{}, error) {
	res, err := database.Session.SetRequest("POST", database.Name, jsonData).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) Read(_id string) (map[string]interface{}, error) {
	res, err := database.Session.SetRequest("GET", database.Name+"/"+_id, nil).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) Copy(_id string) (map[string]interface{}, error) {
	res, err := database.Session.SetRequest("COPY", database.Name+"/"+_id, nil).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) Update(_id string, jsonData []byte) (map[string]interface{}, error) {
	document, err := database.Read(_id)
	_rev := document["_rev"].(string)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	res, err := database.Session.SetRequest("PUT", database.Name+"/"+_id+"?rev="+_rev, jsonData).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) Delete(_id string) (map[string]interface{}, error) {
	_rev, err := database.Read(_id)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	res, err := database.Session.SetRequest("DELETE", database.Name+"/"+_id+"?rev="+_rev["_rev"].(string), nil).Do()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

type UploadRequest struct {
	ID          string       `json:"_id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Date        string       `json:"date"`
	Time        string       `json:"time"`
	Attachments []Attachment `json:"_attachments"`
}

type Attachment struct {
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

func (database *Database) Upload(_id string, file os.File) {

}
