#!/usr/bin/env python3
"""
Information extraction script - extracts structured information from text

Extracts:
- Numbers
- Dates
- Percentages
- Amounts
- Emails
- URLs
- Phone numbers

Usage:
    echo "2024 sales reached 1 million yuan, up 15% year over year" | python extract_info.py

    # Specify the extraction types
    echo "Contact me: test@example.com or 13800138000" | python extract_info.py --types email,phone
"""

import sys
import json
import argparse
import re

def extract_numbers(text: str) -> list:
    """Extract numbers"""
    # Match integers and decimals
    pattern = r'-?\d+(?:\.\d+)?'
    numbers = re.findall(pattern, text)
    # Convert to numeric values
    result = []
    for n in numbers:
        try:
            if '.' in n:
                result.append(float(n))
            else:
                result.append(int(n))
        except ValueError:
            result.append(n)
    return result

def extract_dates(text: str) -> list:
    """Extract dates"""
    patterns = [
        r'\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日]?',  # 2024-01-01 or 2024年1月1日
        r'\d{4}[-/年]\d{1,2}[月]?',                 # 2024-01 or 2024年1月
        r'\d{4}年',                                  # 2024年
        r'\d{1,2}[-/月]\d{1,2}[日]?',              # 01-01 or 1月1日
    ]

    dates = []
    for pattern in patterns:
        matches = re.findall(pattern, text)
        dates.extend(matches)

    return list(set(dates))

def extract_percentages(text: str) -> list:
    """Extract percentages"""
    pattern = r'-?\d+(?:\.\d+)?%'
    return re.findall(pattern, text)

def extract_amounts(text: str) -> list:
    """Extract amounts"""
    patterns = [
        r'[¥$€£]\s*\d+(?:,\d{3})*(?:\.\d+)?',      # ¥100.00
        r'\d+(?:,\d{3})*(?:\.\d+)?\s*[元万亿美金]', # 100万元
        r'\d+(?:\.\d+)?[百千万亿]+[元]?',            # 100万
    ]

    amounts = []
    for pattern in patterns:
        matches = re.findall(pattern, text)
        amounts.extend(matches)

    return list(set(amounts))

def extract_emails(text: str) -> list:
    """Extract email addresses"""
    pattern = r'[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}'
    return re.findall(pattern, text)

def extract_urls(text: str) -> list:
    """Extract URLs"""
    pattern = r'https?://[\w\-._~:/?#\[\]@!$&\'()*+,;=%]+'
    return re.findall(pattern, text)

def extract_phones(text: str) -> list:
    """Extract phone numbers"""
    patterns = [
        r'1[3-9]\d{9}',                           # mobile number
        r'\d{3,4}[-\s]?\d{7,8}',                  # landline
        r'\+\d{1,3}[-\s]?\d{10,12}',             # international number
    ]

    phones = []
    for pattern in patterns:
        matches = re.findall(pattern, text)
        phones.extend(matches)

    return list(set(phones))

def extract_keywords(text: str, min_len: int = 2) -> list:
    """Extract keywords (Chinese and English)"""
    # Chinese keywords
    chinese_pattern = r'[\u4e00-\u9fa5]{2,}'
    chinese_words = re.findall(chinese_pattern, text)

    # English keywords
    english_pattern = r'[a-zA-Z]{3,}'
    english_words = re.findall(english_pattern, text)

    # Count word frequency
    from collections import Counter
    words = chinese_words + [w.lower() for w in english_words]

    # Filter stop words
    stopwords = {'的', '是', '在', '了', '和', '与', '或', '为', '有', '这', '那', '等',
                 'the', 'is', 'are', 'was', 'were', 'and', 'or', 'for', 'with', 'this'}
    words = [w for w in words if w not in stopwords and len(w) >= min_len]

    word_freq = Counter(words)
    return [{"word": w, "count": c} for w, c in word_freq.most_common(20)]

def main():
    parser = argparse.ArgumentParser(description="Information extraction tool")
    parser.add_argument("--types", "-t",
                       help="types to extract, comma-separated (numbers,dates,percentages,amounts,emails,urls,phones,keywords)")
    parser.add_argument("--pretty", "-p", action="store_true", help="pretty-print the output")
    args = parser.parse_args()

    # Read the input
    try:
        text = sys.stdin.read()
        if not text.strip():
            print(json.dumps({"error": "empty input"}))
            return
    except Exception as e:
        print(json.dumps({"error": f"read error: {str(e)}"}))
        return

    # Determine which types to extract
    all_types = ["numbers", "dates", "percentages", "amounts", "emails", "urls", "phones", "keywords"]
    if args.types:
        extract_types = [t.strip().lower() for t in args.types.split(",")]
    else:
        extract_types = all_types

    # Extract the information
    result = {
        "success": True,
        "text_length": len(text),
        "extracted": {}
    }

    extractors = {
        "numbers": extract_numbers,
        "dates": extract_dates,
        "percentages": extract_percentages,
        "amounts": extract_amounts,
        "emails": extract_emails,
        "urls": extract_urls,
        "phones": extract_phones,
        "keywords": extract_keywords,
    }

    for ext_type in extract_types:
        if ext_type in extractors:
            try:
                extracted = extractors[ext_type](text)
                if extracted:
                    result["extracted"][ext_type] = extracted
            except Exception as e:
                result["extracted"][ext_type] = {"error": str(e)}

    # Summary
    result["summary"] = {
        "total_extractions": sum(len(v) if isinstance(v, list) else 0
                                  for v in result["extracted"].values()),
        "types_found": list(result["extracted"].keys())
    }

    # Output
    indent = 2 if args.pretty else None
    print(json.dumps(result, ensure_ascii=False, indent=indent))

if __name__ == "__main__":
    main()
