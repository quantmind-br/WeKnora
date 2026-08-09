---
name: Citation Generator
description: Generates well-formed citations automatically. Use this skill when the user needs to generate references, cite sources, attribute knowledge base content, or provide citation information.
---

# Citation Generator

Generates well-formed citation formats for knowledge base retrieval results.

## Core Capabilities

1. **Source attribution**: mark the source for every piece of knowledge used in an answer
2. **Formatted citations**: supports multiple citation formats (APA, MLA, Chicago, simplified)
3. **Reference lists**: produce a complete reference list at the end of the answer

## Citation Formats

### Simplified Format (default)

For knowledge base content, use the following format:
```
[Document Name, Page X/Paragraph X]
```

Example:
```
According to company policy [Employee Handbook 2024.pdf, Page 15], annual leave requests must be submitted...
```

### APA Format

```
Author. (Year). Title. Source.
```

### Reference List Format

At the end of the answer, list all citations as follows:

```
---
**References**

1. [1] Document A - Chapter X/Page Y
2. [2] Document B - Paragraph Z
```

## Usage Guide

1. **When retrieving content**: record the source info of each result (document name, page, chunk ID)
2. **When citing**: attribute the source immediately after using a piece of knowledge
3. **When summarizing**: list the complete references at the end of the answer

## Notes

- If the retrieval result has no page number, use the chunk or paragraph number
- Multiple citations from the same document can be merged into one
- Citations must accurately correspond to the original content; never fabricate sources
