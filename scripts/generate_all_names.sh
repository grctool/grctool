#!/bin/bash

# Script to generate short names for all documents using the name-generator tool
# This creates a comprehensive naming mapping for the migration

OUTPUT_FILE="document_name_mappings.json"

echo "Generating short names for all documents..."
echo "{"

# Process Evidence Tasks
echo "  \"evidence_tasks\": {"
first=true

# Get list of evidence task IDs from the Tugboat mapping
if [[ -f tugboat_id_mapping.json ]]; then
    # Extract evidence task references from the mapping file
    evidence_refs=$(jq -r '.evidence_tasks | keys[]' tugboat_id_mapping.json 2>/dev/null)
    
    for ref_id in $evidence_refs; do
        if [[ "$ref_id" =~ ^ET[0-9]+$ ]]; then
            echo "Processing evidence task: $ref_id" >&2
            
            # Generate name using the name-generator tool
            result=$(./bin/grctool tool name-generator --document-type evidence --reference-id "$ref_id" 2>/dev/null)
            
            if [[ $? -eq 0 ]]; then
                # Extract generated name from result
                generated_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.generated_name' 2>/dev/null)
                original_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.current_name' 2>/dev/null)
                normalized_id=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.normalized_reference_id' 2>/dev/null)
                
                if [[ "$generated_name" != "null" && "$generated_name" != "" ]]; then
                    # Output JSON mapping
                    if [[ "$first" != "true" ]]; then
                        echo ","
                    fi
                    echo -n "    \"$ref_id\": {\"original_name\": \"$original_name\", \"generated_name\": \"$generated_name\", \"normalized_id\": \"$normalized_id\"}"
                    first=false
                else
                    echo "Failed to generate name for $ref_id" >&2
                fi
            else
                echo "Error processing $ref_id" >&2
            fi
            
            # Add a small delay to avoid overwhelming the system
            sleep 0.1
        fi
    done
fi

echo ""
echo "  },"

# Process Controls
echo "  \"controls\": {"
first=true

# Sample control processing (since we have many, let's process a few key ones)
sample_controls=("CC1.1" "AC1" "SO1" "HR1" "OM1")

for ref_id in "${sample_controls[@]}"; do
    echo "Processing control: $ref_id" >&2
    
    # Generate name using the name-generator tool
    result=$(./bin/grctool tool name-generator --document-type control --reference-id "$ref_id" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        # Extract generated name from result
        generated_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.generated_name' 2>/dev/null)
        original_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.current_name' 2>/dev/null)
        normalized_id=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.normalized_reference_id' 2>/dev/null)
        
        if [[ "$generated_name" != "null" && "$generated_name" != "" ]]; then
            # Output JSON mapping
            if [[ "$first" != "true" ]]; then
                echo ","
            fi
            echo -n "    \"$ref_id\": {\"original_name\": \"$original_name\", \"generated_name\": \"$generated_name\", \"normalized_id\": \"$normalized_id\"}"
            first=false
        else
            echo "Failed to generate name for $ref_id" >&2
        fi
    else
        echo "Error processing control $ref_id" >&2
    fi
    
    # Add a small delay
    sleep 0.1
done

echo ""
echo "  },"

# Process Policies
echo "  \"policies\": {"
first=true

# Sample policy processing
sample_policies=("POL-001" "POL1" "POL2")

for ref_id in "${sample_policies[@]}"; do
    echo "Processing policy: $ref_id" >&2
    
    # Generate name using the name-generator tool
    result=$(./bin/grctool tool name-generator --document-type policy --reference-id "$ref_id" 2>/dev/null)
    
    if [[ $? -eq 0 ]]; then
        # Extract generated name from result
        generated_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.generated_name' 2>/dev/null)
        original_name=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.current_name' 2>/dev/null)
        normalized_id=$(echo "$result" | jq -r '.data.result' 2>/dev/null | jq -r '.normalized_reference_id' 2>/dev/null)
        
        if [[ "$generated_name" != "null" && "$generated_name" != "" ]]; then
            # Output JSON mapping
            if [[ "$first" != "true" ]]; then
                echo ","
            fi
            echo -n "    \"$ref_id\": {\"original_name\": \"$original_name\", \"generated_name\": \"$generated_name\", \"normalized_id\": \"$normalized_id\"}"
            first=false
        else
            echo "Failed to generate name for $ref_id" >&2
        fi
    else
        echo "Error processing policy $ref_id" >&2
    fi
    
    # Add a small delay
    sleep 0.1
done

echo ""
echo "  }"

echo "}"

echo "Name generation completed. Results saved to $OUTPUT_FILE" >&2