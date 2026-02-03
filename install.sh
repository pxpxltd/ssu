#!/usr/bin/env bash
#
# Cross-platform installer for SSU (Smart Submodule Updater)
# Supports: Linux (Arch, Debian, Fedora, etc.), macOS
# Shells: bash, zsh, fish
#
# Usage:
#   ./install.sh          - Install ssu
#   ./install.sh --help   - Show help
#   ./install.sh --uninstall - Remove ssu
#

set -euo pipefail

# =============================================================================
# COLORS AND FORMATTING
# =============================================================================

if [[ -t 1 ]]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    CYAN='\033[0;36m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED='' GREEN='' YELLOW='' BLUE='' CYAN='' BOLD='' NC=''
fi

# =============================================================================
# CONFIGURATION
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SSU_SCRIPT="$SCRIPT_DIR/ssu"
SSU_NAME="ssu"

# Installation options (will be populated based on system)
INSTALL_OPTIONS=()

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

print_status() {
    local status="$1"
    local message="$2"

    case "$status" in
        INFO)     echo -e "${BLUE}[INFO]${NC} $message" ;;
        SUCCESS)  echo -e "${GREEN}[SUCCESS]${NC} $message" ;;
        WARNING)  echo -e "${YELLOW}[WARNING]${NC} $message" ;;
        ERROR)    echo -e "${RED}[ERROR]${NC} $message" ;;
        *)        echo "$message" ;;
    esac
}

print_header() {
    echo ""
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  SSU (Smart Submodule Updater) Installer${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
}

usage() {
    echo -e "${BOLD}SSU Installer${NC}

${BOLD}USAGE:${NC}
    $0 [OPTIONS]

${BOLD}OPTIONS:${NC}
    -h, --help          Show this help message
    --uninstall         Remove installed ssu

${BOLD}EXAMPLES:${NC}
    $0                  Install ssu (interactive)
    $0 --uninstall      Remove ssu from system
"
}

# Detect operating system and distribution
detect_os() {
    local os_type=""
    local os_distro=""

    # Detect OS type
    case "$(uname -s)" in
        Linux*)
            os_type="Linux"
            # Detect Linux distribution
            if [[ -f /etc/os-release ]]; then
                # shellcheck disable=SC1091
                . /etc/os-release
                os_distro="$ID"
            elif [[ -f /etc/arch-release ]]; then
                os_distro="arch"
            elif [[ -f /etc/debian_version ]]; then
                os_distro="debian"
            elif [[ -f /etc/redhat-release ]]; then
                os_distro="fedora"
            else
                os_distro="unknown"
            fi
            ;;
        Darwin*)
            os_type="macOS"
            os_distro="darwin"
            ;;
        *)
            os_type="Unknown"
            os_distro="unknown"
            ;;
    esac

    echo "$os_type|$os_distro"
}

# Detect user's shell and config file
detect_shell() {
    local shell_name=""
    local config_file=""

    # Get shell from environment
    if [[ -n "${SHELL:-}" ]]; then
        shell_name="$(basename "$SHELL")"
    else
        shell_name="bash"
    fi

    # Determine config file
    case "$shell_name" in
        bash)
            if [[ -f "$HOME/.bashrc" ]]; then
                config_file="$HOME/.bashrc"
            elif [[ -f "$HOME/.bash_profile" ]]; then
                config_file="$HOME/.bash_profile"
            else
                config_file="$HOME/.bashrc"
            fi
            ;;
        zsh)
            config_file="$HOME/.zshrc"
            ;;
        fish)
            config_file="$HOME/.config/fish/config.fish"
            ;;
        *)
            config_file="$HOME/.profile"
            ;;
    esac

    echo "$shell_name|$config_file"
}

# Check if a directory is in PATH
check_path() {
    local dir="$1"

    # Expand tilde
    dir="${dir/#\~/$HOME}"

    # Check if directory is in PATH
    if echo "$PATH" | tr ':' '\n' | grep -qx "$dir"; then
        return 0
    else
        return 1
    fi
}

