package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var mainDB *bolt.DB

// API JSON objects
type ErrorResponse struct {
	Code        int    `json: "code"`
	Message     string `json: "message"`
	Description string `json: "description"`
}

type GetObjectRequestJSON struct {
	Username string `json: "username"`
	FileName string `json: "filename"`
}

type CreateObjectRequestJSON struct {
	Username string `json: "username"`
	FileName string `json: "filename"`
	FileSize int64  `json: "filesize"`
}

type UserRequestJSON struct {
	Username string `json: "username"`
}

// Internal use structs
type User struct {
	Username  string `json: "username"`
	ObjectIDs []int  `json: "objectids"`
}

type Object struct {
	ID       int    `json: "id"`
	Name     string `json: "name"`
	FileSize int64  `json: "filesize"`
	Owner    string `json: "owner"`
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func getObject(entryID int, returnObject *Object) error {
	return mainDB.View(func(tx *bolt.Tx) error {
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

func createObjectHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := CreateObjectRequestJSON{}
	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		fmt.Fprintf(res, "Error in decoding message")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	// Confirm that owner exists
	var existingUser []byte
	requestedKey := []byte(requestJSON.Username)
	mainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		existingUser = b.Get(requestedKey)
		return nil
	})
	if existingUser == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "User %v is not a registered user", requestJSON.Username)
		return
	}

	// Create new object in database
	newObject := Object{
		Name:     requestJSON.FileName,
		FileSize: requestJSON.FileSize, Owner: requestJSON.Username,
	}
	err = mainDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("objects"))

		// Generate ID for the object.
		// This returns an error only if the Tx is closed or not writeable.
		// That can't happen in an Update() call so I ignore the error check.
		id, _ := b.NextSequence()
		newObject.ID = int(id)

		// Marshal Object into bytes.
		buf, err := json.Marshal(newObject)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put(itob(newObject.ID), buf)
	})
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error adding object to database.")
		log.Printf("Error adding object to database.\nObject: %v", newObject)
		return
	}

	// Update owner of new object
	err = mainDB.Update(func(tx *bolt.Tx) error {
		// Get owner out of users bucket
		b := tx.Bucket([]byte("users"))
		ownerData := b.Get([]byte(newObject.Owner))
		ownerObject := User{}
		err := json.Unmarshal(ownerData, &ownerObject)
		if err != nil {
			return err
		}

		// Add new objectID
		ownerObject.ObjectIDs = append(ownerObject.ObjectIDs, newObject.ID)

		// Update data in bucket
		buf, err := json.Marshal(ownerObject)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put([]byte(ownerObject.Username), buf)
	})

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error adding object to database.")
		log.Printf("Error updating owner in database.\nOwner: %v\nObject: %v\nError: %v", newObject.Owner, newObject.ID, err)
		return
	}

}

func uploadObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func deleteObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func createUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := UserRequestJSON{}

	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		fmt.Fprintf(res, "Error in decoding message")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var existingObject []byte
	requestedKey := []byte(requestJSON.Username)
	mainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		existingObject = b.Get(requestedKey)
		return nil
	})

	if existingObject != nil {
		res.WriteHeader(http.StatusConflict)
		fmt.Fprintf(res, "That username already exists")
		return
	}

	userObject := User{Username: requestJSON.Username, ObjectIDs: []int{}}

	err = mainDB.Update(func(tx *bolt.Tx) error {
		// Retrieve the objects bucket.
		// This should be created when the DB is first opened.
		b := tx.Bucket([]byte("users"))

		// Marshal Object into bytes.
		buf, err := json.Marshal(userObject)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put([]byte(userObject.Username), buf)
	})

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// we're home free!
}

func deleteUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := UserRequestJSON{}

	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		fmt.Fprintf(res, "Error in decoding message")
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	var existingObject []byte
	requestedKey := []byte(requestJSON.Username)
	mainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		existingObject = b.Get(requestedKey)
		return nil
	})

	if existingObject == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "That username does not exist")
		return
	}

	err = mainDB.Update(func(tx *bolt.Tx) error {
		// Retrieve the objects bucket.
		// This should be created when the DB is first opened.
		b := tx.Bucket([]byte("users"))

		// Persist bytes to users bucket.
		return b.Delete(requestedKey)
	})

	if err != nil {
		log.Printf("Error deleting user in database. %v ", err)
		res.WriteHeader(http.StatusInternalServerError)
		return
	}

	// we're home free!
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Initializing server...")

	// These are set up in code for now, but will eventually be CLI params
	addr := ":8080"
	dbfile := "prod.db"

	// Set up HTTP Handling
	mainRouter := mux.NewRouter()
	// Object Actions
	mainRouter.HandleFunc("/object", getObjectHandler).Methods("GET")
	mainRouter.HandleFunc("/object", createObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object", uploadObjectHandler).Methods("PUT")
	mainRouter.HandleFunc("/object", deleteObjectHandler).Methods("DELETE")
	// User Actions
	mainRouter.HandleFunc("/user", deleteUserHandler).Methods("DELETE")
	mainRouter.HandleFunc("/user", createUserHandler).Methods("POST")

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
	mainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
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
