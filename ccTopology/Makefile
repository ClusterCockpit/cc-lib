
GOBIN = $(shell which go)
HWLOC_ROOT ?= $(shell echo $$LD_LIBRARY_PATH | tr ':' '\n' | grep -E ".+hwloc.+/lib")

.PHONY: test
test:
	CGO_LDFLAGS="-L$(HWLOC_ROOT)" $(GOBIN) test
	cd test && CGO_LDFLAGS="-L$(HWLOC_ROOT)" $(GOBIN) test
