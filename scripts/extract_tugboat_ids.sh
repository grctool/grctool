#!/bin/bash

# Script to extract Tugboat IDs from all document files
# This creates a comprehensive mapping for the naming convention migration

DATA_DIR="/Users/erik/Projects/7thsense/isms/docs"
OUTPUT_FILE="tugboat_id_mapping.json"

echo "Extracting Tugboat IDs from all documents..."
echo "{"

# Process Evidence Tasks
echo "  \"evidence_tasks\": {"
first=true
find "$DATA_DIR/evidence_tasks" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    # Try to extract from filename pattern: ET##_TUGBOAT_ID_Name.json
    tugboat_id=$(echo "$filename" | sed -E 's/^ET[0-9]*_([0-9]{6,})_.*/\1/')
    
    # If that didn't work, try to get from JSON
    if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    # Extract reference ID from filename
    ref_id=$(echo "$filename" | sed -E 's/^(ET[0-9]+).*/\1/')
    
    # Output JSON mapping
    if [[ "$first" != "true" ]]; then
        echo ","
    fi
    echo -n "    \"$ref_id\": {\"tugboat_id\": \"$tugboat_id\", \"filename\": \"$filename\"}"
    first=false
done
echo ""
echo "  },"

# Process Controls
echo "  \"controls\": {"
first=true
find "$DATA_DIR/controls" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    # Try to extract from filename pattern: PREFIX##_TUGBOAT_ID_Name.json
    tugboat_id=$(echo "$filename" | sed -E 's/^[A-Z]{2}[0-9_]*_([0-9]{6,})_.*/\1/')
    
    # If that didn't work, try to get from JSON
    if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    # Extract reference ID from filename or JSON
    ref_id=$(echo "$filename" | sed -E 's/^([A-Z]{2}[0-9_]+).*/\1/' | tr '_' '.')
    if [[ "$ref_id" == "$filename" ]]; then
        # Try to get from JSON if filename parsing failed
        ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null)
        if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
            ref_id="UNKNOWN"
        fi
    fi
    
    # Output JSON mapping
    if [[ "$first" != "true" ]]; then
        echo ","
    fi
    echo -n "    \"$ref_id\": {\"tugboat_id\": \"$tugboat_id\", \"filename\": \"$filename\"}"
    first=false
done
echo ""
echo "  },"

# Process Policies
echo "  \"policies\": {"
first=true
find "$DATA_DIR/policies" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    # Try to extract from filename pattern: P##_TUGBOAT_ID_Name.json
    tugboat_id=$(echo "$filename" | sed -E 's/^P[0-9]+_([0-9]{5,})_.*/\1/')
    
    # If that didn't work, try to get from JSON
    if [[ ! "$tugboat_id" =~ ^[0-9]{5,}$ ]]; then
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    # Extract reference ID from JSON (policies use POL-### format)
    ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null)
    if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
        # Fallback to filename-based extraction
        ref_id=$(echo "$filename" | sed -E 's/^(P[0-9]+).*/POL-\1/')
    fi
    
    # Output JSON mapping
    if [[ "$first" != "true" ]]; then
        echo ","
    fi
    echo -n "    \"$ref_id\": {\"tugboat_id\": \"$tugboat_id\", \"filename\": \"$filename\"}"
    first=false
done
echo ""
echo "  }"

echo "}"

echo "Tugboat ID extraction completed. Results saved to $OUTPUT_FILE"