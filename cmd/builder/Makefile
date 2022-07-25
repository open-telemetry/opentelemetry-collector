include ../../Makefile.Common

.PHONY: ocb
ocb:
	GO111MODULE=on CGO_ENABLED=0 $(GOCMD) build -trimpath -o ../../bin/ocb_$(GOOS)_$(GOARCH) .
