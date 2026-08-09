#!/usr/bin/env python3
"""
Data analysis script - analyzes RAG retrieval results and knowledge base data

Features:
- Basic statistics (count, sum, mean, min/max)
- Numeric distribution analysis
- Text statistics (word frequency, character count)

Usage:
    # Pass JSON data via stdin
    echo '{"items": [1, 2, 3, 4, 5]}' | python analyze.py

    # Pass a file as an argument
    python analyze.py --file data.json

    # Specify the analysis type
    echo '{"items": [1, 2, 3]}' | python analyze.py --type numeric
"""

import sys
import json
import argparse
from collections import Counter

def analyze_numeric(data: list) -> dict:
    """Analyze numeric data"""
    if not data:
        return {"error": "empty dataset"}

    # Filter out numeric values
    numbers = [x for x in data if isinstance(x, (int, float))]
    if not numbers:
        return {"error": "no valid numeric data"}

    numbers.sort()
    n = len(numbers)

    result = {
        "count": n,
        "sum": sum(numbers),
        "mean": sum(numbers) / n,
        "min": min(numbers),
        "max": max(numbers),
        "median": numbers[n // 2] if n % 2 == 1 else (numbers[n // 2 - 1] + numbers[n // 2]) / 2,
    }

    # Compute the standard deviation
    mean = result["mean"]
    variance = sum((x - mean) ** 2 for x in numbers) / n
    result["std_dev"] = variance ** 0.5

    # Distribution statistics
    if n >= 5:
        result["quartiles"] = {
            "q1": numbers[n // 4],
            "q2": result["median"],
            "q3": numbers[3 * n // 4]
        }

    return result

def analyze_text(data: list) -> dict:
    """Analyze text data"""
    if not data:
        return {"error": "empty dataset"}

    texts = [str(x) for x in data if x]

    # Basic statistics
    total_chars = sum(len(t) for t in texts)
    total_words = sum(len(t.split()) for t in texts)

    # Word frequency (simple tokenization)
    all_words = []
    for text in texts:
        words = text.split()
        all_words.extend(w.strip('.,!?;:""\'()[]{}') for w in words if w.strip())

    word_freq = Counter(all_words)

    result = {
        "count": len(texts),
        "total_chars": total_chars,
        "total_words": total_words,
        "avg_chars_per_item": total_chars / len(texts) if texts else 0,
        "avg_words_per_item": total_words / len(texts) if texts else 0,
        "top_words": word_freq.most_common(20),
    }

    return result

def analyze_dict_list(data: list) -> dict:
    """Analyze a list of dict records"""
    if not data:
        return {"error": "empty dataset"}
    if not all(isinstance(x, dict) for x in data):
        return {"error": "expected a list of records"}

    result = {
        "record_count": len(data),
        "fields": {},
    }

    # Collect all fields
    all_keys = set()
    for item in data:
        all_keys.update(item.keys())

    # Analyze each field
    for key in all_keys:
        values = [item.get(key) for item in data if key in item]

        # Determine the field type
        non_null_values = [v for v in values if v is not None]
        if not non_null_values:
            result["fields"][key] = {"type": "all_null", "null_count": len(values)}
            continue

        sample = non_null_values[0]
        if isinstance(sample, (int, float)):
            field_analysis = analyze_numeric(non_null_values)
            field_analysis["type"] = "numeric"
        elif isinstance(sample, str):
            field_analysis = analyze_text(non_null_values)
            field_analysis["type"] = "text"
        else:
            field_analysis = {"type": type(sample).__name__, "count": len(non_null_values)}

        field_analysis["null_count"] = len(values) - len(non_null_values)
        result["fields"][key] = field_analysis

    return result

def analyze_mixed(data: list) -> dict:
    """Analyze mixed-type data"""
    if not data:
        return {"error": "empty dataset"}
    return {
        "count": len(data),
        "types": {type(x).__name__: sum(1 for y in data if type(y) is type(x)) for x in data},
    }

def main():
    parser = argparse.ArgumentParser(description="Data analysis tool")
    parser.add_argument("--file", "-f", help="input file path")
    parser.add_argument("--type", "-t", choices=["numeric", "text", "mixed", "auto"],
                       default="auto", help="analysis type")
    parser.add_argument("--pretty", "-p", action="store_true", help="pretty-print the output")
    args = parser.parse_args()

    # Read the input
    try:
        if args.file:
            with open(args.file, 'r', encoding='utf-8') as f:
                raw_data = f.read()
        else:
            raw_data = sys.stdin.read()

        if not raw_data.strip():
            print(json.dumps({"error": "empty input"}))
            return

        data = json.loads(raw_data)
    except json.JSONDecodeError as e:
        print(json.dumps({"error": f"JSON parse error: {str(e)}"}))
        return
    except FileNotFoundError:
        print(json.dumps({"error": f"file not found: {args.file}"}))
        return
    except Exception as e:
        print(json.dumps({"error": f"read error: {str(e)}"}))
        return

    # Extract the data
    items = None
    if isinstance(data, dict):
        if "items" in data:
            items = data["items"]
        elif "data" in data:
            items = data["data"]
        elif "results" in data:
            items = data["results"]
        else:
            # Assume the whole dict is a single record; wrap it in a list
            items = [data]
    elif isinstance(data, list):
        items = data
    else:
        print(json.dumps({"error": "unsupported data format; expected a list or a dict containing items/data/results"}))
        return

    # Analyze by type
    if args.type == "auto":
        # Auto-detect
        if items and all(isinstance(x, dict) for x in items):
            result = analyze_dict_list(items)
        elif items and all(isinstance(x, (int, float)) for x in items):
            result = analyze_numeric(items)
        elif items and all(isinstance(x, str) for x in items):
            result = analyze_text(items)
        else:
            result = analyze_mixed(items)
    elif args.type == "numeric":
        result = analyze_numeric(items)
    elif args.type == "text":
        result = analyze_text(items)
    else:
        result = analyze_mixed(items)

    # Add metadata
    output = {
        "success": True,
        "analysis": result,
        "metadata": {
            "input_type": type(data).__name__,
            "item_count": len(items) if items else 0,
            "analysis_type": args.type
        }
    }

    # Output
    indent = 2 if args.pretty else None
    print(json.dumps(output, ensure_ascii=False, indent=indent))

if __name__ == "__main__":
    main()
