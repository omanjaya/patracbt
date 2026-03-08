package handler

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/response"
	qrcode "github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

type CardHandler struct {
	db          *gorm.DB
	settingRepo repository.SettingRepository
}

func NewCardHandler(db *gorm.DB, settingRepo repository.SettingRepository) *CardHandler {
	return &CardHandler{
		db:          db,
		settingRepo: settingRepo,
	}
}

type cardStudent struct {
	Name          string  `json:"name"`
	Username      string  `json:"username"`
	PasswordPlain string  `json:"password_plain"`
	NIS           *string `json:"nis"`
	RombelName    string  `json:"rombel_name"`
	RoomName      string  `json:"room_name"`
	AvatarURL     string  `json:"avatar_url"`
}

type cardSettings struct {
	AppName        string `json:"app_name"`
	SchoolName     string `json:"school_name"`
	HeadmasterName string `json:"headmaster_name"`
	HeadmasterNIP  string `json:"headmaster_nip"`
	LogoURL        string `json:"logo_url"`
	HeaderColor    string `json:"header_color"`
}

// GetCards returns student data for card printing.
// GET /admin/cards?rombel_id=X
func (h *CardHandler) GetCards(c *gin.Context) {
	// Build base query: peserta users with profile
	q := h.db.Table("users").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.role = ?", entity.RolePeserta).
		Where("users.deleted_at IS NULL")

	// Optional rombel filter
	rombelIDStr := c.Query("rombel_id")
	if rombelIDStr != "" {
		rombelID, err := strconv.ParseUint(rombelIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "rombel_id tidak valid")
			return
		}
		q = q.Joins("JOIN user_rombels ON user_rombels.user_id = users.id").
			Where("user_rombels.rombel_id = ?", rombelID)
	}

	type userRow struct {
		ID         uint
		Name       string
		Username   string
		NIS        *string
		RoomID     *uint
		AvatarPath *string
	}

	rows := make([]userRow, 0)
	if err := q.Select("users.id, users.name, users.username, users.avatar_path, user_profiles.nis, user_profiles.room_id").Order("users.name ASC").Find(&rows).Error; err != nil {
		response.InternalError(c, "Gagal mengambil data peserta")
		return
	}

	// Collect user IDs for rombel lookup & room IDs
	userIDs := make([]uint, len(rows))
	roomIDSet := make(map[uint]struct{})
	for i, r := range rows {
		userIDs[i] = r.ID
		if r.RoomID != nil {
			roomIDSet[*r.RoomID] = struct{}{}
		}
	}

	// Get rombel names per user
	type rombelRow struct {
		UserID     uint
		RombelName string
	}
	rombelRows := make([]rombelRow, 0)
	if len(userIDs) > 0 {
		h.db.Table("user_rombels").
			Select("user_rombels.user_id, rombels.name as rombel_name").
			Joins("JOIN rombels ON rombels.id = user_rombels.rombel_id").
			Where("user_rombels.user_id IN ?", userIDs).
			Find(&rombelRows)
	}

	rombelMap := make(map[uint]string)
	for _, rr := range rombelRows {
		if existing, ok := rombelMap[rr.UserID]; ok {
			rombelMap[rr.UserID] = existing + ", " + rr.RombelName
		} else {
			rombelMap[rr.UserID] = rr.RombelName
		}
	}

	// Get room names
	roomMap := make(map[uint]string)
	if len(roomIDSet) > 0 {
		roomIDs := make([]uint, 0, len(roomIDSet))
		for id := range roomIDSet {
			roomIDs = append(roomIDs, id)
		}
		type roomRow struct {
			ID   uint
			Name string
		}
		roomRows := make([]roomRow, 0)
		h.db.Table("rooms").Select("id, name").Where("id IN ?", roomIDs).Find(&roomRows)
		for _, rr := range roomRows {
			roomMap[rr.ID] = rr.Name
		}
	}

	// Build student list
	students := make([]cardStudent, 0, len(rows))
	for _, r := range rows {
		roomName := ""
		if r.RoomID != nil {
			roomName = roomMap[*r.RoomID]
		}
		avatarURL := ""
		if r.AvatarPath != nil && *r.AvatarPath != "" {
			avatarURL = *r.AvatarPath
		}
		students = append(students, cardStudent{
			Name:          r.Name,
			Username:      r.Username,
			PasswordPlain: "", // not stored in plain text
			NIS:           r.NIS,
			RombelName:    rombelMap[r.ID],
			RoomName:      roomName,
			AvatarURL:     avatarURL,
		})
	}

	// Get settings
	settings := h.loadCardSettings()

	response.Success(c, gin.H{
		"settings": settings,
		"students": students,
	})
}

// GetCardSettings returns settings for card customization.
// GET /admin/cards/settings
func (h *CardHandler) GetCardSettings(c *gin.Context) {
	response.Success(c, h.loadCardSettings())
}

func (h *CardHandler) loadCardSettings() cardSettings {
	cs := cardSettings{
		HeaderColor: "#2c3e50",
	}

	if s, _ := h.settingRepo.GetByKey("app_name"); s != nil && s.Value != nil {
		cs.AppName = *s.Value
	}
	if s, _ := h.settingRepo.GetByKey("school_name"); s != nil && s.Value != nil {
		cs.SchoolName = *s.Value
	}
	if s, _ := h.settingRepo.GetByKey("headmaster_name"); s != nil && s.Value != nil {
		cs.HeadmasterName = *s.Value
	}
	if s, _ := h.settingRepo.GetByKey("headmaster_nip"); s != nil && s.Value != nil {
		cs.HeadmasterNIP = *s.Value
	}
	if s, _ := h.settingRepo.GetByKey("logo_url"); s != nil && s.Value != nil {
		cs.LogoURL = *s.Value
	}
	if s, _ := h.settingRepo.GetByKey("header_color"); s != nil && s.Value != nil && *s.Value != "" {
		cs.HeaderColor = *s.Value
	}

	return cs
}

