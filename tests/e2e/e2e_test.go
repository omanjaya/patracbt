package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const baseURL = "http://localhost:4000/api/v1"

var adminToken string

// apiResponse is a generic API response wrapper.
type apiResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Meta    *struct {
		Total int `json:"total"`
	} `json:"meta"`
}

// ─── Helpers ────────────────────────────────────────────────────

func doRequest(t *testing.T, method, path string, body interface{}, token string) (*http.Response, apiResponse) {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, baseURL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do request %s %s: %v", method, path, err)
	}

	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var result apiResponse
	json.Unmarshal(raw, &result)
	return resp, result
}

func requireStatus(t *testing.T, resp *http.Response, expected int) {
	t.Helper()
	if resp.StatusCode != expected {
		t.Fatalf("expected status %d, got %d", expected, resp.StatusCode)
	}
}

// ─── Setup: Login as admin ──────────────────────────────────────

func TestMain(m *testing.M) {
	// Login to get admin token
	loginBody, _ := json.Marshal(map[string]string{
		"login":    "admin",
		"password": "admin123",
	})
	resp, err := http.Post(baseURL+"/auth/login", "application/json", bytes.NewReader(loginBody))
	if err != nil {
		fmt.Fprintf(os.Stderr, "SKIP: cannot connect to API at %s: %v\n", baseURL, err)
		os.Exit(0)
	}
	defer resp.Body.Close()

	var loginRes struct {
		Data struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	raw, _ := io.ReadAll(resp.Body)
	json.Unmarshal(raw, &loginRes)

	if loginRes.Data.AccessToken == "" {
		fmt.Fprintf(os.Stderr, "SKIP: login failed (no token). Response: %s\n", string(raw))
		os.Exit(0)
	}
	adminToken = loginRes.Data.AccessToken

	os.Exit(m.Run())
}

// ─── Health ─────────────────────────────────────────────────────

func TestHealth(t *testing.T) {
	resp, result := doRequest(t, "GET", "/health", nil, "")
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("health check not successful")
	}
}

// ─── Auth ───────────────────────────────────────────────────────

func TestAuth_Me(t *testing.T) {
	resp, result := doRequest(t, "GET", "/auth/me", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("auth/me not successful")
	}
	var user struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	json.Unmarshal(result.Data, &user)
	if user.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", user.Username)
	}
	if user.Role != "admin" {
		t.Errorf("expected role 'admin', got '%s'", user.Role)
	}
}

func TestAuth_Unauthorized(t *testing.T) {
	resp, _ := doRequest(t, "GET", "/auth/me", nil, "invalid-token")
	if resp.StatusCode != 401 {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestAuth_LoginFail(t *testing.T) {
	resp, result := doRequest(t, "POST", "/auth/login", map[string]string{
		"login": "admin", "password": "wrongpassword",
	}, "")
	if resp.StatusCode != 401 {
		t.Errorf("expected 401 for wrong password, got %d", resp.StatusCode)
	}
	if result.Success {
		t.Error("expected login to fail")
	}
}

// ─── User CRUD ──────────────────────────────────────────────────

func TestUser_CRUD(t *testing.T) {
	uniqueSuffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// CREATE
	createBody := map[string]interface{}{
		"name":     "E2E Test User " + uniqueSuffix,
		"username": "e2euser" + uniqueSuffix,
		"password": "password123",
		"role":     "peserta",
	}
	resp, result := doRequest(t, "POST", "/admin/users", createBody, adminToken)
	requireStatus(t, resp, 201)
	if !result.Success {
		t.Fatalf("create user failed: %s", result.Message)
	}

	var created struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Role     string `json:"role"`
	}
	json.Unmarshal(result.Data, &created)
	if created.ID == 0 {
		t.Fatal("created user has no ID")
	}
	if created.Username != "e2euser"+uniqueSuffix {
		t.Errorf("expected username 'e2euser%s', got '%s'", uniqueSuffix, created.Username)
	}
	userID := created.ID

	// LIST
	resp, result = doRequest(t, "GET", "/admin/users?search=e2euser"+uniqueSuffix, nil, adminToken)
	requireStatus(t, resp, 200)
	var users []struct {
		ID       uint   `json:"id"`
		Username string `json:"username"`
	}
	json.Unmarshal(result.Data, &users)
	found := false
	for _, u := range users {
		if u.ID == userID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created user not found in list")
	}

	// UPDATE
	updateBody := map[string]interface{}{
		"name": "E2E Updated " + uniqueSuffix,
		"role": "peserta",
	}
	resp, result = doRequest(t, "PUT", fmt.Sprintf("/admin/users/%d", userID), updateBody, adminToken)
	requireStatus(t, resp, 200)
	var updated struct {
		Name string `json:"name"`
	}
	json.Unmarshal(result.Data, &updated)
	if updated.Name != "E2E Updated "+uniqueSuffix {
		t.Errorf("expected updated name, got '%s'", updated.Name)
	}

	// DELETE (soft)
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d", userID), nil, adminToken)
	requireStatus(t, resp, 200)

	// Verify in trashed
	resp, result = doRequest(t, "GET", "/admin/users/trashed?search=e2euser"+uniqueSuffix, nil, adminToken)
	requireStatus(t, resp, 200)
	var trashed []struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &trashed)
	found = false
	for _, u := range trashed {
		if u.ID == userID {
			found = true
			break
		}
	}
	if !found {
		t.Error("deleted user not found in trash")
	}

	// RESTORE
	resp, _ = doRequest(t, "POST", fmt.Sprintf("/admin/users/%d/restore", userID), nil, adminToken)
	requireStatus(t, resp, 200)

	// FORCE DELETE (cleanup)
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d", userID), nil, adminToken)
	requireStatus(t, resp, 200)
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d/force", userID), nil, adminToken)
	requireStatus(t, resp, 200)
}

