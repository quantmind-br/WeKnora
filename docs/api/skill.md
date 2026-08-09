Vou traduzir o documento markdown para inglês, preservando toda a estrutura.

--- DOCUMENT START ---
# Skills API

[Back to Table of Contents](./README.md)

| Method | Path      | Description                    |
| ------ | --------- | ------------------------------- |
| GET    | `/skills` | Get the list of pre-installed Skills |

## GET `/skills` - Get the List of Pre-installed Skills

Retrieves the list of all pre-installed agent skills in the system.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/skills' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "name": "web_search",
            "description": "Search the internet for the latest information"
        },
        {
            "name": "code_interpreter",
            "description": "Execute code and return the results"
        },
        {
            "name": "image_generation",
            "description": "Generate images from text descriptions"
        }
    ],
    "skills_available": true,
    "success": true
}
```

When Skills are not configured in the system, `skills_available` returns `false` and `data` is an empty array:

```json
{
    "data": [],
    "skills_available": false,
    "success": true
}
```

--- DOCUMENT END ---
