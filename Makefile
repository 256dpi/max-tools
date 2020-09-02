.PHONY: ewma mqtt

check:
	go fmt ./...
	go vet ./...
	golint ./...

ewma:
	cd ewma; maxgo -name ewma -install max-tools
	cd perftrack; maxgo -name perftrack -install max-tools
