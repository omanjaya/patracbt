package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	goredis "github.com/redis/go-redis/v9"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/service"
	redisclient "github.com/omanjaya/patra/internal/infrastructure/cache/redis"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/omanjaya/patra/pkg/types"

	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load .env
	_ = godotenv.Load()

	// Config & Logger
	cfg := config.Load()
	logger.Init(cfg.App.Env)
	defer logger.Sync()

	// Database
	db := postgres.NewDB(cfg)

	command := os.Args[1]

	switch command {
	case "recalculate-scores":
		cmdRecalculateScores(db)
	case "warm-up":
		cmdWarmUp(cfg, db)
	case "backup":
		cmdBackup(cfg)
	case "seed-permissions":
		cmdSeedPermissions(db)
	case "migrate-patrabak":
		cmdMigratePatrabak(db)
	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("CBT Patra CLI")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  patra-cli <command> [flags]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  recalculate-scores  Recalculate scores for a schedule")
	fmt.Println("    --schedule-id=N     Schedule ID (required)")
	fmt.Println()
	fmt.Println("  warm-up             Warm up exam question cache in Redis")
	fmt.Println("    --schedule-id=N     Schedule ID (optional, all active if omitted)")
	fmt.Println()
	fmt.Println("  backup              Create a PostgreSQL database backup")
	fmt.Println("    --format=FORMAT     Output format: sql (default) or dump (pg_dump custom)")
	fmt.Println()
	fmt.Println("  seed-permissions    Seed default permissions into database")
	fmt.Println()
	fmt.Println("  migrate-patrabak    Import data from ExamPatra .patrabak backup")
	fmt.Println("    --sql-file=PATH     Path to extracted database.sql (required)")
	fmt.Println("    --skip-users        Skip migrating users")
	fmt.Println("    --skip-questions    Skip migrating question banks & questions")
	fmt.Println("    --dry-run           Parse only, do not write to database")
}

// --- recalculate-scores ---

func cmdRecalculateScores(db *gorm.DB) {
	fs := flag.NewFlagSet("recalculate-scores", flag.ExitOnError)
	scheduleID := fs.Uint("schedule-id", 0, "Exam schedule ID (required)")
	_ = fs.Parse(os.Args[2:])

	if *scheduleID == 0 {
		fmt.Println("Error: --schedule-id is required")
		os.Exit(1)
	}

	sessionRepo := postgres.NewExamSessionRepository(db)
	questionRepo := postgres.NewQuestionRepository(db)
	calculator := service.NewScoreCalculator()

	// Find all finished sessions
	sessions, err := sessionRepo.ListFinishedBySchedule(*scheduleID)
	if err != nil {
		fmt.Printf("Error fetching sessions: %v\n", err)
		os.Exit(1)
	}

	if len(sessions) == 0 {
		fmt.Println("No finished sessions found for this schedule.")
		return
	}

	fmt.Printf("Found %d finished sessions for schedule #%d\n", len(sessions), *scheduleID)

	// Collect all question IDs from all sessions
	questionIDSet := make(map[uint]bool)
	for _, s := range sessions {
		var order []uint
		if err := json.Unmarshal(s.QuestionOrder, &order); err == nil {
			for _, qid := range order {
				questionIDSet[qid] = true
			}
		}
	}

	// Load all questions at once
	var questionIDs []uint
	for id := range questionIDSet {
		questionIDs = append(questionIDs, id)
	}
	questions, err := questionRepo.FindByIDs(questionIDs)
	if err != nil {
		fmt.Printf("Error fetching questions: %v\n", err)
		os.Exit(1)
	}
	questionMap := make(map[uint]*entity.Question, len(questions))
	for _, q := range questions {
		questionMap[q.ID] = q
	}

	// Load all answers by schedule
	answersBySession, err := sessionRepo.GetAllAnswersBySchedule(*scheduleID)
	if err != nil {
		fmt.Printf("Error fetching answers: %v\n", err)
		os.Exit(1)
	}

	var changes []entity.ScoreChange
	changedCount := 0

	for _, session := range sessions {
		oldScore := session.Score
		answers := answersBySession[session.ID]

		// Recalculate
		var newScore float64
		var maxScore float64
		for _, ans := range answers {
			q, ok := questionMap[ans.QuestionID]
			if !ok {
				continue
			}
			maxScore += q.Score
			newScore += calculator.Calculate(q, ans.Answer)
		}

		if newScore != oldScore {
			changedCount++
			fmt.Printf("  Session #%d (User #%d): %.2f -> %.2f\n", session.ID, session.UserID, oldScore, newScore)

			session.Score = newScore
			session.MaxScore = maxScore
			if err := sessionRepo.Update(session); err != nil {
				fmt.Printf("    ERROR updating session: %v\n", err)
				continue
			}

			changes = append(changes, entity.ScoreChange{
				SessionID: session.ID,
				OldScore:  oldScore,
				NewScore:  newScore,
			})
		}
	}

	fmt.Printf("\nRecalculation complete: %d/%d sessions changed\n", changedCount, len(sessions))

	// Create regrade log
	if len(changes) > 0 {
		changesJSON, _ := json.Marshal(changes)
		regradeLog := &entity.RegradeLog{
			ExamScheduleID: *scheduleID,
			RequestedBy:    0, // CLI
			SessionsCount:  len(changes),
			ScoreChanges:   types.JSON(changesJSON),
		}
		if err := sessionRepo.CreateRegradeLog(regradeLog); err != nil {
			fmt.Printf("Warning: failed to create regrade log: %v\n", err)
		} else {
			fmt.Println("Regrade log created.")
		}
	}
}

