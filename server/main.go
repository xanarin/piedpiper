package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	insecureRand "math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/boltdb/bolt"
	"github.com/gorilla/mux"
)

var MainDB *bolt.DB
var DataPath string

const (
	CHARS = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
)

// API JSON objects
type ErrorResponse struct {
	Code        int    `json: "code"`
	Message     string `json: "message"`
	Description string `json: "description"`
}

type GetObjectRequestJSON struct {
	Token    string `json: "token"`
	FileName string `json: "filename"`
}

type CreateObjectRequestJSON struct {
	Token    string `json: "token"`
	FileName string `json: "filename"`
}

type UserCreationJSON struct {
	Username string `json: "username"`
	Password string `json: "password"`
}

type AuthUserRequestJSON struct {
	Username string `json: "username"`
	Password string `json: "password"`
	ReqDate  string `json: "reqdate"`
}

type AuthUserResponseJSON struct {
	ExpirationDate string `json: "expdate"`
	Nonce          string `json: "nonce"`
}

// Internal use structs
type User struct {
	Username     string `json: "username"`
	PasswordHash []byte `json: "passhash"`
	ObjectIDs    []int  `json: "objectids"`
}

type Object struct {
	ID            int    `json: "id"`
	Name          string `json: "name"`
	Owner         string `json: "owner"`
	LocalFileName string `json: "localfilename"`
}

type UploadSession struct {
	ID     int    `json: "id"`
	Object Object `json: "object"`
}

type Token struct {
	Token          []byte `json: "token"`
	User           User   `json: "user"`
	ExpirationDate string `json: "expirationdate"`
}

// itob returns an 8-byte big endian representation of v.
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func checkToken(token string) (*Token, error) {
	// Find token in database, if it exists
	var tokenData []byte
	tokenBytes, err := hex.DecodeString(token)
	if err != nil {
		return nil, err
	}

	err = MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tokens"))
		tokenData = b.Get(tokenBytes)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if tokenData == nil {
		return nil, nil
	}

	tokenObject := Token{}
	err = json.Unmarshal(tokenData, &tokenObject)
	if err != nil {
		return nil, err
	}

	return &tokenObject, nil
}

func checkTokenExpired(token Token) bool {
	tokenExpDate, err := time.Parse("20060102150405", token.ExpirationDate)
	if err != nil {
		log.Panicf("Failed to parse timestamp from our own token: %v", err)
	}

	timeDelta := tokenExpDate.Sub(time.Now().UTC())
	if timeDelta.Hours() < 0.0 {
		return false
	} else if timeDelta.Hours() > 144.0 {
		log.Printf("WARNING: Time has moved backwards. Current time: %s. Token Expiration: %s", time.Now().UTC(), tokenExpDate)
		return false
	}

	return true
}

