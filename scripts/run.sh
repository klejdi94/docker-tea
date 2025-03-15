#!/bin/bash

echo "Building Docker Tea application..."

# Build the application
go build -o docker-tea ./cmd/docker-tea

if [ $? -ne 0 ]; then
    echo "Failed to build Docker Tea application"
    exit 1
fi

echo "Starting Docker Tea application..."
echo ""
echo "Usage tips:"
echo "- Use TAB to switch between resources (Containers, Images, Volumes, Networks)"
echo "- When viewing a container, press 'm' to monitor resource usage in real-time"
echo "- Press '?' for help and to see all available keyboard shortcuts"
echo ""

# Make sure it's executable
chmod +x ./docker-tea

# Run the application
./docker-tea 