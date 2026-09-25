# Milvus multilingual BM25 migration

Legacy WeKnora Milvus collections have only one default text analyzer, so Chinese content may produce no useful BM25 keywords. `milvus-migrate` keeps the old collection and copies the existing dense vectors into a new multilingual collection; Milvus regenerates the BM25 sparse vectors from each row's `language` field.

The dense vector metric (IP / COSINE / L2) is read from the source collection's embedding index by default. Do not change it to a value that differs from the source collection, or the same vectors will be ranked by the wrong distance.

## Usage

Make sure Milvus is running, then run from the project root:

```bash
go run ./cmd/milvus-migrate --source weknora_embeddings --target weknora_embeddings_multilingual
```

If the current shell already exports `MILVUS_ADDRESS` and `MILVUS_COLLECTION`, the corresponding flags can be omitted. Merely writing the variables in `.env.local` does not inject them into the `go run` process; when in doubt, pass the flags explicitly:

```bash
go run ./cmd/milvus-migrate \
  --address 127.0.0.1:19530 \
  --source weknora_embeddings \
  --target weknora_embeddings_multilingual
```

To double-check the metric, pass the same `--metric-type` as the source collection (or the `MILVUS_METRIC_TYPE` environment variable). The value must match the source index, otherwise the migration fails.

After the migration, set WeKnora's `MILVUS_COLLECTION` to the target prefix and restart the service, **keeping the original `MILVUS_METRIC_TYPE` unchanged**:

```dotenv
MILVUS_COLLECTION=weknora_embeddings_multilingual
```

The retrieval list matches `{prefix}_{dimension}` exactly, so `weknora_embeddings` never accidentally searches `weknora_embeddings_multilingual_*`. Only after switching the prefix do new writes and vector/keyword retrieval both go to the new collection.

The migration never deletes the old collection. Once you have confirmed that both Chinese and English BM25 retrieval work, delete the old collection with a Milvus admin tool; back it up before deleting.

By default the migration reads 64 rows per batch and only reads the fields needed to rebuild the target collection; it does not read the BM25 sparse vectors generated in the old collection. If individual chunks are very long, lower the batch size explicitly, e.g. by appending `--batch-size 32`.

## Windows PowerShell notes

WeKnora's `internal/utils` uses `pg_query_go` to parse SQL, and that dependency requires CGO. When running `go run` directly with `CGO_ENABLED=0` in the current session, you get `undefined: pg_query.Parse` or `undefined: pg_query.Deparse`.

The project ships an MSYS2 GCC, so you can temporarily enable CGO in the current PowerShell session and then run the migration. The settings below only affect the current window and do not change the system-wide Go configuration:

```powershell
# Make sure the current directory is the project root containing go.mod
if (!(Test-Path -LiteralPath '.\go.mod')) { throw 'Switch to the WeKnora project root first' }

# Use the project's bundled GCC so Go can find a C compiler
$compilerBin = Join-Path (Get-Location) '.local-tools\msys64\ucrt64\bin'
$env:CGO_ENABLED = '1'
$env:CC = Join-Path $compilerBin 'gcc.exe'
$env:CXX = Join-Path $compilerBin 'g++.exe'
$env:PATH = "$compilerBin;$env:PATH"

# Run the migration; the metric follows the source collection and the old collection is not deleted
go run ./cmd/milvus-migrate --address 127.0.0.1:19530 --source weknora_embeddings --target weknora_embeddings_multilingual
```

If the project directory has no `.local-tools\msys64\ucrt64\bin`, install a working GCC first and point `$env:CC` and `$env:CXX` at the absolute paths of the corresponding `gcc.exe` and `g++.exe`.
