

SRCDIR = src/

all: build run


build:
	mkdir -p bin
	go build -C src -o ../bin/worker worker/worker.go
	go build -C src -o ../bin/manager manager/manager.go

run:
	chmod +x bin/*
	cd bin && ./worker
	#cd bin &&. /manager
