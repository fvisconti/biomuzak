#!/bin/bash

# Multi-platform ffmpeg detection
if command -v ffmpeg >/dev/null 2>&1; then
    FFMPEG="ffmpeg"
elif [ -x "/opt/homebrew/bin/ffmpeg" ]; then
    FFMPEG="/opt/homebrew/bin/ffmpeg"
elif [ -x "/usr/local/bin/ffmpeg" ]; then
    FFMPEG="/usr/local/bin/ffmpeg"
elif [ -x "/usr/bin/ffmpeg" ]; then
    FFMPEG="/usr/bin/ffmpeg"
else
    echo "Error: ffmpeg binary not found in PATH or standard locations."
    exit 1
fi

# Determine number of CPU cores for parallel processing
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    NUM_CORES=$(nproc)
elif [[ "$OSTYPE" == "darwin"* ]]; then
    NUM_CORES=$(sysctl -n hw.ncpu)
else
    NUM_CORES=2 # Fallback
fi

# Directory to search (default to current directory if not provided)
SEARCH_DIR="${1:-.}"

if [ ! -d "$SEARCH_DIR" ]; then
    echo "Error: Directory $SEARCH_DIR does not exist."
    exit 1
fi

# Multi-platform absolute path resolution
SEARCH_DIR_ABS=$(cd "$SEARCH_DIR" && pwd)
echo "Searching for FLAC files in: $SEARCH_DIR_ABS"
echo "Using $NUM_CORES parallel processes."


# Transcoding function
transcode_file() {
    local input_file="$1"
    local ffmpeg_bin="$2"
    local output_file="${input_file%.flac}.mp3"

    if [ -f "$output_file" ]; then
        echo "Skipping (already exists): $output_file"
        return
    fi

    echo "Transcoding: $input_file -> $output_file"
    
    # Run ffmpeg with fixed 320k bitrate
    "$ffmpeg_bin" -y -i "$input_file" -b:a 320k -v error -stats "$output_file" < /dev/null
    
    if [ $? -eq 0 ]; then
        echo "Done: $output_file"
    else
        echo "Failed: $input_file"
    fi
}

# Export function and variables for use in subshells (bash only)
export -f transcode_file
export FFMPEG

# Find all FLAC files and process them in parallel
# -print0 and xargs -0 handle filenames with spaces/special characters
# we pass the function and ffmpeg path via arguments to the subshell for safety
find "$SEARCH_DIR" -type f -name "*.flac" -print0 | xargs -0 -P "$NUM_CORES" -n 1 -I {} bash -c 'transcode_file "$1" "$2"' _ {} "$FFMPEG"


echo "Batch transcoding complete."

