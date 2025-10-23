#!/bin/bash

# Complete migration script for all document types
set -e

DOCS_DIR="/Users/erik/Projects/7thsense/isms/docs"

echo "=== COMPLETE NAMING CONVENTION MIGRATION ==="

# Function to clean filename
clean_filename() {
    local name="$1"
    name=$(echo "$name" | tr ',' '_' | tr '/' '_' | tr ' ' '_' | tr '-' '_')
    if [[ ${#name} -gt 40 ]]; then
        name="${name:0:40}"
    fi
    name=$(echo "$name" | sed 's/_*$//')
    echo "$name"
}

# Function to generate simple short name
generate_simple_name() {
    local current_name="$1"
    local ref_id="$2"
    
    name=$(echo "$current_name" | tr '[:upper:]' '[:lower:]')
    name=$(echo "$name" | sed -E 's/\b(population|list|of|the|and|a|an|for|in|on|at|to|from|with|by|report|document|process)\b//g')
    name=$(echo "$name" | sed 's/[[:space:]]\+/_/g' | sed 's/^_*//' | sed 's/_*$//')
    
    if [[ ${#name} -lt 3 ]]; then
        if [[ "$ref_id" =~ ET ]]; then
            name="evidence_task"
        elif [[ "$ref_id" =~ POL ]]; then
            name="policy"
        elif [[ "$ref_id" =~ ^[A-Z]{2} ]]; then
            name="control"
        else
            name="document"
        fi
    fi
    
    clean_filename "$name"
}

# Migrate ALL Evidence Tasks
echo "=== MIGRATING ALL EVIDENCE TASKS ==="
find "$DOCS_DIR/evidence_tasks" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    if [[ "$filename" =~ ^ET-[0-9]{4}-[0-9]+-[a-z_]+\.json$ ]]; then
        continue
    fi
    
    ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null || echo "")
    if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
        ref_id=$(echo "$filename" | sed -E 's/^(ET[0-9]+).*/\1/')
    fi
    
    tugboat_id=$(jq -r '.id' "$file" 2>/dev/null || echo "")
    if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
        tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{6,})_.*/\1/')
        if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    current_name=$(jq -r '.name' "$file" 2>/dev/null || echo "Evidence Task $ref_id")
    
    if [[ "$ref_id" =~ ^ET([0-9]+)$ ]]; then
        num=${BASH_REMATCH[1]}
        normalized_ref=$(printf "ET-%04d" $((10#$num)))
    else
        continue
    fi
    
    short_name=$(generate_simple_name "$current_name" "$ref_id")
    
    if [[ "$ref_id" != "$normalized_ref" ]]; then
        jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    fi
    
    new_filename="${normalized_ref}-${tugboat_id}-${short_name}.json"
    
    if [[ "$filename" != "$new_filename" ]]; then
        new_path="$(dirname "$file")/$new_filename"
        mv "$file" "$new_path"
    fi
    
    echo "✅ $normalized_ref"
done

# Migrate Controls
echo ""
echo "=== MIGRATING CONTROLS ==="
find "$DOCS_DIR/controls" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    if [[ "$filename" =~ ^[A-Z]{2}-[0-9]{2}(_[0-9]+)?-[0-9]+-[a-z_]+\.json$ ]]; then
        continue
    fi
    
    ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null || echo "")
    if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
        ref_id=$(echo "$filename" | sed -E 's/^([A-Z]{2}[0-9_]+).*/\1/' | tr '_' '.')
    fi
    
    tugboat_id=$(jq -r '.id' "$file" 2>/dev/null || echo "")
    if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
        tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{6,})_.*/\1/')
        if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    current_name=$(jq -r '.name' "$file" 2>/dev/null || echo "Control $ref_id")
    
    # Normalize control reference
    if [[ "$ref_id" =~ ^([A-Z]{2})([0-9]+)\.([0-9]+)$ ]]; then
        prefix=${BASH_REMATCH[1]}
        main=${BASH_REMATCH[2]}
        sub=${BASH_REMATCH[3]}
        normalized_ref=$(printf "%s-%02d.%d" "$prefix" $((10#$main)) $((10#$sub)))
        filename_ref=$(printf "%s-%02d_%d" "$prefix" $((10#$main)) $((10#$sub)))
    elif [[ "$ref_id" =~ ^([A-Z]{2})([0-9]+)$ ]]; then
        prefix=${BASH_REMATCH[1]}
        main=${BASH_REMATCH[2]}
        normalized_ref=$(printf "%s-%02d" "$prefix" $((10#$main)))
        filename_ref="$normalized_ref"
    else
        continue
    fi
    
    short_name=$(generate_simple_name "$current_name" "$ref_id")
    
    if [[ "$ref_id" != "$normalized_ref" ]]; then
        jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    fi
    
    new_filename="${filename_ref}-${tugboat_id}-${short_name}.json"
    
    if [[ "$filename" != "$new_filename" ]]; then
        new_path="$(dirname "$file")/$new_filename"
        mv "$file" "$new_path"
    fi
    
    echo "✅ $normalized_ref"
done

# Migrate Policies
echo ""
echo "=== MIGRATING POLICIES ==="
find "$DOCS_DIR/policies" -name "*.json" -type f | while read file; do
    filename=$(basename "$file")
    
    if [[ "$filename" =~ ^POL-[0-9]{4}-[0-9]+-[a-z_]+\.json$ ]]; then
        continue
    fi
    
    ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null || echo "")
    if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
        ref_id=$(echo "$filename" | sed -E 's/^(P[0-9]+|POL-[0-9]+).*/\1/')
    fi
    
    tugboat_id=$(jq -r '.id' "$file" 2>/dev/null || echo "")
    if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
        tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{5,})_.*/\1/')
        if [[ ! "$tugboat_id" =~ ^[0-9]{5,}$ ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    current_name=$(jq -r '.name' "$file" 2>/dev/null || echo "Policy $ref_id")
    
    # Normalize policy reference
    if [[ "$ref_id" =~ ^POL-?([0-9]+)$ ]]; then
        num=${BASH_REMATCH[1]}
        normalized_ref=$(printf "POL-%04d" $((10#$num)))
    elif [[ "$ref_id" =~ ^P([0-9]+)$ ]]; then
        num=${BASH_REMATCH[1]}
        normalized_ref=$(printf "POL-%04d" $((10#$num)))
    else
        continue
    fi
    
    short_name=$(generate_simple_name "$current_name" "$ref_id")
    
    if [[ "$ref_id" != "$normalized_ref" ]]; then
        jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    fi
    
    new_filename="${normalized_ref}-${tugboat_id}-${short_name}.json"
    
    if [[ "$filename" != "$new_filename" ]]; then
        new_path="$(dirname "$file")/$new_filename"
        mv "$file" "$new_path"
    fi
    
    echo "✅ $normalized_ref"
done

echo ""
echo "=== FINAL SUMMARY ==="
evidence_new=$(find "$DOCS_DIR/evidence_tasks" -name "ET-[0-9][0-9][0-9][0-9]-*-*.json" 2>/dev/null | wc -l)
control_new=$(find "$DOCS_DIR/controls" -name "*-[0-9][0-9]*-[0-9]*-*.json" 2>/dev/null | wc -l)
policy_new=$(find "$DOCS_DIR/policies" -name "POL-[0-9][0-9][0-9][0-9]-*-*.json" 2>/dev/null | wc -l)

echo "✅ Migration completed!"
echo "📊 Files in new format:"
echo "   Evidence tasks: $evidence_new"
echo "   Controls: $control_new"
echo "   Policies: $policy_new"