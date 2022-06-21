VERSION = $(shell git describe --tags --always --dirty)
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
OSARCH=$(shell go env GOHOSTOS)-$(shell go env GOHOSTARCH)

NANOMDM=\
	nanomdm-darwin-amd64 \
	nanomdm-darwin-arm64 \
	nanomdm-linux-amd64

NANO2NANO=\
	nano2nano-darwin-amd64 \
	nano2nano-darwin-arm64 \
	nano2nano-linux-amd64

SUPPLEMENTAL=\
	tools/cmdr.py \
	docs/enroll.mobileconfig

my: nanomdm-$(OSARCH) nano2nano-$(OSARCH)

docker: nanomdm-linux-amd64 nano2nano-linux-amd64

$(NANOMDM): cmd/nanomdm
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

$(NANO2NANO): cmd/nano2nano
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

nanomdm-%-$(VERSION).zip: nanomdm-%.exe nano2nano-%.exe $(SUPPLEMENTAL)
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

nanomdm-%-$(VERSION).zip: nanomdm-% nano2nano-% $(SUPPLEMENTAL)
	rm -rf $(subst .zip,,$@)
	mkdir $(subst .zip,,$@)
	ln $^ $(subst .zip,,$@)
	zip -r $@ $(subst .zip,,$@)
	rm -rf $(subst .zip,,$@)

clean:
	rm -rf nanomdm-* nano2nano-*

release: \
	nanomdm-darwin-amd64-$(VERSION).zip \
	nanomdm-darwin-arm64-$(VERSION).zip \
	nanomdm-linux-amd64-$(VERSION).zip

test:
	go test -v -cover -race ./...

.PHONY: my docker $(NANOMDM) $(NANO2NANO) clean release test