func TestUser_CreateDuplicate(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	body := map[string]interface{}{
		"name": "Dup User", "username": "dupuser" + suffix, "password": "password123", "role": "peserta",
	}

	// Create first
	resp, _ := doRequest(t, "POST", "/admin/users", body, adminToken)
	requireStatus(t, resp, 201)

	// Try duplicate
	resp, result := doRequest(t, "POST", "/admin/users", body, adminToken)
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for duplicate username, got %d", resp.StatusCode)
	}
	_ = result

	// Cleanup
	var users []struct{ ID uint `json:"id"` }
	_, r := doRequest(t, "GET", "/admin/users?search=dupuser"+suffix, nil, adminToken)
	json.Unmarshal(r.Data, &users)
	for _, u := range users {
		doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d", u.ID), nil, adminToken)
		doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d/force", u.ID), nil, adminToken)
	}
}

func TestUser_CreateValidation(t *testing.T) {
	// Missing required fields
	resp, _ := doRequest(t, "POST", "/admin/users", map[string]string{}, adminToken)
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for empty body, got %d", resp.StatusCode)
	}

	// Invalid role
	resp, _ = doRequest(t, "POST", "/admin/users", map[string]interface{}{
		"name": "Test", "username": "testx", "password": "password123", "role": "invalid",
	}, adminToken)
	if resp.StatusCode != 422 {
		t.Errorf("expected 422 for invalid role, got %d", resp.StatusCode)
	}
}

func TestUser_BulkAction(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	var ids []uint

	// Create 3 users
	for i := 0; i < 3; i++ {
		body := map[string]interface{}{
			"name": fmt.Sprintf("Bulk %d", i), "username": fmt.Sprintf("bulk%s_%d", suffix, i),
			"password": "password123", "role": "peserta",
		}
		_, result := doRequest(t, "POST", "/admin/users", body, adminToken)
		var u struct{ ID uint `json:"id"` }
		json.Unmarshal(result.Data, &u)
		ids = append(ids, u.ID)
	}

	// Bulk delete
	resp, _ := doRequest(t, "POST", "/admin/users/bulk-action", map[string]interface{}{
		"action": "delete", "ids": ids,
	}, adminToken)
	requireStatus(t, resp, 200)

	// Bulk restore
	resp, _ = doRequest(t, "POST", "/admin/users/bulk-action", map[string]interface{}{
		"action": "restore", "ids": ids,
	}, adminToken)
	requireStatus(t, resp, 200)

	// Bulk force delete (cleanup)
	resp, _ = doRequest(t, "POST", "/admin/users/bulk-action", map[string]interface{}{
		"action": "delete", "ids": ids,
	}, adminToken)
	requireStatus(t, resp, 200)
	resp, _ = doRequest(t, "POST", "/admin/users/bulk-action", map[string]interface{}{
		"action": "force_delete", "ids": ids,
	}, adminToken)
	requireStatus(t, resp, 200)
}

// ─── Dashboard ──────────────────────────────────────────────────

