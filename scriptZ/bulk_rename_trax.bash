#!/bin/bash

# Check if an argument is provided
if [ $# -ne 1 ]; then
    echo "Usage: $0 <directory>"
    exit 1
fi

# Check if the provided argument is a directory
if [ ! -d "$1" ]; then
    echo "Error: '$1' is not a directory"
    exit 1
fi

# Loop through files in the directory
for file in "$1"/*; do
    # Extract filename without path
    filename=$(basename "$file")
    
    # Check if filename starts with digits followed by ' - '
    if [[ "$filename" =~ ^[0-9]+[[:space:]]+-[[:space:]](.*)$ ]]; then
        # Extract the part after ' - '
        new_filename="${BASH_REMATCH[1]}"
        
        # Move the file to the new filename
        mv "$file" "$1/$new_filename"
        
        # Output the renaming action
        echo "Renamed '$filename' to '$new_filename'"
    fi
done

