hless:
	go build -o hh .

hless-amd64:
	$(MAKE) clean
	GOOS=linux GOARCH=amd64 $(MAKE) hless

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

.PHONY: hless hless-x86_64 logs