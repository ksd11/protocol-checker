
BOLD = \033[1m
CLEAR = \033[0m
CYAN = \033[36m


#### 需要测试的pb文件 ########
PB_DIR := ./testdata/protos/protocol-validate# proto文件目录
PB_FILE := $(PB_DIR)/simple.proto# proto文件
PB_NAME := $(shell basename $(PB_FILE) .proto)# 无后缀的文件名
PB_BIN_DIR := ./testdata/pb_bin
PB_BIN := $(PB_BIN_DIR)/$(PB_NAME).pb.bin

.PHONY: test
test: bin/protoc-gen-check testdata/simple_pb_bin
	rm -rf ./testdata/generated && mkdir -p ./testdata/generated
	./bin/protoc-gen-check \
		$(PB_BIN)

# 根据 protoc-gen-debug生成pb解析数据集
testdata/simple_pb_bin: bin/protoc-gen-debug
	rm -rf $(PB_BIN_DIR) && mkdir -p $(PB_BIN_DIR)
	protoc -I ./testdata/protos/protocol-validate \
		-I ~/go/pkg/mod/github.com/envoyproxy/protoc-gen-validate@v1.0.4 \
		--plugin=protoc-gen-debug=./bin/protoc-gen-debug \
		--debug_out="$(PB_BIN_DIR);$(PB_NAME):$(PB_BIN_DIR)" \
		$(PB_FILE)

# 简单的验证数据集
.PHONY: testdata/simple
testdata/simple: bin/protoc-gen-check
	rm -rf ./testdata/generated && mkdir -p ./testdata/generated
	protoc -I ./testdata/protos/protocol-validate \
		-I ~/go/pkg/mod/github.com/envoyproxy/protoc-gen-validate@v1.0.4 \
		--plugin=protoc-gen-check=./bin/protoc-gen-check \
		--check_out="paths=source_relative:./testdata/generated" \
		./testdata/protos/protocol-validate/simple.proto 

# 编译成可执行二进制文件
build: bin/protoc-gen-check bin/protoc-gen-debug

.PHONY: bin/protoc-gen-check
bin/protoc-gen-check:
	@echo "$(CYAN)Building binary $@ ... $(CLEAR)"
	@go build -o ./bin/protoc-gen-check ./protoc-gen-check

.PHONY: bin/protoc-gen-debug
bin/protoc-gen-debug:
	@echo "$(CYAN)Building binary $@ ... $(CLEAR)"
	@go build -o ./bin/protoc-gen-debug ./protoc-gen-debug

.PHONY: clean
clean:
	@echo "$(CYAN)Cleaning...$(CLEAR)"
	@rm -rf ./bin