package master

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

// TagImportResult holds the result of a tag import operation.
type TagImportResult struct {
	TotalRows int              `json:"total_rows"`
	Assigned  int              `json:"assigned"`
	Removed   int              `json:"removed"`
	Skipped   int              `json:"skipped"`
	Errors    []TagImportError `json:"errors"`
}

// TagImportError represents an error in a specific row during import.
type TagImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

// ImportUserTags reads an Excel file and assigns/removes tags from users.
// Excel format:
//   - Column A: NIS
//   - Columns B onwards: Tag names (header row contains tag names)
//   - Cell values: "Ya" (assign), "Tidak" (remove), "Biarkan" or empty (skip)
func ImportUserTags(data []byte, db *gorm.DB) (*TagImportResult, error) {
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

	if len(rows) < 2 {
		return nil, errors.New("file Excel harus memiliki minimal header dan 1 baris data")
	}

	result := &TagImportResult{}

	// Parse header to get tag names (columns B onwards)
	header := rows[0]
	if len(header) < 2 {
		return nil, errors.New("file Excel harus memiliki kolom NIS dan minimal 1 kolom tag")
	}

	tagNames := make([]string, 0, len(header)-1)
	for i := 1; i < len(header); i++ {
		name := strings.TrimSpace(header[i])
		if name != "" {
			tagNames = append(tagNames, name)
		}
	}

	if len(tagNames) == 0 {
		return nil, errors.New("tidak ditemukan nama tag di header")
	}

	// Load tags from DB by name
	var tags []entity.Tag
	if err := db.Where("name IN ? AND deleted_at IS NULL", tagNames).Find(&tags).Error; err != nil {
		return nil, errors.New("gagal mengambil data tag: " + err.Error())
	}

	tagMap := make(map[string]entity.Tag, len(tags))
	for _, t := range tags {
		tagMap[strings.ToLower(t.Name)] = t
	}

	// Validate all tag names exist
	for _, name := range tagNames {
		if _, ok := tagMap[strings.ToLower(name)]; !ok {
			return nil, fmt.Errorf("tag '%s' tidak ditemukan di database", name)
		}
	}

	// Process data rows
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowNum := i + 1
		result.TotalRows++

		if len(row) == 0 {
			result.Skipped++
			continue
		}

		nis := strings.TrimSpace(safeColTag(row, 0))
		if nis == "" {
			result.Errors = append(result.Errors, TagImportError{
				Row: rowNum, Column: "NIS", Message: "NIS kosong",
			})
			result.Skipped++
			continue
		}

		// Find user by NIS
		var profile entity.UserProfile
		err := db.Where("nis = ?", nis).First(&profile).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result.Errors = append(result.Errors, TagImportError{
					Row: rowNum, Column: "NIS", Message: fmt.Sprintf("user dengan NIS '%s' tidak ditemukan", nis),
				})
				result.Skipped++
				continue
			}
			result.Errors = append(result.Errors, TagImportError{
				Row: rowNum, Column: "NIS", Message: "gagal mencari user: " + err.Error(),
			})
			result.Skipped++
			continue
		}

		user := entity.User{ID: profile.UserID}

		// Process each tag column
		for j, tagName := range tagNames {
			colIdx := j + 1
			value := strings.TrimSpace(strings.ToLower(safeColTag(row, colIdx)))

			tag, ok := tagMap[strings.ToLower(tagName)]
			if !ok {
				continue
			}

			switch value {
			case "ya":
				// syncWithoutDetaching: only append if not already associated
				tagEntity := entity.Tag{ID: tag.ID}
				assoc := db.Model(&tagEntity).Association("Users")
				// Use Append which won't duplicate in many2many
				if err := assoc.Append(&user); err != nil {
					result.Errors = append(result.Errors, TagImportError{
						Row: rowNum, Column: tagName, Message: "gagal menambahkan tag: " + err.Error(),
					})
				} else {
					result.Assigned++
				}
			case "tidak":
				tagEntity := entity.Tag{ID: tag.ID}
				if err := db.Model(&tagEntity).Association("Users").Delete(&user); err != nil {
					result.Errors = append(result.Errors, TagImportError{
						Row: rowNum, Column: tagName, Message: "gagal menghapus tag: " + err.Error(),
					})
				} else {
					result.Removed++
				}
			case "biarkan", "":
				// Skip
			default:
				result.Errors = append(result.Errors, TagImportError{
					Row: rowNum, Column: tagName, Message: fmt.Sprintf("nilai '%s' tidak valid (gunakan Ya/Tidak/Biarkan)", value),
				})
			}
		}
	}

	return result, nil
}

// safeColTag returns the column value at index or empty string if out of bounds.
func safeColTag(row []string, idx int) string {
	if idx < len(row) {
		return row[idx]
	}
	return ""
}
