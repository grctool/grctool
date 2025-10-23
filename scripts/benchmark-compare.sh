#!/bin/bash

# benchmark-compare.sh - Compare benchmark results and detect performance regressions
# Usage: ./scripts/benchmark-compare.sh [baseline-file] [current-results]

set -euo pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BENCHMARKS_DIR="$PROJECT_ROOT/benchmarks"
BASELINE_FILE="${1:-$BENCHMARKS_DIR/baseline.txt}"
CURRENT_FILE="${2:-$BENCHMARKS_DIR/current.txt}"
REPORT_FILE="$BENCHMARKS_DIR/comparison-report.txt"
REGRESSION_THRESHOLD=10 # 10% regression threshold

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create benchmarks directory if it doesn't exist
create_benchmarks_dir() {
    if [[ ! -d "$BENCHMARKS_DIR" ]]; then
        log_info "Creating benchmarks directory: $BENCHMARKS_DIR"
        mkdir -p "$BENCHMARKS_DIR"
    fi
}

# Run all benchmarks and save results
run_benchmarks() {
    local output_file="$1"
    
    log_info "Running all benchmarks..."
    log_info "Output will be saved to: $output_file"
    
    cd "$PROJECT_ROOT"
    
    # Run benchmarks with specific flags for consistent output
    go test -bench=. -benchmem -cpu=1 -count=3 \
        ./internal/tugboat \
        ./internal/tools \
        ./internal/auth \
        ./internal/services \
        ./internal/storage \
        2>&1 | tee "$output_file"
    
    if [[ ${PIPESTATUS[0]} -ne 0 ]]; then
        log_error "Benchmark execution failed"
        return 1
    fi
    
    log_success "Benchmarks completed successfully"
}

# Parse benchmark results from file
parse_benchmark_results() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        log_error "Benchmark file not found: $file"
        return 1
    fi
    
    # Extract benchmark lines (format: BenchmarkName-cpu iteration ns/op MB/s allocs/op B/op)
    grep "^Benchmark" "$file" | sed 's/\t/ /g' | awk '
    {
        name = $1
        iterations = $2
        ns_per_op = $3
        
        # Parse allocations if present
        if (NF >= 5 && $4 ~ /B\/op$/) {
            bytes_per_op = $4
            allocs_per_op = $5
        } else if (NF >= 6 && $5 ~ /B\/op$/) {
            bytes_per_op = $5
            allocs_per_op = $6
        } else {
            bytes_per_op = "0"
            allocs_per_op = "0"
        }
        
        # Remove units
        gsub(/ns\/op/, "", ns_per_op)
        gsub(/B\/op/, "", bytes_per_op)
        gsub(/allocs\/op/, "", allocs_per_op)
        
        print name "|" iterations "|" ns_per_op "|" bytes_per_op "|" allocs_per_op
    }'
}

