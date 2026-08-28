#!/bin/sh
set -eu

usage() {
  cat <<'EOF'
Usage: package-release.sh --version <version> --build <number> [options]

Builds the Universal 2 SKM.app, signs it with Developer ID, notarizes the app
and DMG, staples the tickets, and writes release artifacts.

Options:
  --output <directory>     Artifact directory (default: dist/macos)
  --preview                Build ad-hoc signed, unnotarized preview artifacts
  --skip-notarization      Sign but do not submit to Apple's notary service
  --help                   Show this help

Production environment:
  SKM_RELEASE_ENV_FILE     Optional local env file (default: macos/.release.env.local)
  SKM_SIGNING_IDENTITY     Exact Developer ID Application identity
  SKM_NOTARY_KEY_PATH      App Store Connect API private key (.p8)
  SKM_NOTARY_KEY_ID        App Store Connect API key ID
  SKM_NOTARY_ISSUER_ID     App Store Connect API issuer ID (Team API key)
  SKM_SPARKLE_PUBLIC_KEY   Sparkle EdDSA public key embedded in the App
  SKM_SPARKLE_PRIVATE_KEY_PATH  Sparkle EdDSA private key file used for appcast signing
EOF
}

SCRIPT_DIRECTORY="$(CDPATH= cd -- "$(dirname "$0")" && pwd)"
REPOSITORY_ROOT="$(cd "$SCRIPT_DIRECTORY/../.." && pwd)"
RELEASE_ENV_FILE="${SKM_RELEASE_ENV_FILE:-$REPOSITORY_ROOT/macos/.release.env.local}"

if [ -f "$RELEASE_ENV_FILE" ]; then
  # This file is intentionally local and gitignored because it can reference
  # signing credentials. Environment variables can take precedence when the
  # file uses the provided ${VARIABLE:-value} pattern.
  # shellcheck disable=SC1090
  . "$RELEASE_ENV_FILE"
fi

VERSION=""
BUILD_NUMBER=""
OUTPUT_DIRECTORY="$REPOSITORY_ROOT/dist/macos"
PREVIEW=0
SKIP_NOTARIZATION=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      [ "$#" -ge 2 ] || { echo "error: --version requires a value" >&2; exit 2; }
      VERSION="${2#v}"
      shift 2
      ;;
    --build)
      [ "$#" -ge 2 ] || { echo "error: --build requires a value" >&2; exit 2; }
      BUILD_NUMBER="$2"
      shift 2
      ;;
    --output)
      [ "$#" -ge 2 ] || { echo "error: --output requires a value" >&2; exit 2; }
      OUTPUT_DIRECTORY="$2"
      shift 2
      ;;
    --preview)
      PREVIEW=1
      SKIP_NOTARIZATION=1
      shift
      ;;
    --skip-notarization)
      SKIP_NOTARIZATION=1
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown option $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

case "$VERSION" in
  [0-9]*.[0-9]*.[0-9]*) ;;
  *) echo "error: --version must be a semantic version such as 0.5.3" >&2; exit 2 ;;
esac

case "$BUILD_NUMBER" in
  ''|*[!0-9]*) echo "error: --build must be a positive integer" >&2; exit 2 ;;
  0) echo "error: --build must be greater than zero" >&2; exit 2 ;;
esac

for command_name in xcodebuild lipo ditto hdiutil plutil shasum codesign; do
  command -v "$command_name" >/dev/null 2>&1 || {
    echo "error: required command not found: $command_name" >&2
    exit 1
  }
done

if [ "$PREVIEW" -eq 0 ]; then
  : "${SKM_SIGNING_IDENTITY:?error: SKM_SIGNING_IDENTITY is required for a production package}"
  : "${SKM_SPARKLE_PUBLIC_KEY:?error: SKM_SPARKLE_PUBLIC_KEY is required for a production package}"
  : "${SKM_SPARKLE_PRIVATE_KEY_PATH:?error: SKM_SPARKLE_PRIVATE_KEY_PATH is required for a production package}"
  [ -f "$SKM_SPARKLE_PRIVATE_KEY_PATH" ] || {
    echo "error: Sparkle private key does not exist: $SKM_SPARKLE_PRIVATE_KEY_PATH" >&2
    exit 1
  }
  case "$SKM_SIGNING_IDENTITY" in
    Developer\ ID\ Application:*) ;;
    *) echo "error: SKM_SIGNING_IDENTITY must be a Developer ID Application identity" >&2; exit 1 ;;
  esac
