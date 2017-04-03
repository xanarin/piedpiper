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

func TestCreateDeleteUser(t *testing.T) {
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

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
