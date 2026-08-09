#!/usr/bin/env python3
"""OpenAI-compatible /v1/rerank HTTP micro-service using HuggingFace Sentence Transformers."""
from __future__ import annotations

import argparse
import json
import os
import sys
import time
from typing import Any, Dict, List

try:
    from fastapi import FastAPI, Request
    from fastapi.responses import JSONResponse
except ImportError:
    raise RuntimeError("fastapi not installed — run: uv pip install fastapi[standard]")

_cross_encoder = None


def load_model(model_name: str):
    global _cross_encoder
    if _cross_encoder is None:
        try:
            cache_dir = os.environ.get(
                "HF_HOME", 
                os.path.join(os.path.expanduser("~"), ".cache", "huggingface")
            )
            # sentence-transformers also respects SENTENCE_TRANSFORMERS_HOME for its own cache.
            hf_home  = os.environ.get("SENTENCE_TRANSFORMERS_HOME", cache_dir)
            _cross_encoder = __import__("sentence_transformers", fromlist=["CrossEncoder"]).CrossEncoder(
                model_name_or_path=model_name,
                trust_remote_code=True,
                cache_folder=hf_home,
            )
        except Exception as exc:   # pragma: no cover - defensive logging only
            raise RuntimeError(f"Failed to load cross-encoder {{model_name}}: {{exc}}")
    return _cross_encoder


app = FastAPI(title="Local Reranker Service (bge-reranker-v2-m3)")


@app.post("/v1/rerank")
async def rerank(request: Request) -> JSONResponse:
    body = await request.json()

    query       : str          = body.get("query", "")
    documents_raw = body.get("documents", [])
    top_n       : int          = max(int(body.get("top_n", len(documents_raw))), 0) or len(documents_raw)

    if not query and not documents_raw:
        return JSONResponse(status_code=400, content={
            "error": {"message": "Missing 'query' or 'documents'"}})

    # Accept either plain strings [{"text":"..."}] or dicts — extract the text.
    documents: List[str] = []
    for item in documents_raw:
        if isinstance(item, dict):
            documents.append(str(item.get("text", "")))
        else:
            documents.append(str(item))

    encoder   = app.state.model
    scores    = encoder.predict([(query, d) for d in documents])
    ranked_idx = sorted(range(len(scores)), key=lambda i: float(scores[i]), reverse=True)

    results: List[Dict[str, object]] = []
    seen_ids: set[int] = set()
    count = 0
    for idx in ranked_idx:
        doc_text = documents[idx]
        score_val = round(float(scores[idx]), 6)
        tid = hash(doc_text) & 0xFFFFFFFF
        if tid in seen_ids:
            continue
        seen_ids.add(tid)
        results.append({
            "index"           : idx,
            "document"        : {"text": doc_text},
            "relevance_score" : score_val,
        })
        count += 1
        if count >= top_n:
            break

    total_tokens = sum(max(1, len(d.split())) for d in [query] + documents[:top_n * 2])

    return JSONResponse(content={
        "id"     : f"rerank-{{int(time.time()*1000)}}",
        "object" : "list",
        "model"  : body.get("model", args.model),
        "usage"  : {"total_tokens": int(total_tokens)},
        "results": results,
    })


@app.get("/v1/models")
async def list_models():
    name = args.model.rsplit("/", maxsplit=1)[-1]
    return JSONResponse(content={
        "data": [{
            "id"       : name,
            "object"   : "model",
            "owned_by" : "huggingface/sentence-transformers",
            "name"     : name,
            "display_name": name,
        }],
        "object": "list",
    })


if __name__ == "__main__":
    import uvicorn
    parser = argparse.ArgumentParser(description="Local OpenAI-compatible reranker service")
    parser.add_argument("--model", default="BAAI/bge-reranker-v2-m3", help="HuggingFace model id or local path")
    parser.add_argument("--port", type=int, default=4322)
    parser.add_argument("--host", default="127.0.0.1")
    global args
    args = parser.parse_args()

    print(f"[reranker-service] Loading {{args.model}} ...", file=sys.stderr, flush=True)
    load_model(args.model)
    print(f"[reranker-service] Ready at http://{{args.host}}:{{args.port}}/", file=sys.stderr, flush=True)
    uvicorn.run(app, host=args.host, port=args.port, log_level="warning")