# Build list of valid installation options
get_install_options() {
    INSTALL_OPTIONS=()

    # Option 1: ~/.local/bin (user-local, no sudo)
    local local_bin="$HOME/.local/bin"
    local local_in_path="no"
    if check_path "$local_bin"; then
        local_in_path="yes"
    fi

    if [[ -d "$local_bin" ]] || mkdir -p "$local_bin" 2>/dev/null; then
        INSTALL_OPTIONS+=("$local_bin|no|$local_in_path|User directory (no sudo required)")
    fi

    # Option 2: /usr/local/bin (system-wide, usually requires sudo on Linux)
    local usr_local="/usr/local/bin"
    local needs_sudo="yes"

    # On macOS, check if user owns /usr/local/bin
    if [[ "$(uname -s)" == "Darwin" ]] && [[ -w "$usr_local" ]]; then
        needs_sudo="no"
    fi

    if [[ -d "$usr_local" ]] || [[ -w "$(dirname "$usr_local")" ]] 2>/dev/null; then
        INSTALL_OPTIONS+=("$usr_local|$needs_sudo|yes|System-wide (standard location)")
    fi

    # Option 3: /usr/bin (system-wide, requires sudo)
    local usr_bin="/usr/bin"
    if [[ -d "$usr_bin" ]]; then
        INSTALL_OPTIONS+=("$usr_bin|yes|yes|System directory (requires sudo)")
    fi
}

# Add directory to shell PATH
add_to_path() {
    local dir="$1"
    local shell_name="$2"
    local config_file="$3"

    print_status INFO "Adding $dir to PATH in $config_file"

    # Create config file if it doesn't exist
    touch "$config_file"

    # Add to PATH based on shell type
    case "$shell_name" in
        fish)
            echo "" >> "$config_file"
            echo "# Added by SSU installer" >> "$config_file"
            echo "set -gx PATH $dir \$PATH" >> "$config_file"
            ;;
        *)
            echo "" >> "$config_file"
            echo "# Added by SSU installer" >> "$config_file"
            echo "export PATH=\"$dir:\$PATH\"" >> "$config_file"
            ;;
    esac

    print_status SUCCESS "Added to PATH. Run 'source $config_file' or restart your shell."
}

# Create symlink
create_symlink() {
    local target="$1"
    local needs_sudo="$2"

    # Check if target already exists
    if [[ -e "$target" ]] || [[ -L "$target" ]]; then
        echo ""
        read -rp "$(echo -e "${YELLOW}[WARNING]${NC} $target already exists. Overwrite? [y/N]: ")" confirm
        case "$confirm" in
            y|Y|yes|Yes)
                if [[ "$needs_sudo" == "yes" ]]; then
                    sudo rm -f "$target"
                else
                    rm -f "$target"
                fi
                ;;
            *)
                print_status ERROR "Installation cancelled."
                exit 1
                ;;
        esac
    fi

    # Create parent directory if needed
    local target_dir
    target_dir="$(dirname "$target")"
    if [[ ! -d "$target_dir" ]]; then
        if [[ "$needs_sudo" == "yes" ]]; then
            sudo mkdir -p "$target_dir"
        else
            mkdir -p "$target_dir"
        fi
    fi

    # Create symlink
    print_status INFO "Creating symlink: $target -> $SSU_SCRIPT"
    if [[ "$needs_sudo" == "yes" ]]; then
        sudo ln -sf "$SSU_SCRIPT" "$target"
    else
        ln -sf "$SSU_SCRIPT" "$target"
    fi

    # Verify
    if [[ -L "$target" ]]; then
        return 0
    else
        return 1
    fi
}

