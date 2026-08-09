#!/usr/bin/env python3
"""
Format converter script - converts between JSON / CSV / Markdown

Usage:
    # JSON to CSV
    echo '[{"name": "A", "value": 1}]' | python format_converter.py --to csv

    # JSON to Markdown table
    echo '[{"name": "A", "value": 1}]' | python format_converter.py --to markdown

    # CSV to JSON
    cat data.csv | python format_converter.py --from csv --to json
"""

import sys
import json
import argparse
import csv
import io

def json_to_csv(data: list) -> str:
    """Convert a JSON list to CSV"""
    if not data:
        return ""

    if not all(isinstance(x, dict) for x in data):
        raise ValueError("JSON data must be a list of dicts")

    # Collect all field names
    fieldnames = []
    for item in data:
        for key in item.keys():
            if key not in fieldnames:
                fieldnames.append(key)

    output = io.StringIO()
    writer = csv.DictWriter(output, fieldnames=fieldnames)
    writer.writeheader()
    writer.writerows(data)

    return output.getvalue()

def csv_to_json(csv_text: str) -> list:
    """Convert CSV to a JSON list"""
    reader = csv.DictReader(io.StringIO(csv_text))
    return list(reader)

def json_to_markdown(data: list) -> str:
    """Convert a JSON list to a Markdown table"""
    if not data:
        return ""

    if not all(isinstance(x, dict) for x in data):
        raise ValueError("JSON data must be a list of dicts")

    # Collect all field names
    fieldnames = []
    for item in data:
        for key in item.keys():
            if key not in fieldnames:
                fieldnames.append(key)

    # Build the header
    lines = []
    lines.append("| " + " | ".join(fieldnames) + " |")
    lines.append("| " + " | ".join(["---"] * len(fieldnames)) + " |")

    # Build data rows
    for item in data:
        row = []
        for field in fieldnames:
            value = item.get(field, "")
            # Escape Markdown special characters
            str_value = str(value) if value is not None else ""
            str_value = str_value.replace("|", "\\|")
            row.append(str_value)
        lines.append("| " + " | ".join(row) + " |")

    return "\n".join(lines)

def markdown_to_json(md_text: str) -> list:
    """Convert a Markdown table to a JSON list"""
    lines = [l.strip() for l in md_text.strip().splitlines() if l.strip()]

    if not lines:
        return []

    # Parse the header row
    header_line = lines[0]
    headers = [h.strip() for h in header_line.strip("|").split("|")]

    # Skip the separator row
    data_lines = lines[2:] if len(lines) > 2 else []

    # Parse the data
    result = []
    for line in data_lines:
        if not line.startswith("|"):
            continue
        values = [v.strip() for v in line.strip("|").split("|")]
        item = {}
        for i, header in enumerate(headers):
            if i < len(values):
                item[header] = values[i]
        result.append(item)

    return result

def detect_format(text: str) -> str:
    """Automatically detect the input format"""
    text = text.strip()

    if text.startswith("[") or text.startswith("{"):
        return "json"
    elif text.startswith("|"):
        return "markdown"
    elif "," in text.split("\n")[0]:
        return "csv"
    else:
        return "unknown"

def main():
    parser = argparse.ArgumentParser(description="Data format conversion tool")
    parser.add_argument("--from", "-f", dest="from_format",
                       choices=["json", "csv", "markdown", "auto"],
                       default="auto", help="input format")
    parser.add_argument("--to", "-t", dest="to_format",
                       choices=["json", "csv", "markdown"],
                       required=True, help="output format")
    parser.add_argument("--pretty", "-p", action="store_true", help="pretty-print the output")
    args = parser.parse_args()

    # Read the input
    try:
        raw_input = sys.stdin.read()
        if not raw_input.strip():
            print(json.dumps({"error": "empty input"}))
            return
    except Exception as e:
        print(json.dumps({"error": f"read error: {str(e)}"}))
        return

    # Detect the input format
    from_format = args.from_format
    if from_format == "auto":
        from_format = detect_format(raw_input)
        if from_format == "unknown":
            print(json.dumps({"error": "could not auto-detect the input format"}))
            return

    # Convert to the intermediate format (JSON list)
    try:
        if from_format == "json":
            data = json.loads(raw_input)
            if isinstance(data, dict):
                # Try to extract a list
                if "items" in data:
                    data = data["items"]
                elif "data" in data:
                    data = data["data"]
                elif "results" in data:
                    data = data["results"]
                else:
                    data = [data]
            if not isinstance(data, list):
                data = [data]
        elif from_format == "csv":
            data = csv_to_json(raw_input)
        elif from_format == "markdown":
            data = markdown_to_json(raw_input)
        else:
            print(json.dumps({"error": f"unsupported input format: {from_format}"}))
            return
    except Exception as e:
        print(json.dumps({"error": f"failed to parse the input: {str(e)}"}))
        return

    # Convert to the target format
    try:
        if args.to_format == "json":
            indent = 2 if args.pretty else None
            output = json.dumps(data, ensure_ascii=False, indent=indent)
        elif args.to_format == "csv":
            output = json_to_csv(data)
        elif args.to_format == "markdown":
            output = json_to_markdown(data)
        else:
            print(json.dumps({"error": f"unsupported output format: {args.to_format}"}))
            return

        print(output)
    except Exception as e:
        print(json.dumps({"error": f"conversion failed: {str(e)}"}))
        return

if __name__ == "__main__":
    main()
