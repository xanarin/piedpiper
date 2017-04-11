.PHONY: deploy invalidate listinvalidations

build: 
	go build -o piedpiper .

linux: 
	env GOOS=linux GOARCH=amd64 go build -o piedpiper-linux . 

clean:
	rm -f piedpiper piedpiper-linux
	