// --- warm-up ---

func cmdWarmUp(cfg *config.Config, db *gorm.DB) {
	fs := flag.NewFlagSet("warm-up", flag.ExitOnError)
	scheduleID := fs.Uint("schedule-id", 0, "Exam schedule ID (optional, all active if omitted)")
	_ = fs.Parse(os.Args[2:])

	rdb := redisclient.NewClient(cfg)

	scheduleRepo := postgres.NewExamScheduleRepository(db)
	questionRepo := postgres.NewQuestionRepository(db)

	var scheduleIDs []uint
	if *scheduleID > 0 {
		scheduleIDs = append(scheduleIDs, *scheduleID)
	} else {
		// Find all active/published schedules
		var schedules []entity.ExamSchedule
		db.Where("status IN ? AND deleted_at IS NULL", []string{"active", "published"}).Find(&schedules)
		for _, s := range schedules {
			scheduleIDs = append(scheduleIDs, s.ID)
		}
		if len(scheduleIDs) == 0 {
			fmt.Println("No active schedules found.")
			return
		}
		fmt.Printf("Found %d active schedule(s)\n", len(scheduleIDs))
	}

	ctx := context.Background()

	for _, sid := range scheduleIDs {
		schedule, err := scheduleRepo.FindByID(sid)
		if err != nil {
			fmt.Printf("Schedule #%d: error fetching - %v\n", sid, err)
			continue
		}

		fmt.Printf("Schedule #%d (%s):\n", sid, schedule.Name)

		// Collect question IDs from all question banks
		var allQuestionIDs []uint
		for _, sqb := range schedule.QuestionBanks {
			questions, _, err := questionRepo.ListByBank(sqb.QuestionBankID, paginationAll())
			if err != nil {
				fmt.Printf("  Bank #%d: error - %v\n", sqb.QuestionBankID, err)
				continue
			}
			for _, q := range questions {
				allQuestionIDs = append(allQuestionIDs, q.ID)
			}
			fmt.Printf("  Bank #%d: %d questions loaded\n", sqb.QuestionBankID, len(questions))
		}

		if len(allQuestionIDs) == 0 {
			fmt.Printf("  No questions to cache.\n")
			continue
		}

		// Load all questions and cache to Redis
		questions, err := questionRepo.FindByIDs(allQuestionIDs)
		if err != nil {
			fmt.Printf("  Error loading questions: %v\n", err)
			continue
		}

		cacheKey := fmt.Sprintf("exam_questions:%d", sid)
		data, _ := json.Marshal(questions)
		err = rdb.Set(ctx, cacheKey, data, 24*time.Hour).Err()
		if err != nil {
			fmt.Printf("  Error caching to Redis: %v\n", err)
			continue
		}

		fmt.Printf("  Cached %d questions -> %s (TTL: 24h)\n", len(questions), cacheKey)
	}

	fmt.Println("\nWarm-up complete.")

	// Print cache status
	printCacheStatus(rdb, scheduleIDs)
}

func printCacheStatus(rdb *goredis.Client, scheduleIDs []uint) {
	ctx := context.Background()
	fmt.Println("\nCache status:")
	for _, sid := range scheduleIDs {
		key := fmt.Sprintf("exam_questions:%d", sid)
		ttl, err := rdb.TTL(ctx, key).Result()
		if err != nil || ttl < 0 {
			fmt.Printf("  %s: NOT FOUND\n", key)
		} else {
			size, _ := rdb.StrLen(ctx, key).Result()
			fmt.Printf("  %s: %s (TTL: %s)\n", key, formatBytes(size), ttl.Round(time.Second))
		}
	}
}

