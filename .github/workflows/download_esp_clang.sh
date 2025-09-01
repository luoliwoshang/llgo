#!/bin/bash
set -e

ESP_CLANG_VERSION="19.1.2_20250830"
BASE_URL="https://github.com/goplus/espressif-llvm-project-prebuilt/releases/download/${ESP_CLANG_VERSION}"

get_filename() {
    local platform="$1"
    case "${platform}" in
        "darwin-amd64")
            echo "clang-esp-${ESP_CLANG_VERSION}-x86_64-apple-darwin.tar.xz"
            ;;
        "darwin-arm64")
            echo "clang-esp-${ESP_CLANG_VERSION}-aarch64-apple-darwin.tar.xz"
            ;;
        "linux-amd64")
            echo "clang-esp-${ESP_CLANG_VERSION}-x86_64-linux-gnu.tar.xz"
            ;;
        "linux-arm64")
            echo "clang-esp-${ESP_CLANG_VERSION}-aarch64-linux-gnu.tar.xz"
            ;;
        *)
            echo "Error: Unknown platform ${platform}" >&2
            exit 1
            ;;
    esac
}

download_and_extract() {
    local platform="$1"
    local filename=$(get_filename "${platform}")
    local download_url="${BASE_URL}/${filename}"
    
    echo "Downloading ESP Clang for ${platform}..."
    echo "  URL: ${download_url}"
    
    mkdir -p "crosscompile/clang-${platform}"
    curl -fsSL "${download_url}" | tar -xJ -C "crosscompile/clang-${platform}" --strip-components=1
    
    if [[ ! -f "crosscompile/clang-${platform}/bin/clang++" ]]; then
        echo "Error: clang++ not found in ${platform} toolchain"
        exit 1
    fi
    
    echo "${platform} toolchain ready"
}

mkdir -p crosscompile
echo "Downloading ESP Clang toolchains version ${ESP_CLANG_VERSION}..."

for platform in "darwin-amd64" "darwin-arm64" "linux-amd64" "linux-arm64"; do
    download_and_extract "${platform}" &
done

wait

echo ""
echo "Directory structure:"
find crosscompile -name "clang++" -type f
echo "Total size:"
du -sh crosscompile/