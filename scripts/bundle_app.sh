#!/bin/bash
set -e

# call ./bundle_app.sh binary icon bundle.app release_folder release.zip
# example.: ./scripts/bundle_app.sh conan-mactel src/Icon.icns Conan.app release Conan-Mac-Intel.zip


APP_NAME="Conan"
BIN_PATH="$1"      # compiled app
ICON="$2"
RELEASE_DIR="$4"
APP_BUNDLE="$4/$3"
BUNDLE_DIR="$3"
ZIP_NAME="$5"
FRAMEWORKS="Contents/Frameworks"
MACOS="Contents/MacOS"
PLATFORMS="Contents/PlugIns/platforms"
RESOURCES="Contents/Resources"
VERSION=$(cat VERSION)
BUILD=$(cat BUILD)

# osxcross toolchain prefix
TOOL_PREFIX=$TOOLCHAINPREFIX
OTOOL="${TOOL_PREFIX}-otool"
INSTALL_NAME_TOOL="${TOOL_PREFIX}-install_name_tool"

if ! [ -f "$BIN_PATH" ]; then
    echo "[x] Binary does not exist!"
    exit 1
fi

if ! [ -f "$ICON" ]; then
    echo "[x] Icon file does not exist!"
    exit 1
fi

if [ -d "$APP_BUNDLE" ]; then
    echo "[*] Cleaning previous bundle..."
    rm -rf "$APP_BUNDLE"
fi

echo "[*] Creating .app structure..."
mkdir -p "$APP_BUNDLE/$MACOS"
mkdir -p "$APP_BUNDLE/$FRAMEWORKS"
mkdir -p "$APP_BUNDLE/$PLATFORMS"
mkdir -p "$APP_BUNDLE/$RESOURCES"

cp "$BIN_PATH" "$APP_BUNDLE/$MACOS/$APP_NAME"
cp "$ICON" "$APP_BUNDLE/$RESOURCES/icon.icns"

echo "[*] Generating Info.plist..."
cat > "$APP_BUNDLE/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
 "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>$APP_NAME</string>
  <key>CFBundleIdentifier</key>
  <string>org.e1z0.$APP_NAME</string>
  <key>CFBundleIconFile</key>
  <string>icon.icns</string>
  <key>CFBundleName</key>
  <string>$APP_NAME</string>
  <key>CFBundleShortVersionString</key>
  <string>$VERSION</string>
  <key>CFBundleVersion</key>
  <string>$BUILD</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>NSHighResolutionCapable</key>
  <true/>
  <key>CFBundleSupportedPlatforms</key>
  <array>
    <string>MacOSX</string>
  </array>
  <key>LSApplicationCategoryType</key>
  <string>public.app-category.utilities</string>
  <key>LSMinimumSystemVersion</key>
  <string>10.11</string>
<!-- Hide icon from dock <key>LSUIElement</key>
  <true/>
-->
  <key>NSHumanReadableCopyright</key>
  <string>© 2025 e1z0. All rights reserved.</string>
</dict>
</plist>
EOF

echo "[*] Copying platform plugin..."
cp /osxcross/macports/pkgs/opt/local/libexec/qt5/plugins/platforms/libqcocoa.dylib "$APP_BUNDLE/$PLATFORMS/"

# Qt conf
cat > "$APP_BUNDLE/$RESOURCES/qt.conf" <<EOF
[Paths]
Plugins = PlugIns
EOF

# Declare once for recursion tracking
declare -A visited

