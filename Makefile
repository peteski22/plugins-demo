.PHONY: all plugins clean help

# Output directories
BIN_DIR := bin
PLUGIN_DIR := sample-plugins
PLUGIN_BIN_DIR := $(BIN_DIR)/$(PLUGIN_DIR)

# Build all plugins (default target)
all: clean plugins

# Build plugins
plugins:
	@mkdir -p $(PLUGIN_BIN_DIR)
	@echo "Building sample plugins..."
	@echo ""
	@cd $(PLUGIN_DIR)/rate-limit && go build -o ../../$(PLUGIN_BIN_DIR)/rate-limit-plugin .
	@echo "  ✓ rate-limit-plugin (Go)"
	@cd $(PLUGIN_DIR)/tool-audit && go build -o ../../$(PLUGIN_BIN_DIR)/tool-audit-plugin .
	@echo "  ✓ tool-audit-plugin (Go)"
	@cd $(PLUGIN_DIR)/header-transformer && go build -o ../../$(PLUGIN_BIN_DIR)/header-transformer-plugin .
	@echo "  ✓ header-transformer-plugin (Go)"
	@cd $(PLUGIN_DIR)/prompt-guard && dotnet publish PromptGuard/PromptGuard.csproj -c Release -r osx-arm64 --self-contained /p:PublishSingleFile=true -o ../../$(PLUGIN_BIN_DIR)/prompt-guard-tmp && \
		mv ../../$(PLUGIN_BIN_DIR)/prompt-guard-tmp/PromptGuard ../../$(PLUGIN_BIN_DIR)/prompt-guard-plugin && \
		rm -rf ../../$(PLUGIN_BIN_DIR)/prompt-guard-tmp
	@echo "  ✓ prompt-guard-plugin (C#/.NET SDK-based)"
	@echo ""
	@echo "✅ All plugins built successfully!"
	@echo ""
	@echo "Built binaries:"
	@ls -lh $(PLUGIN_BIN_DIR)/*-plugin 2>/dev/null | awk '{print "  " $$9 " (" $$5 ")"}'
	@echo ""
	@echo "Note: header-injector (Python) requires runtime and is not built."
	@echo "      See sample-plugins/header-injector/ for source code."
	@echo ""

# Clean build artifacts
clean:
	@rm -rf $(BIN_DIR)
	@echo "✓ Cleaned build artifacts"

# Show help
help:
	@echo "Plugin Examples - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  all (default) - Clean and build all plugins"
	@echo "  plugins       - Build all plugins"
	@echo "  clean         - Clean build artifacts"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Prerequisites:"
	@echo "  • Go 1.25+ (for Go plugins)"
	@echo "  • .NET 9.0+ (for C# plugin)"
	@echo "  • Python 3.12+ with uv (optional, for Python plugin)"
	@echo ""
	@echo "Quick start:"
	@echo "  make          # Build all plugins"
	@echo ""