// GenerateQR generates a QR code image for a student's login credentials.
// GET /admin/cards/qr?username=XXX&password=XXX
func (h *CardHandler) GenerateQR(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		response.BadRequest(c, "username required")
		return
	}

	password := c.Query("password")
	qrData := fmt.Sprintf(`{"u":"%s","p":"%s"}`, username, password)

	png, err := qrcode.Encode(qrData, qrcode.Medium, 200)
	if err != nil {
		response.InternalError(c, "Gagal membuat QR code")
		return
	}

	c.Data(http.StatusOK, "image/png", png)
}

type cardStudentWithQR struct {
	cardStudent
	QRBase64 string `json:"qr_base64"`
}

// GetCardsWithQR returns card data including base64 QR codes.
// GET /admin/cards/with-qr?rombel_id=X
func (h *CardHandler) GetCardsWithQR(c *gin.Context) {
	// Build base query: peserta users with profile
	q := h.db.Table("users").
		Joins("LEFT JOIN user_profiles ON user_profiles.user_id = users.id").
		Where("users.role = ?", entity.RolePeserta).
		Where("users.deleted_at IS NULL")

	// Optional rombel filter
	rombelIDStr := c.Query("rombel_id")
	if rombelIDStr != "" {
		rombelID, err := strconv.ParseUint(rombelIDStr, 10, 64)
		if err != nil {
			response.BadRequest(c, "rombel_id tidak valid")
			return
		}
		q = q.Joins("JOIN user_rombels ON user_rombels.user_id = users.id").
			Where("user_rombels.rombel_id = ?", rombelID)
	}

	type userRow struct {
		ID         uint
		Name       string
		Username   string
		NIS        *string
		RoomID     *uint
		AvatarPath *string
	}

	rows := make([]userRow, 0)
	if err := q.Select("users.id, users.name, users.username, users.avatar_path, user_profiles.nis, user_profiles.room_id").Order("users.name ASC").Find(&rows).Error; err != nil {
		response.InternalError(c, "Gagal mengambil data peserta")
		return
	}

	// Collect user IDs for rombel lookup & room IDs
	userIDs := make([]uint, len(rows))
	roomIDSet := make(map[uint]struct{})
	for i, r := range rows {
		userIDs[i] = r.ID
		if r.RoomID != nil {
			roomIDSet[*r.RoomID] = struct{}{}
		}
	}

	// Get rombel names per user
	type rombelRow struct {
		UserID     uint
		RombelName string
	}
	rombelRows := make([]rombelRow, 0)
	if len(userIDs) > 0 {
		h.db.Table("user_rombels").
			Select("user_rombels.user_id, rombels.name as rombel_name").
			Joins("JOIN rombels ON rombels.id = user_rombels.rombel_id").
			Where("user_rombels.user_id IN ?", userIDs).
			Find(&rombelRows)
	}

	rombelMap := make(map[uint]string)
	for _, rr := range rombelRows {
		if existing, ok := rombelMap[rr.UserID]; ok {
			rombelMap[rr.UserID] = existing + ", " + rr.RombelName
		} else {
			rombelMap[rr.UserID] = rr.RombelName
		}
	}

	// Get room names
	roomMap := make(map[uint]string)
	if len(roomIDSet) > 0 {
		roomIDs := make([]uint, 0, len(roomIDSet))
		for id := range roomIDSet {
			roomIDs = append(roomIDs, id)
		}
		type roomRow struct {
			ID   uint
			Name string
		}
		roomRows := make([]roomRow, 0)
		h.db.Table("rooms").Select("id, name").Where("id IN ?", roomIDs).Find(&roomRows)
		for _, rr := range roomRows {
			roomMap[rr.ID] = rr.Name
		}
	}

	// Build student list with QR codes
	students := make([]cardStudentWithQR, 0, len(rows))
	for _, r := range rows {
		roomName := ""
		if r.RoomID != nil {
			roomName = roomMap[*r.RoomID]
		}
		avatarURL := ""
		if r.AvatarPath != nil && *r.AvatarPath != "" {
			avatarURL = *r.AvatarPath
		}
		cs := cardStudent{
			Name:          r.Name,
			Username:      r.Username,
			PasswordPlain: "",
			NIS:           r.NIS,
			RombelName:    rombelMap[r.ID],
			RoomName:      roomName,
			AvatarURL:     avatarURL,
		}

		// Generate QR code as base64
		qrData := fmt.Sprintf(`{"u":"%s","p":""}`, r.Username)
		png, err := qrcode.Encode(qrData, qrcode.Medium, 200)
		qrB64 := ""
		if err == nil {
			qrB64 = "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)
		}

		students = append(students, cardStudentWithQR{
			cardStudent: cs,
			QRBase64:    qrB64,
		})
	}

	settings := h.loadCardSettings()

	response.Success(c, gin.H{
		"settings": settings,
		"students": students,
	})
}