# Print installation menu
print_menu() {
    echo -e "${BOLD}Installation Options:${NC}"
    echo ""

    local i=1
    for option in "${INSTALL_OPTIONS[@]}"; do
        IFS='|' read -r path needs_sudo in_path description <<< "$option"

        local sudo_text=""
        if [[ "$needs_sudo" == "yes" ]]; then
            sudo_text=" ${YELLOW}(requires sudo)${NC}"
        fi

        local path_text=""
        if [[ "$in_path" == "yes" ]]; then
            path_text=" ${GREEN}✓ in PATH${NC}"
        else
            path_text=" ${YELLOW}⚠ not in PATH${NC}"
        fi

        echo -e "  ${BOLD}$i)${NC} $path$sudo_text$path_text"
        echo -e "     $description"
        echo ""
        ((i++))
    done
}

# Verify installation
verify_installation() {
    local target="$1"

    # Check if ssu is available
    if command -v "$SSU_NAME" >/dev/null 2>&1; then
        local installed_path
        installed_path="$(command -v "$SSU_NAME")"
        print_status SUCCESS "Installation verified: $installed_path"

        # Show version
        echo ""
        echo -e "${BOLD}Running 'ssu --help' to verify:${NC}"
        echo ""
        "$SSU_NAME" --help | head -5
        return 0
    else
        print_status WARNING "Installation complete but 'ssu' not found in PATH"
        print_status INFO "You may need to restart your shell or run: export PATH=\"$target:\$PATH\""
        return 1
    fi
}

# Search for installed ssu
find_installed_ssu() {
    local common_locations=(
        "$HOME/.local/bin/$SSU_NAME"
        "/usr/local/bin/$SSU_NAME"
        "/usr/bin/$SSU_NAME"
    )

    for location in "${common_locations[@]}"; do
        if [[ -L "$location" ]] || [[ -f "$location" ]]; then
            echo "$location"
            return 0
        fi
    done

    # Try using command -v
    if command -v "$SSU_NAME" >/dev/null 2>&1; then
        command -v "$SSU_NAME"
        return 0
    fi

    return 1
}

# Uninstall ssu
uninstall() {
    print_header
    print_status INFO "Searching for installed ssu..."
    echo ""

    local installed_path
    if installed_path=$(find_installed_ssu); then
        print_status INFO "Found: $installed_path"
        echo ""
        read -rp "$(echo -e "${YELLOW}Remove $installed_path? [y/N]: ${NC}")" confirm

        case "$confirm" in
            y|Y|yes|Yes)
                # Check if we need sudo
                if [[ -w "$installed_path" ]]; then
                    rm -f "$installed_path"
                else
                    sudo rm -f "$installed_path"
                fi

                if [[ ! -e "$installed_path" ]]; then
                    print_status SUCCESS "Successfully removed $installed_path"
                else
                    print_status ERROR "Failed to remove $installed_path"
                    exit 1
                fi
                ;;
            *)
                print_status INFO "Uninstall cancelled."
                exit 0
                ;;
        esac
    else
        print_status WARNING "No installed ssu found"
        print_status INFO "Checked locations:"
        echo "  - $HOME/.local/bin/$SSU_NAME"
        echo "  - /usr/local/bin/$SSU_NAME"
        echo "  - /usr/bin/$SSU_NAME"
        exit 1
    fi

    echo ""
    print_status SUCCESS "Uninstall complete"
}

# =============================================================================
# MAIN INSTALLATION
# =============================================================================

