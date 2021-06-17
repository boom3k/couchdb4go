package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func main() {
	type User struct {
		Id   string `json:"_id"`
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	couch := NewSession("root", "boomer", "10.0.0.51", false)
	database, err := couch.GetDatabase("dev")
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	userData, err := json.Marshal(&User{Id: "user:rhenderson", Age: 0})
	database.UpdateDocument(userData)
	/*userData, err := json.Marshal(&User{Id: "user:rhenderson", Name: "Ramel", Age: 35})
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	document, err := database.CreateDocument(userData)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	u := &User{}
	mapstructure.Decode(document, &u)
	fmt.Sprint(u)*/
}

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
	RequestHandler   *http.Request
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
	if response.StatusCode == 200 {
		session.ActiveConnection = true
	}
	return session
}

type Database struct {
	Name    string
	Session *Session
}

func (session *Session) GetDatabase(databaseName string) (*Database, error) {
	database := &Database{Name: databaseName, Session: session}
	response, err := ExecuteURL("GET", session.Username, session.Password, session.ServerAddress+database.Name, nil)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	if response.StatusCode != 200 {
		e := errors.New(response.Status + " - " + response.Request.URL.String())
		return nil, e
	}

	return database, nil
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

func (session *Session) ExecuteRequest() (*Res, error) {
	session.RequestHandler.SetBasicAuth(session.Username, session.Password)
	session.RequestHandler.Header.Set("Content-Type", "application/json; charset=UTF-8")
	buffer := new(bytes.Buffer)
	buffer.ReadFrom(session.RequestHandler.Body)
	httpClient := http.Client{}
	httpResponse, err := httpClient.Do(session.RequestHandler)
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	resultMsg := session.RequestHandler.Method + ": " + buffer.String() + " {" + fmt.Sprint(session.RequestHandler.URL) + "} || Status: " + fmt.Sprint(httpResponse.StatusCode)
	response := GetResponse(httpResponse)
	if response.JsonMap["error"] != nil {
		resultMsg += " -- " + (response.JsonMap["reason"]).(string)
	}
	log.Println(resultMsg)
	session.RequestHandler = nil
	return response, nil
}

func (session *Session) SetRequest(method, ask string, body []byte) *Session {
	request, err := http.NewRequest(strings.ToUpper(method), session.ServerAddress+ask, bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	session.RequestHandler = request
	return session
}

func (session *Session) CreateDatabase(databaseName string, isPartitioned bool) (*Database, error) {
	_, err := session.SetRequest("PUT", databaseName+"?partitioned="+fmt.Sprint(isPartitioned), nil).ExecuteRequest()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	db := &Database{}
	db.Name = databaseName
	db.Session = session
	return db, nil
}

func (database *Database) CreateDocument(jsonData []byte) (map[string]interface{}, error) {
	res, err := database.Session.SetRequest("POST", database.Name, jsonData).ExecuteRequest()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) ReadDocument(_id string) (map[string]interface{}, error) {
	res, err := database.Session.SetRequest("GET", database.Name+"/"+_id, nil).ExecuteRequest()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) UpdateDocument(jsonData []byte) (map[string]interface{}, error) {
	_id := "" //Todo: Get ID from jsonData
	_rev, err := database.ReadDocument(_id)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	res, err := database.Session.SetRequest("PUT", database.Name+"/"+_id+"?rev="+_rev["_rev"].(string), jsonData).ExecuteRequest()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}

func (database *Database) DeleteDocument(_id string) (map[string]interface{}, error) {
	_rev, err := database.ReadDocument(_id)
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}
	res, err := database.Session.SetRequest("DELETE", database.Name+"/"+_id+"?rev="+_rev["_rev"].(string), nil).ExecuteRequest()
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return res.JsonMap, nil
}
