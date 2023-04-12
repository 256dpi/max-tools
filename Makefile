.PHONY: ewma perftrack syscmd iasnet

check:
	go fmt ./...
	go vet ./...
	golint ./...

ewma:
	cd ewma; maxgo -name ewma -cross -install max-tools

perftrack:
	cd perftrack; maxgo -name perftrack -cross -install max-tools

syscmd:
	cd syscmd; maxgo -name syscmd -cross -install max-tools

iasnet:
	cd iasnet; maxgo -name iasnet -cross -install max-tools
