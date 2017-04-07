# Server
The server portion of this project is almost feature-complete without considering the encryption aspect. The screenshot in this directory is an example of interacting with the service using the cURL utility. The service currently supports:
  - User Account Creation
  - User Account Deletion
  - Object Creation
  - Object Upload
  - Object Retrieval

Objects are stored on the disk of the server with a randomized filename. This could be moved onto a network share or a cloud storage solution such as Amazon S3. However, the client only knows their username and the filename that they set for the uploaded object. On the server side, all data is kept in a Key Value store. The server is written in Go (Golang).
