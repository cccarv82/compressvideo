#!/usr/bin/env python3
"""
Download Test Videos Script
--------------------------
This script downloads sample videos for testing the CompressVideo application.
All videos are from free and open sources.
"""

import os
import sys
import urllib.request
import shutil
from pathlib import Path
import hashlib

# Define video sources - these are royalty-free test videos
VIDEOS = [
    {
        "name": "car_detection.mp4",
        "url": "https://github.com/intel-iot-devkit/sample-videos/raw/master/car-detection.mp4",
        "description": "Car detection video (similar to animation/gaming)",
        "type": "animation",
        "size_mb": 2
    },
    {
        "name": "people_detection.mp4",
        "url": "https://github.com/intel-iot-devkit/sample-videos/raw/master/people-detection.mp4",
        "description": "People detection footage (similar to gaming)",
        "type": "gaming",
        "size_mb": 2
    },
    {
        "name": "nature_documentary.mp4",
        "url": "https://github.com/intel-iot-devkit/sample-videos/raw/master/face-demographics-walking.mp4",
        "description": "Nature/Documentary sample",
        "type": "documentary",
        "size_mb": 2
    },
    {
        "name": "screencast_sample.mp4",
        "url": "https://github.com/intel-iot-devkit/sample-videos/raw/master/classroom.mp4",
        "description": "Screencast/Classroom sample",
        "type": "screencast",
        "size_mb": 1
    }
]

def calculate_md5(filename):
    """Calculate MD5 hash of a file."""
    hash_md5 = hashlib.md5()
    with open(filename, "rb") as f:
        for chunk in iter(lambda: f.read(4096), b""):
            hash_md5.update(chunk)
    return hash_md5.hexdigest()

def download_file(url, dest_path, description):
    """Download a file with progress indication."""
    try:
        print(f"Downloading {description}...")
        
        # Add User-Agent to prevent 403 errors
        req = urllib.request.Request(
            url, 
            data=None,
            headers={
                'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36'
            }
        )
        
        with urllib.request.urlopen(req) as response, open(dest_path, 'wb') as out_file:
            file_size = int(response.info().get('Content-Length', 0))
            if file_size == 0:  # If content length unknown
                print("Unknown file size, downloading...")
                data = response.read()
                out_file.write(data)
                print(f"\nDownload complete: {dest_path}")
                return True
                
            block_size = 8192
            downloaded = 0
            
            while True:
                buffer = response.read(block_size)
                if not buffer:
                    break
                    
                downloaded += len(buffer)
                out_file.write(buffer)
                
                # Calculate and print progress
                if file_size > 0:
                    progress = int(50 * downloaded / file_size)
                    sys.stdout.write(f"\r[{'=' * progress}{' ' * (50-progress)}] {downloaded/1024/1024:.1f}/{file_size/1024/1024:.1f} MB")
                    sys.stdout.flush()
                else:
                    sys.stdout.write(f"\rDownloaded {downloaded/1024/1024:.1f} MB")
                    sys.stdout.flush()
                
        print(f"\nDownload complete: {dest_path}")
        return True
        
    except Exception as e:
        print(f"\nError downloading {url}: {e}")
        if os.path.exists(dest_path):
            os.remove(dest_path)
        return False

def main():
    """Main function to download test videos."""
    # Create data directory if it doesn't exist
    data_dir = Path("data")
    data_dir.mkdir(exist_ok=True)
    
    # Track success/failure
    success_count = 0
    
    for video in VIDEOS:
        dest_path = data_dir / video["name"]
        
        # Skip if file already exists (MD5 validation removed as links changed)
        if dest_path.exists():
            print(f"{video['name']} already exists, skipping...")
            success_count += 1
            continue
            
        # Download the file
        if download_file(video["url"], dest_path, video["description"]):
            success_count += 1
    
    print(f"\nDownloaded {success_count}/{len(VIDEOS)} test videos to the data/ directory")
    print("These videos can be used for testing the CompressVideo application")

if __name__ == "__main__":
    main() 