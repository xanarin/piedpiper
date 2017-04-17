package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
)

func setup() {
	err := initDB("test.db")
	if err != nil {
		log.Panicf("Database initialization failed with error %v", err)
	}
}

func shutdown() {
	MainDB.Close()
	os.Remove("test.db")
}

func TestCreateUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserRequestJSON{Username: "testguy"}
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
	createUserJSON := UserRequestJSON{Username: "hacker1"}
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

func TestDeleteUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserRequestJSON{Username: "forgettable"}
	buffer, err := json.Marshal(createUserJSON)
	req, err := http.NewRequest("POST", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	createHandler := http.HandlerFunc(createUserHandler)
	createHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Make delete request
	req, err = http.NewRequest("DELETE", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	deleteHandler := http.HandlerFunc(deleteUserHandler)
	deleteHandler.ServeHTTP(rr, req)
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v.",
			status, http.StatusOK)
	}
}

func TestDeleteFictionalUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := UserRequestJSON{Username: "fictional"}
	buffer, err := json.Marshal(createUserJSON)

	// Make delete request
	req, err := http.NewRequest("DELETE", "/user", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	deleteHandler := http.HandlerFunc(deleteUserHandler)
	deleteHandler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v.",
			status, http.StatusNotFound)
	}
}

func TestPutGetObjectValid(t *testing.T) {
	// Create user for this test
	createUserJSON := UserRequestJSON{Username: "#TheRealUploader"}
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

	// Create a new object in the database
	createObjectJSON := CreateObjectRequestJSON{Username: "#TheRealUploader", FileName: "rando239487246char.txt", FileSize: 20}
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
	data := []byte("I am a test file! (not really, but don't tell anyone!")
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
	getObjectJSON := GetObjectRequestJSON{Username: "#TheRealUploader", FileName: "rando239487246char.txt"}
	buffer, err = json.Marshal(getObjectJSON)
	req, err = http.NewRequest("GET", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	rr = httptest.NewRecorder()
	getObjectRunner := http.HandlerFunc(getObjectHandler)
	getObjectRunner.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
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
	createUserJSON := UserRequestJSON{Username: "happyUploader"}
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

	// Create a request to pass to our handler.
	createObjectJSON := CreateObjectRequestJSON{Username: "happyUploader", FileName: "foo.txt", FileSize: 20}
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
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	// Check the response body contains uploadSessionID
	if rr.Body.String() == "" {
		t.Errorf("handler returned empty body, wanted uploadSessionID")
	}
}

func TestCreateObjectInvalidOwner(t *testing.T) {
	// Create a request to pass to our handler.
	createObjectJSON := CreateObjectRequestJSON{Username: "InvalidUploader", FileName: "bar.txt", FileSize: 40}
	buffer, err := json.Marshal(createObjectJSON)
	req, err := http.NewRequest("POST", "/object", bytes.NewBuffer(buffer))
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	rr := httptest.NewRecorder()
	createObjectHandler := http.HandlerFunc(createObjectHandler)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	createObjectHandler.ServeHTTP(rr, req)

	// Check the status code is what we expect.
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("user creator handler returned wrong status code: got %v want %v",
			status, http.StatusNotFound)
	}
}

func TestCreateGetObjectWithoutUpload(t *testing.T) {
	// Create user for this test
	createUserJSON := UserRequestJSON{Username: "SetGetGuy"}
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

	// Create a new object in the database
	createObjectJSON := CreateObjectRequestJSON{Username: "SetGetGuy", FileName: "rando239487246char.txt", FileSize: 20}
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
	getObjectJSON := GetObjectRequestJSON{Username: "SetGetGuy", FileName: "rando239487246char.txt"}
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

func TestCreateGetObjectBadOwner(t *testing.T) {
	// Create user for this test
	createUserJSON := UserRequestJSON{Username: "BadOwner1"}
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

	// Create a new object in the database
	createObjectJSON := CreateObjectRequestJSON{Username: "BadOwner1", FileName: "rando239487246char.txt", FileSize: 20}
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

	// Get object back from database (but we're requesting from a user that doesn't exist)
	getObjectJSON := GetObjectRequestJSON{Username: "BadOwnerInvalid", FileName: "rando239487246char.txt"}
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
	expected := `User BadOwnerInvalid is not a registered user`
	if rr.Body.String() != expected {
		t.Errorf("handler returned unexpected body: got %v want %v",
			rr.Body.String(), expected)
	}
}

func TestCreateGetObjectBadFileName(t *testing.T) {
	// Create user for this test
	createUserJSON := UserRequestJSON{Username: "BadOwner2"}
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

	// Get object back from database (but we're requesting an object that doesn't exist)
	getObjectJSON := GetObjectRequestJSON{Username: "BadOwner2", FileName: "11.txt"}
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

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}