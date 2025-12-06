#!/bin/bash
# Sync matching subfolders between two paths (local or remote).
# Preserves attributes and uses 'rsync -E' for extended attributes (macOS friendly).

set -o errexit
set -o nounset
set -o pipefail

# ANSI color codes
G="\033[1;32m" # Green
C="\033[1;36m" # Cyan
R="\033[1;31m" # Red
nc="\033[0m"   # No Color

log_info() {
    printf "[%s] ${G}INFO${nc}: %s\n" "$(date '+%H:%M:%S')" "$*"
}

log_error() {
    printf "[%s] ${R}ERROR${nc}: %s\n" "$(date '+%H:%M:%S')" "$*" >&2
}

# Usage help
usage() {
    cat <<EOF
Usage: $0 [-p ssh_port] <source_path> <target_path>

Syncs ONLY subdirectories that exist in both source and target.
Preserves modification times and extended attributes (-E).

Examples:
  $0 /local/src /local/dst
  $0 /local/src user@host:/remote/dst
  $0 user@host:/remote/src /local/dst

Options:
  -p <port>   Specify SSH port
EOF
    exit 1
}

# Parse options
SSH_PORT=""
SSH_OPTS=()
RSYNC_SSH="ssh"

while getopts "p:" opt; do
    case "$opt" in
        p)
            SSH_PORT="$OPTARG"
            SSH_OPTS+=("-p" "$SSH_PORT")
            RSYNC_SSH="ssh -p $SSH_PORT"
            ;;
        *) usage ;;
    esac
done
shift $((OPTIND -1))

if [ "$#" -ne 2 ]; then
    usage
fi

SOURCE="$1"
TARGET="$2"

# Parse source location
if [[ "$SOURCE" == *":"* ]]; then
    SOURCE_HOST="${SOURCE%%:*}"
    SOURCE_PATH="${SOURCE#*:}"
else
    SOURCE_HOST=""
    SOURCE_PATH="$SOURCE"
fi

# Parse target location
if [[ "$TARGET" == *":"* ]]; then
    TARGET_HOST="${TARGET%%:*}"
    TARGET_PATH="${TARGET#*:}"
else
    TARGET_HOST=""
    TARGET_PATH="$TARGET"
fi

# Helper to list subdirectories
# Arguments: host (empty for local), path
get_subdirs() {
    local host="$1"
    local path="$2"
    
    if [ -n "$host" ]; then
        # Remote listing
        ssh ${SSH_OPTS[@]+"${SSH_OPTS[@]}"} "$host" "find \"$path\" -mindepth 1 -maxdepth 1 -type d -exec basename {} \;" 2>/dev/null || true
    else
        # Local listing
        if [ ! -d "$path" ]; then
             log_error "Directory '$path' not found."
             return 1
        fi
        find "$path" -mindepth 1 -maxdepth 1 -type d -exec basename {} \; 2>/dev/null || true
    fi
}

log_info "Scanning source: ${SOURCE_HOST:-(local)}:$SOURCE_PATH"
SOURCE_SUBDIRS=$(get_subdirs "$SOURCE_HOST" "$SOURCE_PATH")

log_info "Scanning target: ${TARGET_HOST:-(local)}:$TARGET_PATH"
TARGET_SUBDIRS=$(get_subdirs "$TARGET_HOST" "$TARGET_PATH")

# Find intersection
MATCHING_SUBDIRS=$(comm -12 <(echo "$SOURCE_SUBDIRS" | sort) <(echo "$TARGET_SUBDIRS" | sort))

if [ -z "$MATCHING_SUBDIRS" ]; then
    log_info "No matching subdirectories found to sync."
    exit 0
fi

# Confirm with user
echo
printf "Found matching subdirectories:\n"
printf "${C}%s${nc}\n" "$MATCHING_SUBDIRS"
echo
read -p "Proceed with sync? (y/n): " -n 1 -r REPLY
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Operation cancelled."
    exit 0
fi

# Sync Loop
while read -r SUBDIR; do
    if [ -z "$SUBDIR" ]; then continue; fi

    # Construct full paths
    if [ -n "$SOURCE_HOST" ]; then SRC_FULL="$SOURCE_HOST:$SOURCE_PATH/$SUBDIR/"; else SRC_FULL="$SOURCE_PATH/$SUBDIR/"; fi
    if [ -n "$TARGET_HOST" ]; then DST_FULL="$TARGET_HOST:$TARGET_PATH/$SUBDIR/"; else DST_FULL="$TARGET_PATH/$SUBDIR/"; fi

    log_info "Syncing ${C}$SUBDIR${nc}..."
    
    # rsync flags: 
    # -a: archive mode (recursive, preserves owner, group, permissions, times)
    # -v: verbose
    # -z: compress
    # --ignore-existing: skip updating files that exist on receiver
    # -E: Removed because it causes "unknown option" on non-macOS remotes.
    
    rsync -e "$RSYNC_SSH" -avz --ignore-existing "$SRC_FULL" "$DST_FULL"

done <<< "$MATCHING_SUBDIRS"

log_info "Sync completed."
