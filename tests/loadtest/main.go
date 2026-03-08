// Load test for CBT Patra answer submission flow.
//
// Simulates N concurrent students taking an exam: login, start session, submit
// answers one-by-one (with jittered think time), then optionally submit in
// batches.
//
// Usage:
//
//	go run ./tests/loadtest/ -students=100 -questions=50 -think=2s
//	go run ./tests/loadtest/ -students=100 -questions=50 -batch=10 -think=2s
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var (
	baseURL     = flag.String("url", "http://localhost:8080", "API base URL")
	numStudents = flag.Int("students", 100, "Number of concurrent students")
	numQuestions = flag.Int("questions", 50, "Questions per exam")
	scheduleID  = flag.Int("schedule", 0, "Exam schedule ID (required)")
	thinkTime   = flag.Duration("think", 3*time.Second, "Base think time between answers")
	batchSize   = flag.Int("batch", 0, "Batch size (0 = single-answer mode)")
	prefix      = flag.String("prefix", "student", "Username prefix (student1, student2, ...)")
	password    = flag.String("password", "password", "Password for all student accounts")
	timeout     = flag.Duration("timeout", 10*time.Second, "HTTP request timeout")
	skipLogin   = flag.Bool("skip-login", false, "Skip login phase (use token-file instead)")
	tokenFile   = flag.String("token-file", "", "File with pre-generated tokens (one per line)")
	dryRun      = flag.Bool("dry-run", false, "Print config and exit without running")
)

// Stats collects request-level metrics across all goroutines.
type Stats struct {
	totalRequests  atomic.Int64
	totalErrors    atomic.Int64
	totalLatencyNs atomic.Int64
	maxLatencyNs   atomic.Int64

	latencies []int64
	mu        sync.Mutex
}

func (s *Stats) record(latency time.Duration, isError bool) {
	ns := int64(latency)
	s.totalRequests.Add(1)
	s.totalLatencyNs.Add(ns)
	if isError {
		s.totalErrors.Add(1)
	}

	// Track max using CAS loop
	for {
		old := s.maxLatencyNs.Load()
		if ns <= old || s.maxLatencyNs.CompareAndSwap(old, ns) {
			break
		}
	}

	// Store for percentile calculation
	s.mu.Lock()
	s.latencies = append(s.latencies, ns)
	s.mu.Unlock()
}

func (s *Stats) percentile(p float64) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.latencies) == 0 {
		return 0
	}
	sorted := make([]int64, len(s.latencies))
	copy(sorted, s.latencies)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := int(float64(len(sorted)-1) * p)
	return time.Duration(sorted[idx])
}

