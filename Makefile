version = v1.9.1

build:
	go build .
	docker build -t tungyao/webdav:$(version) --no-cache .

upload:
	docker push tungyao/webdav:$(version)

all: build upload