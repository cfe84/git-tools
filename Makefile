# Makefile for git-tools
# Builds all Go programs into executables

# Platform detection
ifeq ($(OS),Windows_NT)
    PLATFORM := WINDOWS
    EXT := .exe
else
    UNAME_S := $(shell uname -s)
    ifeq ($(UNAME_S),Linux)
        PLATFORM := LINUX
        EXT :=
    endif
    ifeq ($(UNAME_S),Darwin)
        PLATFORM := MACOS
        EXT :=
    endif
endif

# Variables
GO_FILES := $(wildcard git-*.go)
BIN_DIR := bin
EXECUTABLES := $(addprefix $(BIN_DIR)/, $(addsuffix $(EXT), $(basename $(GO_FILES))))

# Default target
all: $(BIN_DIR) $(EXECUTABLES)

# Create bin directory
$(BIN_DIR):
	mkdir $(BIN_DIR)

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

# Clean target to remove all executables
clean:
	if exist $(BIN_DIR) rmdir /S /Q $(BIN_DIR)

# Test target to verify all programs compile
test: all
	@echo "All executables built successfully in $(BIN_DIR)!"

# Install target (optional) - copies executables to a directory in PATH
install: all
	@echo "To install, copy the executable files to a directory in your PATH"
ifeq ($(PLATFORM),WINDOWS)
	@echo "Example: copy $(BIN_DIR)\*$(EXT) C:\Users\%USERNAME%\bin\"
else
	@echo "Example: sudo cp $(BIN_DIR)/* /usr/local/bin/"
endif

# Help target
help:
	@echo "Available targets:"
	@echo "  all       - Build all executables (default) into bin/"
	@echo "  clean     - Remove bin directory and all executables"
	@echo "  test      - Build and verify all programs compile"
	@echo "  install   - Show instructions for installing executables"
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

.PHONY: all clean test install help