# Compare two benchmark results
compare_benchmarks() {
    local baseline_file="$1"
    local current_file="$2"
    local report_file="$3"
    
    log_info "Comparing benchmarks..."
    log_info "Baseline: $baseline_file"
    log_info "Current:  $current_file"
    log_info "Report:   $report_file"
    
    # Parse results
    local baseline_results
    local current_results
    
    baseline_results=$(parse_benchmark_results "$baseline_file")
    current_results=$(parse_benchmark_results "$current_file")
    
    # Create report header
    cat > "$report_file" << EOF
# Benchmark Comparison Report
Generated: $(date)
Baseline: $baseline_file
Current:  $current_file
Regression Threshold: ${REGRESSION_THRESHOLD}%

## Summary
EOF
    
    # Initialize counters
    local total_benchmarks=0
    local improved_benchmarks=0
    local regressed_benchmarks=0
    local unchanged_benchmarks=0
    local new_benchmarks=0
    local missing_benchmarks=0
    
    # Create associative arrays for comparison
    declare -A baseline_ns current_ns baseline_bytes current_bytes baseline_allocs current_allocs
    
    # Parse baseline results
    while IFS='|' read -r name iterations ns_per_op bytes_per_op allocs_per_op; do
        if [[ -n "$name" ]]; then
            baseline_ns["$name"]="$ns_per_op"
            baseline_bytes["$name"]="$bytes_per_op"
            baseline_allocs["$name"]="$allocs_per_op"
        fi
    done <<< "$baseline_results"
    
    # Parse current results
    while IFS='|' read -r name iterations ns_per_op bytes_per_op allocs_per_op; do
        if [[ -n "$name" ]]; then
            current_ns["$name"]="$ns_per_op"
            current_bytes["$name"]="$bytes_per_op"
            current_allocs["$name"]="$allocs_per_op"
        fi
    done <<< "$current_results"
    
    # Detailed comparison
    echo "" >> "$report_file"
    echo "## Detailed Results" >> "$report_file"
    echo "" >> "$report_file"
    printf "| %-50s | %-15s | %-15s | %-10s | %-15s | %-15s | %-10s |\n" \
        "Benchmark" "Baseline (ns/op)" "Current (ns/op)" "Change %" \
        "Baseline (B/op)" "Current (B/op)" "Mem Change %" >> "$report_file"
    printf "|%s|%s|%s|%s|%s|%s|%s|\n" \
        "$(printf '%*s' 50 '' | tr ' ' '-')" \
        "$(printf '%*s' 15 '' | tr ' ' '-')" \
        "$(printf '%*s' 15 '' | tr ' ' '-')" \
        "$(printf '%*s' 10 '' | tr ' ' '-')" \
        "$(printf '%*s' 15 '' | tr ' ' '-')" \
        "$(printf '%*s' 15 '' | tr ' ' '-')" \
        "$(printf '%*s' 10 '' | tr ' ' '-')" >> "$report_file"
    
    # Compare each benchmark
    for benchmark in "${!current_ns[@]}"; do
        ((total_benchmarks++))
        
        local baseline_time="${baseline_ns[$benchmark]:-}"
        local current_time="${current_ns[$benchmark]}"
        local baseline_mem="${baseline_bytes[$benchmark]:-0}"
        local current_mem="${current_bytes[$benchmark]:-0}"
        
        if [[ -z "$baseline_time" ]]; then
            # New benchmark
            ((new_benchmarks++))
            printf "| %-50s | %-15s | %-15s | %-10s | %-15s | %-15s | %-10s |\n" \
                "$benchmark" "N/A" "$current_time" "NEW" "$baseline_mem" "$current_mem" "NEW" >> "$report_file"
        else
            # Calculate percentage change for time
            local time_change
            if [[ "$baseline_time" != "0" ]]; then
                time_change=$(awk "BEGIN {printf \"%.2f\", (($current_time - $baseline_time) / $baseline_time) * 100}")
            else
                time_change="N/A"
            fi
            
            # Calculate percentage change for memory
            local mem_change
            if [[ "$baseline_mem" != "0" ]]; then
                mem_change=$(awk "BEGIN {printf \"%.2f\", (($current_mem - $baseline_mem) / $baseline_mem) * 100}")
            else
                mem_change="N/A"
            fi
            
            # Categorize the change
            local time_change_num
            time_change_num=$(echo "$time_change" | sed 's/%//')
            
            if [[ "$time_change_num" =~ ^-?[0-9]+\.?[0-9]*$ ]]; then
                if (( $(echo "$time_change_num > $REGRESSION_THRESHOLD" | bc -l) )); then
                    ((regressed_benchmarks++))
                elif (( $(echo "$time_change_num < -5" | bc -l) )); then
                    ((improved_benchmarks++))
                else
                    ((unchanged_benchmarks++))
                fi
            else
                ((unchanged_benchmarks++))
            fi
            
            printf "| %-50s | %-15s | %-15s | %-10s | %-15s | %-15s | %-10s |\n" \
                "$benchmark" "$baseline_time" "$current_time" "${time_change}%" \
                "$baseline_mem" "$current_mem" "${mem_change}%" >> "$report_file"
        fi
    done
    
    # Check for missing benchmarks
    for benchmark in "${!baseline_ns[@]}"; do
        if [[ -z "${current_ns[$benchmark]:-}" ]]; then
            ((missing_benchmarks++))
            printf "| %-50s | %-15s | %-15s | %-10s | %-15s | %-15s | %-10s |\n" \
                "$benchmark" "${baseline_ns[$benchmark]}" "N/A" "MISSING" \
                "${baseline_bytes[$benchmark]}" "N/A" "MISSING" >> "$report_file"
        fi
    done
    
    # Update summary
    {
        echo ""
        echo "**Total Benchmarks:** $total_benchmarks"
        echo "**Improved:** $improved_benchmarks (>5% faster)"
        echo "**Unchanged:** $unchanged_benchmarks (±5%)"
        echo "**Regressed:** $regressed_benchmarks (>$REGRESSION_THRESHOLD% slower)"
        echo "**New:** $new_benchmarks"
        echo "**Missing:** $missing_benchmarks"
        echo ""
    } >> "$report_file"
    
    # Add regression analysis if any regressions found
    if [[ $regressed_benchmarks -gt 0 ]]; then
        echo "## ⚠️ Performance Regressions Detected" >> "$report_file"
        echo "" >> "$report_file"
        echo "The following benchmarks show performance regressions greater than ${REGRESSION_THRESHOLD}%:" >> "$report_file"
        echo "" >> "$report_file"
        
        # Extract regressed benchmarks
        for benchmark in "${!current_ns[@]}"; do
            local baseline_time="${baseline_ns[$benchmark]:-}"
            local current_time="${current_ns[$benchmark]}"
            
            if [[ -n "$baseline_time" && "$baseline_time" != "0" ]]; then
                local time_change
                time_change=$(awk "BEGIN {printf \"%.2f\", (($current_time - $baseline_time) / $baseline_time) * 100}")
                local time_change_num
                time_change_num=$(echo "$time_change" | sed 's/%//')
                
                if [[ "$time_change_num" =~ ^-?[0-9]+\.?[0-9]*$ ]] && (( $(echo "$time_change_num > $REGRESSION_THRESHOLD" | bc -l) )); then
                    echo "- **$benchmark**: ${time_change}% slower ($baseline_time → $current_time ns/op)" >> "$report_file"
                fi
            fi
        done
        echo "" >> "$report_file"
    fi
    
    # Print summary to console
    echo ""
    log_info "=== Benchmark Comparison Summary ==="
    echo "Total Benchmarks: $total_benchmarks"
    echo "Improved: $improved_benchmarks"
    echo "Unchanged: $unchanged_benchmarks"
    
    if [[ $regressed_benchmarks -gt 0 ]]; then
        log_warning "Regressed: $regressed_benchmarks"
        log_warning "Performance regressions detected! Check $report_file for details."
        return 1
    else
        log_success "Regressed: $regressed_benchmarks"
        log_success "No significant performance regressions detected."
    fi
    
    echo "New: $new_benchmarks"
    echo "Missing: $missing_benchmarks"
    echo ""
    log_info "Full report available at: $report_file"
    
    return 0
}

