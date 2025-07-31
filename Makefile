SRC := ./src
LINES := $(shell wc -l $(SRC)/*.go | grep total | awk '{print $$1}')
BINARY := conan
BUILD_FILE := BUILD
VERSION := $(shell cat VERSION)
BUILD := $(shell cat $(BUILD_FILE))
QT_PATH_MAC := /opt/homebrew/opt/qt@5
QT_PATH_MSYS := /ucrt64/qt5-static
MAC_DEPLOY_QT := /opt/homebrew/Cellar/qt@5/5.15.16_2/bin/macdeployqt
REL_DIR := release
REL_LINUX_BIN := $(BINARY)-linux
REL_MACINTEL_BIN := $(BINARY)-mactel
REL_MACOS_BIN := $(BINARY)-mac
REL_WINDOWS_BIN := ${BINARY}-win.exe
UID ?= $(shell id -u)
GID ?= $(shell id -g)
WINDOCKERIMAGE := win64-cross-go1.23-qt5.15-static:latest
OSXINTELDOCKER := macos-cross-x86_64-sdk13.1-go1.23-qt5.15-dynamic:latest
OSXARMDOCKER := macos-cross-aarch64-sdk14.5-go1.23-qt5.15-dynamic:latest
VERSION_WIN := $(VERSION).0.$(BUILD)
VERSION_COMMA := $(shell echo $(VERSION_WIN) | tr . ,)

all: help


help:
	@echo "Available targets:"
	@grep -E '^[a-zA-Z0-9._-]+:.*?##' $(MAKEFILE_LIST) | \
		awk -F':|##' '{printf "  \033[1m%-20s\033[0m %s\n", $$1, $$3}'

deps: ## Install dependencies for golang
	go install github.com/mappu/miqt/qt

# Windows x86_64 docker builder
docker_win: ## Make a Windows builder docker container
	@if ! docker image inspect "$(WINDOCKERIMAGE)" > /dev/null 2>&1; then \
	echo "Image not found, building..."; \
	docker build -f docker/win64-cross-go1.23-qt5.15-static.Dockerfile -t "$(WINDOCKERIMAGE)" docker/; \
	else \
	echo "Docker image already built, using it..."; \
	fi

# MacOS Intel docker builder
docker_mactel: ## Make a MacOS Intel builder docker container
	@if ! docker image inspect "$(OSXINTELDOCKER)" > /dev/null 2>&1; then \
	echo "Image not found, building..."; \
	docker build -f docker/macos-cross-x86_64-sdk13.1-go1.23-qt5.15-dynamic.Dockerfile -t "$(OSXINTELDOCKER)" docker/; \
	else \
	echo "Docker image already built, using it..."; \
	fi
# MacOS Apple Silicon docker builder
docker_mac: ## Make a MacOS Apple Silicon builder docker container
	@if ! docker image inspect "$(OSXARMDOCKER)" > /dev/null 2>&1; then \
	echo "Image not found, building..."; \
	docker build -f docker/macos-cross-aarch64-sdk14.5-go1.23-qt5.15-dynamic.Dockerfile -t "$(OSXARMDOCKER)" docker/; \
	else \
	echo "Docker image already built, using it..."; \
	fi

# Generic local build (for development purposes)
build: ## Local build for most systems
	go build -ldflags '-X main.version=$(VERSION) -X main.build=$(BUILD) -X main.debugging=true -v -s -w' -o $(BINARY) $(SRC)

# Linux x64 (local build on linux host)
build_linux: ## Local build for Linux
	go build -ldflags '-X main.version=$(VERSION) -X main.build=$(BUILD) -X main.debugging=false -X main.lines=$(LINES) -v -s -w' -o $(REL_LINUX_BIN) $(SRC)

# macOS ARM (local build)
build_mac: embed_generic ## MacOS Apple Silicon/Intel Local build
	@if test -f $(SRC)/resource.syso; then rm $(SRC)/resource.syso; fi
	CGO_ENABLED=1 \
	PATH="$(QT_PATH_MAC)/bin:$$PATH" \
	LDFLAGS="-L$(QT_PATH_MAC)/lib" \
	CPPFLAGS="-I$(QT_PATH_MAC)/include" \
	PKG_CONFIG_PATH="$(QT_PATH_MAC)/lib/pkgconfig" \
	go build -ldflags '-X main.version=$(VERSION) -X main.build=$(BUILD) -X main.debugging=true -X main.lines=$(LINES) -v -s -w' -o $(REL_MACOS_BIN) $(SRC)

# macOS ARM (using docker cross compile)
# this is under construction
build_docker_mac: docker_mac ## Cross platform docker based build for MacOS Apple Silicon (currently under construction)
	docker run --rm --init -i -t --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=arm64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(OSXARMDOCKER) \
		go build -ldflags "-X main.version=${VERSION} \
		-X main.build=${BUILD} \
		-X main.debugging=false \
		-X main.lines=$(LINES) -v -s -w" \
		-o $(REL_MACOS_BIN) $(SRC)

# macOS Intel build (using docker cross compile)
build_docker_mactel: docker_mactel ## Cross platform docker based build for MacOS Intel
	docker run --rm --init -i --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=amd64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(OSXINTELDOCKER) \
		go build -ldflags "-X main.version=${VERSION} \
		-X main.build=${BUILD} \
		-X main.debugging=false \
		-X main.lines=$(LINES) -v -s -w" \
		-o $(REL_MACINTEL_BIN) $(SRC)

# Windows x86_64 build (using docker cross compile)
build_docker_win: docker_win embed_win ## Cross platform docker based build for Windows x64
	docker run --rm --init -i --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(WINDOCKERIMAGE) \
		go build -ldflags "-X main.version=${VERSION} \
		-X main.build=${BUILD} \
		-X main.lines=$(LINES) \
		-X main.debugging=false \
		-s -w -H windowsgui" --tags=windowsqtstatic -o $(REL_WINDOWS_BIN) ./src/
	@if [ -e $(SRC)/resource.syso ]; then \
		rm $(SRC)/resource.syso; \
	fi

# Please note that this function only works on local apple silicon mac machine
release_mac: ## Release build for MacOS Apple Silicon/Intel (local mac machine only)
	[ -d release/Conan.app ] && rm -rf release/Conan.app || true
	[ -f release/Conan-mac.zip ] && rm release/Conan-mac.zip || true
	cp -r resources/macos-skeleton release/Conan.app
	mkdir release/Conan.app/Contents/MacOS
	cp $(REL_MACOS_BIN) release/Conan.app/Contents/MacOS/Conan
	chmod +x release/Conan.app/Contents/MacOS/Conan
	$(MAC_DEPLOY_QT) release/Conan.app -verbose=1 -always-overwrite -executable=release/Conan.app/Contents/MacOS/Conan
	@# hide app from dock
	@#/usr/libexec/PlistBuddy -c "Add :LSUIElement bool true" "release/Conan.app/Contents/Info.plist"
	@# add copyright
	@/usr/libexec/PlistBuddy -c "Add :NSHumanReadableCopyright string © 2025 e1z0. All rights reserved." "release/Conan.app/Contents/Info.plist"
	@# add version information
	/usr/libexec/PlistBuddy -c "Add :CFBundleShortVersionString string $(VERSION)" "release/Conan.app/Contents/Info.plist"
	@# add build information
	/usr/libexec/PlistBuddy -c "Add :CFBundleVersion string $(BUILD)" "release/Conan.app/Contents/Info.plist"
	mkdir release/Conan.app/Contents/MacOS/scripts
	cp scripts/osahelper release/Conan.app/Contents/MacOS/scripts/
	codesign --force --deep --sign - release/Conan.app
	touch release/Conan.app
	@bash -c 'pushd release > /dev/null; zip -r Conan-Mac.zip Conan.app; popd > /dev/null'
	@echo "Output: release/Conan.app and release/Conan-Mac.zip"

# release mac arm binaries to release folder
# this is under construction
release_mac_non_working:
	@if [ ! -f $(REL_MACOS_BIN) ]; then \
		echo "File $(REL_MACOS_BIN) not found, skipping. You should run make build_mac first"; \
		exit 0; \
	else \
	docker run --rm --init -i -t --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v ${HOME}/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=arm64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
 		$(OSXARMDOCKER) \
                ./scripts/bundle_appv2.sh $(REL_MACOS_BIN) src/Icon.icns Conan.app release Conan-Mac.zip; \
        fi

# release mac intel binaries to release folder
release_mactel: ## Release build for MacOS Intel using docker
	@if [ ! -f $(REL_MACINTEL_BIN) ]; then \
		echo "File $(REL_MACINTEL_BIN) not found, skipping. You should run make build_mactel first"; \
		exit 0; \
	else \
	docker run --rm --init -i --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=amd64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(OSXINTELDOCKER) \
		./scripts/bundle_app.sh $(REL_MACINTEL_BIN) src/Icon.icns Conan.app release Conan-MacIntel.zip; \
	fi

# Make appImage release for Linux
release_linux: ## Release build for Linux (appBundle) local linux machine only
	@if [ ! -f $(REL_LINUX_BIN) ]; then \
		echo "File $(REL_LINUX_BIN) not found, skipping. You should run make build_linux first"; \
		exit 0; \
	else \
	[ -d resources/linux-skeleton/appDir ] && rm -rf resources/linux-skeleton/appDir; \
	./utils/linuxdeploy-x86_64.AppImage \
		--appdir resources/linux-skeleton/appDir \
		--desktop-file resources/linux-skeleton/Conan.desktop \
		--icon-file resources/linux-skeleton/Conan.png \
		--executable $(REL_LINUX_BIN) \
		--plugin qt \
		--output appimage; \
	if [ -f Conan-x86_64.AppImage ]; then \
		rm -rf resources/linux-skeleton/appDir; \
		mv Conan-x86_64.AppImage release/Conan-x86_64.AppImage; \
		echo "Linux target released to release/Conan-x86_64.AppImage"; \
	fi \
	fi
# Make zip file for windows release
release_win: ## Release build for Windows x64 using docker
	@if [ -f $(REL_DIR)/Conan-WinX64.zip ]; then \
	rm $(REL_DIR)/Conan-WinX64.zip; \
        fi
	@if [ -f $(REL_WINDOWS_BIN) ]; then \
	mv $(REL_WINDOWS_BIN) ${REL_DIR}/conan.exe; \
	cd $(REL_DIR) && zip Conan-WinX64.zip conan.exe && rm conan.exe && cd ..; \
	else \
	echo "Binary $(REL_WINDOWS_BIN) cannot be found, first run build_docker_win"; \
	fi

# release combo
release: check-rel-dir release_win release_mactel release_linux ## Release combo for Windows/Mac Intel/Linux


format: ## Format go code
	go fmt $(SRC)
clean: ## Clean binaries and go cache
	rm -f $(BINARY)
	go clean -cache -modcache




# Other misc methods and helpers

# bulder marker (increments build number)
marker: ## Increment build number
	@echo "Incrementing build number..."
	@if [ ! -f $(BUILD_FILE) ]; then echo 1 > $(BUILD_FILE); else \
		n=$$(cat $(BUILD_FILE)); expr $$n + 1 > $(BUILD_FILE); fi

# Embedd qt resources (such as icons)
embed_generic: ## Embed generic QT resources such as icons from src/resources.qrc
	@echo "QT Embed resources"
	miqt-rcc -RccBinary /opt/homebrew/Cellar/qt@5/5.15.16_2/bin/rcc -Input src/resources.qrc -OutputGo src/resources.qrc.go

# Embedd specific windows resources such as version info in PE header
embed_win: ## Embed Windows specific resources (such as version info, build etc...) from src/resource.rc.in 
	@echo "Mingw embed resources"
	sed -e "s/@VERSION_COMMA@/$(VERSION_COMMA)/g" -e "s/@VERSION_DOT@/$(VERSION_WIN)/g" $(SRC)/resource.rc.in > $(SRC)/resource.rc
	docker run --rm --init -i --user $(UID):$(UID) \
	-v ${HOME}/go/pkg/mod:/go/pkg/mod \
	-e GOMODCACHE=/go/pkg/mod \
	-v /home/devnull/.cache/go-build:/.cache/go-build \
	-e GOCACHE=/.cache/go-build \
	-v ${PWD}:/src/conan \
	-w /src/conan \
	-e HOME=/tmp \
	$(WINDOCKERIMAGE) \
	x86_64-w64-mingw32.static-windres $(SRC)/resource.rc -O coff -o $(SRC)/resource.syso





# Enter MacOS x86_64 docker
docker_mactel_enter: ## Enter docker build environment for MacOS Intel
	docker run --rm --init -i -t --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=amd64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(OSXINTELDOCKER) \
		bash
# Enter MacOS Apple Silicon docker
docker_mac_enter: ## Enter docker build environment for MacOS Apple Silicon
	docker run --rm --init -i -t --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOOS=darwin \
		-e GOARCH=arm64 \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(OSXARMDOCKER) \
		bash
# Enter Windows docker
docker_win_enter: ## Enter docker build environment for Windows
	docker run --rm --init -i -t --user $(UID):$(UID) \
		-v ${HOME}/go/pkg/mod:/go/pkg/mod \
		-e GOMODCACHE=/go/pkg/mod \
		-v /home/devnull/.cache/go-build:/.cache/go-build \
		-e GOCACHE=/.cache/go-build \
		-v ${PWD}:/src/conan \
		-w /src/conan \
		-e HOME=/tmp \
		$(WINDOCKERIMAGE) \
		bash

check-rel-dir:
	@if [ ! -d "$(REL_DIR)" ]; then \
		mkdir -p $(REL_DIR); \
		echo "Directory $(REL_DIR) created."; \
	fi

.PHONY: release docker build_linux release_linux help build build_docker_mactel build_docker_win
