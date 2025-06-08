APP_NAME = k8s-controller-tutorial
DOCKER_IMAGE = $(APP_NAME):latest

.PHONY: all build test run docker-build clean

all: build

build:
	go build -o $(APP_NAME) main.go

test:
	go test ./...

run:
	go run main.go

docker-build:
	docker build -t $(DOCKER_IMAGE) .

clean:
	rm -f $(APP_NAME) 