copy_and_patch_deps() {
    local file="$1"
    local app_bin_relpath="@executable_path/../Frameworks"
    local QT_ROOT="/osxcross/macports/pkgs"

    [[ -z "$file" || ! -e "$file" ]] && return
    [[ -n "${visited["$file"]}" ]] && return
    visited["$file"]=1

    echo "[*] Scanning: $file"

    # Extract dependencies
    local deps
    deps=$($OTOOL -L "$file" | tail -n +2 | awk '{print $1}' | grep -E '\.dylib|\.framework')

    for dep in $deps; do
        [[ "$dep" == @* || "$dep" == /System/* || "$dep" == /usr/lib/* ]] && continue

        if [[ "$dep" == *".framework/"* ]]; then
            local fwname=$(basename "$dep" | cut -d. -f1)
            local dep_fixed="${dep/\/opt\/local/$QT_ROOT/opt/local}"
            local fwroot=$(echo "$dep_fixed" | sed -E "s|(.*${fwname}\.framework).*|\1|")
            local fwdest="$APP_BUNDLE/$FRAMEWORKS/${fwname}.framework"

            if [[ ! -d "$fwdest" ]]; then
                echo "  ? Copying framework: $fwname"
                cp -R "$fwroot" "$fwdest"
                chmod -R +w "$fwdest"
            fi

            local fwlib=""
            if [[ -f "$fwdest/Versions/5/$fwname" ]]; then
                fwlib="$fwdest/Versions/5/$fwname"
            elif [[ -f "$fwdest/$fwname" ]]; then
                fwlib="$fwdest/$fwname"
            else
                fwlib=$(find "$fwdest" -type f -name "$fwname" | head -n1)
            fi

            if [[ -f "$fwlib" ]]; then
                echo "  ? Patching $file (framework: $fwname)"
                $INSTALL_NAME_TOOL -change "$dep" "$app_bin_relpath/${fwname}.framework/$fwname" "$file"
                copy_and_patch_deps "$fwlib"
            else
                echo "  ⚠️  Framework lib not found: $fwdest"
            fi
        else
            local dep_fixed="${dep/\/opt\/local/$QT_ROOT/opt/local}"
            local dep_name=$(basename "$dep")
            local dep_dest="$APP_BUNDLE/$FRAMEWORKS/$dep_name"

            if [[ ! -f "$dep_dest" ]]; then
                echo "  ? Copying dylib: $dep_name"
                cp "$dep_fixed" "$dep_dest" 2>/dev/null || {
                    echo "  ⚠️  Skipping missing: $dep_fixed"
                    continue
                }
                chmod +w "$dep_dest"
            fi

            echo "  ? Patching $file (dylib: $dep_name)"
            $INSTALL_NAME_TOOL -change "$dep" "$app_bin_relpath/$dep_name" "$file"
            copy_and_patch_deps "$dep_dest"
        fi
    done
}

echo "[*] Patching binary and dependencies..."
copy_and_patch_deps "$APP_BUNDLE/$MACOS/$APP_NAME"

# additional fixes
mkdir "$APP_BUNDLE/$MACOS/scripts"
cp ./scripts/osahelper "$APP_BUNDLE/$MACOS/scripts/"
# frameworks
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtCore.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtDBus.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtGui.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtNetwork.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtNetwork.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtPrintSupport.framework $APP_BUNDLE/$FRAMEWORKS/
#cp -r /osxcross/macports/pkgs/opt/local/libexec/qt5/lib/QtWidgets.framework $APP_BUNDLE/$FRAMEWORKS/
# libraries
cp /osxcross/macports/pkgs/opt/local/lib/libdbus-1.3.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libbrotlicommon.1.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libbrotlidec.1.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libbz2.1.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libdouble-conversion.3.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libfreetype.6.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libglib-2.0.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libgraphite2.3.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libgthread-2.0.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libharfbuzz.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libiconv.2.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libicudata.76.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libicui18n.76.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libicuuc.76.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libintl.8.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libpcre2-16.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libpcre2-8.0.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libpng16.16.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libz.1.dylib $APP_BUNDLE/$FRAMEWORKS
#cp /osxcross/macports/pkgs/opt/local/lib/libzstd.1.dylib $APP_BUNDLE/$FRAMEWORKS


mkdir -p $APP_BUNDLE/Contents/PlugIns/styles
cp /osxcross/macports/pkgs/opt/local/libexec/qt5/plugins/styles/* $APP_BUNDLE/Contents/PlugIns/styles/
./scripts/macdeployqtfix.py $APP_BUNDLE/Contents/MacOS/Conan /osxcross/macports/pkgs/opt/local/libexec/qt5
install_name_tool \
  -change /opt/local/lib/libdbus-1.3.dylib \
          @executable_path/../Frameworks/libdbus-1.3.dylib \
  $APP_BUNDLE/Contents/Frameworks/QtDBus.framework/Versions/5/QtDBus

pushd $RELEASE_DIR
echo "[*] Creating ZIP package..."
if [ -f "$ZIP_NAME" ]; then
    echo "[*] Removing old zip..."
    rm -f "$ZIP_NAME"
fi
zip -r "$ZIP_NAME" "$BUNDLE_DIR" > /dev/null
rm -rf "$BUNDLE_DIR"
popd
echo "[*] App bundled successfully into $APP_BUNDLE"