func TestDashboard_Stats(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/stats", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("dashboard stats failed")
	}

	var stats struct {
		TotalPeserta int64 `json:"total_peserta"`
		TotalGuru    int64 `json:"total_guru"`
		TotalRombel  int64 `json:"total_rombel"`
	}
	json.Unmarshal(result.Data, &stats)
	// These should be >= 0 (valid numbers)
	if stats.TotalPeserta < 0 || stats.TotalGuru < 0 {
		t.Error("invalid stats values")
	}
}

func TestDashboard_Alerts(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/alerts", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("dashboard alerts failed")
	}

	var alerts []struct {
		Type  string `json:"type"`
		Title string `json:"title"`
		Count int64  `json:"count"`
		Link  string `json:"link"`
	}
	json.Unmarshal(result.Data, &alerts)
	for _, a := range alerts {
		if a.Type == "" || a.Title == "" || a.Link == "" {
			t.Errorf("alert missing fields: %+v", a)
		}
		if a.Count < 0 {
			t.Errorf("alert count negative: %+v", a)
		}
	}
}

func TestDashboard_ServerStats(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/server-stats", nil, adminToken)
	requireStatus(t, resp, 200)

	var stats struct {
		CPUPercent float64 `json:"cpu_percent"`
		RAMUsed    string  `json:"ram_used"`
		GoVersion  string  `json:"go_version"`
	}
	json.Unmarshal(result.Data, &stats)
	if stats.GoVersion == "" {
		t.Error("missing go_version")
	}
	if stats.RAMUsed == "" {
		t.Error("missing ram_used")
	}
}

func TestDashboard_OngoingExams(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/ongoing-exams", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("ongoing exams failed")
	}
}

func TestDashboard_UpcomingExams(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/upcoming-exams", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("upcoming exams failed")
	}
}

func TestDashboard_RecentActivity(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/dashboard/recent-activity", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("recent activity failed")
	}
}

// ─── Rombel CRUD ────────────────────────────────────────────────

func TestRombel_CRUD(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// CREATE
	resp, result := doRequest(t, "POST", "/admin/rombels", map[string]interface{}{
		"name": "E2E Rombel " + suffix,
	}, adminToken)
	requireStatus(t, resp, 201)
	var rombel struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(result.Data, &rombel)
	if rombel.ID == 0 {
		t.Fatal("rombel ID is 0")
	}

	// LIST
	resp, result = doRequest(t, "GET", "/admin/rombels?search=E2E+Rombel+"+suffix, nil, adminToken)
	requireStatus(t, resp, 200)

	// UPDATE
	resp, result = doRequest(t, "PUT", fmt.Sprintf("/admin/rombels/%d", rombel.ID), map[string]interface{}{
		"name": "E2E Rombel Updated " + suffix,
	}, adminToken)
	requireStatus(t, resp, 200)

	// Create user + assign to rombel
	userBody := map[string]interface{}{
		"name": "Rombel Student " + suffix, "username": "rombelstudent" + suffix,
		"password": "password123", "role": "peserta",
	}
	_, userResult := doRequest(t, "POST", "/admin/users", userBody, adminToken)
	var user struct{ ID uint `json:"id"` }
	json.Unmarshal(userResult.Data, &user)

	resp, _ = doRequest(t, "POST", fmt.Sprintf("/admin/rombels/%d/assign-users", rombel.ID), map[string]interface{}{
		"user_ids": []uint{user.ID},
	}, adminToken)
	requireStatus(t, resp, 200)

	// Remove user from rombel
	resp, _ = doRequest(t, "POST", fmt.Sprintf("/admin/rombels/%d/remove-users", rombel.ID), map[string]interface{}{
		"user_ids": []uint{user.ID},
	}, adminToken)
	requireStatus(t, resp, 200)

	// DELETE rombel
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/rombels/%d", rombel.ID), nil, adminToken)
	requireStatus(t, resp, 200)

	// Cleanup user
	doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d", user.ID), nil, adminToken)
	doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d/force", user.ID), nil, adminToken)
}

// ─── Subject CRUD ───────────────────────────────────────────────

func TestSubject_CRUD(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// CREATE
	resp, result := doRequest(t, "POST", "/admin/subjects", map[string]interface{}{
		"name": "E2E Subject " + suffix, "code": "E2E" + suffix,
	}, adminToken)
	requireStatus(t, resp, 201)
	var subj struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &subj)

	// LIST
	resp, _ = doRequest(t, "GET", "/admin/subjects?search=E2E+Subject", nil, adminToken)
	requireStatus(t, resp, 200)

	// UPDATE
	resp, _ = doRequest(t, "PUT", fmt.Sprintf("/admin/subjects/%d", subj.ID), map[string]interface{}{
		"name": "E2E Subject Updated " + suffix, "code": "E2E" + suffix,
	}, adminToken)
	requireStatus(t, resp, 200)

	// DELETE
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/subjects/%d", subj.ID), nil, adminToken)
	requireStatus(t, resp, 200)
}

