VM_IP=192.168.56.101
VM_USER=core
VM_TARGET_DIR=/home/core/tp2
TEST_NAME=basic

compile: sync
	ssh $(VM_USER)@$(VM_IP) "cd $(VM_TARGET_DIR); make build"

build:
	go build -o bin/bootstrapper ./cmd/bootstrapper
	go build -o bin/client ./cmd/client
	go build -o bin/node ./cmd/node
	go build -o bin/server ./cmd/server

test: compile
	ssh $(VM_USER)@$(VM_IP) "/bin/bash -c cd $(VM_TARGET_DIR); chmod +x test/test.sh; ./test/test.sh $(TEST_NAME)" 

sync:
	scp -r bin/ $(VM_USER)@$(VM_IP):$(VM_TARGET_DIR)