// --- backup ---

func cmdBackup(cfg *config.Config) {
	fs := flag.NewFlagSet("backup", flag.ExitOnError)
	format := fs.String("format", "sql", "Backup format: sql (plain text) or dump (pg_dump custom format)")
	_ = fs.Parse(os.Args[2:])

	if *format != "sql" && *format != "dump" {
		fmt.Printf("Error: --format must be 'sql' or 'dump', got '%s'\n", *format)
		os.Exit(1)
	}

	backupDir := "./backups"
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		fmt.Printf("Error creating backup directory: %v\n", err)
		os.Exit(1)
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("cbt_patra_%s.%s", timestamp, *format)
	filePath := filepath.Join(backupDir, filename)

	fmt.Printf("Creating backup of database '%s' (format: %s)...\n", cfg.DB.Database, *format)

	var args []string
	if *format == "dump" {
		args = []string{"-Fc"}
	}
	args = append(args,
		"-h", cfg.DB.Host,
		"-p", cfg.DB.Port,
		"-U", cfg.DB.Username,
		"-d", cfg.DB.Database,
		"-f", filePath,
	)

	cmd := exec.Command("pg_dump", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", cfg.DB.Password))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error running pg_dump: %v\n", err)
		os.Exit(1)
	}

	// Print file info
	info, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Backup created but could not stat file: %v\n", err)
		return
	}

	absPath, _ := filepath.Abs(filePath)
	fmt.Printf("Backup created successfully:\n")
	fmt.Printf("  Path: %s\n", absPath)
	fmt.Printf("  Size: %s\n", formatBytes(info.Size()))
}

// --- seed-permissions ---

func cmdSeedPermissions(db *gorm.DB) {
	type permDef struct {
		Name      string
		GroupName string
	}

	defaults := []permDef{
		// User Management
		{"users.view", "User Management"},
		{"users.create", "User Management"},
		{"users.edit", "User Management"},
		{"users.delete", "User Management"},
		{"users.import", "User Management"},

		// Master Data
		{"rombels.manage", "Master Data"},
		{"subjects.manage", "Master Data"},
		{"tags.manage", "Master Data"},
		{"rooms.manage", "Master Data"},

		// Question Banks
		{"question-banks.view", "Question Banks"},
		{"question-banks.create", "Question Banks"},
		{"question-banks.edit", "Question Banks"},
		{"question-banks.delete", "Question Banks"},
		{"questions.import", "Question Banks"},

		// Exam Schedules
		{"exam-schedules.view", "Exam Schedules"},
		{"exam-schedules.create", "Exam Schedules"},
		{"exam-schedules.edit", "Exam Schedules"},
		{"exam-schedules.delete", "Exam Schedules"},
		{"exam-schedules.publish", "Exam Schedules"},

		// Monitoring & Supervision
		{"monitoring.view", "Monitoring"},
		{"monitoring.actions", "Monitoring"},
		{"monitoring.reset-session", "Monitoring"},

		// Reports
		{"reports.view", "Reports"},
		{"reports.export", "Reports"},
		{"reports.regrade", "Reports"},
		{"reports.grade-essay", "Reports"},

		// Settings
		{"settings.view", "Settings"},
		{"settings.edit", "Settings"},
		{"settings.backup", "Settings"},
		{"settings.panic-mode", "Settings"},

		// Roles & Permissions
		{"roles.manage", "Roles & Permissions"},
		{"permissions.manage", "Roles & Permissions"},
	}

	created := 0
	skipped := 0

	for _, p := range defaults {
		var count int64
		db.Model(&entity.Permission{}).Where("name = ?", p.Name).Count(&count)
		if count > 0 {
			skipped++
			continue
		}

		perm := entity.Permission{
			Name:      p.Name,
			GroupName: p.GroupName,
		}
		if err := db.Create(&perm).Error; err != nil {
			fmt.Printf("  Error creating '%s': %v\n", p.Name, err)
			continue
		}
		created++
	}

	fmt.Printf("Permissions seeded: %d created, %d skipped (already exist)\n", created, skipped)
}

// --- Helpers ---

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// paginationAll returns pagination params that fetch all records.
func paginationAll() pagination.Params {
	return pagination.Params{Page: 1, PerPage: 10000}
}
