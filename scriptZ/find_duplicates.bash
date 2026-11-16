#!/bin/bash

# Check if correct number of arguments are provided
if [ $# -ne 2 ]; then
    echo "Usage: $0 <directory> <output_file>"
    exit 1
fi

# Input directory and output file
root_dir="$1"
output_file="$2"

# Check if the provided argument is a directory
if [ ! -d "$root_dir" ]; then
    echo "Error: '$root_dir' is not a directory"
    exit 1
fi

# Clear the output file if it exists
> "$output_file"

# Temporary file for storing filenames and their full paths
temp_file=$(mktemp)

echo "Collecting file list from $root_dir..."
# Collect all filenames and their paths efficiently
# Output format: basename|fullpath
find "$root_dir" -type f -exec bash -c '
    for file; do
        printf "%s|%s\n" "${file##*/}" "$file"
    done
' _ {} + > "$temp_file"

echo "Processing duplicates..."

# Use awk to find duplicates and group them on the same line
# Items are grouped by filename (case-insensitive)
# Separated by 4 empty spaces as requested
awk '
{
    # Robustly split into filename and path at the first pipe
    p = index($0, "|")
    if (p == 0) next
    fname = substr($0, 1, p-1)
    path = substr($0, p+1)
    
    fname_lower = tolower(fname)
    if (paths[fname_lower] == "") {
        paths[fname_lower] = path
    } else {
        paths[fname_lower] = paths[fname_lower] "    " path
    }
    count[fname_lower]++
}
END {
    for (f in count) {
        if (count[f] > 1) {
            print paths[f]
        }
    }
}
' "$temp_file" >> "$output_file"

# Clean up temporary file
rm -f "$temp_file"

echo "Search complete. Results saved to $output_file."

