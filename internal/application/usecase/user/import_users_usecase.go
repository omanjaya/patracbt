package user

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	pkgbcrypt "github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/xuri/excelize/v2"
)

type ImportResult struct {
	TotalRows    int           `json:"total_rows"`
	SuccessCount int           `json:"success_count"`
	FailedCount  int           `json:"failed_count"`
	Created      int           `json:"created"`
	Updated      int           `json:"updated"`
	Skipped      int           `json:"skipped"`
	Errors       []ImportError `json:"errors"`
}

type ImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

// validRoles defines the allowed roles for import.
var validRoles = map[string]bool{
	entity.RoleAdmin: true, entity.RoleGuru: true, entity.RolePengawas: true, entity.RolePeserta: true,
}

// parsedRow holds data extracted from a single Excel row.
type parsedRow struct {
	Row      int
	Name     string
	Username string
	Password string
	Role     string
	Email    string
	NIS      string
	NIP      string
	Class    string
	Major    string
	Phone    string
}

// ImportUsersFromExcel reads an Excel file with two-phase validation:
// Phase 1: Parse all rows and collect validation errors (including in-file and DB duplicates).
// Phase 2: If no critical errors, create all valid users.
// Expected columns: A=name, B=username, C=password, D=role, E=email, F=nis, G=nip, H=class, I=major, J=phone
func ImportUsersFromExcel(data []byte, userRepo repository.UserRepository) (*ImportResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, errors.New("gagal membaca file Excel: " + err.Error())
	}

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, errors.New("file Excel tidak memiliki sheet")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, err
	}

	result := &ImportResult{}

	// ── Phase 1: Parse and validate all rows ──
	var parsed []parsedRow
	seenUsernames := map[string]int{} // username -> first row number
	seenEmails := map[string]int{}    // email -> first row number
	seenNIS := map[string]int{}       // nis -> first row number
	seenNIP := map[string]int{}       // nip -> first row number
	allUsernames := []string{}
	allEmails := []string{}
	allNIS := []string{}
	allNIP := []string{}

	for i, row := range rows {
		if i == 0 {
			continue // skip header
		}
		rowNum := i + 1
		result.TotalRows++

		if len(row) < 3 {
			result.Errors = append(result.Errors, ImportError{
				Row: rowNum, Column: "-", Message: "kolom tidak lengkap (minimal: name, username, password)",
			})
			result.Skipped++
			continue
		}

		p := parsedRow{Row: rowNum}
		p.Name = strings.TrimSpace(safeCol(row, 0))
		p.Username = strings.TrimSpace(safeCol(row, 1))
		p.Password = strings.TrimSpace(safeCol(row, 2))
		p.Role = strings.TrimSpace(strings.ToLower(safeCol(row, 3)))
		p.Email = strings.TrimSpace(safeCol(row, 4))
		p.NIS = strings.TrimSpace(safeCol(row, 5))
		p.NIP = strings.TrimSpace(safeCol(row, 6))
		p.Class = strings.TrimSpace(safeCol(row, 7))
		p.Major = strings.TrimSpace(safeCol(row, 8))
		p.Phone = strings.TrimSpace(safeCol(row, 9))

		if p.Role == "" {
			p.Role = entity.RolePeserta
		}

		rowHasError := false

		// Required field validation
		if p.Name == "" {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "name", Message: "name wajib diisi"})
			rowHasError = true
		}
		if p.Username == "" {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "username", Message: "username wajib diisi"})
			rowHasError = true
		}
		if p.Password == "" {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "password", Message: "password wajib diisi"})
			rowHasError = true
		}

		// Role validation
		if !validRoles[p.Role] {
			result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "role", Message: fmt.Sprintf("role '%s' tidak valid (admin/guru/pengawas/peserta)", p.Role)})
			rowHasError = true
		}

		// In-file duplicate detection: username
		if p.Username != "" {
			lower := strings.ToLower(p.Username)
			if firstRow, exists := seenUsernames[lower]; exists {
				result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "username", Message: fmt.Sprintf("username '%s' duplikat dalam file (sama dengan baris %d)", p.Username, firstRow)})
				rowHasError = true
			} else {
				seenUsernames[lower] = rowNum
				allUsernames = append(allUsernames, p.Username)
			}
		}

		// In-file duplicate detection: email
		if p.Email != "" {
			lower := strings.ToLower(p.Email)
			if firstRow, exists := seenEmails[lower]; exists {
				result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "email", Message: fmt.Sprintf("email '%s' duplikat dalam file (sama dengan baris %d)", p.Email, firstRow)})
				rowHasError = true
			} else {
				seenEmails[lower] = rowNum
				allEmails = append(allEmails, p.Email)
			}
		}

		// In-file duplicate detection: NIS (only for peserta)
		if p.Role == entity.RolePeserta && p.NIS != "" {
			if firstRow, exists := seenNIS[p.NIS]; exists {
				result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "nis", Message: fmt.Sprintf("NIS '%s' duplikat dalam file (sama dengan baris %d)", p.NIS, firstRow)})
				rowHasError = true
			} else {
				seenNIS[p.NIS] = rowNum
				allNIS = append(allNIS, p.NIS)
			}
		}

		// In-file duplicate detection: NIP (only for non-peserta)
		if p.Role != entity.RolePeserta && p.NIP != "" {
			if firstRow, exists := seenNIP[p.NIP]; exists {
				result.Errors = append(result.Errors, ImportError{Row: rowNum, Column: "nip", Message: fmt.Sprintf("NIP '%s' duplikat dalam file (sama dengan baris %d)", p.NIP, firstRow)})
				rowHasError = true
			} else {
				seenNIP[p.NIP] = rowNum
				allNIP = append(allNIP, p.NIP)
			}
		}

		if rowHasError {
			result.Skipped++
			continue
		}

		parsed = append(parsed, p)
	}

	// DB duplicate detection (batch queries)
	dbDupUsernames, err := userRepo.FindExistingUsernames(allUsernames)
	if err != nil {
		return nil, errors.New("gagal memeriksa duplikat username: " + err.Error())
	}
	dbDupEmails, err := userRepo.FindExistingEmails(allEmails)
	if err != nil {
		return nil, errors.New("gagal memeriksa duplikat email: " + err.Error())
	}
	dbDupNIS, err := userRepo.FindExistingNIS(allNIS)
	if err != nil {
		return nil, errors.New("gagal memeriksa duplikat NIS: " + err.Error())
	}
	dbDupNIP, err := userRepo.FindExistingNIP(allNIP)
	if err != nil {
		return nil, errors.New("gagal memeriksa duplikat NIP: " + err.Error())
	}

	dupUsernameSet := toSet(dbDupUsernames)
	dupEmailSet := toSet(dbDupEmails)
	dupNISSet := toSet(dbDupNIS)
	dupNIPSet := toSet(dbDupNIP)

	// Filter out rows with DB duplicates
	var validRows []parsedRow
	for _, p := range parsed {
		rowHasError := false

		if _, dup := dupUsernameSet[strings.ToLower(p.Username)]; dup {
			result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "username", Message: fmt.Sprintf("username '%s' sudah ada di database", p.Username)})
			rowHasError = true
		}
		if p.Email != "" {
			if _, dup := dupEmailSet[strings.ToLower(p.Email)]; dup {
				result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "email", Message: fmt.Sprintf("email '%s' sudah ada di database", p.Email)})
				rowHasError = true
			}
		}
		if p.Role == entity.RolePeserta && p.NIS != "" {
			if _, dup := dupNISSet[p.NIS]; dup {
				result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "nis", Message: fmt.Sprintf("NIS '%s' sudah ada di database", p.NIS)})
				rowHasError = true
			}
		}
		if p.Role != entity.RolePeserta && p.NIP != "" {
			if _, dup := dupNIPSet[p.NIP]; dup {
				result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "nip", Message: fmt.Sprintf("NIP '%s' sudah ada di database", p.NIP)})
				rowHasError = true
			}
		}

		if rowHasError {
			result.Skipped++
			continue
		}
		validRows = append(validRows, p)
	}

	// If there are validation errors, return without saving
	if len(result.Errors) > 0 {
		result.FailedCount = result.Skipped
		return result, nil
	}

	// ── Phase 2: Create valid users inside a database transaction ──
	tx, txErr := userRepo.BeginTx()
	if txErr != nil {
		return nil, errors.New("gagal memulai transaksi: " + txErr.Error())
	}
	defer userRepo.RollbackTx(tx) // no-op if committed

	for _, p := range validRows {
		hashed, err := pkgbcrypt.HashPassword(p.Password)
		if err != nil {
			result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "password", Message: "gagal hash password"})
			result.FailedCount++
			continue
		}

		user := &entity.User{
			Name:     p.Name,
			Username: p.Username,
			Password: hashed,
			Role:     p.Role,
		}
		if p.Email != "" {
			user.Email = &p.Email
		}

		// Build profile if any profile field is provided
		if p.NIS != "" || p.NIP != "" || p.Class != "" || p.Major != "" || p.Phone != "" {
			profile := &entity.UserProfile{}
			if p.NIS != "" {
				profile.NIS = &p.NIS
			}
			if p.NIP != "" {
				profile.NIP = &p.NIP
			}
			if p.Class != "" {
				profile.Class = &p.Class
			}
			if p.Major != "" {
				profile.Major = &p.Major
			}
			if p.Phone != "" {
				profile.Phone = &p.Phone
			}
			user.Profile = profile
		}

		if err := userRepo.CreateInTx(tx, user); err != nil {
			result.Errors = append(result.Errors, ImportError{Row: p.Row, Column: "-", Message: err.Error()})
			result.FailedCount++
			continue
		}
		result.Created++
	}

	// Commit transaction — all successful inserts are atomic
	if result.Created > 0 {
		if err := userRepo.CommitTx(tx); err != nil {
			return nil, errors.New("gagal commit transaksi: " + err.Error())
		}
	}

	result.SuccessCount = result.Created
	result.FailedCount += result.Skipped

	return result, nil
}

// safeCol returns the column value at index or empty string if out of bounds.
func safeCol(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}

// toSet converts a string slice to a lowercase set map.
func toSet(items []string) map[string]struct{} {
	m := make(map[string]struct{}, len(items))
	for _, item := range items {
		m[strings.ToLower(item)] = struct{}{}
	}
	return m
}