// ─── Tag CRUD ───────────────────────────────────────────────────

func TestTag_CRUD(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// CREATE
	resp, result := doRequest(t, "POST", "/admin/tags", map[string]interface{}{
		"name": "E2E Tag " + suffix,
	}, adminToken)
	requireStatus(t, resp, 201)
	var tag struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &tag)

	// LIST
	resp, _ = doRequest(t, "GET", "/admin/tags?search=E2E+Tag", nil, adminToken)
	requireStatus(t, resp, 200)

	// UPDATE
	resp, _ = doRequest(t, "PUT", fmt.Sprintf("/admin/tags/%d", tag.ID), map[string]interface{}{
		"name": "E2E Tag Updated " + suffix,
	}, adminToken)
	requireStatus(t, resp, 200)

	// DELETE
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/tags/%d", tag.ID), nil, adminToken)
	requireStatus(t, resp, 200)
}

// ─── Question Bank CRUD ─────────────────────────────────────────

func TestQuestionBank_CRUD(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// Need a subject first
	_, subjRes := doRequest(t, "POST", "/admin/subjects", map[string]interface{}{
		"name": "QB Subject " + suffix, "code": "QBS" + suffix,
	}, adminToken)
	var subj struct{ ID uint `json:"id"` }
	json.Unmarshal(subjRes.Data, &subj)

	// CREATE bank
	resp, result := doRequest(t, "POST", "/question-banks", map[string]interface{}{
		"name": "E2E Bank " + suffix, "subject_id": subj.ID,
	}, adminToken)
	requireStatus(t, resp, 201)
	var bank struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &bank)
	if bank.ID == 0 {
		t.Fatal("bank ID is 0")
	}

	// LIST
	resp, _ = doRequest(t, "GET", "/question-banks?search=E2E+Bank", nil, adminToken)
	requireStatus(t, resp, 200)

	// GET detail
	resp, result = doRequest(t, "GET", fmt.Sprintf("/question-banks/%d", bank.ID), nil, adminToken)
	requireStatus(t, resp, 200)

	// DELETE bank
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/question-banks/%d", bank.ID), nil, adminToken)
	requireStatus(t, resp, 200)

	// Cleanup subject
	doRequest(t, "DELETE", fmt.Sprintf("/admin/subjects/%d", subj.ID), nil, adminToken)
}

// ─── User Filter: no_rombel / exclude_rombel_id ─────────────────

func TestUser_RombelFilters(t *testing.T) {
	// List users without rombel
	resp, result := doRequest(t, "GET", "/admin/users?role=peserta&no_rombel=true", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("no_rombel filter failed")
	}

	// List users excluding a rombel
	resp, result = doRequest(t, "GET", "/admin/users?role=peserta&exclude_rombel_id=1", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("exclude_rombel_id filter failed")
	}
}

// ─── Import Template Download ───────────────────────────────────

func TestUser_DownloadTemplate(t *testing.T) {
	req, _ := http.NewRequest("GET", baseURL+"/admin/users/import/template", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("download template: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct != "text/csv" {
		t.Errorf("expected text/csv, got %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Error("empty template")
	}
	// Should contain rombel column header
	if !bytes.Contains(body, []byte("rombel")) {
		t.Error("template missing 'rombel' column")
	}
}

// ─── Search Peserta ─────────────────────────────────────────────

func TestUser_SearchPeserta(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/users/search-peserta?q=&limit=5", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("search peserta failed")
	}
}

// ─── Permissions ────────────────────────────────────────────────

func TestPermissions_List(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/permissions", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("list permissions failed")
	}
}

func TestPermissions_Groups(t *testing.T) {
	resp, result := doRequest(t, "GET", "/admin/permissions/groups", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("list permission groups failed")
	}
}

// ─── Room CRUD ──────────────────────────────────────────────────

