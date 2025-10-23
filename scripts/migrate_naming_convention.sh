#!/bin/bash

# Comprehensive naming convention migration script
# This script will:
# 1. Update all JSON reference_id fields to new format
# 2. Rename all files to new convention
# 3. Create backup of original files

set -e  # Exit on any error

DOCS_DIR="/Users/erik/Projects/7thsense/isms/docs"
BACKUP_DIR="/Users/erik/Projects/7thsense/isms/docs_backup_$(date +%Y%m%d_%H%M%S)"
MIGRATION_LOG="migration_log.txt"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date '+%Y-%m-%d %H:%M:%S')]${NC} $1" | tee -a "$MIGRATION_LOG"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1" | tee -a "$MIGRATION_LOG"
}

warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1" | tee -a "$MIGRATION_LOG"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "$MIGRATION_LOG"
}

# Function to normalize reference IDs
normalize_evidence_ref() {
    local ref_id="$1"
    if [[ "$ref_id" =~ ^ET-?([0-9]+)$ ]]; then
        printf "ET-%04d" "${BASH_REMATCH[1]}"
    else
        echo "$ref_id"
    fi
}

normalize_policy_ref() {
    local ref_id="$1"
    if [[ "$ref_id" =~ ^POL-?([0-9]+)$ ]]; then
        printf "POL-%04d" "${BASH_REMATCH[1]}"
    elif [[ "$ref_id" =~ ^P([0-9]+)$ ]]; then
        printf "POL-%04d" "${BASH_REMATCH[1]}"
    else
        echo "$ref_id"
    fi
}

normalize_control_ref() {
    local ref_id="$1"
    # Handle various control formats: CC1.1, AC1, SO19, etc.
    if [[ "$ref_id" =~ ^([A-Z]{2})-?([0-9]+)\.([0-9]+)$ ]]; then
        printf "%s-%02d_%d" "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}" "${BASH_REMATCH[3]}"
    elif [[ "$ref_id" =~ ^([A-Z]{2})-?([0-9]+)$ ]]; then
        printf "%s-%02d" "${BASH_REMATCH[1]}" "${BASH_REMATCH[2]}"
    else
        echo "$ref_id"
    fi
}

# Function to generate short name using the name generator tool
generate_short_name() {
    local doc_type="$1"
    local ref_id="$2"
    
    log "Generating short name for $doc_type $ref_id" >&2
    
    result=$(./bin/grctool tool name-generator --document-type "$doc_type" --reference-id "$ref_id" 2>/dev/null)
    if [[ $? -eq 0 ]]; then
        short_name=$(echo "$result" | jq -r '.data.result | fromjson | .generated_name' 2>/dev/null)
        if [[ "$short_name" != "null" && "$short_name" != "" ]]; then
            echo "$short_name"
        else
            # Fallback to generic name
            echo "${doc_type}_${ref_id,,}"
        fi
    else
        # Fallback to generic name
        echo "${doc_type}_${ref_id,,}"
    fi
}

# Function to backup original files
create_backup() {
    log "Creating backup of original files..."
    
    if [[ -d "$BACKUP_DIR" ]]; then
        error "Backup directory already exists: $BACKUP_DIR"
        exit 1
    fi
    
    cp -r "$DOCS_DIR" "$BACKUP_DIR"
    success "Backup created at: $BACKUP_DIR"
}

