#!/bin/bash

echo "Building Docker Tea application..."

# Save current directory and navigate to project root
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"
cd "$PROJECT_ROOT"

# Build the application
go build -o docker-tea ./cmd/docker-tea

if [ $? -ne 0 ]; then
    echo "Failed to build Docker Tea application"
    # Return to the original directory
    cd - > /dev/null
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

# Return to the original directory
cd - > /dev/null 