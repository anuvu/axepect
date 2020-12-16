VERSION := $(shell x=$$(git describe --tags 2>/dev/null) && echo $${x\#v} || echo unknown)
VERSION_SUFFIX := $(shell [ -z "$$(git status --porcelain --untracked-files=no)" ] || echo -dirty)
VERSION_FULL := $(VERSION)$(VERSION_SUFFIX)
LDFLAGS := "${ldflags:+$ldflags }-X main.version=${ver}${suff}"
BUILD_FLAGS := -ldflags "-X main.version=$(VERSION_FULL)"

CMDS := cimc/cmd/demo

GO_FILES := $(wildcard *.go)
ALL_GO_FILES := $(wildcard *.go */*.go)

build: .build $(CMDS)

.build: $(GO_FILES)
	go build ./...
	@touch $@

cimc/cmd/demo: $(wildcard cimc/cmd/*.go) $(GO_FILES)
	cd $(dir $@) && go build -o $(notdir $@) $(BUILD_FLAGS) ./...

gofmt: .gofmt

.gofmt: $(ALL_GO_FILES)
	o=$$(gofmt -l -w .) && [ -z "$$o" ] || { echo "gofmt made changes: $$o"; exit 1; }
	@touch $@
