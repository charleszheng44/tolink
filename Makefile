.PHONY: build install clean uninstall

build:
	go build -o bin/tolink ./cmd/tolink
	go build -o bin/tolinkctl ./cmd/tolinkctl

install: build
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp tolink.service /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
	sudo systemctl enable tolink
	sudo systemctl is-active --quiet tolink && sudo systemctl restart tolink || sudo systemctl start tolink

clean:
	rm -rf bin/

uninstall:
	sudo systemctl stop tolink || true
	sudo systemctl disable tolink || true
	sudo rm -f /usr/local/bin/tolink /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
