#!/bin/bash

# Enable safer shell behaviour and a small logging helper
set -o errexit
set -o nounset
set -o pipefail

log() {
    printf '%s %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*"
}

# Optional: enable shell tracing if SYNC_DEBUG=1 in environment
if [ "${SYNC_DEBUG:-0}" -eq 1 ] 2>/dev/null; then
    log "DEBUG: enabling shell trace"
    set -x
fi

# Parse optional SSH port and positional args
SSH_PORT=""
while getopts "p:" opt; do
    case "$opt" in
        p) SSH_PORT="$OPTARG" ;;
        *) 
           echo "Usage: $0 [-p ssh_port] <source> <target>"
           exit 1
           ;;
    esac
done
shift $((OPTIND -1))

# Ensure the correct number of positional arguments
if [ "$#" -ne 2 ]; then
    echo "Usage: $0 [-p ssh_port] <source> <target>"
    echo "Examples:"
    echo "  Local to Local:   $0 /local/source /local/target"
    echo "  Local to Remote:  $0 /local/source user@remote:/target"
    echo "  Remote to Local:  $0 user@remote:/source /local/target"
    echo "  Remote to Remote: $0 user@remote:/source user@remote:/target"
    echo "  With port:        $0 -p 2222 user@host:/source /local/target"
    exit 1
fi

SOURCE=$1
TARGET=$2

# Show confirmation prompt
# Using printf instead of echo for better escape sequence handling
printf "\nI'll synchronize the files from \033[1;36m%s\033[0m to \033[1;36m%s\033[0m\n" "$SOURCE" "$TARGET"
read -p "Proceed? (y/n): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Operation cancelled."
    exit 0
fi

# Validate rsync installation
if ! command -v rsync &> /dev/null; then
    echo "Error: rsync is not installed. Please install rsync and try again."
    exit 1
fi

# Determine if SOURCE is remote
if [[ "$SOURCE" == *":"* ]]; then
    SOURCE_HOST=$(echo "$SOURCE" | cut -d':' -f1)
    SOURCE_DIR=$(echo "$SOURCE" | cut -d':' -f2-)
    REMOTE_SOURCE=true
else
    SOURCE_DIR="$SOURCE"
    REMOTE_SOURCE=false
fi

# Determine if TARGET is remote
if [[ "$TARGET" == *":"* ]]; then
    TARGET_HOST=$(echo "$TARGET" | cut -d':' -f1)
    TARGET_DIR=$(echo "$TARGET" | cut -d':' -f2-)
    REMOTE_TARGET=true
else
    TARGET_DIR="$TARGET"
    REMOTE_TARGET=false
fi

# Configure SSH options and rsync -e string if a port was provided
SSH_OPTS=()
RSYNC_SSH="ssh"
if [ -n "${SSH_PORT:-}" ]; then
    SSH_OPTS+=("-p" "$SSH_PORT")
    RSYNC_SSH="ssh -p $SSH_PORT"
    log "Using custom SSH port: $SSH_PORT"
fi
# A short preview string for logs
SSH_PREVIEW="$RSYNC_SSH"

# Find subdirectories in source (local or remote)
if $REMOTE_SOURCE; then
    log "Running remote find on source: $SSH_PREVIEW $SOURCE_HOST find \"$SOURCE_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'"
    SOURCE_SUBDIRS=$(ssh "${SSH_OPTS[@]}" "$SOURCE_HOST" "find \"$SOURCE_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'" 2>&1) || {
        log "Error while listing source subdirs: $SOURCE_SUBDIRS"
        exit 1
    }
else
    log "Running local find on source: find \"$SOURCE_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'"
    SOURCE_SUBDIRS=$(find "$SOURCE_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>&1) || {
        log "Error while listing source subdirs: $SOURCE_SUBDIRS"
        exit 1
    }
fi

# Find subdirectories in target (local or remote)
if $REMOTE_TARGET; then
    log "Running remote find on target: $SSH_PREVIEW $TARGET_HOST find \"$TARGET_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'"
    TARGET_SUBDIRS=$(ssh "${SSH_OPTS[@]}" "$TARGET_HOST" "find \"$TARGET_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'" 2>&1) || {
        log "Error while listing target subdirs: $TARGET_SUBDIRS"
        exit 1
    }
else
    log "Running local find on target: find \"$TARGET_DIR\" -mindepth 1 -maxdepth 1 -type d -printf '%f\n'"
    TARGET_SUBDIRS=$(find "$TARGET_DIR" -mindepth 1 -maxdepth 1 -type d -printf '%f\n' 2>&1) || {
        log "Error while listing target subdirs: $TARGET_SUBDIRS"
        exit 1
    }
fi

# Find common subdirectories by name
COMMON_SUBDIRS=$(echo "$SOURCE_SUBDIRS" | grep -F -x -f <(echo "$TARGET_SUBDIRS"))

# Sync each subdirectory
echo "Starting sync..."
while read -r DIR_NAME; do
    if [ -z "$DIR_NAME" ]; then
        continue
    fi

    # Construct source and target paths
    if $REMOTE_SOURCE; then
        SOURCE_FULL="$SOURCE_HOST:$SOURCE_DIR/$DIR_NAME"
    else
        SOURCE_FULL="$SOURCE_DIR/$DIR_NAME"
    fi

    if $REMOTE_TARGET; then
        FINAL_TARGET="$TARGET_HOST:$TARGET_DIR/$DIR_NAME"
    else
        FINAL_TARGET="$TARGET_DIR/$DIR_NAME"
    fi

    # Print syncing info
    printf "Syncing: \033[1;32m%s\033[0m -> \033[1;36m%s\033[0m\n" "$SOURCE_FULL" "$FINAL_TARGET"

    # Perform the sync and capture output for new files
    # Modified to only include .flac files and exclude ._ files
    RSYNC_OUTPUT=$(mktemp)
    log "Running rsync for $DIR_NAME"
    log "Command preview: rsync -e \"$RSYNC_SSH\" -avz --times --itemize-changes -vv --progress --include='*.flac' --exclude='._*' --exclude='*' \"$SOURCE_FULL/\" \"$FINAL_TARGET/\""
    log "If this hangs, an SSH prompt may be waiting for a password or host key acceptance."

    if $REMOTE_SOURCE; then
        rsync -e "$RSYNC_SSH" -avz --times --itemize-changes -vv --progress \
              --include="*.flac" \
              --exclude="._*" \
              --exclude="*" \
              "$SOURCE_FULL/" "$FINAL_TARGET/" 2>&1 | tee "$RSYNC_OUTPUT"
    elif $REMOTE_TARGET; then
        rsync -e "$RSYNC_SSH" -avz --times --itemize-changes -vv --progress \
              --include="*.flac" \
              --exclude="._*" \
              --exclude="*" \
              "$SOURCE_FULL/" "$FINAL_TARGET/" 2>&1 | tee "$RSYNC_OUTPUT"
    else
        rsync -avz --times --itemize-changes -vv --progress \
              --include="*.flac" \
              --exclude="._*" \
              --exclude="*" \
              "$SOURCE_FULL/" "$FINAL_TARGET/" 2>&1 | tee "$RSYNC_OUTPUT"
    fi

    # Filter and display new files from rsync output
    echo "Newly synced files:"
    grep "^>f" "$RSYNC_OUTPUT" | awk '{print $2}'

    # Clean up temporary file
    rm -f "$RSYNC_OUTPUT"
done <<< "$COMMON_SUBDIRS"

echo "Sync completed."
