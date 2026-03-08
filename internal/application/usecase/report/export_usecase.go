package report

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"github.com/xuri/excelize/v2"
)

// ExportLedger generates an Excel ledger of all finished sessions for a schedule.
func ExportLedger(
	scheduleID uint,
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) ([]byte, string, error) {
	return exportLedger(scheduleID, false, sessionRepo, scheduleRepo, questionRepo)
}

// ExportLedgerMultiSheet generates an Excel ledger with one sheet per rombel.
func ExportLedgerMultiSheet(
	scheduleID uint,
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) ([]byte, string, error) {
	return exportLedger(scheduleID, true, sessionRepo, scheduleRepo, questionRepo)
}

func exportLedger(
	scheduleID uint,
	multiSheet bool,
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
	questionRepo repository.QuestionRepository,
) ([]byte, string, error) {
	schedule, err := scheduleRepo.FindByID(scheduleID)
	if err != nil || schedule == nil {
		return nil, "", fmt.Errorf("jadwal tidak ditemukan")
	}

	sessions, err := sessionRepo.ListFinishedBySchedule(scheduleID)
	if err != nil {
		return nil, "", err
	}

	// Load all questions for all banks in the schedule
	var questions []*entity.Question
	p := pagination.Params{Page: 1, PerPage: 9999}
	for _, bankRef := range schedule.QuestionBanks {
		qs, _, _ := questionRepo.ListByBank(bankRef.QuestionBankID, p)
		questions = append(questions, qs...)
	}

	f := excelize.NewFile()

	if !multiSheet || len(sessions) == 0 {
		// Single-sheet mode (default)
		sheet := "Ledger Nilai"
		f.SetSheetName("Sheet1", sheet)
		writeSheetData(f, sheet, sessions, questions)
	} else {
		// Multi-sheet mode: group by rombel
		userIDs := make([]uint, 0, len(sessions))
		for _, s := range sessions {
			userIDs = append(userIDs, s.UserID)
		}
		userRombels, _ := sessionRepo.GetUserRombelNames(userIDs)

		// Group sessions by rombel name
		grouped := make(map[string][]*entity.ExamSession)
		for _, s := range sessions {
			rombels := userRombels[s.UserID]
			if len(rombels) == 0 {
				rombels = []string{"Tanpa Rombel"}
			}
			for _, rName := range rombels {
				grouped[rName] = append(grouped[rName], s)
			}
		}

		// Remove default Sheet1 only after creating at least one sheet
		first := true
		for rombelName, rombelSessions := range grouped {
			sheetName := sanitizeSheetName(rombelName)
			if first {
				f.SetSheetName("Sheet1", sheetName)
				first = false
			} else {
				f.NewSheet(sheetName)
			}
			writeSheetData(f, sheetName, rombelSessions, questions)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("ledger_%s.xlsx", sanitizeFilename(schedule.Name))
	return buf.Bytes(), filename, nil
}

// writeSheetData writes the header and session rows to a sheet.
func writeSheetData(f *excelize.File, sheet string, sessions []*entity.ExamSession, questions []*entity.Question) {
	// Header row
	headers := []string{"No", "Nama", "NIS", "Username", "Nilai", "Pelanggaran"}
	for i, q := range questions {
		headers = append(headers, fmt.Sprintf("S%d (%s)", i+1, strings.ToUpper(q.QuestionType)))
	}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Bold header
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	endCell, _ := excelize.CoordinatesToCellName(len(headers), 1)
	f.SetCellStyle(sheet, "A1", endCell, boldStyle)

	// Build answer map per session
	opts := []string{"A", "B", "C", "D", "E"}
	for rowIdx, session := range sessions {
		row := rowIdx + 2

		nis := ""
		if session.User.Profile != nil && session.User.Profile.NIS != nil {
			nis = *session.User.Profile.NIS
		}

		colSet(f, sheet, 1, row, rowIdx+1)
		colSet(f, sheet, 2, row, session.User.Name)
		colSet(f, sheet, 3, row, nis)
		colSet(f, sheet, 4, row, session.User.Username)
		colSet(f, sheet, 5, row, session.Score)
		colSet(f, sheet, 6, row, session.ViolationCount)

		answerMap := make(map[uint]entity.ExamAnswer)
		for _, a := range session.Answers {
			answerMap[a.QuestionID] = a
		}

		for qIdx, q := range questions {
			col := qIdx + 7
			text := "-"
			if a, ok := answerMap[q.ID]; ok && a.Answer != nil {
				switch q.QuestionType {
				case entity.QuestionTypePG, entity.QuestionTypeBenarSalah:
					var ans map[string]interface{}
					if json.Unmarshal(a.Answer, &ans) == nil {
						if idx, ok := ans["option_index"].(float64); ok && int(idx) < len(opts) {
							text = opts[int(idx)]
						}
					}
				default:
					text = "✓"
				}
			}
			colSet(f, sheet, col, row, text)
		}
	}
}

func colSet(f *excelize.File, sheet string, col, row int, value interface{}) {
	cell, _ := excelize.CoordinatesToCellName(col, row)
	f.SetCellValue(sheet, cell, value)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, " ", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// ExportUnfinished generates an Excel file listing students who haven't finished the exam.
func ExportUnfinished(
	scheduleID uint,
	sessionRepo repository.ExamSessionRepository,
	scheduleRepo repository.ExamScheduleRepository,
) ([]byte, string, error) {
	schedule, err := scheduleRepo.FindByID(scheduleID)
	if err != nil || schedule == nil {
		return nil, "", fmt.Errorf("jadwal tidak ditemukan")
	}

	// Get ongoing sessions
	ongoing, err := sessionRepo.ListOngoingBySchedule(scheduleID)
	if err != nil {
		return nil, "", fmt.Errorf("gagal mengambil sesi berlangsung: %w", err)
	}

	// Get not-started sessions
	notStarted, err := sessionRepo.ListNotStartedBySchedule(scheduleID)
	if err != nil {
		return nil, "", fmt.Errorf("gagal mengambil sesi belum mulai: %w", err)
	}

	// Collect all user IDs for rombel lookup
	allUserIDs := make([]uint, 0, len(ongoing)+len(notStarted))
	for _, s := range ongoing {
		allUserIDs = append(allUserIDs, s.UserID)
	}
	for _, s := range notStarted {
		allUserIDs = append(allUserIDs, s.UserID)
	}

	userRombels := make(map[uint][]string)
	if len(allUserIDs) > 0 {
		userRombels, _ = sessionRepo.GetUserRombelNames(allUserIDs)
	}

	f := excelize.NewFile()
	sheet := "Belum Selesai"
	f.SetSheetName("Sheet1", sheet)

	// Header
	headers := []string{"No", "Nama", "NIS", "Username", "Rombel", "Status"}
	for col, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Bold header style
	boldStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	endCell, _ := excelize.CoordinatesToCellName(len(headers), 1)
	f.SetCellStyle(sheet, "A1", endCell, boldStyle)

	row := 2
	no := 1

	// Write not-started sessions first
	for _, s := range notStarted {
		nis := ""
		if s.User.Profile != nil && s.User.Profile.NIS != nil {
			nis = *s.User.Profile.NIS
		}
		rombel := strings.Join(userRombels[s.UserID], ", ")
		if rombel == "" {
			rombel = "-"
		}

		colSet(f, sheet, 1, row, no)
		colSet(f, sheet, 2, row, s.User.Name)
		colSet(f, sheet, 3, row, nis)
		colSet(f, sheet, 4, row, s.User.Username)
		colSet(f, sheet, 5, row, rombel)
		colSet(f, sheet, 6, row, "Belum Mulai")
		row++
		no++
	}

	// Write ongoing sessions
	for _, s := range ongoing {
		nis := ""
		if s.User.Profile != nil && s.User.Profile.NIS != nil {
			nis = *s.User.Profile.NIS
		}
		rombel := strings.Join(userRombels[s.UserID], ", ")
		if rombel == "" {
			rombel = "-"
		}

		status := "Sedang Mengerjakan"
		if s.Status == entity.SessionStatusTerminated {
			status = "Terblokir"
		}

		colSet(f, sheet, 1, row, no)
		colSet(f, sheet, 2, row, s.User.Name)
		colSet(f, sheet, 3, row, nis)
		colSet(f, sheet, 4, row, s.User.Username)
		colSet(f, sheet, 5, row, rombel)
		colSet(f, sheet, 6, row, status)
		row++
		no++
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, "", err
	}

	filename := fmt.Sprintf("belum_selesai_%s.xlsx", sanitizeFilename(schedule.Name))
	return buf.Bytes(), filename, nil
}

// sanitizeSheetName ensures the sheet name is valid for Excel (max 31 chars, no special chars).
func sanitizeSheetName(s string) string {
	// Remove characters invalid in Excel sheet names: \ / * ? : [ ]
	invalid := []string{"\\", "/", "*", "?", ":", "[", "]"}
	for _, ch := range invalid {
		s = strings.ReplaceAll(s, ch, "")
	}
	s = strings.TrimSpace(s)
	if len(s) > 31 {
		s = s[:31]
	}
	if s == "" {
		s = "Sheet"
	}
	return s
}
