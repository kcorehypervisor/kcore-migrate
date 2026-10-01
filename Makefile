# Build and publish from `nix develop` so go, make, git, gh, and sha256sum match.
VERSION := $(shell tr -d '[:space:]' < VERSION)
BIN := kcore-migrate
DIST := dist
LDFLAGS := -s -w -X main.version=$(VERSION)
PLATFORMS := linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

.PHONY: build test dist release release-publish clean

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags '$(LDFLAGS)' -o bin/$(BIN) ./cmd/kcore-migrate

test:
	go test ./...

dist: test
	rm -rf $(DIST)
	mkdir -p $(DIST)
	set -e; for p in $(PLATFORMS); do \
		os=$${p%-*}; arch=$${p#*-}; \
		stage=$(DIST)/$(BIN)-$(VERSION)-$$p; \
		mkdir -p $$stage; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags '$(LDFLAGS)' -o $$stage/$(BIN) ./cmd/kcore-migrate; \
		tar -C $(DIST) -czf $$stage.tar.gz $(BIN)-$(VERSION)-$$p; \
		rm -rf $$stage; \
	done
	cd $(DIST) && sha256sum *.tar.gz > SHA256SUMS

# Tag v$(VERSION), package dist/, upload a GitHub Release.
release:
	bash ./scripts/release.sh release

# Upload dist/ to the existing v$(VERSION) release. Refuses a dirty tree.
release-publish: dist
	bash ./scripts/release.sh publish

clean:
	rm -rf bin $(DIST)
