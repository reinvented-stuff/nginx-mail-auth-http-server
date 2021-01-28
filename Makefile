VERSION := $(shell cat .version )
PLATFORM := $(shell uname -s | tr [A-Z] [a-z])
GO = go

PROGNAME = nginx-mail-auth-http-server
PROGNAME_VERSION = $(PROGNAME)-$(VERSION)
TARGZ_FILENAME = $(PROGNAME)-$(VERSION).tar.gz
TARGZ_CONTENTS = nginx-mail-auth-http-server README.md Makefile .version

PREFIX = /tmp
PWD = $(shell pwd)

export PROGROOT=$(PWD)/$(PROGNAME_VERSION)

.PHONY: all version build clean install test

$(TARGZ_FILENAME):
	mkdir -vp "$(PROGNAME_VERSION)"
	cp -v $(TARGZ_CONTENTS) "$(PROGNAME_VERSION)/"
	tar -zvcf "$(TARGZ_FILENAME)" "$(PROGNAME_VERSION)"

$(PROGNAME):
	env GOOS="$(PLATFORM)" $(GO) build -ldflags="-X 'main.BuildVersion=$(VERSION)'" -v -o "$(PROGNAME)" .

test:
	@echo "Not implemented yet"

install:
	install -d $(DESTDIR)/usr/share/doc/$(PROGNAME_VERSION)
	install -d $(DESTDIR)/usr/bin
	install -m 755 $(PROGNAME) $(DESTDIR)/usr/bin
	install -m 644 README.md $(DESTDIR)/usr/share/doc/$(PROGNAME_VERSION)

clean:
	rm -vf "$(PROGNAME)"

build: $(PROGNAME)

compress: $(TARGZ_FILENAME)
