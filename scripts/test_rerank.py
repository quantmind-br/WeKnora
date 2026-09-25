#!/usr/bin/env python3
from sentence_transformers import CrossEncoder
enc = CrossEncoder("BAAI/bge-reranker-v2-m3", trust_remote_code=True)
query   = "What is a light source?"
docs    = ["The sun emits visible light.", 
           "Quantum mechanics describes subatomic behavior."]
scores  = enc.predict([(query, d) for d in docs])
for i,(d,s) in enumerate(zip(docs,scores)):
    print(f"score={s:.6f} text="{d[:50]}"")
