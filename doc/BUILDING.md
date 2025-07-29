# Intro

# Compile using Local machine

## Linux

Requirements:

```
apt install qtbase5-dev build-essential golang-go
go install github.com/mappu/miqt/cmd/miqt-rcc
go install github.com/mappu/miqt/cmd/miqt-uic
```

Build:
```
make build
```

## Apple Silicon Mac (arm64)

Requirements:

```
xcode-select --install
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
brew install golang
brew install pkg-config
brew install qt@5
brew install make
go install github.com/mappu/miqt/cmd/miqt-rcc
go install github.com/mappu/miqt/cmd/miqt-uic
```

Building:

```
make build_mac
```
The resulting binary will be called: conan-mac

If you need to make a release .app bundle and zipped distribution archive then run:
```
make release_mac
```
The output will be generated in release/ folder called Conan.app and Conan-Mac.zip

# Cross platform compile

## Requirements

* Docker
* Linux Machine (Debian/Ubuntu/ArchLinux etc..)
* GNU make

## Build

```
make build_docker_mactel # Build for MacOS x64 Intel
make build_docker_mac # Build for MacOS Apple Silicon ARM64 (currently not working, use local build)
make build_docker_win # Build for Windows x64 (static binary)
make build_linux # Build for Linux x64 (local build only)
 
```

## Release

```
make release_mactel # Build .app bundle for MacOS x64 Intel
make release_mac    # Build .app bundle for MacOS Apple Silicon ARM64 (only works from local Mac machine)
make release_linux  # Build appImage bundle for x64 Linux
make release_win    # Build .zip with statically linked .exe inside for Windows x64
```

All build commands can be viewed using `make help`.
