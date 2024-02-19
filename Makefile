hless:
	go build .

hless-x86_64:
	env GOOS=linux GOARCH=x86_64 go build .

clean:
	go clean

.PHONY: hless hless-x86_64