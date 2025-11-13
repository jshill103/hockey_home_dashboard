#!/bin/sh
# Simple wrapper script to initialize and start the application

# Call the entrypoint script
/app/docker-entrypoint.sh

# Start the web server
exec /app/web_server -team "${TEAM_CODE}"

