.PHONY: build install

build:
	go build -o bin/tolink ./cmd/tolink

install: build
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp tolink.service /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
	sudo systemctl enable tolink
