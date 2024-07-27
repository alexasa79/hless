all: hless hless-amd64

hless hh:
	go build -o hh .

hless-amd64 hh-amd64:
	GOOS=linux GOARCH=amd64 go build -o hh-amd64 .

clean:
	go clean
	rm -f hh

test:
	go test ./...

logs:
	go build cmd/log-gen/log-gen.go
	./log-gen -s 1000 logs/thousand.log
	./log-gen -s 10000 logs/ten-thousand.log
	./log-gen -s 100000 logs/hundred-thousand.log
	./log-gen -s 1000000 logs/million.log
	./log-gen -s 5000000 logs/five-millions.log

.PHONY: hless hless-amd64 logs
