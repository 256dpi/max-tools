.PHONY: ewma perftrack

check:
	go fmt ./...
	go vet ./...
	golint ./...

ewma:
	cd ewma; maxgo -name ewma -cross -install max-tools

perftrack:
	cd perftrack; maxgo -name perftrack -cross -install max-tools
