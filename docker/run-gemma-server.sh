#!/bin/bash

echo "Starting Gemma 4 sidecar on port 8081"

# Start the llama.cpp server
exec ./server -m "/models/gemma-4-26b-q8_0/gguf-file.gguf" -c 4096 --host 0.0.0.0 --port 8081 -t 8 --no-mmap --log-disable