# Save current results as baseline
save_baseline() {
    local current_file="$1"
    local baseline_file="$2"
    
    if [[ ! -f "$current_file" ]]; then
        log_error "Current results file not found: $current_file"
        return 1
    fi
    
    log_info "Saving current results as new baseline..."
    cp "$current_file" "$baseline_file"
    log_success "Baseline saved to: $baseline_file"
}

# Display usage information
usage() {
    echo "Usage: $0 [OPTIONS] [baseline-file] [current-file]"
    echo ""
    echo "Options:"
    echo "  -h, --help                 Show this help message"
    echo "  -r, --run                  Run benchmarks and compare with baseline"
    echo "  -s, --save-baseline        Save current results as new baseline"
    echo "  -b, --baseline-only        Run benchmarks and save as baseline"
    echo "  -c, --compare-only         Compare existing results files"
    echo "  -t, --threshold N          Set regression threshold percentage (default: 10)"
    echo ""
    echo "Examples:"
    echo "  $0 -r                      # Run benchmarks and compare with baseline"
    echo "  $0 -s                      # Run benchmarks and save as new baseline"
    echo "  $0 -c baseline.txt current.txt  # Compare specific files"
    echo "  $0 -t 15 -r                # Use 15% regression threshold"
}

# Main function
main() {
    local run_benchmarks_flag=false
    local save_baseline_flag=false
    local compare_only_flag=false
    local baseline_only_flag=false
    
    # Parse command line arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                exit 0
                ;;
            -r|--run)
                run_benchmarks_flag=true
                shift
                ;;
            -s|--save-baseline)
                run_benchmarks_flag=true
                save_baseline_flag=true
                shift
                ;;
            -b|--baseline-only)
                run_benchmarks_flag=true
                baseline_only_flag=true
                shift
                ;;
            -c|--compare-only)
                compare_only_flag=true
                shift
                ;;
            -t|--threshold)
                REGRESSION_THRESHOLD="$2"
                shift 2
                ;;
            -*)
                log_error "Unknown option: $1"
                usage
                exit 1
                ;;
            *)
                # Positional arguments (baseline and current files)
                if [[ -z "${BASELINE_FILE:-}" || "$BASELINE_FILE" == "$BENCHMARKS_DIR/baseline.txt" ]]; then
                    BASELINE_FILE="$1"
                elif [[ -z "${CURRENT_FILE:-}" || "$CURRENT_FILE" == "$BENCHMARKS_DIR/current.txt" ]]; then
                    CURRENT_FILE="$1"
                else
                    log_error "Too many positional arguments"
                    usage
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # Create benchmarks directory
    create_benchmarks_dir
    
    # Execute based on flags
    if [[ "$run_benchmarks_flag" == true ]]; then
        if ! run_benchmarks "$CURRENT_FILE"; then
            exit 1
        fi
        
        if [[ "$baseline_only_flag" == true ]]; then
            save_baseline "$CURRENT_FILE" "$BASELINE_FILE"
            exit 0
        fi
        
        if [[ "$save_baseline_flag" == true ]]; then
            save_baseline "$CURRENT_FILE" "$BASELINE_FILE"
            exit 0
        fi
        
        # Compare with baseline if it exists
        if [[ -f "$BASELINE_FILE" ]]; then
            if ! compare_benchmarks "$BASELINE_FILE" "$CURRENT_FILE" "$REPORT_FILE"; then
                exit 1
            fi
        else
            log_warning "No baseline file found at: $BASELINE_FILE"
            log_info "Saving current results as baseline..."
            save_baseline "$CURRENT_FILE" "$BASELINE_FILE"
        fi
        
    elif [[ "$compare_only_flag" == true ]]; then
        if ! compare_benchmarks "$BASELINE_FILE" "$CURRENT_FILE" "$REPORT_FILE"; then
            exit 1
        fi
    else
        log_error "No action specified. Use -h for help."
        exit 1
    fi
}

# Check dependencies
check_dependencies() {
    local missing_deps=()
    
    if ! command -v go >/dev/null 2>&1; then
        missing_deps+=("go")
    fi
    
    if ! command -v bc >/dev/null 2>&1; then
        missing_deps+=("bc")
    fi
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing required dependencies: ${missing_deps[*]}"
        log_error "Please install the missing dependencies and try again."
        exit 1
    fi
}

# Run main function
check_dependencies
main "$@"