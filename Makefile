-include .env

# Goodies
V = 0
Q = $(if $(filter 1,$V),,@)
E := 
S := $E $E
M = $(shell printf "\033[34;1m▶\033[0m")
rwildcard = $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2) $(filter $(subst *,%,$2),$d))

# Folders
BIN_DIR     ?= $(CURDIR)/bin
LOG_DIR     ?= log
TMP_DIR     ?= tmp
INSTALL_DIR ?= /usr/local/bin

# Version, branch, and project
BRANCH    != git symbolic-ref --short HEAD
COMMIT    != git rev-parse --short HEAD
STAMP     != date +%Y%m%d%H%M%S
BUILD     := "$(STAMP).$(COMMIT)"
VERSION   != awk '/^var +VERSION +=/{gsub("\"", "", $$4) ; print $$4}' version.go
ifeq ($VERSION,)
VERSION   != git describe --tags --always --dirty="-dev"
endif
PROJECT   != awk '/^const +APP += +/{gsub("\"", "", $$4); print $$4}' version.go
ifeq (${PROJECT},)
PROJECT   != basename "$(PWD)"
endif
PLATFORMS ?= darwin linux windows pi

# Files
GOTESTS   := $(call rwildcard,,*_test.go)
GOFILES   := $(filter-out $(GOTESTS), $(call rwildcard,,*.go))
ASSETS    :=

# Tools
GO       ?= go
GOOS     != $(GO) env GOOS
HTTPIE   ?= http
LOGGER    =  bunyan -L -o short
GOBIN     = $(BIN_DIR)
GOLINT   ?= golangci-lint
#COMPRESS ?= 7z
COMPRESS ?= zip

# Flags
#MAKEFLAGS += --silent
# GO
LDFLAGS = -ldflags "-X main.commit=$(COMMIT) -X main.branch=$(BRANCH) -X main.stamp=$(STAMP)"
ifneq ($(what),)
TEST_ARG := -run '$(what)'
else
TEST_ARG :=
endif

# Main Recipes
.PHONY: all archive build clean dep fmt help install lint version vet

help: Makefile; ## Display this help
	@echo
	@echo "$(PROJECT) version $(VERSION) build $(BUILD) in $(BRANCH) branch"
	@echo "Make recipes you can run: "
	@echo
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) |\
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo

archive: __archive_init__ __archive_all__; @ ## Archive the binaries

build: __build_init__ __build_all__; @ ## Build the application for all platforms

dep:; $(info $(M) Updating Modules...) @ ## Updates the GO Modules
	$Q $(GO) get -u ./...
	$Q $(GO) mod tidy

install: build __install_init__ __install_all__; @ ## Installs the binaries on the local host

lint:;  $(info $(M) Linting application...) @ ## Lint Golang files
	$Q $(GOLINT) run *.go

fmt:; $(info $(M) Formatting the code...) @ ## Format the code following the go-fmt rules
	$Q $(GO) fmt *.go

vet:; $(info $(M) Vetting application...) @ ## Run go vet
	$Q $(GO) vet *.go

clean:; $(info $(M) Cleaning up folders and files...) @ ## Clean up
	$Q rm -rf $(BIN_DIR)  2> /dev/null
	$Q rm -rf $(LOG_DIR)  2> /dev/null
	$Q rm -rf $(TMP_DIR)  2> /dev/null

version:; @ ## Get the version of this project
	@echo $(VERSION)

# install recipes
.PHONY: __install_all__ __install_init__
__install_init__:; $(info $(M) Installing application $(PROJECT))
__install_all__:   $(INSTALL_DIR)/$(PROJECT);

$(INSTALL_DIR)/$(PROJECT): $(BIN_DIR)/linux/$(PROJECT)
	sudo install -s -t $(INSTALL_DIR) $<

# build recipes for various platforms
.PHONY: __build_all__ __build_init__ __fetch_modules__
__build_init__:;     $(info $(M) Building application $(PROJECT))
__build_all__:       $(foreach platform, $(PLATFORMS), $(BIN_DIR)/$(platform)/$(PROJECT));
__fetch_modules__: ; $(info $(M) Fetching Modules...)
	$Q $(GO) mod download