# Function to migrate evidence tasks
migrate_evidence_tasks() {
    log "Migrating evidence tasks..."
    
    local count=0
    find "$DOCS_DIR/evidence_tasks" -name "*.json" -type f | while read file; do
        filename=$(basename "$file")
        
        # Skip files that are already in new format
        if [[ "$filename" =~ ^ET-[0-9]{4}-[0-9]+-[a-z_]+\.json$ ]]; then
            log "Skipping already migrated file: $filename"
            continue
        fi
        
        log "Processing evidence task: $filename"
        
        # Extract current reference ID from JSON
        current_ref=$(jq -r '.reference_id' "$file" 2>/dev/null)
        if [[ "$current_ref" == "null" || "$current_ref" == "" ]]; then
            # Try to extract from filename
            current_ref=$(echo "$filename" | sed -E 's/^(ET[0-9]+).*/\1/')
        fi
        
        # Extract Tugboat ID
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            # Try to extract from filename
            tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{6,})_.*/\1/')
            if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
                warning "Could not extract Tugboat ID for $filename"
                tugboat_id="MISSING"
            fi
        fi
        
        # Normalize reference ID
        normalized_ref=$(normalize_evidence_ref "$current_ref")
        
        # Generate short name
        short_name=$(generate_short_name "evidence" "$current_ref")
        
        # Update reference_id in JSON
        if [[ "$current_ref" != "$normalized_ref" ]]; then
            log "Updating reference_id from '$current_ref' to '$normalized_ref' in $filename"
            jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
        fi
        
        # Create new filename
        ref_num=${normalized_ref#ET-}
        ref_num=$((10#$ref_num))  # Force base 10 interpretation
        new_filename="ET-$(printf '%04d' $ref_num)-${tugboat_id}-${short_name}.json"
        
        if [[ "$filename" != "$new_filename" ]]; then
            new_path="$(dirname "$file")/$new_filename"
            log "Renaming: $filename -> $new_filename"
            mv "$file" "$new_path"
            count=$((count + 1))
        fi
    done
    
    success "Migrated evidence tasks"
}

# Function to migrate controls
migrate_controls() {
    log "Migrating controls..."
    
    local count=0
    find "$DOCS_DIR/controls" -name "*.json" -type f | while read file; do
        filename=$(basename "$file")
        
        # Skip files that are already in new format
        if [[ "$filename" =~ ^[A-Z]{2}-[0-9]{2}(_[0-9]+)?-[0-9]+-[a-z_]+\.json$ ]]; then
            log "Skipping already migrated file: $filename"
            continue
        fi
        
        log "Processing control: $filename"
        
        # Extract current reference ID from JSON
        current_ref=$(jq -r '.reference_id' "$file" 2>/dev/null)
        if [[ "$current_ref" == "null" || "$current_ref" == "" ]]; then
            # Try to extract from filename and convert underscores to dots
            current_ref=$(echo "$filename" | sed -E 's/^([A-Z]{2}[0-9_]+).*/\1/' | tr '_' '.')
            if [[ "$current_ref" == "$filename" ]]; then
                warning "Could not extract reference ID for $filename"
                continue
            fi
        fi
        
        # Extract Tugboat ID
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            # Try to extract from filename
            tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{6,})_.*/\1/')
            if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
                # Try another pattern
                tugboat_id=$(echo "$filename" | sed -E 's/^[A-Z]{2}[0-9_]*_([0-9]{6,})_.*/\1/')
                if [[ ! "$tugboat_id" =~ ^[0-9]{6,}$ ]]; then
                    warning "Could not extract Tugboat ID for $filename"
                    tugboat_id="MISSING"
                fi
            fi
        fi
        
        # Normalize reference ID (keep dots in JSON, convert to underscores for filename)
        normalized_ref=$(normalize_control_ref "$current_ref")
        filename_ref=$(echo "$normalized_ref" | tr '.' '_')
        
        # Generate short name
        short_name=$(generate_short_name "control" "$current_ref")
        
        # Update reference_id in JSON (keep dots for JSON format)
        json_ref=$(echo "$normalized_ref" | tr '_' '.')
        if [[ "$current_ref" != "$json_ref" ]]; then
            log "Updating reference_id from '$current_ref' to '$json_ref' in $filename"
            jq --arg new_ref "$json_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
        fi
        
        # Create new filename (using underscores)
        new_filename="${filename_ref}-${tugboat_id}-${short_name}.json"
        
        if [[ "$filename" != "$new_filename" ]]; then
            new_path="$(dirname "$file")/$new_filename"
            log "Renaming: $filename -> $new_filename"
            mv "$file" "$new_path"
            count=$((count + 1))
        fi
    done
    
    success "Migrated controls"
}

# Function to migrate policies
migrate_policies() {
    log "Migrating policies..."
    
    local count=0
    find "$DOCS_DIR/policies" -name "*.json" -type f | while read file; do
        filename=$(basename "$file")
        
        # Skip files that are already in new format
        if [[ "$filename" =~ ^POL-[0-9]{4}-[0-9]+-[a-z_]+\.json$ ]]; then
            log "Skipping already migrated file: $filename"
            continue
        fi
        
        log "Processing policy: $filename"
        
        # Extract current reference ID from JSON
        current_ref=$(jq -r '.reference_id' "$file" 2>/dev/null)
        if [[ "$current_ref" == "null" || "$current_ref" == "" ]]; then
            # Try to extract from filename
            current_ref=$(echo "$filename" | sed -E 's/^(P[0-9]+|POL-[0-9]+).*/\1/')
            if [[ "$current_ref" == "$filename" ]]; then
                warning "Could not extract reference ID for $filename"
                continue
            fi
        fi
        
        # Extract Tugboat ID
        tugboat_id=$(jq -r '.id' "$file" 2>/dev/null)
        if [[ "$tugboat_id" == "null" || "$tugboat_id" == "" ]]; then
            # Try to extract from filename
            tugboat_id=$(echo "$filename" | sed -E 's/.*_([0-9]{5,})_.*/\1/')
            if [[ ! "$tugboat_id" =~ ^[0-9]{5,}$ ]]; then
                warning "Could not extract Tugboat ID for $filename"
                tugboat_id="MISSING"
            fi
        fi
        
        # Normalize reference ID
        normalized_ref=$(normalize_policy_ref "$current_ref")
        
        # Generate short name
        short_name=$(generate_short_name "policy" "$current_ref")
        
        # Update reference_id in JSON
        if [[ "$current_ref" != "$normalized_ref" ]]; then
            log "Updating reference_id from '$current_ref' to '$normalized_ref' in $filename"
            jq --arg new_ref "$normalized_ref" '.reference_id = $new_ref' "$file" > "${file}.tmp" && mv "${file}.tmp" "$file"
        fi
        
        # Create new filename
        ref_num=${normalized_ref#POL-}
        ref_num=$((10#$ref_num))  # Force base 10 interpretation
        new_filename="POL-$(printf '%04d' $ref_num)-${tugboat_id}-${short_name}.json"
        
        if [[ "$filename" != "$new_filename" ]]; then
            new_path="$(dirname "$file")/$new_filename"
            log "Renaming: $filename -> $new_filename"
            mv "$file" "$new_path"
            count=$((count + 1))
        fi
    done
    
    success "Migrated policies"
}

# Function to validate migration
validate_migration() {
    log "Validating migration..."
    
    # Count files in new format
    evidence_count=$(find "$DOCS_DIR/evidence_tasks" -name "ET-[0-9][0-9][0-9][0-9]-*-*.json" | wc -l)
    control_count=$(find "$DOCS_DIR/controls" -name "*-[0-9][0-9]-*-*.json" | wc -l)
    policy_count=$(find "$DOCS_DIR/policies" -name "POL-[0-9][0-9][0-9][0-9]-*-*.json" | wc -l)
    
    log "Files in new format:"
    log "  Evidence tasks: $evidence_count"
    log "  Controls: $control_count"
    log "  Policies: $policy_count"
    
    # Check for any files that might not have been migrated
    old_evidence=$(find "$DOCS_DIR/evidence_tasks" -name "ET[0-9]*_*" | wc -l)
    old_controls=$(find "$DOCS_DIR/controls" -name "*[0-9]_[0-9]*_*" | wc -l)
    old_policies=$(find "$DOCS_DIR/policies" -name "P[0-9]*_*" | wc -l)
    
    if [[ $old_evidence -gt 0 || $old_controls -gt 0 || $old_policies -gt 0 ]]; then
        warning "Some files may not have been migrated:"
        warning "  Old evidence format: $old_evidence"
        warning "  Old control format: $old_controls"
        warning "  Old policy format: $old_policies"
    else
        success "All files appear to be in new format"
    fi
}

# Main execution
main() {
    log "Starting naming convention migration..."
    log "Migration log: $MIGRATION_LOG"
    
    # Check prerequisites
    if [[ ! -f "./bin/grctool" ]]; then
        error "grctool binary not found. Please run 'go build -o bin/grctool' first."
        exit 1
    fi
    
    if [[ ! -d "$DOCS_DIR" ]]; then
        error "Docs directory not found: $DOCS_DIR"
        exit 1
    fi
    
    # Create backup
    create_backup
    
    # Perform migrations
    migrate_evidence_tasks
    migrate_controls
    migrate_policies
    
    # Validate results
    validate_migration
    
    success "Migration completed successfully!"
    log "Backup location: $BACKUP_DIR"
    log "Migration log: $MIGRATION_LOG"
}

# Run main function
main "$@"