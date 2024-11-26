# Directories
BACKEND_DIR := backend
FRONTEND_DIR := frontend
DIST_DIR := dist

# Backend variables
BACKEND_BINARY := app

# Frontend variables
FRONTEND_BUILD_DIR := $(FRONTEND_DIR)/dist

# Commands
GO := go
NPM := npm

.PHONY: all clean build-backend build-frontend package

all: clean build-backend build-frontend package

# Clean up previous builds
clean:
	@echo "Cleaning up previous builds..."
	rm -rf $(DIST_DIR)
	rm -f $(BACKEND_DIR)/$(BACKEND_BINARY)

# Build the Go backend
build-backend:
	@echo "Building Go backend..."
	cd $(BACKEND_DIR) && $(GO) build -o $(BACKEND_BINARY) ./cmd/main.go

# Build the Node.js frontend
build-frontend:
	@echo "Building Node.js frontend..."
	cd $(FRONTEND_DIR) && $(NPM) install && $(NPM) run build

# Package everything into the dist directory
package:
	@echo "Packaging backend and frontend into $(DIST_DIR)..."
	mkdir -p $(DIST_DIR)
	# Copy backend binary
	cp $(BACKEND_DIR)/$(BACKEND_BINARY) $(DIST_DIR)/
	# Copy frontend build output
	cp -r $(FRONTEND_BUILD_DIR)/* $(DIST_DIR)/
	@echo "Package ready in $(DIST_DIR)"
