#!/usr/bin/env python3
"""Generate Terraform import config from Habitica API with proper tag references."""

import json
import os
import re
import subprocess
import sys
from pathlib import Path

def fetch(endpoint):
    """Fetch from Habitica API."""
    user_id = os.environ.get("HABITICA_USER_ID")
    api_token = os.environ.get("HABITICA_API_TOKEN")
    client_id = os.environ.get("HABITICA_CLIENT_AUTHOR_ID")

    if not all([user_id, api_token, client_id]):
        print("Error: Set HABITICA_USER_ID, HABITICA_API_TOKEN, HABITICA_CLIENT_AUTHOR_ID", file=sys.stderr)
        sys.exit(1)

    result = subprocess.run([
        "curl", "-s",
        "-H", f"x-api-user: {user_id}",
        "-H", f"x-api-key: {api_token}",
        "-H", f"x-client: {client_id}-TerraformHabitica",
        f"https://habitica.com/api/v3{endpoint}"
    ], capture_output=True, text=True)

    return json.loads(result.stdout)["data"]

def to_resource_name(text):
    """Convert text to valid Terraform resource name."""
    name = re.sub(r'[^a-zA-Z0-9]', '_', text).lower()[:50]
    # Ensure doesn't start with number
    if name and name[0].isdigit():
        name = "_" + name
    return name

def main():
    output_dir = Path("examples/import")

    print("Fetching tags...", file=sys.stderr)
    tags = fetch("/tags")

    # Build UUID -> resource name lookup
    tag_lookup = {t["id"]: to_resource_name(t["name"]) for t in tags}

    print("Fetching habits...", file=sys.stderr)
    habits = fetch("/tasks/user?type=habits")

    print("Fetching dailies...", file=sys.stderr)
    dailies = fetch("/tasks/user?type=dailys")

    # === imports.tf (temporary - delete after terraform apply) ===
    imports = [
        '# TEMPORARY: Delete this file after running terraform apply',
        '',
    ]

    for t in tags:
        name = to_resource_name(t["name"])
        imports.extend([
            'import {',
            f'  to = habitica_tag.{name}',
            f'  id = "{t["id"]}"',
            '}',
            '',
        ])

    for h in habits:
        name = to_resource_name(h["text"])
        imports.extend([
            'import {',
            f'  to = habitica_habit.{name}',
            f'  id = "{h["id"]}"',
            '}',
            '',
        ])

    for d in dailies:
        name = to_resource_name(d["text"])
        imports.extend([
            'import {',
            f'  to = habitica_daily.{name}',
            f'  id = "{d["id"]}"',
            '}',
            '',
        ])

    # === resources.tf (permanent) ===
    resources = [
        'terraform {',
        '  required_providers {',
        '    habitica = { source = "registry.terraform.io/inannamalick/habitica" }',
        '  }',
        '}',
        '',
        'provider "habitica" {}',
        '',
        '# === TAGS ===',
        '',
    ]

    for t in tags:
        name = to_resource_name(t["name"])
        resources.extend([
            f'resource "habitica_tag" "{name}" {{',
            f'  name = "{t["name"]}"',
            '}',
            '',
        ])

    resources.append('# === HABITS ===')
    resources.append('')
    for h in habits:
        name = to_resource_name(h["text"])
        tag_refs = [f'habitica_tag.{tag_lookup[tid]}.id' for tid in h.get("tags", []) if tid in tag_lookup]
        tags_line = f'  tags     = [{", ".join(tag_refs)}]' if tag_refs else ''
        notes = h.get("notes", "").replace('\\', '\\\\').replace('"', '\\"')

        resources.extend([
            f'resource "habitica_habit" "{name}" {{',
            f'  text     = "{h["text"].replace(chr(34), chr(92)+chr(34))}"',
            f'  notes    = "{notes}"',
            f'  up       = {str(h.get("up", True)).lower()}',
            f'  down     = {str(h.get("down", False)).lower()}',
            f'  priority = {h.get("priority", 1)}',
        ])
        if tags_line:
            resources.append(tags_line)
        resources.extend(['}', ''])

    resources.append('# === DAILIES ===')
    resources.append('')
    for d in dailies:
        name = to_resource_name(d["text"])
        tag_refs = [f'habitica_tag.{tag_lookup[tid]}.id' for tid in d.get("tags", []) if tid in tag_lookup]
        tags_line = f'  tags       = [{", ".join(tag_refs)}]' if tag_refs else ''
        notes = d.get("notes", "").replace('\\', '\\\\').replace('"', '\\"')

        # Parse start_date
        start_date = d.get("startDate", "")
        if start_date:
            start_date = start_date[:10]  # Extract YYYY-MM-DD from ISO string

        # Parse repeat config
        repeat = d.get("repeat", {})

        repeat_str = (
            f'monday = {str(repeat.get("m", True)).lower()}, '
            f'tuesday = {str(repeat.get("t", True)).lower()}, '
            f'wednesday = {str(repeat.get("w", True)).lower()}, '
            f'thursday = {str(repeat.get("th", True)).lower()}, '
            f'friday = {str(repeat.get("f", True)).lower()}, '
            f'saturday = {str(repeat.get("s", False)).lower()}, '
            f'sunday = {str(repeat.get("su", False)).lower()}'
        )

        resources.extend([
            f'resource "habitica_daily" "{name}" {{',
            f'  text       = "{d["text"].replace(chr(34), chr(92)+chr(34))}"',
            f'  notes      = "{notes}"',
            f'  frequency  = "{d.get("frequency", "weekly")}"',
            f'  every_x    = {d.get("everyX", 1)}',
            f'  priority   = {d.get("priority", 1)}',
            f'  start_date = "{start_date}"',
            f'  repeat     = {{ {repeat_str} }}',
        ])
        if tags_line:
            resources.append(tags_line)
        resources.extend(['}', ''])

    # Write files
    (output_dir / "imports.tf").write_text('\n'.join(imports))
    (output_dir / "resources.tf").write_text('\n'.join(resources))

    print(f"Generated {output_dir}/imports.tf (delete after apply)", file=sys.stderr)
    print(f"Generated {output_dir}/resources.tf (keep)", file=sys.stderr)

if __name__ == "__main__":
    main()
