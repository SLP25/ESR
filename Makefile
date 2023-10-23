VM_IP=192.168.56.101
VM_USER=core
VM_TARGET_DIR=/home/core/tp2
TEST_NAME=basic

compile: sync
	ssh $(VM_USER)@$(VM_IP) "cd $(VM_TARGET_DIR); make build"

build:
	echo "Hello"
#go build -o bin/server server.go

test: compile
	ssh $(VM_USER)@$(VM_IP) "cd $(VM_TARGET_DIR); chmod +x test/test.sh; ./test/test.sh $(TEST_NAME)" 

sync:
	ssh $(VM_USER)@$(VM_IP) "mkdir -p $(VM_TARGET_DIR)/bin"
	scp -r Makefile $(VM_USER)@$(VM_IP):$(VM_TARGET_DIR)
	scp -r src/ $(VM_USER)@$(VM_IP):$(VM_TARGET_DIR)
	scp -r test/ $(VM_USER)@$(VM_IP):$(VM_TARGET_DIR)