main() {
    # Parse arguments
    case "${1:-}" in
        -h|--help)
            usage
            exit 0
            ;;
        --uninstall)
            uninstall
            exit 0
            ;;
        "")
            # Continue with installation
            ;;
        *)
            print_status ERROR "Unknown option: $1"
            usage
            exit 1
            ;;
    esac

    # Print header
    print_header

    # Pre-flight checks
    if [[ ! -f "$SSU_SCRIPT" ]]; then
        print_status ERROR "SSU script not found at: $SSU_SCRIPT"
        print_status INFO "Make sure you're running this from the ssu directory"
        exit 1
    fi

    if [[ ! -x "$SSU_SCRIPT" ]]; then
        print_status WARNING "SSU script is not executable, fixing..."
        chmod +x "$SSU_SCRIPT"
    fi

    # Detect environment
    IFS='|' read -r os_type os_distro <<< "$(detect_os)"
    IFS='|' read -r shell_name config_file <<< "$(detect_shell)"

    print_status INFO "Detected OS: $os_type ($os_distro)"
    print_status INFO "Detected shell: $shell_name"
    print_status INFO "Config file: $config_file"
    echo ""

    # Check dependencies
    if ! command -v git >/dev/null 2>&1; then
        print_status WARNING "Git not found. SSU requires Git 2.0+."
        read -rp "Continue anyway? [y/N]: " confirm
        if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
            exit 1
        fi
    fi

    # Build installation options
    get_install_options

    if [[ ${#INSTALL_OPTIONS[@]} -eq 0 ]]; then
        print_status ERROR "No valid installation locations found"
        exit 1
    fi

    # Display menu
    print_menu

    # Get user selection
    echo -e "${BOLD}Select installation location [1-${#INSTALL_OPTIONS[@]}]:${NC}"
    read -rp "> " selection

    # Validate selection
    if ! [[ "$selection" =~ ^[0-9]+$ ]] || [[ "$selection" -lt 1 ]] || [[ "$selection" -gt ${#INSTALL_OPTIONS[@]} ]]; then
        print_status ERROR "Invalid selection"
        exit 1
    fi

    # Get selected option
    local selected_option="${INSTALL_OPTIONS[$((selection - 1))]}"
    IFS='|' read -r install_path needs_sudo in_path description <<< "$selected_option"

    echo ""
    print_status INFO "Selected: $install_path"

    # Confirm if sudo is needed
    if [[ "$needs_sudo" == "yes" ]]; then
        print_status WARNING "This installation requires sudo privileges"
        read -rp "Continue? [y/N]: " confirm
        if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
            print_status INFO "Installation cancelled"
            exit 0
        fi
    fi

    # Create symlink
    local target="$install_path/$SSU_NAME"
    if ! create_symlink "$target" "$needs_sudo"; then
        print_status ERROR "Failed to create symlink"
        exit 1
    fi

    print_status SUCCESS "Created symlink: $target"

    # Handle PATH if needed
    if [[ "$in_path" == "no" ]]; then
        echo ""
        print_status WARNING "$install_path is not in your PATH"
        read -rp "Add it to PATH automatically? [y/N]: " add_path

        if [[ "$add_path" == "y" ]] || [[ "$add_path" == "Y" ]]; then
            add_to_path "$install_path" "$shell_name" "$config_file"
        else
            echo ""
            print_status INFO "To add manually, add this to your $config_file:"
            if [[ "$shell_name" == "fish" ]]; then
                echo -e "${CYAN}set -gx PATH $install_path \$PATH${NC}"
            else
                echo -e "${CYAN}export PATH=\"$install_path:\$PATH\"${NC}"
            fi
        fi
    fi

    # Verify installation
    echo ""
    print_status INFO "Verifying installation..."
    echo ""

    if verify_installation "$(dirname "$target")"; then
        echo ""
        print_status SUCCESS "Installation complete!"
        echo ""
        echo -e "${BOLD}Next steps:${NC}"
        echo "  1. Run: ${CYAN}ssu --help${NC}"
        echo "  2. Navigate to a project with submodules"
        echo "  3. Run: ${CYAN}ssu --status${NC}"
        echo ""
    else
        echo ""
        print_status INFO "Installation complete, but you may need to:"
        echo "  1. Restart your shell or run: source $config_file"
        echo "  2. Then try: ${CYAN}ssu --help${NC}"
        echo ""
    fi
}

# =============================================================================
# ENTRY POINT
# =============================================================================

main "$@"