func getObjectHandler(res http.ResponseWriter, req *http.Request) {
	// Decode Parameters from URL
	queryParams := req.URL.Query()

	// Check to ensure that correct parameters exist
	if len(queryParams["token"]) == 0 || len(queryParams["filename"]) == 0 {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding url parameters")
		return
	}

	requestToken, err := url.QueryUnescape(queryParams["token"][0])
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Parameter 'token' was not URL encoded properly")
		return
	}

	requestFileName, err := url.QueryUnescape(queryParams["filename"][0])
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Parameter 'filename' was not URL encoded properly")
		return
	}

	if requestToken == "" || requestFileName == "" {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding url parameters")
		return
	}

	// Check and validate token
	token, err := checkToken(requestToken)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error retrieving token from datastore: %v", err)
		return
	}

	if token == nil {
		log.Printf("Tried to use invalid token: %s", requestToken)
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "Token '%v' is not a valid token", requestToken)
		return
	}

	// Check if token is expired
	if !checkTokenExpired(*token) {
		log.Printf("Expired token presented for user %v.\n\tToken Expired at: %s\n\tCurrent Time: %s", token.User.Username, token.ExpirationDate, time.Now().UTC())
		res.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprintf(res, "Token is expired.")

		// If token is expired, remove it from database
		err = MainDB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("tokens"))
			return b.Delete(token.Token)
		})

		// Sorry for putting an error handler in an error handler
		if err != nil {
			log.Printf("Failed to remove expired token from the datastore: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(res, "")
			return
		}
		return
	}

	// Get user from database
	var existingUser []byte
	requestedKey := []byte(token.User.Username)
	MainDB.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		existingUser = b.Get(requestedKey)
		return nil
	})
	ownerObject := User{}
	err = json.Unmarshal(existingUser, &ownerObject)

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

			if object.Name == requestFileName {
				finalObject = &object
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
		fmt.Fprintf(res, "Failed to find object with filename %v belonging to user %v", requestFileName, ownerObject.Username)
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

	// Check and validate token
	token, err := checkToken(requestJSON.Token)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error retrieving token from datastore: %v", err)
		return
	}

	if token == nil {
		log.Printf("Tried to use invalid token: '%v'", requestJSON.Token)
		res.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(res, "Token '%v' is not a valid token", requestJSON.Token)
		return
	}

	// Check if token is expired
	if !checkTokenExpired(*token) {
		log.Printf("Expired token presented for user %v.\n\tToken Expired at: %s\n\tCurrent Time: %s", token.User.Username, token.ExpirationDate, time.Now().UTC())
		res.WriteHeader(http.StatusPreconditionFailed)
		fmt.Fprintf(res, "Token is expired.")

		// If token is expired, remove it from database
		err = MainDB.Update(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("tokens"))
			return b.Delete(token.Token)
		})

		// Sorry for putting an error handler in an error handler
		if err != nil {
			log.Printf("Failed to remove expired token from the datastore: %v", err)
			res.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(res, "")
			return
		}
		return
	}

	// Create new object in database
	// Create Random filename
	seed := insecureRand.NewSource(time.Now().UnixNano())
	bag := insecureRand.New(seed)

	b := make([]byte, 36)
	for i := range b {
		b[i] = CHARS[bag.Intn(len(CHARS))]
	}
	randomFileName := string(b)

	newObject := Object{
		Name:          requestJSON.FileName,
		Owner:         token.User.Username,
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

func createUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := UserCreationJSON{}

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

	// Hash password
	plainData := []byte(requestJSON.Password + requestJSON.Username)
	// Hashing the password with the default cost of 10
	hashedData, err := bcrypt.GenerateFromPassword(plainData, bcrypt.DefaultCost)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(res, "Server encountered an error hashing the password")
		log.Printf("Encountered an error hashing the password '%s' with username '%s' using Bcrypt.", requestJSON.Password, requestJSON.Username)
		return
	}

	userObject := User{
		Username:     requestJSON.Username,
		PasswordHash: hashedData,
		ObjectIDs:    []int{},
	}

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

