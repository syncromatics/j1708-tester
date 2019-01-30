
APP_NAME := j1708-tester
BUILD_PATH := ./artifacts
LINUX_BUILD_PATH = $(BUILD_PATH)/linux/$(APP_NAME)
LINUX_ARM_BUILD_PATH = $(BUILD_PATH)/arm/$(APP_NAME)
WINDOWS_BUILD_PATH = $(BUILD_PATH)/windows/$(APP_NAME).exe
MAC_BUILD_PATH = $(BUILD_PATH)/darwin/$(APP_NAME)

.PHONY: build run package

build:
	mkdir -p artifacts/linux artifacts/arm artifacts/windows artifacts/darwin
	GOOS=linux GOARCH=amd64 go build -o $(LINUX_BUILD_PATH) cmd/$(APP_NAME)/main.go
	GOOS=linux GOARCH=arm go build -o $(LINUX_ARM_BUILD_PATH) cmd/$(APP_NAME)/main.go
	GOOS=darwin GOARCH=amd64 go build -o $(MAC_BUILD_PATH) cmd/$(APP_NAME)/main.go
	GOOS=windows GOARCH=amd64 go build -o $(WINDOWS_BUILD_PATH) cmd/$(APP_NAME)/main.go

run:
	go run cmd/$(APP_NAME)/main.go

package: build
	cd $(BUILD_PATH)/darwin && tar -zcvf ../darwin.tar.gz *
	cd $(BUILD_PATH)/linux && tar -zcvf ../linux.tar.gz *
	cd $(BUILD_PATH)/arm && tar -zcvf ../arm.tar.gz *
	cd $(BUILD_PATH)/windows && zip -r ../windows.zip *
	rm -R $(BUILD_PATH)/darwin $(BUILD_PATH)/linux $(BUILD_PATH)/arm $(BUILD_PATH)/windows
