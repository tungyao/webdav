version = v1.7

build:
	go build .
	docker build -t tungyao/webdav:$(version) --no-cache .

upload:
	docker push tungyao/webdav:$(version)

all: build upload