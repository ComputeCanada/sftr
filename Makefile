SOURCE_DIR	:= .
OUTPUT_DIR	:= dist

version_go	:= $(SOURCE_DIR)/version.go
SOURCES		:= $(SOURCE_DIR)/*.go

prog_x86_64	:= $(OUTPUT_DIR)/sftr_linux_x86-64
prog_osx	:= $(OUTPUT_DIR)/sftr_osx

ALL_BUILDS	:= $(prog_x86_64) $(prog_osx)

# build flags
LD_FLAGS_COMMON	:= -w -s
LD_FLAGS	:= $(LD_FLAGS_COMMON)
LD_FLAGS_LINUX	:= -extldflags \"-static\" $(LD_FLAGS)

# I don't anticipate using these or know if the target platforms support
# OpenSSH anyway.  If needed, see:
# https://github.com/cloudfoundry/cli/blob/master/Makefile
prog_i686	:= $(OUTPUT_DIR)/sftr_linux_i686
prog_win32	:= $(OUTPUT_DIR)/sftr_win32.exe
prog_winx64	:= $(OUTPUT_DIR)/sftr_winx64.exe

# determine version
VERSION		:= $(shell git describe)

all: $(ALL_BUILDS)

$(version_go): .git/refs/heads
	@echo "package main\n\nconst PROGRAM_VERSION = \"$(VERSION)\"" > $@

$(prog_osx):	$(SOURCES) $(version_go)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build \
		-ldflags "$(LD_FLAGS)" -o $@

$(prog_x86_64):	$(SOURCES) $(version_go)
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build \
		-ldflags "$(LD_FLAGS_LINUX)" -o $@

sftr: $(SOURCES)
	@go build sftr.go

test: $(ALL_BUILDS)
	@./test.sh
