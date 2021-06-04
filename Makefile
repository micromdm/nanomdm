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

my: nanomdm-$(OSARCH) nano2nano-$(OSARCH)

docker: nanomdm-linux-amd64 nano2nano-linux-amd64

$(NANOMDM): cmd/nanomdm
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

$(NANO2NANO): cmd/nano2nano
	GOOS=$(word 2,$(subst -, ,$@)) GOARCH=$(word 3,$(subst -, ,$(subst .exe,,$@))) go build $(LDFLAGS) -o $@ ./$<

%-$(VERSION).zip: %.exe
	rm -f $@
	zip $@ $<

%-$(VERSION).zip: %
	rm -f $@
	zip $@ $<

clean:
	rm -f nanomdm-* nano2nano-*

release: \
	$(foreach bin,$(NANOMDM),$(subst .exe,,$(bin))-$(VERSION).zip) \
	$(foreach bin,$(NANO2NANO),$(subst .exe,,$(bin))-$(VERSION).zip)

test:
	go test -v -cover -race ./...

.PHONY: my docker $(NANOMDM) $(NANO2NANO) clean release test
