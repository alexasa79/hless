hless:
	go build cmd/hless.go

hless-x86_64:
	env GOOS=linux GOARCH=x86_64 go build cmd/hless.go

.PHONY: hless hless-x86_64