func main() {
	flag.Parse()

	fmt.Println("========================================")
	fmt.Println("  CBT Patra Load Test")
	fmt.Println("========================================")
	fmt.Printf("Target:      %s\n", *baseURL)
	fmt.Printf("Students:    %d\n", *numStudents)
	fmt.Printf("Questions:   %d\n", *numQuestions)
	fmt.Printf("Think time:  %s\n", *thinkTime)
	fmt.Printf("Batch size:  %d (0=single)\n", *batchSize)
	fmt.Printf("Schedule ID: %d\n", *scheduleID)
	fmt.Println("========================================")

	if *dryRun {
		fmt.Println("[dry-run] Exiting.")
		return
	}

	if *scheduleID == 0 {
		fmt.Println("\nERROR: -schedule flag is required.")
		fmt.Println("Create an exam schedule first, then pass its ID.")
		fmt.Println("Example: go run ./tests/loadtest/ -schedule=1 -students=100")
		return
	}

	stats := &Stats{}

	// Phase 1: Login all students
	tokens := make([]string, *numStudents)
	fmt.Printf("\nPhase 1: Logging in %d students...\n", *numStudents)
	loginStart := time.Now()

	var wg sync.WaitGroup
	loginErrors := atomic.Int64{}
	for i := 0; i < *numStudents; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			username := fmt.Sprintf("%s%d", *prefix, idx+1)
			token, err := login(username, *password)
			if err != nil {
				loginErrors.Add(1)
				if idx < 5 { // only print first few errors
					fmt.Printf("  [WARN] Login failed for %s: %v\n", username, err)
				}
				return
			}
			tokens[idx] = token
		}(i)
	}
	wg.Wait()
	loginElapsed := time.Since(loginStart)
	loggedIn := int64(*numStudents) - loginErrors.Load()
	fmt.Printf("  Logged in: %d/%d (%.1fs)\n", loggedIn, *numStudents, loginElapsed.Seconds())

	if loggedIn == 0 {
		fmt.Println("\nERROR: No students could log in. Check server and credentials.")
		return
	}

	// Phase 2: Start exam sessions
	sessionIDs := make([]int, *numStudents)
	fmt.Printf("\nPhase 2: Starting exam sessions (schedule=%d)...\n", *scheduleID)
	sessionStart := time.Now()
	sessionErrors := atomic.Int64{}

	for i := 0; i < *numStudents; i++ {
		if tokens[i] == "" {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sid, err := startExam(tokens[idx], *scheduleID)
			if err != nil {
				sessionErrors.Add(1)
				if idx < 5 {
					fmt.Printf("  [WARN] Start exam failed for student%d: %v\n", idx+1, err)
				}
				return
			}
			sessionIDs[idx] = sid
		}(i)
	}
	wg.Wait()
	sessionElapsed := time.Since(sessionStart)
	activeSessions := int64(0)
	for _, sid := range sessionIDs {
		if sid > 0 {
			activeSessions++
		}
	}
	fmt.Printf("  Sessions started: %d/%d (%.1fs)\n", activeSessions, loggedIn, sessionElapsed.Seconds())

	if activeSessions == 0 {
		fmt.Println("\nERROR: No exam sessions could be started. Check schedule ID and student enrollment.")
		return
	}

	// Phase 3: Submit answers concurrently
	mode := "single"
	if *batchSize > 0 {
		mode = fmt.Sprintf("batch(%d)", *batchSize)
	}
	fmt.Printf("\nPhase 3: Submitting answers (%s mode)...\n", mode)
	answerStart := time.Now()

	for i := 0; i < *numStudents; i++ {
		if tokens[i] == "" || sessionIDs[i] == 0 {
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if *batchSize > 0 {
				simulateStudentBatch(tokens[idx], sessionIDs[idx], *numQuestions, *batchSize, *thinkTime, stats)
			} else {
				simulateStudent(tokens[idx], sessionIDs[idx], *numQuestions, *thinkTime, stats)
			}
		}(i)
	}
	wg.Wait()
	answerElapsed := time.Since(answerStart)

	// Report
	printReport(stats, answerElapsed, int(activeSessions))
}

// simulateStudent sends answers one at a time with jittered think time.
func simulateStudent(token string, sessionID, numQ int, think time.Duration, stats *Stats) {
	client := &http.Client{Timeout: *timeout}

	for q := 1; q <= numQ; q++ {
		// Jittered think time: 50%-150% of base
		jitter := time.Duration(float64(think) * (0.5 + rand.Float64()))
		time.Sleep(jitter)

		body, _ := json.Marshal(map[string]interface{}{
			"question_id": q,
			"answer":      json.RawMessage(fmt.Sprintf(`{"option_index":%d}`, rand.Intn(4))),
			"is_flagged":  rand.Float32() < 0.1,
		})

		url := fmt.Sprintf("%s/api/v1/exam/sessions/%d/answers", *baseURL, sessionID)
		req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		t0 := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(t0)

		isErr := err != nil
		if resp != nil {
			if resp.StatusCode != 200 {
				isErr = true
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		stats.record(latency, isErr)
	}
}

// simulateStudentBatch accumulates answers into batches and sends them via the batch endpoint.
func simulateStudentBatch(token string, sessionID, numQ, batchSz int, think time.Duration, stats *Stats) {
	client := &http.Client{Timeout: *timeout}
	batch := make([]map[string]interface{}, 0, batchSz)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		body, _ := json.Marshal(map[string]interface{}{
			"answers": batch,
		})

		url := fmt.Sprintf("%s/api/v1/exam/sessions/%d/answers/batch", *baseURL, sessionID)
		req, _ := http.NewRequest("POST", url, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		t0 := time.Now()
		resp, err := client.Do(req)
		latency := time.Since(t0)

		isErr := err != nil
		if resp != nil {
			if resp.StatusCode != 200 {
				isErr = true
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		stats.record(latency, isErr)
		batch = batch[:0]
	}

	for q := 1; q <= numQ; q++ {
		// Jittered think time per question (shorter for batch)
		jitter := time.Duration(float64(think) * (0.3 + rand.Float64()*0.4))
		time.Sleep(jitter)

		batch = append(batch, map[string]interface{}{
			"question_id": q,
			"answer":      json.RawMessage(fmt.Sprintf(`{"option_index":%d}`, rand.Intn(4))),
			"is_flagged":  rand.Float32() < 0.1,
		})

		if len(batch) >= batchSz {
			flush()
		}
	}
	// Flush remaining
	flush()
}

// login authenticates a student and returns the access token.
func login(username, pwd string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"username": username,
		"password": pwd,
	})
	client := &http.Client{Timeout: *timeout}
	resp, err := client.Post(*baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode error: %w", err)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response: no data field")
	}
	token, ok := data["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response: no access_token")
	}
	return token, nil
}

