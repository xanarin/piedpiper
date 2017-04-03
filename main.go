package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var mainDB *bolt.DB

type ErrorResponse struct {
	Code        int    `json: "code"`
	Message     string `json: "message"`
	Description string `json: "description"`
}

type Object struct {
	ID       int    `json: "id"`
	Name     string `json: "name"`
	FilePath string `json: "filepath"`
	FileSize int64  `json: "filesize"`
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func putObject(entry *Object, db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		// Retrieve the objects bucket.
		// This should be created when the DB is first opened.
		b := tx.Bucket([]byte("objects"))

		// Generate ID for the user.
		// This returns an error only if the Tx is closed or not writeable.
		// That can't happen in an Update() call so I ignore the error check.
		id, _ := b.NextSequence()
		entry.ID = int(id)

		// Marshal user data into bytes.
		buf, err := json.Marshal(entry)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put(itob(entry.ID), buf)
	})
}

func getObject(entryID int, returnObject *Object, db *bolt.DB) error {
	return db.View(func(tx *bolt.Tx) error {
		// Get reference to objects bucket
		bucket := tx.Bucket([]byte("objects"))

		// Get value from k,v pair
		value := bucket.Get(itob(entryID))

		// Demarshall data back from v in database, place into returnObject
		err := json.Unmarshal(value, returnObject)
		if err != nil {
			return err
		}
		return nil
	})
}

func getObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func putObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func deleteObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func arrayContains(element string, array []string) bool {
	for _, v := range array {
		if v == element {
			return true
		}
	}
	return false
}

func main() {
	log.Println("Initializing server...")

	// These are set up in code for now, but will eventually be CLI params
	addr := ":8080"
	dbfile := "test.db"

	// Set up HTTP Handling
	mainRouter := mux.NewRouter()
	// Object Actions
	mainRouter.HandleFunc("/object", getObjectHandler).Methods("GET")
	mainRouter.HandleFunc("/object", putObjectHandler).Methods("PUT")
	mainRouter.HandleFunc("/object", putObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object", deleteObjectHandler).Methods("DELETE")

	// Open Database Connection
	var err error
	// Open database, with a 1 second timeout in case something goes wrong
	mainDB, err = bolt.Open(dbfile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	defer mainDB.Close()

	// Instantiate buckets if they don't exist
	mainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("objects"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})

	// Kick off Server
	log.Printf("Server Initialized. Listening on %v", addr)
	err = http.ListenAndServe(addr, mainRouter)
	log.Panicf("Main Router has crashed: %v", err)
}