fi

if [ "$SKIP_NOTARIZATION" -eq 0 ]; then
  : "${SKM_NOTARY_KEY_PATH:?error: SKM_NOTARY_KEY_PATH is required}"
  : "${SKM_NOTARY_KEY_ID:?error: SKM_NOTARY_KEY_ID is required}"
  : "${SKM_NOTARY_ISSUER_ID:?error: SKM_NOTARY_ISSUER_ID is required}"
  [ -f "$SKM_NOTARY_KEY_PATH" ] || {
    echo "error: notary API key does not exist: $SKM_NOTARY_KEY_PATH" >&2
    exit 1
  }
fi

mkdir -p "$OUTPUT_DIRECTORY"
OUTPUT_DIRECTORY="$(cd "$OUTPUT_DIRECTORY" && pwd)"
TEMPORARY_DIRECTORY="$(mktemp -d "${TMPDIR:-/tmp}/skm-macos-release.XXXXXX")"

cleanup() {
  rm -rf "$TEMPORARY_DIRECTORY"
}
trap cleanup EXIT HUP INT TERM

DERIVED_DATA="$TEMPORARY_DIRECTORY/DerivedData"
APP_PATH="$DERIVED_DATA/Build/Products/Release/SKM.app"
APP_EXECUTABLE="$APP_PATH/Contents/MacOS/SKM"
CORE_EXECUTABLE="$APP_PATH/Contents/Resources/skm-core"
SPARKLE_FRAMEWORK="$APP_PATH/Contents/Frameworks/Sparkle.framework"
ENGLISH_LOCALIZATION="$APP_PATH/Contents/Resources/en.lproj/Localizable.strings"
EXPECTED_BUNDLE_ID="com.zzzzzyijie.skm"
ZIP_PATH="$OUTPUT_DIRECTORY/SKM-$VERSION-universal.zip"
DMG_PATH="$OUTPUT_DIRECTORY/SKM-$VERSION-universal.dmg"
CHECKSUM_PATH="$OUTPUT_DIRECTORY/SKM-$VERSION-checksums.txt"
APPCAST_PATH="$OUTPUT_DIRECTORY/appcast.xml"

rm -f "$ZIP_PATH" "$DMG_PATH" "$CHECKSUM_PATH" "$APPCAST_PATH"

echo "Building SKM $VERSION ($BUILD_NUMBER) for arm64 and x86_64..."
xcodebuild \
  -quiet \
  -workspace "$REPOSITORY_ROOT/macos/SKM.xcworkspace" \
  -scheme SKM \
  -configuration Release \
  -destination "generic/platform=macOS" \
  -derivedDataPath "$DERIVED_DATA" \
  ARCHS="arm64 x86_64" \
  ONLY_ACTIVE_ARCH=NO \
  MARKETING_VERSION="$VERSION" \
  CURRENT_PROJECT_VERSION="$BUILD_NUMBER" \
  SKM_SPARKLE_PUBLIC_KEY="${SKM_SPARKLE_PUBLIC_KEY:-}" \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_INJECT_BASE_ENTITLEMENTS=NO \
  build

[ -d "$APP_PATH" ] || { echo "error: Xcode did not produce SKM.app" >&2; exit 1; }
[ -x "$CORE_EXECUTABLE" ] || { echo "error: SKM.app does not contain executable skm-core" >&2; exit 1; }
[ -d "$SPARKLE_FRAMEWORK" ] || { echo "error: SKM.app does not contain Sparkle.framework" >&2; exit 1; }
[ -f "$ENGLISH_LOCALIZATION" ] || { echo "error: SKM.app does not contain the English localization" >&2; exit 1; }

