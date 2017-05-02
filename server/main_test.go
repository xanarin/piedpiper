package main

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/boltdb/bolt"
)

func setup() {
	err := initDB("test.db")
	if err != nil {
		log.Panicf("Database initialization failed with error %v", err)
	}
	DataPath = "test_data/"
	os.Mkdir(DataPath, 0777)
}

func shutdown() {
	MainDB.Close()
	os.Remove("test.db")
	os.RemoveAll("./test_data/")
}

func TestCreateUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserCreationJSON{Username: "testguy", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestCreateConflictingUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserCreationJSON{Username: "hacker1", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Make second, conflicting request
	handlerb := http.HandlerFunc(createUserHandler)
	reqb, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}
	rrb := httptest.NewRecorder()
	handlerb.ServeHTTP(rrb, reqb)

	if status := rrb.Code; status != http.StatusConflict {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusConflict)
	}
}

func TestAuthUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserCreationJSON{Username: "authguy", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Auth the user
	reqDateString := time.Now().UTC().Format("20060102150405")
	authUserJSON := AuthUserRequestJSON{Username: "authguy", Password: "foobar", ReqDate: reqDateString}
	buffer, err = json.Marshal(authUserJSON)
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	authHandler := http.HandlerFunc(authUserHandler)
	authHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("auth handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestPutGetObjectValid(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "#TheRealUploader", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Auth the user
	reqDateString := time.Now().UTC().Format("20060102150405")
	authUserJSON := AuthUserRequestJSON{Username: "#TheRealUploader", Password: "foobar", ReqDate: reqDateString}
	buffer, err = json.Marshal(authUserJSON)
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	authHandler := http.HandlerFunc(authUserHandler)
	authHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("auth handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get response JSON
	response := AuthUserResponseJSON{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate hash
	hashInput := []byte("#TheRealUploader")
	hashInput = append(hashInput, response.Nonce...)
	hashInput = append(hashInput, []byte(response.ExpirationDate)...)
	hasher := sha512.New()
	hasher.Write(hashInput)
	sha := hex.EncodeToString(hasher.Sum(nil))

	// Create a new object in the database
	createObjectJSON := CreateObjectRequestJSON{Token: sha, FileName: "rando239487246char.txt"}
	buffer, err = json.Marshal(createObjectJSON)
	req, err = http.NewRequest("POST", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	createObjectRunner := http.HandlerFunc(createObjectHandler)
	createObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	uploadID, err := strconv.Atoi(rr.Body.String())
	if err != nil {
		t.Errorf("Failed to convert UploadID '%v' to integer", uploadID)
	}

	// Upload file to database
	data := []byte("I am a test file! (not really, but don't tell anyone)!")
	req, err = http.NewRequest("POST", "/object/"+strconv.Itoa(uploadID)+"/", bytes.NewBuffer(data))
	if err != nil {
		t.Fatal(err)
	}
	rr = httptest.NewRecorder()
	uploadObjectRunner := http.HandlerFunc(uploadObjectHandler)
	uploadObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("object upload handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get object back from database
	getObjectJSON := GetObjectRequestJSON{Token: sha, FileName: "rando239487246char.txt"}
	buffer, err = json.Marshal(getObjectJSON)
	req, err = http.NewRequest("GET", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(getObjectHandler)
	getObjectRunner.ServeHTTP(rr, req)

	status := rr.Code
	if status != http.StatusOK {
		t.Errorf("object get handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	if status == http.StatusNotFound {
		MainDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("objects"))
			log.Printf("Dumping object bucket")
			return b.ForEach(func(k, v []byte) error {
				object := Object{}
				json.Unmarshal(v, &object)
				log.Printf("%v: %v", k, object)
				return nil
			})
		})
		MainDB.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("users"))
			log.Printf("Dumping users bucket")
			return b.ForEach(func(k, v []byte) error {
				object := User{}
				json.Unmarshal(v, &object)
				log.Printf("%v: %v", k, object.ObjectIDs)
				return nil
			})
		})
	}

	bs, err := ioutil.ReadAll(rr.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(bs) != string(data) {
		t.Errorf("Returned data does not match uploaded data.\nExpected: %v\nActual: %v", string(data), string(bs))
	}
}

func TestCreateObjectValid(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserCreationJSON{Username: "happyUploader", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Auth the user
	reqDateString := time.Now().UTC().Format("20060102150405")
	authUserJSON := AuthUserRequestJSON{Username: "happyUploader", Password: "foobar", ReqDate: reqDateString}
	buffer, err = json.Marshal(authUserJSON)
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	authHandler := http.HandlerFunc(authUserHandler)
	authHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("auth handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get response JSON
	response := AuthUserResponseJSON{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate hash
	hashInput := []byte("happyUploader")
	hashInput = append(hashInput, response.Nonce...)
	hashInput = append(hashInput, []byte(response.ExpirationDate)...)
	hasher := sha512.New()
	hasher.Write(hashInput)
	tokenString := hex.EncodeToString(hasher.Sum(nil))

	// Create a request to pass to our handler.
	createObjectJSON := CreateObjectRequestJSON{Token: tokenString, FileName: "foo.txt"}
	buffer, err = json.Marshal(createObjectJSON)
	req, err = http.NewRequest("POST", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr = httptest.NewRecorder()
	createObjectHandler := http.HandlerFunc(createObjectHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	createObjectHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		bs, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			t.Fatal(err)
		}
		t.Errorf("user creator handler returned wrong status code: got %v want %v.\n\tBody was: %v",
			status, http.StatusOK, string(bs))
	}

	// Check the response body contains uploadSessionID
	if rr.Body.String() == "" {
		t.Errorf("handler returned empty body, wanted uploadSessionID")
	}
}

func TestCreateGetObjectWithoutUpload(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "SetGetGuy", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Auth the user
	reqDateString := time.Now().UTC().Format("20060102150405")
	authUserJSON := AuthUserRequestJSON{Username: "SetGetGuy", Password: "foobar", ReqDate: reqDateString}
	buffer, err = json.Marshal(authUserJSON)
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	authHandler := http.HandlerFunc(authUserHandler)
	authHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("auth handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get response JSON
	response := AuthUserResponseJSON{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate hash
	hashInput := []byte("SetGetGuy")
	hashInput = append(hashInput, response.Nonce...)
	hashInput = append(hashInput, []byte(response.ExpirationDate)...)
	hasher := sha512.New()
	hasher.Write(hashInput)
	tokenString := hex.EncodeToString(hasher.Sum(nil))

	// Create a new object in the database
	createObjectJSON := CreateObjectRequestJSON{Token: tokenString, FileName: "rando239487246char.txt"}
	buffer, err = json.Marshal(createObjectJSON)
	req, err = http.NewRequest("POST", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	createObjectRunner := http.HandlerFunc(createObjectHandler)
	createObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get object back from database
	getObjectJSON := GetObjectRequestJSON{Token: tokenString, FileName: "rando239487246char.txt"}
	buffer, err = json.Marshal(getObjectJSON)
	req, err = http.NewRequest("GET", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(getObjectHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusPreconditionFailed {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
}

func TestCreateGetObjectBadFileName(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "BadOwner2", Password: "foobar"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Auth the user
	reqDateString := time.Now().UTC().Format("20060102150405")
	authUserJSON := AuthUserRequestJSON{Username: "BadOwner2", Password: "foobar", ReqDate: reqDateString}
	buffer, err = json.Marshal(authUserJSON)
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	authHandler := http.HandlerFunc(authUserHandler)
	authHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("auth handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Get response JSON
	response := AuthUserResponseJSON{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate hash
	hashInput := []byte("BadOwner2")
	hashInput = append(hashInput, response.Nonce...)
	hashInput = append(hashInput, []byte(response.ExpirationDate)...)
	hasher := sha512.New()
	hasher.Write(hashInput)
	tokenString := hex.EncodeToString(hasher.Sum(nil))

	// Get object back from database (but we're requesting an object that doesn't exist)
	getObjectJSON := GetObjectRequestJSON{Token: tokenString, FileName: "11.txt"}
	buffer, err = json.Marshal(getObjectJSON)
	req, err = http.NewRequest("GET", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(getObjectHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}

	// Check the response body is what we expect.
	expected := `Failed to find object with filename 11.txt belonging to user BadOwner2`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestAuthUserAgain(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "Authenticator", Password: "password"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Authenticate User
	authUserJSON := AuthUserRequestJSON{
		Username: "Authenticator",
		Password: "password",
		ReqDate:  time.Now().UTC().Format("20060102150405"),
	}
	buffer, err = json.Marshal(authUserJSON)
	if err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(authUserHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response JSON
	response := AuthUserResponseJSON{}
	err = json.NewDecoder(rr.Body).Decode(&response)
	if err != nil {
		t.Fatal(err)
	}

	if len(response.Nonce) != 24 {
		t.Errorf("Nonce was not 24 characters long. Got '%v'", response.Nonce)
	}

	expTime, err := time.Parse("20060102150405", response.ExpirationDate)
	if err != nil {
		t.Errorf("Invalid timestamp returned from server. Got '%v'. Error from parser: %v", response.ExpirationDate, err)
	}

	expDuration := expTime.Sub(time.Now().UTC())
	if expDuration.Hours() > 144.0 || expDuration.Hours() < 143.0 {
		t.Errorf("Token expiration duration is out of range for spec. Got %v", expDuration)
	}
}

func TestAuthUserBadPassword(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "badauthguy", Password: "password1"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Authenticate User
	authUserJSON := AuthUserRequestJSON{
		Username: "badauthguy",
		Password: "password2", // Note: Not the password that was given during registration
		ReqDate:  time.Now().UTC().Format("20060102150405"),
	}
	buffer, err = json.Marshal(authUserJSON)
	if err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(authUserHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusForbidden {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusForbidden)
	}
}

func TestAuthUserReplayAttack(t *testing.T) {
	// Create user for this test
	createUserJSON := UserCreationJSON{Username: "naiveuser", Password: "verysecurepassword"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(createUserHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// -10 minute duration
	negativeTenMinutes, err := time.ParseDuration("-10m")
	if err != nil {
		t.Fatal(err)
	}

	badTimeStamp := time.Now().UTC().Add(negativeTenMinutes).Format("20060102150405")
	// Authenticate User
	authUserJSON := AuthUserRequestJSON{
		Username: "naiveuser",
		Password: "verysecurepassword", // Note: Not the password that was given during registration
		ReqDate:  badTimeStamp,
	}
	buffer, err = json.Marshal(authUserJSON)
	if err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest("GET", "/auth", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(authUserHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusExpectationFailed {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusExpectationFailed)
	}

	// Check the response body contains uploadSessionID
	if rr.Body.String() == "Invalid time stamp." {
		t.Errorf("handler returned empty body, wanted uploadSessionID")
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
