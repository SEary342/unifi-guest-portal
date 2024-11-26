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

.PHONY: all clean build-backend build-frontend package copy-env

all: clean build-backend build-frontend package

# Clean up previous builds
clean:
	@echo "Cleaning up previous builds..."
	rm -rf $(DIST_DIR)
	rm -f $(BACKEND_DIR)/$(BACKEND_BINARY)
	rm -rf ${BACKEND_DIR}/cmd/${DIST_DIR}

# Build the Go backend
build-backend:
	@echo "Building Go backend..."
	cd $(BACKEND_DIR) && $(GO) build -o $(BACKEND_BINARY) ./cmd/main.go

# Build the Node.js frontend
build-frontend:
	@echo "Building Node.js frontend..."
	cd $(FRONTEND_DIR) && $(NPM) ci && $(NPM) run build

# Copy the .env file if it exists
copy-env:
	@echo "Checking for .env file..."
	mkdir -p $(DIST_DIR)  # Ensure dist/ exists before copying the .env file
	@if [ -f .env ]; then \
		echo "Copying .env file to $(DIST_DIR)..."; \
		cp .env $(DIST_DIR)/; \
	else \
		echo ".env file not found, skipping..."; \
	fi

backend-debug: clean build-frontend
	@echo "Prepping Front-End Debug for Backend"
	mkdir -p $(BACKEND_DIR)/cmd/$(DIST_DIR)
	cp -r $(FRONTEND_BUILD_DIR)/* $(BACKEND_DIR)/cmd/$(DIST_DIR)/

# Package everything into the dist directory
package: copy-env
	@echo "Packaging backend and frontend into $(DIST_DIR)..."
	# Copy backend binary
	chmod u+x $(BACKEND_DIR)/$(BACKEND_BINARY)
	mv $(BACKEND_DIR)/$(BACKEND_BINARY) $(DIST_DIR)/
	# Copy frontend build output
	cp -r $(FRONTEND_BUILD_DIR)/* $(DIST_DIR)/
	@echo "Package ready in $(DIST_DIR)"
