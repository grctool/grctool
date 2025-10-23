#!/bin/bash

# Simple, focused migration script for naming convention
# Processes files one by one with clear output

set -e

DOCS_DIR="/Users/erik/Projects/7thsense/isms/docs"
BACKUP_DIR="/Users/erik/Projects/7thsense/isms/docs_backup_$(date +%Y%m%d_%H%M%S)"

echo "=== NAMING CONVENTION MIGRATION ==="
echo "Starting migration of documents to new naming convention..."

# Create backup
echo "Creating backup..."
cp -r "$DOCS_DIR" "$BACKUP_DIR"
echo "✅ Backup created at: $BACKUP_DIR"

# Function to clean filename (remove special chars, limit length)
clean_filename() {
    local name="$1"
    # Remove or replace problematic characters
    name=$(echo "$name" | tr ',' '_' | tr '/' '_' | tr ' ' '_' | tr '-' '_')
    # Limit to 40 characters
    if [[ ${#name} -gt 40 ]]; then
        name="${name:0:40}"
    fi
    # Remove trailing underscores
    name=$(echo "$name" | sed 's/_*$//')
    echo "$name"
}

# Function to generate simple short name from current name
generate_simple_name() {
    local current_name="$1"
    local ref_id="$2"
    
    # Convert to lowercase and extract key words
    name=$(echo "$current_name" | tr '[:upper:]' '[:lower:]')
    
    # Remove common filler words
    name=$(echo "$name" | sed -E 's/\b(population|list|of|the|and|a|an|for|in|on|at|to|from|with|by|report|document|process)\b//g')
    
    # Clean up extra spaces and convert to underscores
    name=$(echo "$name" | sed 's/[[:space:]]\+/_/g' | sed 's/^_*//' | sed 's/_*$//')
    
    # If name is empty or too short, use ref_id based fallback
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

# Process Evidence Tasks
echo ""
echo "=== MIGRATING EVIDENCE TASKS ==="
find "$DOCS_DIR/evidence_tasks" -name "*.json" -type f | head -10 | while read file; do
    filename=$(basename "$file")
    
    # Skip if already in new format
    if [[ "$filename" =~ ^ET-[0-9]{4}-[0-9]+-[a-z_]+\.json$ ]]; then
        echo "⏭️  Skipping already migrated: $filename"
        continue
    fi
    
    echo "📄 Processing: $filename"
    
    # Extract reference ID from JSON
    ref_id=$(jq -r '.reference_id' "$file" 2>/dev/null || echo "")
    if [[ "$ref_id" == "null" || "$ref_id" == "" ]]; then
        # Extract from filename
        ref_id=$(echo "$filename" | sed -E 's/^(ET[0-9]+).*/\1/')
    fi
    
    # Extract Tugboat ID
    tugboat_id=$(jq -r '.id' "$file" 2>/dev/null || echo "")
    if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
        tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{6,})_.*/\1/')
        if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
            tugboat_id="MISSING"
        fi
    fi
    
    # Extract current name
    current_name=$(jq -r '.name' "$file" 2>/dev/null || echo "")
    if [[ "$current_name" == "null" || "$current_name" == "" ]]; then
        current_name="Evidence Task $ref_id"
    fi
    
    # Normalize reference ID to ET-0001 format
    if [[ "$ref_id" =~ ^ET([0-9]+)$ ]]; then
        num=${BASH_REMATCH[1]}
        normalized_ref=$(printf "ET-%04d" $((10#$num)))
    else
        echo "❌ Could not parse reference ID: $ref_id"
        continue
    fi
    
    # Generate short name
    short_name=$(generate_simple_name "$current_name" "$ref_id")
    
    # Update reference_id in JSON if needed
    if [[ "$ref_id" != "$normalized_ref" ]]; then
        echo "   📝 Updating reference_id: $ref_id → $normalized_ref"
        jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
    fi
    
    # Create new filename
    new_filename="${normalized_ref}-${tugboat_id}-${short_name}.json"
    
    if [[ "$filename" != "$new_filename" ]]; then
        new_path="$(dirname "$file")/$new_filename"
        echo "   📁 Renaming: $filename → $new_filename"
        mv "$file" "$new_path"
    fi
    
    echo "   ✅ Completed: $normalized_ref"
done

echo ""
echo "=== MIGRATION SUMMARY ==="

# Count results
evidence_new=$(find "$DOCS_DIR/evidence_tasks" -name "ET-[0-9][0-9][0-9][0-9]-*-*.json" 2>/dev/null | wc -l)
evidence_old=$(find "$DOCS_DIR/evidence_tasks" -name "ET[0-9]*_*" 2>/dev/null | wc -l)

echo "Evidence tasks in new format: $evidence_new"
echo "Evidence tasks in old format: $evidence_old"

echo ""
echo "✅ Migration completed!"
echo "📂 Backup location: $BACKUP_DIR"

# Show a few examples
echo ""
echo "📋 Example new filenames:"
find "$DOCS_DIR/evidence_tasks" -name "ET-[0-9][0-9][0-9][0-9]-*-*.json" 2>/dev/null | head -5