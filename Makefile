# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
all: test build
build:
	rm -rf target/
	mkdir target/
	cp -r config target/config
	$(GOBUILD) -o target/gochat -tags=etcd main.go

test:
	$(GOTEST) -v ./...

clean:
	rm -rf target/

run:
	nohup target/gochat -module logic 2>&1 > target/logic.log &
	nohup target/gochat -module connect_websocket 2>&1 > target/connect_websocket.log &
	nohup target/gochat -module task 2>&1 > target/task.log &
	nohup target/gochat -module api 2>&1 > target/api.log &
	nohup target/gochat -module site 2>&1 > target/site.log &

stop:
	pkill -f target/gochat
