package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ─── Old schema structs (for parsing) ───────────────────────────────────────

type oldUser struct {
	ID          uint
	Name        string
	Username    string
	Email       *string
	IsActive    bool
	Password    string
	LoginToken  *string
	DeletedAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastLoginAt *time.Time
}

type oldUserProfile struct {
	ID        uint
	UserID    uint
	NIS       *string
	NIP       *string
	RombelID  *uint
	RoomID    *uint
	CreatedAt time.Time
	UpdatedAt time.Time
}

type oldRoom struct {
	ID        uint
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type oldRombel struct {
	ID         uint
	Name       string
	GradeLevel int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type oldSubject struct {
	ID        uint
	Name      string
	Code      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type oldQuestionBank struct {
	ID        uint
	Name      string
	SubjectID *uint
	UserID    uint
	Desc      string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type oldStimulus struct {
	ID             uint
	QuestionBankID uint
	Content        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type oldQuestion struct {
	ID             uint
	QuestionBankID uint
	Type           string
	Body           string
	AudioPath      *string
	AudioLimit     int
	Options        string // raw JSON
	CorrectAnswer  *string
	SortOrder      int
	DefaultMark    float64
	StimulusID     *uint
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type roleEntry struct {
	RoleID  uint
	UserID  uint
}

// ─── New schema structs ──────────────────────────────────────────────────────

type newUser struct {
	ID          uint       `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	Username    string     `gorm:"column:username"`
	Email       *string    `gorm:"column:email"`
	Password    string     `gorm:"column:password"`
	Role        string     `gorm:"column:role"`
	IsActive    bool       `gorm:"column:is_active"`
	LoginToken  *string    `gorm:"column:login_token"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	LastLoginAt *time.Time `gorm:"column:last_login_at"`
}

func (newUser) TableName() string { return "users" }

type newUserProfile struct {
	ID        uint      `gorm:"column:id"`
	UserID    uint      `gorm:"column:user_id"`
	NIS       *string   `gorm:"column:nis"`
	NIP       *string   `gorm:"column:n_ip"`
	RombelID  *uint     `gorm:"column:rombel_id"`
	RoomID    *uint     `gorm:"column:room_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (newUserProfile) TableName() string { return "user_profiles" }

type newRoom struct {
	ID        uint       `gorm:"column:id"`
	Name      string     `gorm:"column:name"`
	Capacity  int        `gorm:"column:capacity"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	UpdatedAt time.Time  `gorm:"column:updated_at"`
}

func (newRoom) TableName() string { return "rooms" }

type newRombel struct {
	ID         uint      `gorm:"column:id"`
	Name       string    `gorm:"column:name"`
	GradeLevel string    `gorm:"column:grade_level"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}

func (newRombel) TableName() string { return "rombels" }

type newSubject struct {
	ID        uint      `gorm:"column:id"`
	Name      string    `gorm:"column:name"`
	Code      string    `gorm:"column:code"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (newSubject) TableName() string { return "subjects" }

type newQuestionBank struct {
	ID          uint       `gorm:"column:id"`
	Name        string     `gorm:"column:name"`
	SubjectID   *uint      `gorm:"column:subject_id"`
	CreatedBy   uint       `gorm:"column:created_by"`
	Description string     `gorm:"column:description"`
	Status      string     `gorm:"column:status"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
}

func (newQuestionBank) TableName() string { return "question_banks" }

type newStimulus struct {
	ID             uint      `gorm:"column:id"`
	QuestionBankID uint      `gorm:"column:question_bank_id"`
	Content        string    `gorm:"column:content"`
	CreatedAt      time.Time `gorm:"column:created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (newStimulus) TableName() string { return "stimulus" }

type newQuestion struct {
	ID             uint       `gorm:"column:id"`
	QuestionBankID uint       `gorm:"column:question_bank_id"`
	StimulusID     *uint      `gorm:"column:stimulus_id"`
	QuestionType   string     `gorm:"column:question_type"`
	Body           string     `gorm:"column:body"`
	Score          float64    `gorm:"column:score"`
	Difficulty     string     `gorm:"column:difficulty"`
	Options        string     `gorm:"column:options"`
	CorrectAnswer  string     `gorm:"column:correct_answer"`
	AudioPath      *string    `gorm:"column:audio_path"`
	AudioLimit     int        `gorm:"column:audio_limit"`
	BloomLevel     int        `gorm:"column:bloom_level"`
	TopicCode      string     `gorm:"column:topic_code"`
	OrderIndex     int        `gorm:"column:order_index"`
	DeletedAt      *time.Time `gorm:"column:deleted_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at"`
}

func (newQuestion) TableName() string { return "questions" }

// ─── Command ─────────────────────────────────────────────────────────────────

func cmdMigratePatrabak(db *gorm.DB) {
	fs := flag.NewFlagSet("migrate-patrabak", flag.ExitOnError)
	sqlFile := fs.String("sql-file", "", "Path to extracted database.sql from .patrabak (required)")
	skipUsers := fs.Bool("skip-users", false, "Skip migrating users")
	skipQuestions := fs.Bool("skip-questions", false, "Skip migrating question banks & questions")
	dryRun := fs.Bool("dry-run", false, "Parse only, do not write to database")
	_ = fs.Parse(os.Args[2:])

	if *sqlFile == "" {
		fmt.Println("Error: --sql-file is required")
		fmt.Println("  Example: patra-cli migrate-patrabak --sql-file /tmp/patrabak_extract/database.sql")
		os.Exit(1)
	}

	fmt.Printf("Parsing: %s\n", *sqlFile)
	if *dryRun {
		fmt.Println("[DRY RUN] No data will be written.")
	}

	data, err := parseDump(*sqlFile)
	if err != nil {
		fmt.Printf("Error parsing dump: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed: %d subjects, %d rombels, %d rooms, %d question_banks, %d stimuli, %d questions, %d users, %d user_profiles\n",
		len(data.Subjects), len(data.Rombels), len(data.Rooms),
		len(data.QuestionBanks), len(data.Stimuli), len(data.Questions),
		len(data.Users), len(data.UserProfiles))

	if *dryRun {
		return
	}

	// Migrate in order (respects FK deps)
	migrateSubjects(db, data.Subjects)
	migrateRombels(db, data.Rombels)
	migrateRooms(db, data.Rooms)

	if !*skipQuestions {
		migrateQuestionBanks(db, data.QuestionBanks)
		migrateStimuli(db, data.Stimuli)
		migrateQuestions(db, data.Questions)
	}

	if !*skipUsers {
		migrateUsers(db, data.Users, data.Roles)
		migrateUserProfiles(db, data.UserProfiles)
	}

	// Reset sequences so next auto-increment works correctly
	resetSequences(db)

	fmt.Println("\nMigration complete!")
}

// ─── Parsed data container ────────────────────────────────────────────────────

type dumpData struct {
	Subjects      []oldSubject
	Rombels       []oldRombel
	Rooms         []oldRoom
	QuestionBanks []oldQuestionBank
	Stimuli       []oldStimulus
	Questions     []oldQuestion
	Users         []oldUser
	UserProfiles  []oldUserProfile
	Roles         []roleEntry
}

// ─── SQL Dump Parser ──────────────────────────────────────────────────────────

func parseDump(path string) (*dumpData, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	data := &dumpData{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024) // 10MB buffer per line

	var currentTable string
	var inCopy bool

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "COPY public.") {
			// Extract table name
			rest := strings.TrimPrefix(line, "COPY public.")
			tableName := strings.SplitN(rest, " ", 2)[0]
			currentTable = tableName
			inCopy = true
			continue
		}

		if line == "\\." {
			inCopy = false
			currentTable = ""
			continue
		}

		if !inCopy || line == "" {
			continue
		}

		cols := strings.Split(line, "\t")

		switch currentTable {
		case "subjects":
			if r := parseSubject(cols); r != nil {
				data.Subjects = append(data.Subjects, *r)
			}
		case "rombels":
			if r := parseRombel(cols); r != nil {
				data.Rombels = append(data.Rombels, *r)
			}
		case "rooms":
			if r := parseRoom(cols); r != nil {
				data.Rooms = append(data.Rooms, *r)
			}
		case "question_banks":
			if r := parseQuestionBank(cols); r != nil {
				data.QuestionBanks = append(data.QuestionBanks, *r)
			}
		case "stimuluses":
			if r := parseStimulus(cols); r != nil {
				data.Stimuli = append(data.Stimuli, *r)
			}
		case "questions":
			if r := parseQuestion(cols); r != nil {
				data.Questions = append(data.Questions, *r)
			}
		case "users":
			if r := parseUser(cols); r != nil {
				data.Users = append(data.Users, *r)
			}
		case "user_profiles":
			if r := parseUserProfile(cols); r != nil {
				data.UserProfiles = append(data.UserProfiles, *r)
			}
		case "model_has_roles":
			if r := parseRole(cols); r != nil {
				data.Roles = append(data.Roles, *r)
			}
		}
	}

	return data, scanner.Err()
}

// ─── Row parsers ──────────────────────────────────────────────────────────────

// COPY public.subjects (id, name, code, created_at, updated_at)
func parseSubject(c []string) *oldSubject {
	if len(c) < 5 {
		return nil
	}
	return &oldSubject{
		ID:        mustUint(c[0]),
		Name:      pgStr(c[1]),
		Code:      pgStr(c[2]),
		CreatedAt: mustTime(c[3]),
		UpdatedAt: mustTime(c[4]),
	}
}

// COPY public.rombels (id, name, grade_level, created_at, updated_at)
func parseRombel(c []string) *oldRombel {
	if len(c) < 5 {
		return nil
	}
	return &oldRombel{
		ID:         mustUint(c[0]),
		Name:       pgStr(c[1]),
		GradeLevel: mustInt(c[2]),
		CreatedAt:  mustTime(c[3]),
		UpdatedAt:  mustTime(c[4]),
	}
}

// COPY public.rooms (id, name, description, created_at, updated_at)
func parseRoom(c []string) *oldRoom {
	if len(c) < 5 {
		return nil
	}
	return &oldRoom{
		ID:        mustUint(c[0]),
		Name:      pgStr(c[1]),
		// c[2] = description (skip)
		CreatedAt: mustTime(c[3]),
		UpdatedAt: mustTime(c[4]),
	}
}

// COPY public.question_banks (id, name, subject_id, user_id, description, is_active, created_at, updated_at)
func parseQuestionBank(c []string) *oldQuestionBank {
	if len(c) < 8 {
		return nil
	}
	return &oldQuestionBank{
		ID:        mustUint(c[0]),
		Name:      pgStr(c[1]),
		SubjectID: pgUintPtr(c[2]),
		UserID:    mustUint(c[3]),
		Desc:      pgStr(c[4]),
		IsActive:  c[5] == "t",
		CreatedAt: mustTime(c[6]),
		UpdatedAt: mustTime(c[7]),
	}
}

// COPY public.stimuluses (id, question_bank_id, content, topic, created_at, updated_at)
func parseStimulus(c []string) *oldStimulus {
	if len(c) < 6 {
		return nil
	}
	return &oldStimulus{
		ID:             mustUint(c[0]),
		QuestionBankID: mustUint(c[1]),
		Content:        pgStr(c[2]),
		// c[3] = topic (skip)
		CreatedAt: mustTime(c[4]),
		UpdatedAt: mustTime(c[5]),
	}
}

// COPY public.questions (id, question_bank_id, type, question_body, audio_path, audio_limit, options, correct_answer_text, sort_order, default_mark, created_at, updated_at, stimulus_id, options_updated_at)
func parseQuestion(c []string) *oldQuestion {
	if len(c) < 13 {
		return nil
	}
	return &oldQuestion{
		ID:             mustUint(c[0]),
		QuestionBankID: mustUint(c[1]),
		Type:           pgStr(c[2]),
		Body:           pgStr(c[3]),
		AudioPath:      pgStrPtr(c[4]),
		AudioLimit:     mustInt(c[5]),
		Options:        pgStr(c[6]),
		CorrectAnswer:  pgStrPtr(c[7]),
		SortOrder:      mustInt(c[8]),
		DefaultMark:    mustFloat(c[9]),
		CreatedAt:      mustTime(c[10]),
		UpdatedAt:      mustTime(c[11]),
		StimulusID:     pgUintPtr(c[12]),
	}
}

// COPY public.users (id, name, username, email, is_active, email_verified_at, password, login_token, deleted_at, remember_token, created_at, updated_at, last_login_at)
func parseUser(c []string) *oldUser {
	if len(c) < 13 {
		return nil
	}
	return &oldUser{
		ID:          mustUint(c[0]),
		Name:        pgStr(c[1]),
		Username:    pgStr(c[2]),
		Email:       pgStrPtr(c[3]),
		IsActive:    c[4] == "t",
		Password:    pgStr(c[6]),
		LoginToken:  pgStrPtr(c[7]),
		DeletedAt:   pgTimePtr(c[8]),
		CreatedAt:   mustTime(c[10]),
		UpdatedAt:   mustTime(c[11]),
		LastLoginAt: pgTimePtr(c[12]),
	}
}

// COPY public.user_profiles (id, user_id, avatar, nis, nip, rombel_id, room_id, created_at, updated_at)
func parseUserProfile(c []string) *oldUserProfile {
	if len(c) < 9 {
		return nil
	}
	return &oldUserProfile{
		ID:        mustUint(c[0]),
		UserID:    mustUint(c[1]),
		// c[2] = avatar (skip)
		NIS:       pgStrPtr(c[3]),
		NIP:       pgStrPtr(c[4]),
		RombelID:  pgUintPtr(c[5]),
		RoomID:    pgUintPtr(c[6]),
		CreatedAt: mustTime(c[7]),
		UpdatedAt: mustTime(c[8]),
	}
}

// COPY public.model_has_roles (role_id, model_type, model_id)
func parseRole(c []string) *roleEntry {
	if len(c) < 3 {
		return nil
	}
	return &roleEntry{
		RoleID: mustUint(c[0]),
		UserID: mustUint(c[2]),
	}
}

// ─── Migrate functions ────────────────────────────────────────────────────────

func migrateSubjects(db *gorm.DB, rows []oldSubject) {
	records := make([]newSubject, 0, len(rows))
	for _, r := range rows {
		records = append(records, newSubject{
			ID: r.ID, Name: r.Name, Code: r.Code,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 100)
	fmt.Printf("  subjects: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateRombels(db *gorm.DB, rows []oldRombel) {
	records := make([]newRombel, 0, len(rows))
	for _, r := range rows {
		records = append(records, newRombel{
			ID: r.ID, Name: r.Name, GradeLevel: strconv.Itoa(r.GradeLevel),
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 100)
	fmt.Printf("  rombels: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateRooms(db *gorm.DB, rows []oldRoom) {
	records := make([]newRoom, 0, len(rows))
	for _, r := range rows {
		records = append(records, newRoom{
			ID: r.ID, Name: r.Name, Capacity: 30,
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 100)
	fmt.Printf("  rooms: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateQuestionBanks(db *gorm.DB, rows []oldQuestionBank) {
	records := make([]newQuestionBank, 0, len(rows))
	for _, r := range rows {
		status := "active"
		if !r.IsActive {
			status = "inactive"
		}
		records = append(records, newQuestionBank{
			ID:          r.ID,
			Name:        r.Name,
			SubjectID:   r.SubjectID,
			CreatedBy:   r.UserID,
			Description: r.Desc,
			Status:      status,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 100)
	fmt.Printf("  question_banks: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateStimuli(db *gorm.DB, rows []oldStimulus) {
	records := make([]newStimulus, 0, len(rows))
	for _, r := range rows {
		records = append(records, newStimulus{
			ID:             r.ID,
			QuestionBankID: r.QuestionBankID,
			Content:        r.Content,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 100)
	fmt.Printf("  stimuli: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateQuestions(db *gorm.DB, rows []oldQuestion) {
	records := make([]newQuestion, 0, len(rows))
	skipped := 0

	for _, r := range rows {
		newOpts, correctAnswer, err := transformOptions(r.Options, r.Type)
		if err != nil {
			skipped++
			continue
		}

		records = append(records, newQuestion{
			ID:             r.ID,
			QuestionBankID: r.QuestionBankID,
			StimulusID:     r.StimulusID,
			QuestionType:   r.Type,
			Body:           r.Body,
			Score:          r.DefaultMark,
			Difficulty:     "medium",
			Options:        newOpts,
			CorrectAnswer:  correctAnswer,
			AudioPath:      r.AudioPath,
			AudioLimit:     r.AudioLimit,
			BloomLevel:     0,
			TopicCode:      "",
			OrderIndex:     r.SortOrder,
			CreatedAt:      r.CreatedAt,
			UpdatedAt:      r.UpdatedAt,
		})
	}

	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 200)
	fmt.Printf("  questions: %d rows, %d skipped (err: %v)\n", len(records), skipped, result.Error)
}

func migrateUsers(db *gorm.DB, rows []oldUser, roles []roleEntry) {
	// Build role map: userID → role string
	roleMap := make(map[uint]string)
	for _, r := range roles {
		switch r.RoleID {
		case 1, 2:
			roleMap[r.UserID] = "admin"
		case 3:
			roleMap[r.UserID] = "guru"
		case 4:
			roleMap[r.UserID] = "peserta"
		}
	}

	records := make([]newUser, 0, len(rows))
	for _, r := range rows {
		role := roleMap[r.ID]
		if role == "" {
			role = "peserta"
		}
		records = append(records, newUser{
			ID:          r.ID,
			Name:        r.Name,
			Username:    r.Username,
			Email:       r.Email,
			Password:    r.Password,
			Role:        role,
			IsActive:    r.IsActive,
			LoginToken:  r.LoginToken,
			DeletedAt:   r.DeletedAt,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
			LastLoginAt: r.LastLoginAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 200)
	fmt.Printf("  users: %d rows (err: %v)\n", len(records), result.Error)
}

func migrateUserProfiles(db *gorm.DB, rows []oldUserProfile) {
	records := make([]newUserProfile, 0, len(rows))
	for _, r := range rows {
		records = append(records, newUserProfile{
			ID:        r.ID,
			UserID:    r.UserID,
			NIS:       r.NIS,
			NIP:       r.NIP,
			RombelID:  r.RombelID,
			RoomID:    r.RoomID,
			CreatedAt: r.CreatedAt,
			UpdatedAt: r.UpdatedAt,
		})
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(records, 200)
	fmt.Printf("  user_profiles: %d rows (err: %v)\n", len(records), result.Error)
}

func resetSequences(db *gorm.DB) {
	tables := []string{"subjects", "rombels", "rooms", "question_banks", "stimulus", "questions", "users", "user_profiles"}
	fmt.Println("\nResetting sequences:")
	for _, t := range tables {
		sql := fmt.Sprintf("SELECT setval(pg_get_serial_sequence('%s', 'id'), COALESCE(MAX(id), 1)) FROM %s", t, t)
		if err := db.Exec(sql).Error; err != nil {
			fmt.Printf("  %s: WARN - %v\n", t, err)
		} else {
			fmt.Printf("  %s: ok\n", t)
		}
	}
}

// ─── Options JSON transformation ──────────────────────────────────────────────

// Old: [{"text":"...", "weight": 0/1}]
// New options: [{"id":"a","text":"...","score":0}]
// New correct_answer: ["a"] (letters where weight>0)
func transformOptions(raw string, qtype string) (string, string, error) {
	if raw == "" || raw == "\\N" || raw == "null" {
		// No options (esai/isian): empty arrays
		return "[]", "[]", nil
	}

	type oldOpt struct {
		Text   string  `json:"text"`
		Weight float64 `json:"weight"`
	}

	var opts []oldOpt
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		// Try to return as-is if already in new format or broken
		return "[]", "[]", nil
	}

	type newOpt struct {
		ID    string  `json:"id"`
		Text  string  `json:"text"`
		Score float64 `json:"score"`
	}

	letters := "abcdefghijklmnopqrstuvwxyz"
	newOpts := make([]newOpt, 0, len(opts))
	correctIDs := make([]string, 0)

	for i, o := range opts {
		letter := string(letters[i])
		newOpts = append(newOpts, newOpt{
			ID:    letter,
			Text:  o.Text,
			Score: o.Weight,
		})
		if o.Weight > 0 {
			correctIDs = append(correctIDs, letter)
		}
	}

	newOptsJSON, _ := json.Marshal(newOpts)
	correctJSON, _ := json.Marshal(correctIDs)

	return string(newOptsJSON), string(correctJSON), nil
}

// ─── PG COPY format helpers ───────────────────────────────────────────────────

func pgStr(s string) string {
	if s == "\\N" {
		return ""
	}
	// Unescape PostgreSQL COPY escape sequences
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\r", "\r")
	s = strings.ReplaceAll(s, "\\\\", "\\")
	return s
}

func pgStrPtr(s string) *string {
	if s == "\\N" {
		return nil
	}
	v := pgStr(s)
	return &v
}

func pgUintPtr(s string) *uint {
	if s == "\\N" {
		return nil
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return nil
	}
	u := uint(v)
	return &u
}

func pgTimePtr(s string) *time.Time {
	if s == "\\N" {
		return nil
	}
	t, err := parseTime(s)
	if err != nil {
		return nil
	}
	return &t
}

func mustUint(s string) uint {
	if s == "\\N" {
		return 0
	}
	v, _ := strconv.ParseUint(s, 10, 64)
	return uint(v)
}

func mustInt(s string) int {
	if s == "\\N" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func mustFloat(s string) float64 {
	if s == "\\N" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func mustTime(s string) time.Time {
	t, _ := parseTime(s)
	return t
}

func parseTime(s string) (time.Time, error) {
	if s == "\\N" || s == "" {
		return time.Time{}, nil
	}
	// Try common PostgreSQL timestamp formats
	formats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04:05.999999",
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05.999999-07",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable time: %s", s)
}
