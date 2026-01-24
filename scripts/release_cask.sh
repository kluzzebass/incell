#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if [[ -f "$ROOT_DIR/.env" ]]; then
  # shellcheck disable=SC1091
  source "$ROOT_DIR/.env"
fi

APP_NAME="${APP_NAME:-Incell}"
BUNDLE_ID="${BUNDLE_ID:-com.kluzzebass.incell}"
SIGN_IDENTITY="${SIGN_IDENTITY:-}"
NOTARY_PROFILE="${NOTARY_PROFILE:-}"
GITHUB_REPO="${GITHUB_REPO:-}"
SKIP_NOTARIZE="${SKIP_NOTARIZE:-0}"

# Get version from git tag or default
VERSION="${GITHUB_REF_NAME:-v0.0.0}"
VERSION="${VERSION#v}"

if [[ -z "$SIGN_IDENTITY" ]]; then
  echo "SIGN_IDENTITY is required (e.g. 'Developer ID Application: Your Name (TEAMID)')." >&2
  exit 1
fi

if [[ -z "$GITHUB_REPO" ]]; then
  echo "GITHUB_REPO is required (e.g. 'kluzzebass/incell')." >&2
  exit 1
fi

if [[ "$SKIP_NOTARIZE" != "1" && -z "$NOTARY_PROFILE" ]]; then
  echo "NOTARY_PROFILE is required unless SKIP_NOTARIZE=1." >&2
  exit 1
fi

cd "$ROOT_DIR"

DIST_DIR="$ROOT_DIR/dist"
APP_DIR="$DIST_DIR/${APP_NAME}.app"
CONTENTS_DIR="$APP_DIR/Contents"
MACOS_DIR="$CONTENTS_DIR/MacOS"
RESOURCES_DIR="$CONTENTS_DIR/Resources"

echo "Creating app bundle structure..."
rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

echo "Building Go binary..."
CGO_ENABLED=1 go build -o "$MACOS_DIR/$APP_NAME" ./cmd/incell

echo "Creating icns from icon..."
ICONSET_DIR="$DIST_DIR/icon.iconset"
mkdir -p "$ICONSET_DIR"

# Create iconset from source PNG
sips -z 16 16     "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_16x16.png"
sips -z 32 32     "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_16x16@2x.png"
sips -z 32 32     "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_32x32.png"
sips -z 64 64     "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_32x32@2x.png"
sips -z 128 128   "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_128x128.png"
sips -z 256 256   "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_128x128@2x.png"
sips -z 256 256   "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_256x256.png"
sips -z 512 512   "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_256x256@2x.png"
sips -z 512 512   "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_512x512.png"
sips -z 1024 1024 "$ROOT_DIR/resources/icon.png" --out "$ICONSET_DIR/icon_512x512@2x.png"

iconutil -c icns "$ICONSET_DIR" -o "$RESOURCES_DIR/icon.icns"
rm -rf "$ICONSET_DIR"

echo "Creating Info.plist..."
cat > "$CONTENTS_DIR/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleExecutable</key>
    <string>${APP_NAME}</string>
    <key>CFBundleIconFile</key>
    <string>icon</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>NSSupportsAutomaticGraphicsSwitching</key>
    <true/>
</dict>
</plist>
EOF

echo "Signing app..."
codesign --deep --force --options runtime --sign "$SIGN_IDENTITY" "$APP_DIR"
codesign --verify --deep --strict "$APP_DIR"
spctl --assess --type execute --verbose "$APP_DIR" || true

ZIP_NAME="${APP_NAME}-${VERSION}.zip"
ZIP_PATH="$DIST_DIR/$ZIP_NAME"

echo "Packaging zip..."
rm -f "$ZIP_PATH"
ditto -c -k --keepParent "$APP_DIR" "$ZIP_PATH"

if [[ "$SKIP_NOTARIZE" != "1" ]]; then
  echo "Notarizing..."
  xcrun notarytool submit "$ZIP_PATH" --keychain-profile "$NOTARY_PROFILE" --wait
  echo "Stapling..."
  xcrun stapler staple "$APP_DIR"
fi

SHA256="$(shasum -a 256 "$ZIP_PATH" | awk '{print $1}')"

echo ""
echo "Release artifacts:"
echo "  App: $APP_DIR"
echo "  Zip: $ZIP_PATH"
echo "  Version: $VERSION"
echo "  SHA256: $SHA256"
echo ""

cat > "$DIST_DIR/cask.rb" <<EOF
cask "incell" do
  version "$VERSION"
  sha256 "$SHA256"

  url "https://github.com/$GITHUB_REPO/releases/download/v#{version}/$ZIP_NAME"
  name "Incell"
  desc "FreeCell solitaire"
  homepage "https://github.com/$GITHUB_REPO"

  app "${APP_NAME}.app"
end
EOF

echo "Cask snippet written to $DIST_DIR/cask.rb"
