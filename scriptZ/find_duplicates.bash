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

# Input directory
root_dir="$1"

# Output file
output_file="dump_duplicates.txt"
> "$output_file"  # Clear the file if it exists

# Temporary file for storing filenames and their full paths
temp_file=$(mktemp)

# Collect all filenames and their paths
find "$root_dir" -type f -exec bash -c 'printf "%s|%s\n" "$(basename "$1")" "$1"' _ {} \; > "$temp_file"

# Debug: Print temp_file contents
echo "Collected files:"
cat "$temp_file"

# Function to find duplicates for a single filename
find_duplicates() {
    local filename="$1"

    # Search for all occurrences of the filename
    matches=$(grep -i -F "|$filename" "$temp_file" | cut -d'|' -f2)
    count=$(echo "$matches" | wc -l)

    # Debug: Print matches for the current file
    echo "Checking filename: $filename"
    echo "$matches"

    # If more than one occurrence is found, log the duplicates
    if [ "$count" -gt 1 ]; then
        echo "$matches" >> "$output_file"
    fi
}

export -f find_duplicates
export temp_file
export output_file

# Determine the number of processors to use (leave one out)
if command -v nproc &>/dev/null; then
    num_jobs=$(nproc --ignore=1)
else
    num_jobs=$(($(sysctl -n hw.ncpu) - 1))  # macOS alternative
fi

# Extract unique filenames and process them in parallel
cut -d'|' -f1 "$temp_file" | sort | uniq | parallel -j "$num_jobs" find_duplicates {}

# Clean up temporary file
rm -f "$temp_file"

echo "Search complete. Results saved to $output_file."

