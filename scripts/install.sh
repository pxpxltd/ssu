#!/usr/bin/env bash
set -euo pipefail

# SSU Install Script
# Usage: curl -sSL https://raw.githubusercontent.com/pxpxltd/ssu/master/scripts/install.sh | bash
#
# Environment variables:
#   VERSION   - specific version to install (e.g. v1.0.0). Default: latest release.
#   INSTALL_DIR - override install directory. Default: /usr/local/bin or ~/.local/bin.

OWNER="pxpxltd"
REPO="ssu"
GITHUB_API="https://api.github.com/repos/${OWNER}/${REPO}"
GITHUB_DOWNLOAD="https://github.com/${OWNER}/${REPO}/releases/download"

# --- Output helpers ---

has_color() {
    test -t 1
}

print_status() {
    if has_color; then
        printf '\033[0;32m%s\033[0m\n' "$1"
    else
        printf '%s\n' "$1"
    fi
}

print_warning() {
    if has_color; then
        printf '\033[0;33m%s\033[0m\n' "$1" >&2
    else
        printf '%s\n' "$1" >&2
    fi
}

print_error() {
    if has_color; then
        printf '\033[0;31mError: %s\033[0m\n' "$1" >&2
    else
        printf 'Error: %s\n' "$1" >&2
    fi
}

# --- Detection functions ---

detect_os() {
    local os
    os="$(uname -s | tr '[:upper:]' '[:lower:]')"
    case "$os" in
        linux)            echo "linux" ;;
        darwin)           echo "darwin" ;;
        freebsd)          echo "freebsd" ;;
        msys*|mingw*|cygwin*)  echo "windows" ;;
        *)
            print_error "unsupported OS: $os"
            exit 1
            ;;
    esac
}

detect_arch() {
    local arch
    arch="$(uname -m)"
    case "$arch" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)
            print_error "unsupported architecture: $arch"
            exit 1
            ;;
    esac
}

detect_install_dir() {
    # Allow explicit override
    if [ -n "${INSTALL_DIR:-}" ]; then
        mkdir -p "$INSTALL_DIR"
        echo "$INSTALL_DIR"
        return
    fi

    if [ -w /usr/local/bin ]; then
        echo "/usr/local/bin"
    else
        mkdir -p "${HOME}/.local/bin"
        print_warning "Note: installing to ~/.local/bin (add it to your PATH if not already)"
        echo "${HOME}/.local/bin"
    fi
}

# --- HTTP helpers ---

has_curl() {
    command -v curl >/dev/null 2>&1
}

has_wget() {
    command -v wget >/dev/null 2>&1
}

http_download() {
    local dest="$1"
    local url="$2"

    if has_curl; then
        if ! curl -sSL -o "$dest" "$url"; then
            print_error "failed to download: $url"
            return 1
        fi
    elif has_wget; then
        if ! wget -q -O "$dest" "$url"; then
            print_error "failed to download: $url"
            return 1
        fi
    else
        print_error "neither curl nor wget found; install one and try again"
        exit 1
    fi

    if [ ! -f "$dest" ]; then
        print_error "download did not produce a file: $dest"
        return 1
    fi
}

http_get() {
    local url="$1"

    if has_curl; then
        curl -sSL "$url"
    elif has_wget; then
        wget -q -O - "$url"
    else
        print_error "neither curl nor wget found; install one and try again"
        exit 1
    fi
}

# --- Version resolution ---

get_latest_version() {
    local response
    response="$(http_get "${GITHUB_API}/releases/latest")"

    local tag
    tag="$(printf '%s' "$response" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | head -1 | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')"

    if [ -z "$tag" ]; then
        print_error "failed to determine latest version from GitHub API"
        exit 1
    fi

    echo "$tag"
}

# --- Download ---

download() {
    local tmpdir="$1"
    local os="$2"
    local arch="$3"
    local version="$4"

    # Strip leading 'v' for filename (goreleaser uses version without v prefix)
    local ver="${version#v}"

    local ext="tar.gz"
    if [ "$os" = "windows" ]; then
        ext="zip"
    fi

    local filename="ssu_${ver}_${os}_${arch}.${ext}"
    local url="${GITHUB_DOWNLOAD}/${version}/${filename}"

    printf 'Downloading %s ... ' "$filename"
    http_download "${tmpdir}/${filename}" "$url"
    print_status "ok"

    printf 'Downloading checksums.txt ... '
    http_download "${tmpdir}/checksums.txt" "${GITHUB_DOWNLOAD}/${version}/checksums.txt"
    print_status "ok"

    # Return the filename via a global variable (POSIX-compatible)
    ARCHIVE_FILENAME="$filename"
}

# --- Checksum verification ---

compute_sha256() {
    local file="$1"

    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$file" | awk '{print $1}'
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$file" | awk '{print $1}'
    elif command -v openssl >/dev/null 2>&1; then
        openssl dgst -sha256 "$file" | awk '{print $NF}'
    else
        print_error "no SHA256 tool found (tried sha256sum, shasum, openssl)"
        exit 1
    fi
}

verify_checksum() {
    local tmpdir="$1"
    local filename="$2"

    printf 'Verifying checksum ... '

    local expected
    expected="$(grep "${filename}" "${tmpdir}/checksums.txt" | awk '{print $1}')"

    if [ -z "$expected" ]; then
        echo ""
        print_error "checksum not found for ${filename} in checksums.txt"
        exit 1
    fi

    local actual
    actual="$(compute_sha256 "${tmpdir}/${filename}")"

    if [ "$expected" != "$actual" ]; then
        echo ""
        print_error "checksum mismatch for ${filename}"
        print_error "  expected: ${expected}"
        print_error "  actual:   ${actual}"
        exit 1
    fi

    print_status "ok"
}

# --- Installation ---

install_binary() {
    local tmpdir="$1"
    local install_dir="$2"
    local filename="$3"

    local ext="${filename##*.}"

    printf 'Extracting binary ... '
    if [ "$ext" = "gz" ]; then
        # tar.gz archive
        tar -xzf "${tmpdir}/${filename}" -C "$tmpdir" ssu
    elif [ "$ext" = "zip" ]; then
        # zip archive (Windows)
        unzip -o "${tmpdir}/${filename}" ssu.exe -d "$tmpdir" >/dev/null
    fi
    print_status "ok"

    printf 'Installing to %s ... ' "$install_dir"
    if command -v install >/dev/null 2>&1; then
        install -m 755 "${tmpdir}/ssu" "${install_dir}/ssu"
    else
        cp "${tmpdir}/ssu" "${install_dir}/ssu"
        chmod 755 "${install_dir}/ssu"
    fi
    print_status "ok"
}

# --- Main ---

main() {
    local os arch install_dir version

    os="$(detect_os)"
    arch="$(detect_arch)"
    install_dir="$(detect_install_dir)"

    if [ -n "${VERSION:-}" ]; then
        version="$VERSION"
        # Ensure version has v prefix for download URL
        case "$version" in
            v*) ;;
            *)  version="v${version}" ;;
        esac
    else
        printf 'Fetching latest version ... '
        version="$(get_latest_version)"
        print_status "$version"
    fi

    local tmpdir
    tmpdir="$(mktemp -d)"
    trap "rm -rf \"$tmpdir\"" EXIT

    ARCHIVE_FILENAME=""
    download "$tmpdir" "$os" "$arch" "$version"

    verify_checksum "$tmpdir" "$ARCHIVE_FILENAME"
    install_binary "$tmpdir" "$install_dir" "$ARCHIVE_FILENAME"

    echo ""
    print_status "ssu ${version} installed to ${install_dir}/ssu"
}

main "$@"
