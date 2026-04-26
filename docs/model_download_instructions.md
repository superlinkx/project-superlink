# Gemma 4 Model Download Instructions

## Official Gemma 4 Model Sources

The Gemma 4 model is available from Google's official sources. Here are the options:

### Option 1: Google Cloud Vertex AI (Recommended for Production)
Google hosts Gemma models on Vertex AI:
- https://console.cloud.google.com/vertex-ai
- Search for "Gemma" in the model catalog
- Follow Google's documentation: https://cloud.google.com/vertex-ai/docs/generative-ai/model-reference/gemma

### Option 2: Hugging Face (GGUF Format)
For local deployment with llama.cpp:

1. **Download from Hugging Face**:
   ```bash
   # Install huggingface_hub if needed
   pip install huggingface_hub

   # Download the model
   git lfs install
   git clone https://huggingface.co/google/gemma-4-26b-it-GGUF
   cd gemma-4-26b-it-GGUF
   ```

2. **Convert to Q8_0 format** (if needed):
   ```bash
   # Use the llama.cpp convert script
   python3 -m llama_cpp.convert_model \
     --input_type huggingface \
     --output_type gguf \
     --quant_type q8_0 \
     gemma-4-26b-it-GGUF \
     gemma-4-26b-q8_0.gguf
   ```

### Option 3: Direct GGUF Download Links

The Q8_0 quantized version (recommended for balance of speed and quality):

- **GGUF Q8_0**: https://huggingface.co/google/gemma-4-26b-it-GGUF/resolve/main/gemma-4-26b-it-q8_0.gguf
  - File size: ~13-16GB
  - Recommended for most use cases

Alternative quantizations:
- Q4_K_M (smaller, faster but slightly less accurate): https://huggingface.co/google/gemma-4-26b-it-GGUF/resolve/main/gemma-4-26b-it-q4_k_m.gguf (~8GB)
- F16 (largest, most accurate): https://huggingface.co/google/gemma-4-26b-it-GGUF/resolve/main/gemma-4-26b-it-f16.gguf (~27GB)

## Setup Instructions for Superlink

Once downloaded:

1. Create the models directory:
   ```bash
   mkdir -p ./models/gemma-4-26b-q8_0
   ```

2. Copy the model file:
   ```bash
   # Replace with your actual download path
   cp /path/to/gemma-4-26b-it-q8_0.gguf ./models/gemma-4-26b-q8_0/gguf-file.gguf
   ```

3. Update docker-compose.yml to use the correct path if needed

## Alternative: Smaller Test Models

If you want to test without downloading the full Gemma 4 model, consider these smaller alternatives:

### TinyLlama (for testing)
```bash
wget https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v0.6-GGUF/resolve/main/tinyllama-1.1b-chat-v0.6.Q4_K_M.gguf
cp tinyllama-1.1b-chat-v0.6.Q4_K_M.gguf ./models/gemma-4-26b-q8_0/gguf-file.gguf
```

### Phi-3 (Medium size)
```bash
wget https://huggingface.co/microsoft/Phi-3-mini-4k-instruct-GGUF/resolve/main/phi-3-mini-4k-instruct-q4_k_m.gguf
cp phi-3-mini-4k-instruct-q4_k_m.gguf ./models/gemma-4-26b-q8_0/gguf-file.gguf
```

## Notes

1. **Storage Requirements**: Ensure you have at least 20GB free disk space for the Q8_0 model
2. **Download Time**: Expect several hours to download depending on your internet speed
3. **Verification**: After downloading, verify the file integrity using SHA hashes from Hugging Face
4. **Legal Compliance**: Ensure compliance with Google's Gemma terms of service

## Troubleshooting

If you encounter issues:
- Check disk space: `df -h`
- Verify download completeness by checking file size
- Test with a smaller model first to verify the infrastructure works