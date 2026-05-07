BINARY      := pgagent
PKG         := .
PREFIX      ?= $(HOME)/.local
BIN_DIR     ?= $(PREFIX)/bin
CONFIG_DIR  ?= $(HOME)/.pgagent
CONFIG_FILE := $(CONFIG_DIR)/config.yml
EXAMPLE     := config.yml.example
DIST_DIR    := dist

GO          ?= go
GOFLAGS     ?=
LDFLAGS     ?= -s -w

.PHONY: all build install uninstall config clean test help

all: build

help:
	@echo "Targets:"
	@echo "  build      Compile the pgagent binary into ./$(DIST_DIR)/"
	@echo "  install    Build, install to $(BIN_DIR), seed $(CONFIG_FILE)"
	@echo "  config     Create $(CONFIG_FILE) from $(EXAMPLE) (no overwrite)"
	@echo "  uninstall  Remove $(BIN_DIR)/$(BINARY) (config is preserved)"
	@echo "  test       Run go test ./..."
	@echo "  clean      Remove ./$(DIST_DIR)/"

build:
	@mkdir -p $(DIST_DIR)
	$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) $(PKG)
	@echo "built $(DIST_DIR)/$(BINARY)"

install: build config
	@mkdir -p $(BIN_DIR)
	@if [ -w "$(BIN_DIR)" ]; then \
		install -m 0755 $(DIST_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY); \
	else \
		echo "$(BIN_DIR) is not writable; using sudo"; \
		sudo install -m 0755 $(DIST_DIR)/$(BINARY) $(BIN_DIR)/$(BINARY); \
	fi
	@echo ""
	@echo "installed: $(BIN_DIR)/$(BINARY)"
	@echo "config:    $(CONFIG_FILE)"
	@echo ""
	@echo "Next: edit $(CONFIG_FILE) to add your databases, then run:"
	@echo "  $(BINARY) -db <name> -sql \"SELECT ...\""

config:
	@mkdir -p $(CONFIG_DIR)
	@if [ -f "$(CONFIG_FILE)" ]; then \
		echo "config exists, leaving it alone: $(CONFIG_FILE)"; \
	else \
		cp $(EXAMPLE) $(CONFIG_FILE); \
		chmod 600 $(CONFIG_FILE); \
		echo "seeded: $(CONFIG_FILE)"; \
	fi

uninstall:
	@if [ -w "$(BIN_DIR)" ]; then \
		rm -f $(BIN_DIR)/$(BINARY); \
	else \
		sudo rm -f $(BIN_DIR)/$(BINARY); \
	fi
	@echo "removed: $(BIN_DIR)/$(BINARY)"
	@echo "config preserved at $(CONFIG_FILE) — delete manually if no longer needed"

test:
	$(GO) test ./...

clean:
	rm -rf $(DIST_DIR)
