
bindir  = ./bin
prefix  = /usr/local/bin
appname = git-credential-bitbucket

.PHONY: all install

all: $(bindir)/$(appname)

install: $(prefix)/$(appname)

$(bindir)/$(appname): *.go go.mod go.sum
	go build -o $@

$(prefix)/$(appname): $(bindir)/$(appname)
	sudo install -s -t $(prefix) $<