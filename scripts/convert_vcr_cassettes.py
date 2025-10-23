#!/usr/bin/env python3
"""
Convert VCR cassettes from JSON format to go-vcr YAML format.

JSON format (custom):
{
  "name": "cassette.yaml",
  "interactions": [
    {
      "request": {
        "method": "GET",
        "url": "...",
        "headers": {"Accept": "..."}  # string values
      },
      "response": {
        "status_code": 200,
        "status": "200 OK",
        "headers": {"Content-Type": "..."},  # string values
        "body": "{...}"  # plain string
      }
    }
  ]
}

YAML format (go-vcr):
interactions:
- request:
    body: null
    form: {}
    headers:
      Accept:
      - "..."  # array of strings
    method: GET
    uri: "..."
  response:
    body:
      string: |
        {...}  # nested under 'string' key
    code: 200
    headers:
      Content-Type:
      - "..."  # array of strings
"""

import json
import sys
import yaml
from pathlib import Path


def convert_headers(headers_dict):
    """Convert headers from map[string]string to map[string][]string."""
    return {k: [v] for k, v in headers_dict.items()}


def convert_interaction(json_interaction):
    """Convert a single interaction from JSON to go-vcr format."""
    req = json_interaction["request"]
    resp = json_interaction["response"]

    # Convert request
    govcr_request = {
        "body": req.get("body", None),
        "form": {},
        "headers": convert_headers(req["headers"]),
        "method": req["method"],
        "uri": req["url"]
    }

    # Convert response
    govcr_response = {
        "body": {
            "string": resp["body"]
        },
        "code": resp["status_code"],
        "headers": convert_headers(resp["headers"])
    }

    return {
        "request": govcr_request,
        "response": govcr_response
    }


def convert_cassette(input_file, output_file):
    """Convert a JSON cassette file to go-vcr YAML format."""
    print(f"Converting {input_file} -> {output_file}")

    # Read JSON
    with open(input_file, 'r') as f:
        json_data = json.load(f)

    # Convert interactions
    govcr_interactions = [
        convert_interaction(interaction)
        for interaction in json_data["interactions"]
    ]

    # Create go-vcr format
    govcr_cassette = {
        "interactions": govcr_interactions
    }

    # Write YAML
    with open(output_file, 'w') as f:
        yaml.dump(govcr_cassette, f, default_flow_style=False, allow_unicode=True, width=1000)

    print(f"✓ Converted {len(govcr_interactions)} interactions")


def main():
    if len(sys.argv) > 1:
        # Convert specific files
        for input_file in sys.argv[1:]:
            input_path = Path(input_file)
            output_path = input_path
            convert_cassette(str(input_path), str(output_path))
    else:
        print("Usage: python3 convert_vcr_cassettes.py <cassette-file> [<cassette-file> ...]")
        sys.exit(1)


if __name__ == "__main__":
    main()
