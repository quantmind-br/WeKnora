#!/usr/bin/env python3
"""
RAG retrieval results → OpenMAIC Requirement converter

Converts WeKnora RAG retrieval results into a structured OpenMAIC course
generation requirement description. This script only transforms data and
makes no network calls.

Usage:
  echo '{"chunks": [...], "audience": "beginners"}' | python scripts/rag-to-requirement.py
  python scripts/rag-to-requirement.py --file results.json

Input format (JSON):
{
  "chunks": [
    {
      "document_name": "doc-A.pdf",
      "content": "document content snippet...",
      "metadata": {"page": 5, "section": "Chapter 3"}
    }
  ],
  "query": "user's original query (optional)",
  "audience": "target audience description (optional, default 'learners in the relevant field')",
  "depth": "teaching depth: beginner|intermediate|advanced (optional, default 'intermediate')",
  "language": "zh-CN|en-US (optional, default 'en-US')",
  "focus_areas": ["focus area 1", "focus area 2"]  # optional
}

Output format (JSON):
{
  "requirement": "structured teaching requirement description",
  "metadata": {
    "source_documents": ["doc-A.pdf", "doc-B.pdf"],
    "total_chunks": 5,
    "audience": "beginners",
    "depth": "intermediate",
    "language": "en-US"
  }
}
"""

import json
import sys
from typing import Any

def extract_key_topics(chunks: list[dict], max_topics: int = 5) -> list[str]:
    """Extract key topics from document chunks (based on content summaries and section info)."""
    topics = []
    seen = set()

    for chunk in chunks:
        metadata = chunk.get("metadata", {})
        # Prefer section/chapter info
        for key in ("section", "chapter", "heading", "title"):
            if key in metadata and metadata[key] not in seen:
                topics.append(metadata[key])
                seen.add(metadata[key])
                if len(topics) >= max_topics:
                    return topics

    # If section info is insufficient, extract from the first 50 characters of content
    for chunk in chunks:
        content = chunk.get("content", "")[:50].strip()
        if content and content not in seen:
            topics.append(content + "...")
            seen.add(content)
            if len(topics) >= max_topics:
                break

    return topics

def build_requirement(data: dict[str, Any]) -> str:
    """Build an OpenMAIC requirement string from the RAG results."""
    chunks = data.get("chunks", [])
    query = data.get("query", "")
    audience = data.get("audience", "learners in the relevant field")
    depth = data.get("depth", "intermediate")
    language = data.get("language", "en-US")
    focus_areas = data.get("focus_areas", [])

    depth_map = {
        "beginner": "beginner",
        "intermediate": "intermediate",
        "advanced": "advanced",
    }
    depth_en = depth_map.get(depth, "intermediate")

    # Extract document names
    source_docs = list({c.get("document_name", "unknown document") for c in chunks})

    # Extract key topics
    key_topics = extract_key_topics(chunks)

    # Build the requirement
    parts = []

    # Opening: based on what content, for whom, create what course
    if query:
        parts.append(f"Based on the following knowledge base content, create a {depth_en} course for {audience}.")
        parts.append(f"User's original request: {query}")
    else:
        parts.append(f"Based on the following knowledge base content, create a {depth_en} course for {audience}.")

    # Content sources
    if source_docs:
        docs_str = ", ".join(source_docs[:5])
        if len(source_docs) > 5:
            docs_str += f" and {len(source_docs)} more documents"
        parts.append(f"\nContent sources: {docs_str}")

    # Key topics
    if key_topics:
        parts.append("\nCore topics:")
        for i, topic in enumerate(key_topics, 1):
            parts.append(f"  {i}. {topic}")

    # Focus areas
    if focus_areas:
        parts.append("\nKey coverage:")
        for area in focus_areas:
            parts.append(f"  - {area}")

    # Language directive
    if language == "zh-CN":
        parts.append("\nPlease generate the course content in Chinese.")
    else:
        parts.append("\nPlease generate the course content in English.")

    return "\n".join(parts)

def process(input_data: dict[str, Any]) -> dict[str, Any]:
    """Main processing function."""
    chunks = input_data.get("chunks", [])
    if not chunks:
        return {
            "requirement": input_data.get("query", ""),
            "metadata": {
                "source_documents": [],
                "total_chunks": 0,
                "error": "no retrieval results provided; using the raw query as the requirement",
            },
        }

    requirement = build_requirement(input_data)

    source_docs = list({c.get("document_name", "unknown document") for c in chunks})

    return {
        "requirement": requirement,
        "metadata": {
            "source_documents": source_docs,
            "total_chunks": len(chunks),
            "audience": input_data.get("audience", "learners in the relevant field"),
            "depth": input_data.get("depth", "intermediate"),
            "language": input_data.get("language", "en-US"),
        },
    }

def main() -> None:
    import argparse

    parser = argparse.ArgumentParser(description="Convert RAG retrieval results into an OpenMAIC requirement")
    parser.add_argument("--file", "-f", help="input JSON file path")
    args = parser.parse_args()

    # Read the input
    if args.file:
        with open(args.file, "r", encoding="utf-8") as f:
            input_data = json.load(f)
    else:
        input_text = sys.stdin.read()
        if not input_text.strip():
            print(
                "Error: no input data provided. Usage:\n"
                "  echo '{\"chunks\": [...]}' | python rag-to-requirement.py\n"
                "  python rag-to-requirement.py --file input.json",
                file=sys.stderr,
            )
            sys.exit(1)
        try:
            input_data = json.loads(input_text)
        except json.JSONDecodeError as e:
            print(f"Error: failed to parse input JSON: {e}", file=sys.stderr)
            sys.exit(1)

    # Process and output
    result = process(input_data)
    print(json.dumps(result, ensure_ascii=False, indent=2))

if __name__ == "__main__":
    main()
