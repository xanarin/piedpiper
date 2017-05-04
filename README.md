# Pied Piper

## Description
Cloud storage has been supported by all major smartphone operating systems. This platform is a prototype for a secured cloud storage system between an Android phone and a laptop/desktop. The phone can download files from and upload files to the server.

Our state of the art, secured protocol provides (1) integrity, (2) confidentiality, and (3) authentication. The protocol is immune to message replay.

## Technology
The server component of this platform is written in Golang, and is hosted in this repository. There is also an Android app in development that will allow users to interact and manage their files that are stored in Pied Piper.

## API Documentation

All requests that require authentication must use HTTP basic auth to attach a username and password. _Eventually, a secret token-based system will take the place of this_

### Account Registration
Request: POST /user 
```json
{
  "username":<username>,
  "password":<password>
}
```

Response: HTTP Status Code indicating success or giving specific error.

### Ticket Generation and Request
Request: POST /auth
```json
{
  "reqdate":<YYYYMMDDHHmmss>,
  "username":<username>,
  "password":<password>
}
```

Response:
```json
{
    "expdate":<YYYYMMDDHHmmss>,
    "nonce":<128 random characters>
}
```

Device Token: hash(\<username\>\<nonce\>\<reqdate\>)

The hash we will use is SHA512. Nonces and hashes are represented in hexidecimal in all requests and responses.

### Create Object
Request: POST /object
```json
{
  "token": <token>,
  "filename": <filename>
}
```

Returns an UploadID, to be used in the next step of object initialization.

### Upload Object
Request: POST /object/\<UploadID\>

Object is sent in the Body of the request, encoded as a series of bytes.

### Get Object
Request: GET /object
```json
{
  "token": <token>,
  "filename": <filename>
}
```

Object is returned in the Body of the response, encoded as a series of bytes.

## Choice of Crypto
Currently, the client-server API is protected with TLS that uses a valid SSL certificate issued by Let’s Encrypt. The user authentication token consists of a SHA-512 hash over a username, a 128 character nonce, and the timestamp of when the token was requested. The android client uses AES-256 in ECB mode for now but this will be replaced with CBC or GCM mode in the future. 

## Implementation/Testing
Android app was developed using Android studio.  It sends JSON over HTTPS to the server, and sends the files in byte arrays to the server in the body of the HTTP POST.

The app was tested by creating files and sending them to the server from the app. Then retrieving the file back from the server and saving the file.  The integrity of the file was checked and it was also made sure the contents of the file were the same as the original. In addition to using the app, files were uploaded from the command line and pull from the server by the app. Files were also uploaded from the app and pulled from the server from the command line. The same file checks were done on these files.

The server was implemented using the Go language. Testing of the server was carried out using Go’s built-in testing framework. The handlers were tested using simulated API calls, and the server-side code currently has 60.5% test coverage of all statements. The password hashing on the server side is done by first salting the password with the user’s username, then using the bcrypt library to hash the salted password, and then storing the result in a database. All request handling is done using Goroutines, which can be thought of as threads. This ensures that all requests are responded to as quickly as possible, since all processors on the server can be utilized simultaneously. The database of choice was Bolt, a disk-based key-value store written in Go. It is highly performant while still reliable, and can generate snapshots so the datastore can be read in parallel, and not block queued writes. This helps reduce the possibility for race conditions in the code. This was essential since many requests would be accessing the main datastores simultaneously, though only a few operations require writing to them. All functionality was first tested with the Go testing framework, then was tested on the production server using cURL in verbose mode.

## Bugs/Weaknesses
Currently the client-side cryptography is using 256-bit AES-ECB to encrypt user files before sending them to the server. A more secure mode of operation (AES-CBC or AES-GCM) will be used by the time the project is completed in order to better protect user files from cryptanalytic attacks. 

Additionally, the symmetric key is derived using SHA1PRNG because it was the easiest key derivation mechanism that was available in the widest range of android systems. This will be replaced with a more secure key derivation function once time allows.

It is currently possible for a client to request an unlimited number of tokens, and the system would keep generating and returning them as long as the requests are valid. After the tokens have expired, they would be removed, but a malicious client could exhaust the server resources before that time. Because the user would not be making these requests directly, there could be a rule set for all API users that would stipulate that they may not intentionally or unintentionally exploit this vulnerability. Another solution would be to add a cool-down period to each user’s authentication requests, so another token could not be administered until the cool-down time had expired. However, this solution would add additional complexity and computational cost to the application.
