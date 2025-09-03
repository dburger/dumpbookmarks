SRC = $(wildcard *.go)  $(wildcard **/*.go)

bin/dumpbookmarks: $(SRC)
	GOOS=linux GOARCH=amd64 go build -o bin/dumpbookmarks

bin/dumpbookmarks.exe: $(SRC)
	GOOS=windows GOARCH=amd64 go build -o bin/dumpbookmarks.exe

buildl: bin/dumpbookmarks

buildw: bin/dumpbookmarks.exe

build: buildl buildw

runl: buildl
	./bin/dumpbookmarks

runw: buildw
	./bin/dumpbookmarks.exe

clean:
	rm -rf ./bin