assert_universal() {
  binary_path="$1"
  architectures="$(lipo -archs "$binary_path")"
  case " $architectures " in *" arm64 "*) ;; *) echo "error: $binary_path is missing arm64" >&2; exit 1 ;; esac
  case " $architectures " in *" x86_64 "*) ;; *) echo "error: $binary_path is missing x86_64" >&2; exit 1 ;; esac
}

assert_universal "$APP_EXECUTABLE"
assert_universal "$CORE_EXECUTABLE"

APP_VERSION="$(plutil -extract CFBundleShortVersionString raw "$APP_PATH/Contents/Info.plist")"
APP_BUILD="$(plutil -extract CFBundleVersion raw "$APP_PATH/Contents/Info.plist")"
APP_BUNDLE_ID="$(plutil -extract CFBundleIdentifier raw "$APP_PATH/Contents/Info.plist")"
DEVELOPMENT_REGION="$(plutil -extract CFBundleDevelopmentRegion raw "$APP_PATH/Contents/Info.plist")"
CORE_VERSION="$("$CORE_EXECUTABLE" version)"
[ "$APP_VERSION" = "$VERSION" ] || { echo "error: App version is $APP_VERSION, expected $VERSION" >&2; exit 1; }
[ "$APP_BUILD" = "$BUILD_NUMBER" ] || { echo "error: App build is $APP_BUILD, expected $BUILD_NUMBER" >&2; exit 1; }
[ "$APP_BUNDLE_ID" = "$EXPECTED_BUNDLE_ID" ] || { echo "error: App Bundle ID is $APP_BUNDLE_ID, expected $EXPECTED_BUNDLE_ID" >&2; exit 1; }
[ "$DEVELOPMENT_REGION" = "zh-Hans" ] || { echo "error: App development region is $DEVELOPMENT_REGION, expected zh-Hans" >&2; exit 1; }
[ "$CORE_VERSION" = "$VERSION" ] || { echo "error: Core version is $CORE_VERSION, expected $VERSION" >&2; exit 1; }

sign_sparkle_nested_code() {
  signing_identity="$1"
  timestamp_option="$2"
  sparkle_version="$SPARKLE_FRAMEWORK/Versions/B"
  for nested_bundle in \
    "$sparkle_version/XPCServices/Downloader.xpc" \
    "$sparkle_version/XPCServices/Installer.xpc" \
    "$sparkle_version/Updater.app"; do
    codesign --force --options runtime "$timestamp_option" --preserve-metadata=entitlements --sign "$signing_identity" "$nested_bundle"
  done
  codesign --force --options runtime "$timestamp_option" --sign "$signing_identity" "$SPARKLE_FRAMEWORK"
}

if [ "$PREVIEW" -eq 0 ]; then
  echo "Signing Sparkle, bundled Core, and SKM.app..."
  sign_sparkle_nested_code "$SKM_SIGNING_IDENTITY" --timestamp
  codesign --force --options runtime --timestamp --sign "$SKM_SIGNING_IDENTITY" "$CORE_EXECUTABLE"
  codesign --force --options runtime --timestamp --sign "$SKM_SIGNING_IDENTITY" "$APP_PATH"
  codesign --verify --deep --strict --verbose=2 "$APP_PATH"
else
  echo "Ad-hoc signing Sparkle, bundled Core, and SKM.app for local preview..."
  sign_sparkle_nested_code - --timestamp=none
  codesign --force --options runtime --timestamp=none --sign - "$CORE_EXECUTABLE"
  codesign \
    --force \
    --options runtime \
    --timestamp=none \
    --entitlements "$REPOSITORY_ROOT/macos/SKMApp/SKMPreview.entitlements" \
    --sign - \
    "$APP_PATH"
  codesign --verify --deep --strict --verbose=2 "$APP_PATH"
  echo "warning: preview artifacts use ad-hoc signing and are not suitable for distribution" >&2
fi

notarize() {
  artifact_path="$1"
  xcrun notarytool submit "$artifact_path" \
    --key "$SKM_NOTARY_KEY_PATH" \
    --key-id "$SKM_NOTARY_KEY_ID" \
    --issuer "$SKM_NOTARY_ISSUER_ID" \
    --wait \
    --timeout 30m
}

