#!/bin/bash

# Ensure VAULT_ADDR and VAULT_TOKEN are set
if [ -z "$VAULT_ADDR" ] || [ -z "$VAULT_TOKEN" ]; then
  echo "Please set VAULT_ADDR and VAULT_TOKEN environment variables."
  exit 1
fi

# Path to the keys file
KEYS_FILE="keys.json"

# Check if keys file exists
if [ ! -f "$KEYS_FILE" ]; then
  echo "Keys file ($KEYS_FILE) not found!"
  exit 1
fi

# Enable KV secrets engine at the desired path if not already enabled
vault secrets enable -path=private-keys kv >/dev/null 2>&1

# Read and parse the keys.json file
KEY_COUNT=$(jq '.keys | length' "$KEYS_FILE")

for (( i=0; i<KEY_COUNT; i++ )); do
  PUBLIC_KEY=$(jq -r ".keys[$i].public_key" "$KEYS_FILE")
  PRIVATE_KEY=$(jq -r ".keys[$i].private_key" "$KEYS_FILE")

  # Store the private key in Vault with the public key as the key
  vault kv put "private-keys/${PUBLIC_KEY}" private_key="${PRIVATE_KEY}"
  
  echo "Loaded private key for public key: ${PUBLIC_KEY}"
done

echo "All keys have been loaded into Vault."
