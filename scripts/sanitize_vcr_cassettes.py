#!/usr/bin/env python3
"""
Sanitize VCR cassettes by removing PII and replacing with test data.

This script processes all JSON cassette files and:
1. Replaces org_id/org values with a test value (12345)
2. Replaces user IDs with test values
3. Replaces emails with test emails
4. Replaces names with generic names
5. Preserves template variables like {{organization.name}}
"""

import json
import os
import re
from pathlib import Path


# Test values to replace PII
TEST_ORG_ID = 12345
TEST_USER_ID = 99999

# Fields that contain organization IDs
ORG_ID_FIELDS = ['org_id', 'org']

# Fields that contain user IDs
USER_ID_FIELDS = ['owner_id', 'published_by_id', 'user_id', 'assignee_id',
                  'language_okay_updated_by_id', 'infosec_program_id']

# Fields that contain email addresses
EMAIL_FIELDS = ['email']

# Fields that contain names to anonymize
NAME_FIELDS = ['first_name', 'last_name', 'display_name', 'short_display_name',
               'audit_team_name']

# Test values for names
TEST_NAMES = {
    'first_name': 'Test',
    'last_name': 'User',
    'display_name': 'Test User',
    'short_display_name': 'TU',
    'audit_team_name': 'Test Audit Team',
}

# Mapping of real org IDs to test value
ORG_ID_MAP = {}

# Mapping of real user IDs to test values
USER_ID_MAP = {}


def sanitize_value(obj, key, value):
    """Sanitize a single value based on its key."""
    if value is None:
        return value

    if key in ORG_ID_FIELDS and isinstance(value, int) and value > 0:
        if value not in ORG_ID_MAP:
            ORG_ID_MAP[value] = TEST_ORG_ID
        return ORG_ID_MAP[value]

    if key in USER_ID_FIELDS and isinstance(value, int) and value > 0:
        if value not in USER_ID_MAP:
            USER_ID_MAP[value] = TEST_USER_ID
        return USER_ID_MAP[value]

    if key in EMAIL_FIELDS and isinstance(value, str) and '@' in value:
        return 'test@example.com'

    if key in NAME_FIELDS and isinstance(value, str) and value:
        return TEST_NAMES.get(key, 'Test Value')

    return value


def sanitize_dict(obj):
    """Recursively sanitize a dictionary."""
    if isinstance(obj, dict):
        result = {}
        for key, value in obj.items():
            if isinstance(value, dict):
                result[key] = sanitize_dict(value)
            elif isinstance(value, list):
                result[key] = sanitize_list(value)
            else:
                result[key] = sanitize_value(obj, key, value)
        return result
    return obj


def sanitize_list(arr):
    """Recursively sanitize a list."""
    result = []
    for item in arr:
        if isinstance(item, dict):
            result.append(sanitize_dict(item))
        elif isinstance(item, list):
            result.append(sanitize_list(item))
        else:
            result.append(item)
    return result


def sanitize_body_string(body_str):
    """Sanitize a JSON body string embedded in response."""
    try:
        body_obj = json.loads(body_str)
        sanitized = sanitize_dict(body_obj) if isinstance(body_obj, dict) else sanitize_list(body_obj) if isinstance(body_obj, list) else body_obj
        return json.dumps(sanitized, separators=(',', ':'))
    except json.JSONDecodeError:
        # Not JSON, return as-is (e.g., HTML error pages)
        return body_str


def sanitize_cassette(cassette_path):
    """Sanitize a single cassette file."""
    with open(cassette_path, 'r') as f:
        cassette = json.load(f)

    # Sanitize interactions
    if 'interactions' in cassette:
        for interaction in cassette['interactions']:
            # Sanitize request URL (replace org IDs in URLs)
            if 'request' in interaction and 'url' in interaction['request']:
                url = interaction['request']['url']
                # Replace org IDs in URL paths like /org/13888/ or org_id=13888
                for real_id in list(ORG_ID_MAP.keys()):
                    url = url.replace(f'/org/{real_id}/', f'/org/{TEST_ORG_ID}/')
                    url = url.replace(f'org_id={real_id}', f'org_id={TEST_ORG_ID}')
                interaction['request']['url'] = url

            # Sanitize response body
            if 'response' in interaction and 'body' in interaction['response']:
                body = interaction['response']['body']
                if body:
                    interaction['response']['body'] = sanitize_body_string(body)

    # Write back
    with open(cassette_path, 'w') as f:
        json.dump(cassette, f, indent=2)


def find_org_ids(cassette_path):
    """First pass: find all org IDs in a cassette."""
    with open(cassette_path, 'r') as f:
        content = f.read()

    # Find org_id and org patterns
    for match in re.finditer(r'"org_id"\s*:\s*(\d+)', content):
        org_id = int(match.group(1))
        if org_id > 0:
            ORG_ID_MAP[org_id] = TEST_ORG_ID

    for match in re.finditer(r'"org"\s*:\s*(\d+)', content):
        org_id = int(match.group(1))
        if org_id > 0:
            ORG_ID_MAP[org_id] = TEST_ORG_ID

    # Find user ID patterns
    for field in USER_ID_FIELDS:
        for match in re.finditer(rf'"{field}"\s*:\s*(\d+)', content):
            user_id = int(match.group(1))
            if user_id > 0:
                USER_ID_MAP[user_id] = TEST_USER_ID


def main():
    # Find cassette directory
    script_dir = Path(__file__).parent.parent
    cassette_dir = script_dir / 'internal' / 'tugboat' / 'testdata' / 'vcr_cassettes'

    if not cassette_dir.exists():
        print(f"Cassette directory not found: {cassette_dir}")
        return 1

    cassette_files = list(cassette_dir.glob('*.json'))
    print(f"Found {len(cassette_files)} cassette files")

    # First pass: find all org IDs
    print("First pass: discovering IDs...")
    for cassette_path in cassette_files:
        find_org_ids(cassette_path)

    print(f"Found org IDs to replace: {list(ORG_ID_MAP.keys())}")
    print(f"Found user IDs to replace: {list(USER_ID_MAP.keys())}")

    # Second pass: sanitize
    print("\nSecond pass: sanitizing cassettes...")
    for cassette_path in cassette_files:
        print(f"  Sanitizing: {cassette_path.name}")
        sanitize_cassette(cassette_path)

    print(f"\nDone! Sanitized {len(cassette_files)} cassettes.")
    return 0


if __name__ == '__main__':
    exit(main())
