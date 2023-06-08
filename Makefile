# Go parameters
GOCMD=GO111MODULE=on go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
Mode=dev
all: test build
build:
	rm -rf target/
	mkdir target/
	mkdir target/config
	mkdir target/log
	cp -r config/$(Mode) target/config/$(Mode)
	$(GOBUILD) -o target/gochat -tags=etcd main.go

test:
	$(GOTEST) -v ./...

clean:
	rm -rf target/

run:
	nohup target/gochat -module logic -conf="./target/config/$(Mode)" 2>&1 > target/log/logic.log &
	nohup target/gochat -module connect_websocket -conf="./target/config/$(Mode)" 2>&1 > target/log/connect_websocket.log &
	nohup target/gochat -module api -conf="./target/config/$(Mode)" 2>&1 > target/log/api.log &
	nohup target/gochat -module site -conf="./target/config/$(Mode)" 2>&1 > target/log/site.log &
	nohup target/gochat -module task -conf="./target/config/$(Mode)" 2>&1 > target/log/task.log &

stop:
	pkill -f target/gochat
