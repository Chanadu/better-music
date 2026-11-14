#!/usr/bin/env bash
set -euo pipefail

repo=$(cd "$(dirname "$0")/../.." && pwd)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
project="$work/project"
config="$repo/deploy/ios/config.yml"
app="$work/Payload/Better Music.app"
sdk=$(xcrun --sdk iphoneos --show-sdk-path)
minimum=$(awk -F'"' '/minIOSVersion:/ {print $2}' "$config")
target="arm64-apple-ios${minimum}"

# Wails reads build/appicon.png relative to the generator's working directory.
mkdir -p "$work/build"
cp "$repo/frontend/static/pwa-icon-512.png" "$work/build/appicon.png"
(cd "$work" && wails3 ios xcode:gen -outdir "$project" -config "$config")

cd "$repo/client"
wails3 ios overlay:gen -out "$project/overlay.json" -config "$config"

GOOS=ios GOARCH=arm64 CGO_ENABLED=1 \
    CGO_CFLAGS="-isysroot $sdk -target $target -miphoneos-version-min=$minimum" \
    CGO_LDFLAGS="-isysroot $sdk -target $target" \
    go build -buildmode=c-archive -tags production -trimpath -buildvcs=false \
    -overlay "$project/overlay.json" -o "$work/client.a" .

mkdir -p "$app"
cp "$project/main/Info.plist" "$app/Info.plist"
executable=$(/usr/libexec/PlistBuddy -c 'Print :CFBundleExecutable' "$app/Info.plist")
xcrun --sdk iphoneos clang -target "$target" -isysroot "$sdk" \
    -framework Foundation -framework UIKit -framework WebKit \
    -framework Security -framework CoreFoundation -framework UniformTypeIdentifiers \
    -framework LocalAuthentication -framework UserNotifications -framework AVFoundation \
    -framework CoreLocation -framework CoreMotion -framework SystemConfiguration \
    -lresolv -o "$app/$executable" "$project/main/main.m" "-Wl,-force_load,$work/client.a"

xcrun --sdk iphoneos ibtool --compile "$app/LaunchScreen.storyboardc" "$project/main/LaunchScreen.storyboard"
xcrun --sdk iphoneos actool --compile "$app" --app-icon AppIcon \
    --platform iphoneos --minimum-deployment-target "$minimum" \
    --product-type com.apple.product-type.application --target-device iphone --target-device ipad \
    --output-partial-info-plist "$work/icon-info.plist" "$project/main/Assets.xcassets"
/usr/libexec/PlistBuddy -c "Merge $work/icon-info.plist" "$app/Info.plist"

# SideStore replaces this ad hoc signature with the user's development signature.
codesign --force --sign - "$app"
codesign --verify "$app"
xcrun vtool -show-build "$app/$executable"

output="$repo/client/packages/Better-Music.ipa"
mkdir -p "$(dirname "$output")"
(cd "$work" && zip -qry Better-Music.ipa Payload)
mv "$work/Better-Music.ipa" "$output"
unzip -tq "$output"
