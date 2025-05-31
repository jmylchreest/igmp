#!/bin/bash
set -e

zip_cmd=$(which zip)
if [ -z "$zip_cmd" ]; then
  echo "Error: zip command not found. Please install it."
  exit 1
fi

[[ ! -d dist/ ]] && mkdir -p dist

for dir_path in bin/*; do
  if [ ! -d "$dir_path" ]; then
    continue # Skip if not a directory
  fi

  dir_name=$(basename "$dir_path")

  # Split dir_name into OS and ARCH, assuming format "OS-ARCH"
  IFS='-' read -r platform arch <<< "$dir_name"

  if [ -z "$platform" ] || [ -z "$arch" ]; then
    echo "Warning: Skipping directory $dir_path, does not match OS-ARCH pattern."
    continue
  fi

  echo "Processing $platform/$arch from $dir_path..."

  # Find the binary file (igmpqd or igmpqd.exe)
  binary_file=""
  if [ -f "$dir_path/igmpqd" ]; then
    binary_file="$dir_path/igmpqd"
  elif [ -f "$dir_path/igmpqd.exe" ]; then
    binary_file="$dir_path/igmpqd.exe"
  else
    echo "Warning: No binary found in $dir_path for $platform/$arch."
    continue
  fi

  echo "Packaging $binary_file into dist/${platform}_${arch}.zip"
  # The -j option junks paths, storing only the file in the zip.
  # The -v option is for verbose output.
  "$zip_cmd" -v -j "dist/${platform}_${arch}.zip" "$binary_file"
done

echo "All binaries packaged."