if [ "$SKIP_NOTARIZATION" -eq 0 ]; then
  echo "Submitting SKM.app for notarization..."
  SUBMISSION_ZIP="$TEMPORARY_DIRECTORY/SKM-notarization.zip"
  ditto -c -k --sequesterRsrc --keepParent "$APP_PATH" "$SUBMISSION_ZIP"
  notarize "$SUBMISSION_ZIP"
  xcrun stapler staple "$APP_PATH"
  xcrun stapler validate "$APP_PATH"
fi

echo "Creating ZIP and DMG..."
ditto -c -k --sequesterRsrc --keepParent "$APP_PATH" "$ZIP_PATH"

DMG_ROOT="$TEMPORARY_DIRECTORY/dmg-root"
mkdir -p "$DMG_ROOT"
ditto "$APP_PATH" "$DMG_ROOT/SKM.app"
ln -s /Applications "$DMG_ROOT/Applications"
hdiutil create \
  -volname "SKM $VERSION" \
  -srcfolder "$DMG_ROOT" \
  -ov \
  -format UDZO \
  "$DMG_PATH"

if [ "$PREVIEW" -eq 0 ]; then
  echo "Signing DMG..."
  codesign --force --timestamp --sign "$SKM_SIGNING_IDENTITY" "$DMG_PATH"
else
  echo "Ad-hoc signing DMG for local preview..."
  codesign --force --timestamp=none --sign - "$DMG_PATH"
fi
codesign --verify --verbose=2 "$DMG_PATH"

if [ "$SKIP_NOTARIZATION" -eq 0 ]; then
  echo "Submitting DMG for notarization..."
  notarize "$DMG_PATH"
  xcrun stapler staple "$DMG_PATH"
  xcrun stapler validate "$DMG_PATH"
  spctl --assess --type execute --verbose=2 "$APP_PATH"
  spctl --assess --type open --context context:primary-signature --verbose=2 "$DMG_PATH"
fi

if [ "$PREVIEW" -eq 0 ]; then
  SPARKLE_APPCAST_TOOL="$DERIVED_DATA/SourcePackages/artifacts/sparkle/Sparkle/bin/generate_appcast"
  [ -x "$SPARKLE_APPCAST_TOOL" ] || {
    echo "error: Sparkle generate_appcast tool was not found in Xcode package artifacts" >&2
    exit 1
  }
  APPCAST_SOURCE="$TEMPORARY_DIRECTORY/appcast-source"
  mkdir -p "$APPCAST_SOURCE"
  ditto "$ZIP_PATH" "$APPCAST_SOURCE/$(basename "$ZIP_PATH")"
  "$SPARKLE_APPCAST_TOOL" \
    --ed-key-file "$SKM_SPARKLE_PRIVATE_KEY_PATH" \
    --download-url-prefix "https://github.com/zzzzzyijie/skm/releases/download/v$VERSION/" \
    --link "https://github.com/zzzzzyijie/skm/releases/tag/v$VERSION" \
    --maximum-versions 3 \
    -o "$APPCAST_PATH" \
    "$APPCAST_SOURCE"
  [ -s "$APPCAST_PATH" ] || { echo "error: Sparkle appcast was not generated" >&2; exit 1; }
fi

(
  cd "$OUTPUT_DIRECTORY"
  if [ -f "$APPCAST_PATH" ]; then
    shasum -a 256 "$(basename "$ZIP_PATH")" "$(basename "$DMG_PATH")" "$(basename "$APPCAST_PATH")" > "$(basename "$CHECKSUM_PATH")"
  else
    shasum -a 256 "$(basename "$ZIP_PATH")" "$(basename "$DMG_PATH")" > "$(basename "$CHECKSUM_PATH")"
  fi
)

echo "Release artifacts:"
echo "  $ZIP_PATH"
echo "  $DMG_PATH"
echo "  $CHECKSUM_PATH"
[ ! -f "$APPCAST_PATH" ] || echo "  $APPCAST_PATH"