$(BIN_DIR)/darwin: $(BIN_DIR) ; @mkdir -p $@
$(BIN_DIR)/darwin/$(PROJECT): $(GOFILES) $(ASSETS) | $(BIN_DIR)/darwin; $(info $(M) building application for darwin)
	$Q CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build $(if $V,-v) $(LDFLAGS) -o $@ .

$(BIN_DIR)/linux:   $(BIN_DIR) ; @mkdir -p $@
$(BIN_DIR)/linux/$(PROJECT): $(GOFILES) $(ASSETS) | $(BIN_DIR)/linux; $(info $(M) building application for linux)
	$Q CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(if $V,-v) $(LDFLAGS) -o $@ .

$(BIN_DIR)/windows: $(BIN_DIR) ; @mkdir -p $@
$(BIN_DIR)/windows/$(PROJECT): $(BIN_DIR)/windows/$(PROJECT).exe;
$(BIN_DIR)/windows/$(PROJECT).exe: $(GOFILES) $(ASSETS) | $(BIN_DIR)/windows; $(info $(M) building application for windows)
	$Q CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build $(if $V,-v) $(LDFLAGS) -o $@ .

$(BIN_DIR)/pi:   $(BIN_DIR) ; @mkdir -p $@
$(BIN_DIR)/pi/$(PROJECT): $(GOFILES) $(ASSETS) | $(BIN_DIR)/pi; $(info $(M) building application for pi)
	$Q CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=6 $(GO) build $(if $V,-v) $(LDFLAGS) -o $@ .

# archive recipes
.PHONY: __archive_all__ __archive_init__
__archive_init__:;     $(info $(M) Archiving binaries for application $(PROJECT))
__archive_all__:       $(foreach platform, $(PLATFORMS), $(BIN_DIR)/$(platform)/$(PROJECT)-$(VERSION).$(platform).$(COMPRESS));

$(BIN_DIR)/darwin/$(PROJECT)-$(VERSION).darwin.7z: $(BIN_DIR)/darwin/$(PROJECT)
	7z a -r $@ $<
$(BIN_DIR)/linux/$(PROJECT)-$(VERSION).linux.7z: $(BIN_DIR)/linux/$(PROJECT)
	7z a -r $@ $<
$(BIN_DIR)/windows/$(PROJECT)-$(VERSION).windows.7z: $(BIN_DIR)/windows/$(PROJECT).exe
	7z a -r $@ $<
$(BIN_DIR)/pi/$(PROJECT)-$(VERSION).pi.7z: $(BIN_DIR)/pi/$(PROJECT)
	7z a -r $@ $<

$(BIN_DIR)/darwin/$(PROJECT)-$(VERSION).darwin.zip: $(BIN_DIR)/darwin/$(PROJECT)
	zip -u -j9 $@ $<
$(BIN_DIR)/linux/$(PROJECT)-$(VERSION).linux.zip: $(BIN_DIR)/linux/$(PROJECT)
	zip -u -j9 $@ $<
$(BIN_DIR)/windows/$(PROJECT)-$(VERSION).windows.zip: $(BIN_DIR)/windows/$(PROJECT).exe
	zip -u -j9 $@ $<
$(BIN_DIR)/pi/$(PROJECT)-$(VERSION).pi.zip: $(BIN_DIR)/pi/$(PROJECT)
	zip -u -j9 $@ $<

# Download recipes
$(BIN_DIR)/%: | $(BIN_DIR) ; $(info $(M) installing $(PACKAGE)...)
	$Q tmp=$$(mktemp -d) ; \
	  env GOPATH=$$tmp GOBIN=$(BIN_DIR) $(GO) get $(PACKAGE) || status=$$? ; \
	  chmod -R u+w $$tmp ; rm -rf $$tmp ; \
	  exit $$status
