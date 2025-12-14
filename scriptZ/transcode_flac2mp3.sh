#!/bin/bash

# Define the ffmpeg binary path
FFMPEG="/opt/homebrew/bin/ffmpeg"

# Check if ffmpeg exists and is executable
if [ ! -x "$FFMPEG" ]; then
    echo "Error: ffmpeg binary not found or not executable at $FFMPEG"
    exit 1
fi

# Change directory to the script's location
cd "$(dirname "$0")"

# Find all FLAC files and process them
find . -type f -name "*.flac" | while read -r input_file; do
    # specific output file path (same directory, different extension)
    output_file="${input_file%.flac}.mp3"
    
    # Check if the output file already exists
    if [ -f "$output_file" ]; then
        echo "Skipping (already exists): $output_file"
        continue
    fi
    
    echo "Transcoding: $input_file -> $output_file"
    
    # Run ffmpeg with fixed 320k bitrate
    # -i: input file
    # -b:a 320k: audio bitrate
    # -y: overwrite output files (we checked existence above, but this strictly handles the case if we remove the check or want to force)
    # Actually, we checked existence, so -y isn't strictly needed unless we race condition, but -n (no overwrite) is safer if we want to rely on ffmpeg's check. 
    # Since we do our own check, I'll allow overwrite in the command but the logic above prevents it. 
    # To be cleaner, I will use -n in the command or just let it run.
    # I'll use -v error -stats to keep output clean but visible.
    
    "$FFMPEG" -y -i "$input_file" -b:a 320k -v error -stats "$output_file" < /dev/null
    
    if [ $? -eq 0 ]; then
        echo "Done: $output_file"
    else
        echo "Failed: $input_file"
    fi
done

echo "Batch transcoding complete."
