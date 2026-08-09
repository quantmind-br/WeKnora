# Enable Knowledge Graph Feature Guide

This document explains how to enable and verify the Knowledge Graph (Neo4j) feature in WeKnora, guiding you through the entire process from environment preparation to frontend configuration.

## Prerequisites

- The basic deployment of the WeKnora backend and frontend has been completed.
- A working Docker/Docker Compose runtime environment is available.
- A locally or remotely accessible Neo4j service (using the project's bundled Docker Compose setup is recommended).

## Step 1: Configure Environment Variables

Add or modify the following variables in the `.env` file at the project root:

```
NEO4J_ENABLE=true
NEO4J_URI=bolt://neo4j:7687
NEO4J_USERNAME=neo4j
NEO4J_PASSWORD=your_strong_password
# Optional: NEO4J_DATABASE=neo4j
```

Notes:

- `NEO4J_ENABLE` must be set to `true` to enable the knowledge graph logic.
- `neo4j` in `NEO4J_URI` is the docker-compose service name; if you're using an external instance, replace it with the actual address.
- If your production environment uses secrets management, make sure the password is injected via a secure method.

## Step 2: Start the Neo4j Service

The project ships with a Neo4j component, which can be started directly with the following command:

```bash
docker-compose --profile neo4j up -d
```

Common verification command:

```bash
docker ps | grep neo4j
```

If you need custom mounts or memory settings, edit the `neo4j` service configuration in `docker-compose.yml`.

## Step 3: Restart WeKnora Services

To make the new environment variables take effect, restart the backend and frontend (example for reference only):

```bash
make stop && make start
# or
docker compose up -d --build
```

Make sure the backend logs show a message indicating that `neo4j` initialized successfully.

## Step 4: Enable Entity/Relation Extraction in the Frontend

1. Log in to the WeKnora frontend admin page.
2. Open "Knowledge Base Settings" or create a new knowledge base.
3. Check the "Enable Entity Extraction" and "Enable Relation Extraction" toggles.
4. Following the on-screen prompts, fill in any required LLM, callback, or model parameters (if applicable).

After saving, the system will automatically trigger entity and relation extraction tasks during the document ingestion stage.

## Step 5: Verify the Knowledge Graph

### Method 1: Neo4j Console

1. Visit `http://localhost:7474` (or the corresponding host/port).
2. Log in using the username and password from your `.env` file.
3. Run `MATCH (n) RETURN n LIMIT 50;` to check for new nodes/relationships.

### Method 2: WeKnora Interface

After uploading a document on the knowledge base or conversation page, the frontend should display a graph visualization entry point; during conversations, the system will automatically query the graph based on intent and return supplementary information.

## Troubleshooting Common Issues

- **Unable to connect to Neo4j**: Confirm network reachability, verify that `NEO4J_URI` and the username/password are correct, and check the Neo4j container logs.
- **No nodes generated**: Confirm that entity/relation extraction is enabled for the knowledge base and that the uploaded document has finished parsing; check the backend logs for any extraction task errors.
- **No query results**: Try running `CALL db.schema.visualization;` in the Neo4j console to check whether a schema exists, and re-import the document if necessary.

Once you've completed the steps above, the Knowledge Graph feature is successfully enabled, and can be combined with the RAG and Agent workflows to improve answer quality.