func authUserHandler(res http.ResponseWriter, req *http.Request) {
	requestJSON := AuthUserRequestJSON{}
	err := json.NewDecoder(req.Body).Decode(&requestJSON)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Error in decoding message")
		return
	}
	log.Printf("Request: %v", requestJSON)

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
	// Unmarshal User Object
	userObject := User{}
	err = json.Unmarshal(userData, &userObject)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error unmarshalling owner object from database. %v", err)
		return
	}

	// Bcrypt
	err = bcrypt.CompareHashAndPassword(userObject.PasswordHash, []byte(requestJSON.Password+requestJSON.Username))
	if err != nil {
		res.WriteHeader(http.StatusForbidden)
		fmt.Fprintf(res, "Invalid password given for user %v", requestJSON.Username)
		return
	}

	// Check RequestDate (to prevent replay attack)
	requestDate, err := time.Parse("20060102150405", requestJSON.ReqDate)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(res, "Invalid time stamp.")
		log.Printf("Invalid time stamp from user: '%v'. Parser gave error: %v", requestJSON.ReqDate, err)
		return
	}

	timeSinceRequest := time.Now().UTC().Sub(requestDate)
	if timeSinceRequest.Minutes() > 5.0 {
		res.WriteHeader(http.StatusExpectationFailed)
		fmt.Fprintf(res, "Request Time is greater than 5 minutes ago.")
		log.Printf("Request time >5 minutes from current time. \n\tRequest Time: '%v' \n\tCurrent Time: '%v'", requestDate, time.Now().UTC())
		return
	}

	// At this point, user has been successfully authenticated. Generate a nonce and send it back.
	// This simply creates a random byte array
	var nonce [24]byte

	lengthOfCHARS := int64(len(CHARS))
	for i := 0; i < 24; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(lengthOfCHARS))
		if err != nil {
			res.WriteHeader(http.StatusInternalServerError)
			log.Printf("Error turning big/Int into int64: ", err)
			return
		}
		nonce[i] = CHARS[int(n.Int64())]
	}
	log.Printf("Generated nonce: %v", string(nonce[:]))

	// This is the life of the token
	timeDuration, err := time.ParseDuration("144h")
	if err != nil {
		log.Panicf("%v", err)
	}
	timeOffset, err := time.ParseDuration("-1s")
	if err != nil {
		log.Panicf("%v", err)
	}
	expDateString := time.Now().UTC().Add(timeDuration).Add(timeOffset).Format("20060102150405")

	responseJSON := AuthUserResponseJSON{
		ExpirationDate: expDateString,
		Nonce:          string(nonce[:]),
	}

	// Write response back to client
	responseData, err := json.Marshal(responseJSON)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error marshalling response for client", err)
		return
	}

	res.Header().Set("Content-Type", "application/json")
	res.Write(responseData)

	// Write token into database with user and timestamp for expiration

	// Create hash
	log.Printf("hashInput: '%v'", userObject.Username+string(nonce[:])+expDateString)
	hashInput := []byte(userObject.Username + string(nonce[:]) + expDateString)
	tokenBytes := sha512.Sum512(hashInput)
	log.Printf("hashInput(text): '%v'", hex.EncodeToString(hashInput))

	log.Printf("Token for user %v is: '%v'", userObject.Username, hex.EncodeToString(tokenBytes[:]))

	token := Token{
		Token:          tokenBytes[:],
		User:           userObject,
		ExpirationDate: expDateString,
	}

	err = MainDB.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("tokens"))

		// Marshal Object into bytes.
		buf, err := json.Marshal(token)
		if err != nil {
			return err
		}

		// Persist bytes to users bucket.
		return b.Put(token.Token, buf)
	})
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error storing token in database", err)
		return
	}

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

	err = MainDB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("tokens"))
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
	portPtr := flag.Int("port", 5678, "port for the server to bind")
	dbfilePtr := flag.String("dbfile", "prod.db", "file to be used as database")
	datapathPtr := flag.String("datapath", "./data/", "directory where data files will be stored")
	tlsPtr := flag.Bool("ssl", false, "Whether SSL will be used when serving data")
	fullChainPtr := flag.String("fullchain", "./fullchain.pem", "Full chain file (only used in SSL mode)")
	privKeyPtr := flag.String("privatekey", "./privkey.pem", "Private key file (only used in SSL mode)")
	flag.Parse()

	// Set up HTTP Handling
	mainRouter := mux.NewRouter()

	// Auth Actions
	mainRouter.HandleFunc("/auth", authUserHandler)
	// Object Actions
	mainRouter.HandleFunc("/object", getObjectHandler).Methods("GET")
	mainRouter.HandleFunc("/object", createObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object", createObjectHandler).Methods("PUT")
	mainRouter.HandleFunc("/object/{uploadid}", uploadObjectHandler).Methods("POST")
	mainRouter.HandleFunc("/object/{uploadid}", uploadObjectHandler).Methods("PUT")

	// User Actions
	mainRouter.HandleFunc("/user", createUserHandler).Methods("POST")

	// Initialize database
	err := initDB(*dbfilePtr)
	defer MainDB.Close()
	if err != nil {
		log.Printf("Database initialization failed with error %v", err)
	}

	// Set Data Directory
	DataPath = *datapathPtr

	// Kick off Server
	serveString := fmt.Sprintf(":%v", *portPtr)
	if *tlsPtr {
		log.Printf("Server Initialized. Listening on %s. Serving with SSL.", serveString)
		err = http.ListenAndServeTLS(serveString, *fullChainPtr, *privKeyPtr, mainRouter)
	} else {
		log.Printf("Server Initialized. Listening on %s.", serveString)
		err = http.ListenAndServe(serveString, mainRouter)
	}
	log.Panicf("Main Router has crashed: %v", err)
}
