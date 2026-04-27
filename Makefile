.PHONY: build install clean uninstall

UNAME := $(shell uname -s)

build:
	go build -o bin/tolink ./cmd/tolink
	go build -o bin/tolinkctl ./cmd/tolinkctl

install: build
ifeq ($(UNAME),Darwin)
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp bin/tolinkctl /usr/local/bin/tolinkctl
	mkdir -p $(HOME)/Library/LaunchAgents
	cp tolink.plist $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
	launchctl unload $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist 2>/dev/null || true
	launchctl load $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
else
	sudo cp bin/tolink /usr/local/bin/tolink
	sudo cp bin/tolinkctl /usr/local/bin/tolinkctl
	sudo cp tolink.service /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
	sudo systemctl enable tolink
	sudo systemctl is-active --quiet tolink && sudo systemctl restart tolink || sudo systemctl start tolink
endif

clean:
	rm -rf bin/

uninstall:
ifeq ($(UNAME),Darwin)
	launchctl unload $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist 2>/dev/null || true
	rm -f $(HOME)/Library/LaunchAgents/com.charleszheng44.tolink.plist
	sudo rm -f /usr/local/bin/tolink /usr/local/bin/tolinkctl
else
	sudo systemctl stop tolink || true
	sudo systemctl disable tolink || true
	sudo rm -f /usr/local/bin/tolink /usr/local/bin/tolinkctl /etc/systemd/system/tolink.service
	sudo systemctl daemon-reload
endif
