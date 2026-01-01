IMAGENAME = ipmipower

.PHONY: all clean build

all: build

clean:
	@rm -f $(BASEDIR)/ipmipower
	$(call egreen,Done.)

build:
	cd $(BASEDIR) && \
	go build -v -trimpath -ldflags "-s -w" -o ipmipower .
	$(call egreen,Done.)

### Common ### Do not modify
include build-common/home.mk
include build-common/docker.mk
include Makefile.local