// startExam starts an exam session for the given schedule.
func startExam(token string, schedID int) (int, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"exam_schedule_id": schedID,
	})
	client := &http.Client{Timeout: *timeout}
	req, _ := http.NewRequest("POST", *baseURL+"/api/v1/exam/sessions/start", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("status %d: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode error: %w", err)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("unexpected response: no data field")
	}

	// Try session_id or id
	for _, key := range []string{"session_id", "id"} {
		if v, ok := data[key]; ok {
			switch id := v.(type) {
			case float64:
				return int(id), nil
			case json.Number:
				n, _ := id.Int64()
				return int(n), nil
			}
		}
	}
	return 0, fmt.Errorf("unexpected response: no session id in %v", data)
}

// printReport outputs a summary of the load test results.
func printReport(stats *Stats, elapsed time.Duration, activeStudents int) {
	total := stats.totalRequests.Load()
	errors := stats.totalErrors.Load()
	avgLatency := time.Duration(0)
	if total > 0 {
		avgLatency = time.Duration(stats.totalLatencyNs.Load() / total)
	}
	maxLatency := time.Duration(stats.maxLatencyNs.Load())
	rps := float64(total) / elapsed.Seconds()

	p50 := stats.percentile(0.50)
	p95 := stats.percentile(0.95)
	p99 := stats.percentile(0.99)

	errorRate := float64(0)
	if total > 0 {
		errorRate = float64(errors) / float64(total) * 100
	}

	fmt.Println()
	fmt.Println("+======================================+")
	fmt.Println("|       LOAD TEST RESULTS              |")
	fmt.Println("+======================================+")
	fmt.Printf("| Active Students: %d\n", activeStudents)
	fmt.Printf("| Duration:        %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("| Total Requests:  %d\n", total)
	fmt.Printf("| Errors:          %d (%.2f%%)\n", errors, errorRate)
	fmt.Println("+--------------------------------------+")
	fmt.Printf("| Avg Latency:     %s\n", avgLatency.Round(time.Microsecond))
	fmt.Printf("| P50 Latency:     %s\n", p50.Round(time.Microsecond))
	fmt.Printf("| P95 Latency:     %s\n", p95.Round(time.Microsecond))
	fmt.Printf("| P99 Latency:     %s\n", p99.Round(time.Microsecond))
	fmt.Printf("| Max Latency:     %s\n", maxLatency.Round(time.Microsecond))
	fmt.Println("+--------------------------------------+")
	fmt.Printf("| Throughput:      %.0f req/s\n", rps)
	fmt.Println("+======================================+")

	// Assessment for 2-core 8GB target
	fmt.Println()
	fmt.Println("Assessment (target: 2 core CPU, 8GB RAM, 100+ students):")
	if avgLatency < 10*time.Millisecond && errorRate < 1.0 {
		fmt.Println("  [EXCELLENT] Ready for production.")
		fmt.Println("  Avg latency <10ms, error rate <1%.")
	} else if avgLatency < 50*time.Millisecond && errorRate < 5.0 {
		fmt.Println("  [GOOD] Acceptable performance.")
		fmt.Println("  Consider optimization if targeting lower latency.")
	} else if avgLatency < 200*time.Millisecond && errorRate < 10.0 {
		fmt.Println("  [FAIR] May need optimization.")
		fmt.Println("  Check DB query performance, Redis pipeline usage, and connection pooling.")
	} else {
		fmt.Println("  [POOR] Needs immediate attention.")
		fmt.Println("  Investigate bottlenecks: CPU profiling, DB slow queries, Redis latency.")
	}

	fmt.Println()
	if p99 > 500*time.Millisecond {
		fmt.Println("  WARNING: P99 latency >500ms. Tail latency may cause timeout issues.")
	}
	if rps < float64(activeStudents) {
		fmt.Printf("  WARNING: Throughput (%.0f rps) is below student count (%d).\n", rps, activeStudents)
		fmt.Println("  This means students experience delays during rapid answer submission.")
	}
}
