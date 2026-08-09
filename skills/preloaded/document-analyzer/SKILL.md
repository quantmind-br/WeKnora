---
name: Document Analyzer
description: Performs deep analysis of document structure and content. Use this skill when the user needs to analyze document structure, extract key information, identify document types, assess content quality, or understand how a document is organized.
---

# Document Analyzer

Performs deep structural analysis and content understanding of documents in the knowledge base.

## Core Capabilities

1. **Structure analysis**: identify the document's heading hierarchy and organization
2. **Key information extraction**: extract core arguments, key data, and important conclusions
3. **Document type identification**: determine the document type (report, manual, paper, contract, etc.)
4. **Content quality assessment**: evaluate the document's completeness, consistency, and readability

## Analysis Process

### 1. Document Overview

Start by gathering the document's overall information:
- Document name and type
- Total pages / chunks
- Creation / update time
- Main sections / headings

### 2. Structure Analysis

Identify and describe:
- Heading hierarchy
- Section organization
- Logical flow (chronological, causal, parallel structures)

### 3. Content Extraction

Focus on:
- **Core topic**: the document's central subject
- **Key arguments**: the main points and reasoning
- **Supporting data**: important data, statistics, and facts
- **Conclusions & recommendations**: the document's conclusions or suggestions

### 4. Quality Assessment

Evaluation dimensions:
- Completeness: whether all necessary content is covered
- Consistency: whether the content is logically consistent
- Clarity: whether the expression is clear and easy to understand

## Output Format

```markdown
## Document Analysis Report

### Basic Info
- Document name: XXX
- Document type: XXX
- Structure depth: X levels

### Document Structure
1. Chapter 1: XXX
   1.1 ...
   1.2 ...
2. Chapter 2: XXX

### Core Content
- Topic: XXX
- Key arguments:
  1. ...
  2. ...
- Important data: XXX

### Analysis Conclusions
XXX
```

## Notes

- Stay objective and neutral, faithful to the original text
- Distinguish factual statements from opinions
- Mark the source location of the information
