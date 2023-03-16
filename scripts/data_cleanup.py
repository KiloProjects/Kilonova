import toml
import os
import subprocess
import re

# Written using ChatGPT...

max_size = 1024 * 1024 * 1024 # 1024 MB = 1 GB

with open("./config.toml", "r") as f:
    path: str = toml.load(f)["common"]["data_dir"] + "/subtests" # Remove only subtests

def get_directory_size(path):
    """Returns the size of a directory in bytes."""
    result = subprocess.run(["du", "-s", path], capture_output=True, text=True)
    output = result.stdout.strip().split("\t")[0]
    size_bytes = int(output) * 1024
    return size_bytes

def format_size(size_bytes):
    """Returns a human-readable representation of a size in bytes."""
    for unit in ['', 'K', 'M', 'G', 'T', 'P', 'E', 'Z']:
        if abs(size_bytes) < 1024.0:
            return f"{size_bytes:.2f}{unit}B"
        size_bytes /= 1024.0
    return f"{size_bytes:.2f}YB"

def get_numeric_value(filename):
    """Extracts the numeric value from a filename."""
    match = re.search(r"\d+", filename)
    if match:
        return int(match.group(0))
    else:
        return float('inf')

def delete_files(path):
    """Deletes files in a directory in alphabetical order until the directory size is under a set size."""
    while get_directory_size(path) > max_size:
        files = os.listdir(path)
        files = sorted(files, key=lambda x: get_numeric_value(os.path.splitext(os.path.basename(x))[0]))
        for file in files:
            file_path = os.path.join(path, file)
            if os.path.isfile(file_path):
                os.remove(file_path)
                print(f"Deleted file: {file_path}")
                if get_directory_size(path) <= max_size:
                    return


print(f"Disk usage for {path} before: {format_size(get_directory_size(path))}")
delete_files(path)
print(f"Disk usage for {path} after: {format_size(get_directory_size(path))}")

