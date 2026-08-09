---
name: Data Processor
description: Data processing and analysis skill. Use this skill when the user needs data analysis, statistical calculations, format conversion, data extraction, or report generation on knowledge base retrieval results. Supports advanced data processing through Python script execution.
---

# Data Processor

Enterprise-grade knowledge base data processing and analysis skill for handling RAG retrieval results and performing data analysis tasks.

## Core Capabilities

1. **Data analysis**: perform statistical analysis on retrieved document data
2. **Format conversion**: convert between JSON, CSV, Markdown and other formats
3. **Data extraction**: extract structured information from unstructured text
4. **Report generation**: generate data analysis reports and summaries

## Use Cases

Use this skill when the user's request involves:
- "analyze this data", "run statistics", "calculate totals/averages"
- "convert to JSON/CSV format"
- "extract key information", "organize into a table"
- "generate a report", "summarize the data"

## Available Scripts

### 1. analyze.py - Data Analysis Script

Analyzes input JSON data and produces a statistical report.

**Command-line usage** (for reference):
```bash
# Pass JSON data via stdin
echo '{"items": [1, 2, 3, 4, 5]}' | python scripts/analyze.py

# Or pass a file path (the file must actually exist)
python scripts/analyze.py --file data.json
```

**When using the execute_skill_script tool**:
- If you have in-memory data (such as a JSON string), pass it via the `input` parameter, not `args`
- The `--file` parameter is only for reading files that already exist in the skill directory, not for passing in-memory data

```json
// ✅ Correct: pass data via input
{
  "skill_name": "data-processor",
  "script_path": "scripts/analyze.py",
  "input": "{\"items\": [1, 2, 3], \"query\": \"statistical analysis\"}"
}

// ❌ Wrong: --file requires a real file path and cannot be used alone
{
  "skill_name": "data-processor",
  "script_path": "scripts/analyze.py",
  "args": ["--file"],
  "input": "{...}"
}
```

**Input format**:
```json
{
  "items": [array of data items],
  "query": "optional query description"
}
```

**Output**: statistical results in JSON, including count, sum, average, etc.

### 2. format_converter.py - Format Conversion Script

Converts data between JSON, CSV, and Markdown tables.

**Usage**:
```bash
# JSON to CSV
echo '[{"name": "A", "value": 1}]' | python scripts/format_converter.py --to csv

# JSON to Markdown table
echo '[{"name": "A", "value": 1}]' | python scripts/format_converter.py --to markdown

# CSV to JSON
echo 'name,value\nA,1' | python scripts/format_converter.py --from csv --to json
```

### 3. extract_info.py - Information Extraction Script

Extracts structured information from text (numbers, dates, keywords, etc.).

**Usage**:
```bash
echo "2024 sales reached 1 million yuan, up 15% year over year" | python scripts/extract_info.py
```

**Output**:
```json
{
  "numbers": ["1", "15"],
  "dates": ["2024"],
  "percentages": ["15%"],
  "amounts": ["1 million yuan"]
}
```

## Processing Workflow

### Analyzing RAG Retrieval Results

When you need to analyze knowledge base retrieval results:

1. Collect the retrieved document snippets
2. Extract key data points
3. Use `analyze.py` for statistics
4. Organize and present the analysis results

**Example**:
```
User: "Help me summarize all product sales data mentioned in the knowledge base"

Steps:
1. Use knowledge_search to retrieve relevant documents
2. Organize the data into JSON format
3. Call execute_skill_script:
   - skill_name: "data-processor"
   - script_path: "scripts/analyze.py"
   - Pass the data via stdin
4. Parse the output and generate a report
```

### Data Format Conversion

When the user needs output in a specific format:

1. Organize the data into standard JSON format
2. Use `format_converter.py` to convert
3. Return the target-format result

## Best Practices

1. **Data preprocessing**: ensure the data format is correct before calling scripts
2. **Error handling**: check script execution results and handle exceptions
3. **Result validation**: verify the reasonableness of the output
4. **Incremental processing**: process large datasets in batches

## Output Format

Example analysis result:
```markdown
## Data Analysis Report

### Basic Statistics
- Data count: 50
- Total sum: 1,234,567
- Average: 24,691.34
- Max: 99,999
- Min: 100

### Distribution
| Range | Count | Percentage |
|------|------|------|
| 0-1000 | 10 | 20% |
| 1000-10000 | 25 | 50% |
| >10000 | 15 | 30% |

### Conclusion
Based on the data analysis, XXX...
```

## Notes

- Scripts run in a Docker sandbox for safe isolation
- Execution timeout defaults to 60 seconds
- Input data size is limited; split large files into batches
- Script output is JSON for easy downstream processing
