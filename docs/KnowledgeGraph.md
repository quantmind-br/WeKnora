# WeKnora Knowledge Graph

## Quick Start

- Configure the relevant environment variables in .env
    - Enable Neo4j: `NEO4J_ENABLE=true`
    - Neo4j URI: `NEO4J_URI=bolt://neo4j:7687`
    - Neo4j username: `NEO4J_USERNAME=neo4j`
    - Neo4j password: `NEO4J_PASSWORD=password`

- Start Neo4j
```bash
docker-compose --profile neo4j up -d
```

- On the knowledge base settings page, enable entity and relationship extraction, and configure the related settings as prompted

## Generating the Graph

After uploading any document, the system automatically extracts entities and relationships and generates the corresponding knowledge graph.

![Knowledge graph example](./images/graph3.png)

## Viewing the Graph

Log in at `http://localhost:7474` and run `match (n) return (n)` to view the generated knowledge graph.

During conversations, the system automatically queries the knowledge graph and retrieves relevant knowledge.
