# Makefile for git-tools
# Builds all Go programs into executables

# Platform detection
ifeq ($(OS),Windows_NT)
    PLATFORM := WINDOWS
	INSTALL_DIR := $(LOCALAPPDATA)\Microsoft\WindowsApps
    EXT := .exe
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        PLATFORM := LINUX
		INSTALL_DIR := $(HOME)/bin
        EXT :=
    endif
    ifeq ($(UNAME_S),Darwin)
        PLATFORM := MACOS
		INSTALL_DIR := $(HOME)/bin
        EXT :=
    endif
endif

# Variables
GO_FILES := $(wildcard git-*.go)
BIN_DIR := bin
EXECUTABLES := $(addprefix $(BIN_DIR)/, $(addsuffix $(EXT), $(basename $(GO_FILES))))
INSTALLED_EXECUTABLES := $(addprefix $(INSTALL_DIR)/, $(notdir $(EXECUTABLES)))

# Default target
all: $(BIN_DIR) $(EXECUTABLES)

$(BIN_DIR):
	mkdir $(BIN_DIR)

$(INSTALL_DIR):
	mkdir $(INSTALL_DIR)

# Pattern rule to build executables from Go files
$(BIN_DIR)/%$(EXT): %.go common/*.go go.mod | $(BIN_DIR)
	go build -o $@ $<

# Individual targets for each executable
$(BIN_DIR)/git-backup$(EXT): git-backup.go common/*.go go.mod | $(BIN_DIR)
	go build -o $(BIN_DIR)/git-backup$(EXT) git-backup.go

$(BIN_DIR)/git-move-branch$(EXT): git-move-branch.go common/*.go go.mod | $(BIN_DIR)
	go build -o $(BIN_DIR)/git-move-branch$(EXT) git-move-branch.go

$(BIN_DIR)/git-reparent$(EXT): git-reparent.go common/*.go go.mod | $(BIN_DIR)
	go build -o $(BIN_DIR)/git-reparent$(EXT) git-reparent.go

$(BIN_DIR)/git-split$(EXT): git-split.go common/*.go go.mod | $(BIN_DIR)
	go build -o $(BIN_DIR)/git-split$(EXT) git-split.go

$(BIN_DIR)/git-bookmark$(EXT): git-bookmark.go common/*.go go.mod | $(BIN_DIR)
	go build -o $(BIN_DIR)/git-bookmark$(EXT) git-bookmark.go

$(INSTALL_DIR)/%: $(BIN_DIR)/%
	cp $< $@

# Clean target to remove all executables
clean:
ifeq ($(PLATFORM),WINDOWS)
	if exist $(BIN_DIR) rmdir /S /Q $(BIN_DIR)
else
	rm -rf $(BIN_DIR)
endif

# Test target to verify all programs compile
test: all
	@echo "All executables built successfully in $(BIN_DIR)!"

install: $(INSTALL_DIR) $(INSTALLED_EXECUTABLES)
	@echo "Installing to $(INSTALL_DIR)"

# Help target
help:
	@echo "Available targets:"
	@echo "  all       - Build all executables (default) into bin/"
	@echo "  clean     - Remove bin directory and all executables"
	@echo "  test      - Build and verify all programs compile"
	@echo "  install   - Install binaries"
	@echo "  help      - Show this help message"
	@echo ""
	@echo "Platform: $(PLATFORM)"
	@echo "Extension: $(EXT)"
	@echo ""
	@echo "Individual targets (all built in bin/):"
	@echo "  $(BIN_DIR)/git-backup$(EXT)"
	@echo "  $(BIN_DIR)/git-move-branch$(EXT)"
	@echo "  $(BIN_DIR)/git-reparent$(EXT)"
	@echo "  $(BIN_DIR)/git-split$(EXT)"
	@echo "  $(BIN_DIR)/git-bookmark$(EXT)"

.PHONY: all clean test install help
