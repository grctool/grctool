#!/usr/bin/env python3
"""
Coverage Badge Generator for GRCTool

Generates coverage badges in SVG format based on Go test coverage output.
Supports both overall coverage and per-package coverage badges.
"""

import re
import sys
import argparse
import subprocess
from pathlib import Path
from typing import Tuple, Optional


def get_color_for_coverage(coverage: float) -> str:
    """
    Determine badge color based on coverage percentage.
    
    Args:
        coverage: Coverage percentage (0-100)
        
    Returns:
        Color code for the badge
    """
    if coverage >= 80:
        return "brightgreen"
    elif coverage >= 70:
        return "green"
    elif coverage >= 60:
        return "yellowgreen"
    elif coverage >= 50:
        return "yellow"
    elif coverage >= 30:
        return "orange"
    else:
        return "red"


def extract_overall_coverage(coverage_file: str) -> Optional[float]:
    """
    Extract overall coverage percentage from coverage file.
    
    Args:
        coverage_file: Path to the coverage.out file
        
    Returns:
        Coverage percentage as float, or None if extraction fails
    """
    try:
        result = subprocess.run(
            ["go", "tool", "cover", "-func", coverage_file],
            capture_output=True,
            text=True,
            check=True
        )
        
        # Parse the last line which contains total coverage
        lines = result.stdout.strip().split('\n')
        if not lines:
            return None
            
        last_line = lines[-1]
        # Format: "total: (statements) XX.X%"
        match = re.search(r'(\d+\.?\d*)%', last_line)
        if match:
            return float(match.group(1))
    except (subprocess.CalledProcessError, ValueError, AttributeError):
        pass
    
    return None


def extract_package_coverage(package: str, coverage_file: str) -> Optional[float]:
    """
    Extract coverage percentage for a specific package.
    
    Args:
        package: Package name to extract coverage for
        coverage_file: Path to the coverage.out file
        
    Returns:
        Coverage percentage as float, or None if package not found
    """
    try:
        result = subprocess.run(
            ["go", "tool", "cover", "-func", coverage_file],
            capture_output=True,
            text=True,
            check=True
        )
        
        # Parse function coverage and calculate package coverage
        package_functions = []
        for line in result.stdout.split('\n'):
            if package in line and not line.strip().startswith('total:'):
                match = re.search(r'(\d+\.?\d*)%', line)
                if match:
                    package_functions.append(float(match.group(1)))
        
        if package_functions:
            return sum(package_functions) / len(package_functions)
            
    except (subprocess.CalledProcessError, ValueError, AttributeError):
        pass
    
    return None


def generate_svg_badge(label: str, message: str, color: str) -> str:
    """
    Generate SVG badge with the given parameters.
    
    Args:
        label: Left side label text
        message: Right side message text
        color: Badge color
        
    Returns:
        SVG badge as string
    """
    # Calculate text widths (approximate)
    label_width = len(label) * 6 + 10
    message_width = len(message) * 6 + 10
    total_width = label_width + message_width
    
    svg_template = f'''<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="{total_width}" height="20">
    <linearGradient id="b" x2="0" y2="100%">
        <stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
        <stop offset="1" stop-opacity=".1"/>
    </linearGradient>
    <clipPath id="a">
        <rect width="{total_width}" height="20" rx="3" fill="#fff"/>
    </clipPath>
    <g clip-path="url(#a)">
        <path fill="#555" d="M0 0h{label_width}v20H0z"/>
        <path fill="{get_color_hex(color)}" d="M{label_width} 0h{message_width}v20H{label_width}z"/>
        <path fill="url(#b)" d="M0 0h{total_width}v20H0z"/>
    </g>
    <g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="110">
        <text x="{label_width//2 + 1}" y="15" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="{(label_width-10)*10}">{label}</text>
        <text x="{label_width//2 + 1}" y="14" transform="scale(.1)" textLength="{(label_width-10)*10}">{label}</text>
        <text x="{label_width + message_width//2 - 1}" y="15" fill="#010101" fill-opacity=".3" transform="scale(.1)" textLength="{(message_width-10)*10}">{message}</text>
        <text x="{label_width + message_width//2 - 1}" y="14" transform="scale(.1)" textLength="{(message_width-10)*10}">{message}</text>
    </g>
</svg>'''
    
    return svg_template


def get_color_hex(color_name: str) -> str:
    """Convert color name to hex code."""
    colors = {
        "brightgreen": "#4c1",
        "green": "#97CA00",
        "yellowgreen": "#a4a61d",
        "yellow": "#dfb317",
        "orange": "#fe7d37",
        "red": "#e05d44"
    }
    return colors.get(color_name, "#9f9f9f")


def main():
    """Main function to handle command line arguments and generate badges."""
    parser = argparse.ArgumentParser(description="Generate coverage badges for GRCTool")
    parser.add_argument(
        "--coverage-file",
        default="coverage.out",
        help="Path to coverage.out file (default: coverage.out)"
    )
    parser.add_argument(
        "--output",
        default="docs/coverage/badge.svg",
        help="Output path for badge (default: docs/coverage/badge.svg)"
    )
    parser.add_argument(
        "--package",
        help="Generate badge for specific package instead of overall coverage"
    )
    parser.add_argument(
        "--label",
        default="coverage",
        help="Label text for badge (default: coverage)"
    )
    parser.add_argument(
        "--format",
        choices=["svg", "markdown"],
        default="svg",
        help="Output format (default: svg)"
    )
    
    args = parser.parse_args()
    
    # Ensure coverage file exists
    if not Path(args.coverage_file).exists():
        print(f"Error: Coverage file '{args.coverage_file}' not found.", file=sys.stderr)
        print("Run 'go test -coverprofile=coverage.out ./...' first.", file=sys.stderr)
        sys.exit(1)
    
    # Extract coverage percentage
    if args.package:
        coverage = extract_package_coverage(args.package, args.coverage_file)
        if coverage is None:
            print(f"Error: Could not extract coverage for package '{args.package}'", file=sys.stderr)
            sys.exit(1)
        label = f"{args.package.split('/')[-1]} coverage"
    else:
        coverage = extract_overall_coverage(args.coverage_file)
        if coverage is None:
            print(f"Error: Could not extract overall coverage from '{args.coverage_file}'", file=sys.stderr)
            sys.exit(1)
        label = args.label
    
    # Format coverage message
    coverage_text = f"{coverage:.1f}%"
    color = get_color_for_coverage(coverage)
    
    # Generate output
    if args.format == "svg":
        # Ensure output directory exists
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        
        # Generate and save SVG badge
        svg_content = generate_svg_badge(label, coverage_text, color)
        
        with open(output_path, 'w') as f:
            f.write(svg_content)
        
        print(f"✓ Coverage badge generated: {output_path}")
        print(f"  Coverage: {coverage_text}")
        print(f"  Color: {color}")
        
    elif args.format == "markdown":
        # Generate markdown badge using shields.io
        encoded_label = label.replace(" ", "%20")
        encoded_message = coverage_text.replace(" ", "%20")
        shield_url = f"https://img.shields.io/badge/{encoded_label}-{encoded_message}-{color}"
        markdown = f"![{label}]({shield_url})"
        print(markdown)
    
    # Show coverage status
    if coverage >= 80:
        status = "EXCELLENT"
    elif coverage >= 70:
        status = "GOOD"
    elif coverage >= 50:
        status = "FAIR"
    else:
        status = "NEEDS IMPROVEMENT"
    
    print(f"  Status: {status}")


if __name__ == "__main__":
    main()