# Pied Piper

## Description
Cloud storage has been supported by all major smartphone operating systems. This platform is a prototype for a secured cloud storage system between an Android phone and a laptop/desktop. The phone can download files from and upload files to the server.

Our state of the art, secured protocol provides (1) integrity, (2) confidentiality, and (3) authentication. The protocol is immune to message replay.

## Technology
The server component of this platform is written in Golang, and is hosted in this repository. There is also an Android app in development that will allow users to interact and manage their files that are stored in Pied Piper.

## API Documentation

All requests that require authentication must use HTTP basic auth to attach a username and password. _Eventually, a secret token-based system will take the place of this_

### Ticket Generation and Request

```
C->S    {
            request:ticket
            reqdate:<YYYMMDDHHmmss>,
            username:<username>
            password:<password>
        }
S->C    {
            request:ticket,
            repldate:<YYYYMMDDHHmmss>
            nonce:<128 random characters>,
        }

TOKEN: hash(<username><nonce><reqdate>)

The hash we will use is SHA512
```
