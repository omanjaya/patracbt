#!/bin/bash
set -euo pipefail

cd /Users/omanjaya/project/patra

echo "=== CBT Patra Benchmark Suite ==="
echo ""

echo "--- Score Calculator Benchmarks ---"
go test -bench=. -benchmem -benchtime=3s ./internal/domain/service/
echo ""

echo "--- Redis ExamCache Benchmarks ---"
go test -bench=. -benchmem -benchtime=3s ./internal/infrastructure/cache/
echo ""

echo "=== Benchmarks Complete ==="
echo ""
echo "--- Load Test ---"
echo "Make sure the server is running on localhost:8080 before running the load test."
echo ""
echo "Usage:"
echo "  # Single-answer mode (100 students, 50 questions, 2s think time)"
echo "  go run ./tests/loadtest/ -schedule=<ID> -students=100 -questions=50 -think=2s"
echo ""
echo "  # Batch mode (10 answers per batch)"
echo "  go run ./tests/loadtest/ -schedule=<ID> -students=100 -questions=50 -batch=10 -think=2s"
echo ""
echo "  # Quick test (10 students)"
echo "  go run ./tests/loadtest/ -schedule=<ID> -students=10 -questions=10 -think=500ms"
echo ""
echo "  # Dry run (print config only)"
echo "  go run ./tests/loadtest/ -schedule=1 -dry-run"
