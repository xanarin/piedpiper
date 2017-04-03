package main

import (
	"fmt"
	"github.com/boltdb/bolt"
	"log"
	"os"
	"testing"
	"time"
)

var testdb *bolt.DB

func setup() {
	// Open Database Connection
	var err error
	// Open database, with a 1 second timeout in case something goes wrong
	testdb, err = bolt.Open("test.db", 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	// Instantiate buckets if they don't exist
	testdb.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("objects"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
}

func shutdown() {
	testdb.Close()
	os.Remove("test.db")
}

func TestGetPutGetDatabase(t *testing.T) {
	testObject := Object{Name: "testObject", FilePath: "/tmp/testObject1", FileSize: 64000}

	// Put object in database
	err := putObject(&testObject, testdb)
	if err != nil {
		t.Errorf("Failed to store testObject in collection. error: %v", err)
	}
	id := testObject.ID
	log.Printf("id: %v", id)

	// Get object back from database
	returnObject := Object{}
	err = getObject(id, &returnObject, testdb)
	if err != nil {
		t.Errorf("Retrieve object operation failed. Error: %v", err)
	}

	// Test validity of object
	if returnObject.Name != "testObject" || returnObject.FilePath != "/tmp/testObject1" || returnObject.FileSize != 64000 {
		t.Errorf("Object returned from database does not match inserted object.\n\tExpected: %v \n\tActual: %v", testObject, returnObject)
	}
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}