func TestRoom_CRUD(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// CREATE
	resp, result := doRequest(t, "POST", "/admin/rooms", map[string]interface{}{
		"name": "E2E Room " + suffix, "capacity": 30,
	}, adminToken)
	requireStatus(t, resp, 201)
	var room struct {
		ID       uint   `json:"id"`
		Name     string `json:"name"`
		Capacity int    `json:"capacity"`
	}
	json.Unmarshal(result.Data, &room)
	if room.ID == 0 {
		t.Fatal("room ID is 0")
	}
	if room.Name != "E2E Room "+suffix {
		t.Errorf("expected name 'E2E Room %s', got '%s'", suffix, room.Name)
	}
	if room.Capacity != 30 {
		t.Errorf("expected capacity 30, got %d", room.Capacity)
	}

	// LIST
	resp, result = doRequest(t, "GET", "/admin/rooms?search=E2E+Room+"+suffix, nil, adminToken)
	requireStatus(t, resp, 200)
	var rooms []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	json.Unmarshal(result.Data, &rooms)
	found := false
	for _, r := range rooms {
		if r.ID == room.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created room not found in list")
	}

	// UPDATE
	resp, result = doRequest(t, "PUT", fmt.Sprintf("/admin/rooms/%d", room.ID), map[string]interface{}{
		"name": "E2E Room Updated " + suffix, "capacity": 40,
	}, adminToken)
	requireStatus(t, resp, 200)
	var updated struct {
		Name     string `json:"name"`
		Capacity int    `json:"capacity"`
	}
	json.Unmarshal(result.Data, &updated)
	if updated.Name != "E2E Room Updated "+suffix {
		t.Errorf("expected updated name, got '%s'", updated.Name)
	}

	// DELETE
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/rooms/%d", room.ID), nil, adminToken)
	requireStatus(t, resp, 200)
}

// ─── Room Assign/Remove Users ───────────────────────────────────

func TestRoom_AssignRemoveUsers(t *testing.T) {
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%100000)

	// Create a room
	resp, result := doRequest(t, "POST", "/admin/rooms", map[string]interface{}{
		"name": "E2E Assign Room " + suffix, "capacity": 20,
	}, adminToken)
	requireStatus(t, resp, 201)
	var room struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &room)
	if room.ID == 0 {
		t.Fatal("room ID is 0")
	}

	// Create a peserta user
	resp, result = doRequest(t, "POST", "/admin/users", map[string]interface{}{
		"name": "Room Peserta " + suffix, "username": "roompeserta" + suffix,
		"password": "password123", "role": "peserta",
	}, adminToken)
	requireStatus(t, resp, 201)
	var user struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &user)
	if user.ID == 0 {
		t.Fatal("user ID is 0")
	}

	// Assign user to room
	resp, _ = doRequest(t, "POST", fmt.Sprintf("/admin/rooms/%d/assign-users", room.ID), map[string]interface{}{
		"user_ids": []uint{user.ID},
	}, adminToken)
	requireStatus(t, resp, 200)

	// Verify user is listed in room
	resp, result = doRequest(t, "GET", fmt.Sprintf("/admin/rooms/%d/users", room.ID), nil, adminToken)
	requireStatus(t, resp, 200)
	var roomUsers []struct{ ID uint `json:"id"` }
	json.Unmarshal(result.Data, &roomUsers)
	found := false
	for _, u := range roomUsers {
		if u.ID == user.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("assigned user not found in room users list")
	}

	// Remove user from room
	resp, _ = doRequest(t, "POST", fmt.Sprintf("/admin/rooms/%d/remove-users", room.ID), map[string]interface{}{
		"user_ids": []uint{user.ID},
	}, adminToken)
	requireStatus(t, resp, 200)

	// Cleanup: delete room
	resp, _ = doRequest(t, "DELETE", fmt.Sprintf("/admin/rooms/%d", room.ID), nil, adminToken)
	requireStatus(t, resp, 200)

	// Cleanup: delete user
	doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d", user.ID), nil, adminToken)
	doRequest(t, "DELETE", fmt.Sprintf("/admin/users/%d/force", user.ID), nil, adminToken)
}

// ─── User Room Filters ──────────────────────────────────────────

func TestUser_RoomFilters(t *testing.T) {
	// List users without a room
	resp, result := doRequest(t, "GET", "/admin/users?role=peserta&no_room=true", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("no_room filter failed")
	}

	// List users excluding a specific room
	resp, result = doRequest(t, "GET", "/admin/users?role=peserta&exclude_room_id=1", nil, adminToken)
	requireStatus(t, resp, 200)
	if !result.Success {
		t.Error("exclude_room_id filter failed")
	}
}
