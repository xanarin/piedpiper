package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var MainDB *bolt.DB
var DataPath string

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
	ID            int    `json: "id"`
	Name          string `json: "name"`
	FileSize      int64  `json: "filesize"`
	Owner         string `json: "owner"`
	LocalFileName string `json: "localfilename"`
}

type UploadSession struct {
	ID     int    `json: "id"`
	Object Object `json: "object"`
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func getObjectHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := GetObjectRequestJSON{}
	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding message")
		return
	}

	// Confirm that owner exists
	var userData []byte
	requestedKey := []byte(requestJSON.Username)
	MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		userData = b.Get(requestedKey)
		return nil
	})
	if userData == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "User %v is not a registered user", requestJSON.Username)
		return
	}

	ownerObject := User{}
	err = json.Unmarshal(userData, &ownerObject)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error unmarshalling owner object from database. %v", err)
		return
	}

	// Get object from database (using owner's own index)
	var finalObject *Object
	err = MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("objects"))
		for _, v := range ownerObject.ObjectIDs {
			object := Object{}
			objectData := b.Get(itob(v))
			err := json.Unmarshal(objectData, &object)
			if err != nil {
				return err
			}

			if object.Name == requestJSON.FileName {
				finalObject = &object
				return nil
			}
		}
		return nil
	})

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error retrieving object from database where owner is known. %v", err)
		return
	}

	if finalObject == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "Failed to find object with filename %v belonging to user %v", requestJSON.FileName, requestJSON.Username)
		return
	}

	// Read object back to user
	filepath := path.Join(DataPath, finalObject.LocalFileName)
	// Check that file has been initialized
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		res.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprintf(res, "Object was never uploaded, only created")
		return
	}
	http.ServeFile(res, req, filepath)
  log.Printf("Object %v has been GOTten", finalObject.ID)
}

func createObjectHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := CreateObjectRequestJSON{}
	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding message")
		return
	}

	// Confirm that owner exists
	var existingUser []byte
	requestedKey := []byte(requestJSON.Username)
	MainDB.View(func(tx *bolt.Tx) error {
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
	// Create Random filename
	CHARS := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	seed := rand.NewSource(time.Now().UnixNano())
	bag := rand.New(seed)

	b := make([]byte, 36)
	for i := range b {
		b[i] = CHARS[bag.Intn(len(CHARS))]
	}
	randomFileName := string(b)

	newObject := Object{
		Name:          requestJSON.FileName,
		FileSize:      requestJSON.FileSize,
		Owner:         requestJSON.Username,
		LocalFileName: randomFileName,
	}

	err = MainDB.Update(func(tx *bolt.Tx) error {
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
	err = MainDB.Update(func(tx *bolt.Tx) error {
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

	// Send back upload code to user
	uploadSession := UploadSession{Object: newObject}
	err = MainDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uploads"))

		// Generate ID for the object.
		// This returns an error only if the Tx is closed or not writeable.
		// That can't happen in an Update() call so I ignore the error check.
		id, _ := b.NextSequence()
		uploadSession.ID = int(id)

		// Marshal Object into bytes.
		buf, err := json.Marshal(uploadSession)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put(itob(uploadSession.ID), buf)
	})

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Error adding object to database.")
		log.Printf("Error creating new uploadsession. Error: %v", err)
		return
	}

	fmt.Fprintf(res, "%v", uploadSession.ID)
  log.Printf("Object %v has been created with UploadID %v", uploadSession.Object.ID, uploadSession.ID)
}

func uploadObjectHandler(res http.ResponseWriter, req *http.Request) {
	// Parse data from request
	urlPath := req.URL.Path
	pathParts := strings.Split(urlPath, "/")
	uploadString := pathParts[2]
	uploadID, err := strconv.Atoi(uploadString)
	if err != nil || uploadID == 0 {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding uploadId")
		return
	}

	// Get upload object from store
	uploadData := []byte{}
	MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uploads"))
		uploadData = b.Get(itob(uploadID))
		return nil
	})
	if uploadData == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "UploadID %v is not valid", uploadID)
		return
	}

	uploadSession := UploadSession{}
	err = json.Unmarshal(uploadData, &uploadSession)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error unmarshalling UploadSession object from database. %v", err)
		return
	}

	// Write data to file
	buf := bytes.NewBuffer(make([]byte, 0, req.ContentLength))
	_, err = buf.ReadFrom(req.Body)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Failed to read bytes from request")
		return
	}

	body := buf.Bytes()
	filepath := path.Join(DataPath, uploadSession.Object.LocalFileName)

	err = ioutil.WriteFile(filepath, body, 0600)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Writing uploaded object to disk. Filepath: %v. Error: %v", DataPath+"/"+uploadSession.Object.LocalFileName, err)
		return
	}

	// Remove UploadSession from store
	err = MainDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("uploads"))
		return b.Delete(itob(uploadSession.ID))
	})
  log.Printf("Object %v has been uploaded with UploadID %v", uploadSession.Object.ID, uploadSession.ID)
}

func deleteObjectHandler(res http.ResponseWriter, req *http.Request) {

}

func createUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := UserRequestJSON{}

	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding message")
		return
	}

	var existingObject []byte
	requestedKey := []byte(requestJSON.Username)
	MainDB.View(func(tx *bolt.Tx) error {
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

	err = MainDB.Update(func(tx *bolt.Tx) error {
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
		log.Printf("Database insert of user %v failed with error %v", requestJSON.Username, err)
		return
	}
  log.Printf("User %v has been created", requestJSON.Username)
}

func deleteUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := UserRequestJSON{}

	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding message")
		return
	}

	var existingObject []byte
	requestedKey := []byte(requestJSON.Username)
	MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		existingObject = b.Get(requestedKey)
		return nil
	})

	if existingObject == nil {
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "That username does not exist")
		return
	}

	err = MainDB.Update(func(tx *bolt.Tx) error {
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
  log.Printf("User %v has been deleted", requestJSON.Username)
}

func initDB(dbfile string) error {
  log.Printf("Database initializing....")
	// Open Database Connection
	var err error
	// Open database, with a 1 second timeout in case something goes wrong
	MainDB, err = bolt.Open(dbfile, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return err
	}

	// Instantiate buckets if they don't exist
	err = MainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("objects"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = MainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	err = MainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("uploads"))
		if err != nil {
			return fmt.Errorf("Error creating bucket: %s", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	log.SetOutput(os.Stderr)
	log.Println("Initializing server...")

	// These are set up in code for now, but will eventually be CLI params
	addr := ":3478"
	dbfile := "prod.db"
	DataPath = "data/"

	// Set up HTTP Handling
	mainRouter := mux.NewRouter()
	// Object Actions
	mainRouter.HandleFunc("/object", getObjectHandler).Methods("GET")
	mainRouter.HandleFunc("/object", createObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object", createObjectHandler).Methods("PUT")
	mainRouter.HandleFunc("/object/{uploadid}", uploadObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object/{uploadid}", uploadObjectHandler).Methods("PUT")
	mainRouter.HandleFunc("/object", deleteObjectHandler).Methods("DELETE")
	// User Actions
	mainRouter.HandleFunc("/user", deleteUserHandler).Methods("DELETE")
	mainRouter.HandleFunc("/user", createUserHandler).Methods("POST")

	// Initialize database
	err := initDB(dbfile)
	defer MainDB.Close()
	if err != nil {
		log.Printf("Database initialization failed with error %v", err)
	}

	// Kick off Server
	log.Printf("Server Initialized. Listening on %v", addr)
	err = http.ListenAndServe(addr, mainRouter)
	log.Panicf("Main Router has crashed: %v", err)
}
