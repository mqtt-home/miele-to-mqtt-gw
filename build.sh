#!/bin/bash
set -e
cd "$(dirname "$0")"
cd app
npm run build
cd ..
docker compose build
