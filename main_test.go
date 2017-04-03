package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/boltdb/bolt"
)

func setup() {
	// Open Database Connection
	var err error
	// Open database, with a 1 second timeout in case something goes wrong
	mainDB, err = bolt.Open("test.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate buckets if they don't exist
	mainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("objects"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
	mainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
}

func shutdown() {
	mainDB.Close()
	os.Remove("test.db")
}

func TestGetPutGetDatabase(t *testing.T) {
	testObject := Object{Name: "testObject", FilePath: "/tmp/testObject1", FileSize: 64000}

	// Put object in database
	err := putObject(&testObject)
	if err != nil {
		t.Errorf("Failed to store testObject in collection. error: %v", err)
	}
	id := testObject.ID

	// Get object back from database
	returnObject := Object{}
	err = getObject(id, &returnObject)
	if err != nil {
		t.Errorf("Retrieve object operation failed. Error: %v", err)
	}

	// Test validity of object
	if returnObject.Name != "testObject" || returnObject.FilePath != "/tmp/testObject1" || returnObject.FileSize != 64000 {
		t.Errorf("Object returned from database does not match inserted object.\n\tExpected: %v \n\tActual: %v", testObject, returnObject)
	}
}

func TestCreateUser(t *testing.T) {
	// Create a request to pass to our handler.
	createUserJSON := CreateUserRequestJSON{Username: "testguy"}
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
	createUserJSON := CreateUserRequestJSON{Username: "hacker1"}
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

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
