package handler

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	pkgbcrypt "github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/response"
)

type ProfileHandler struct {
	userRepo repository.UserRepository
}

func NewProfileHandler(userRepo repository.UserRepository) *ProfileHandler {
	return &ProfileHandler{userRepo: userRepo}
}

// GET /profile
func (h *ProfileHandler) Get(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		response.NotFound(c, "User tidak ditemukan")
		return
	}
	response.Success(c, user)
}

// PUT /profile
func (h *ProfileHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		response.NotFound(c, "User tidak ditemukan")
		return
	}

	var req struct {
		Name  string  `json:"name"`
		Email *string `json:"email"`
		NIS   *string `json:"nis"`
		NIP   *string `json:"nip"`
		Class *string `json:"class"`
		Major *string `json:"major"`
		Phone *string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if req.Name != "" {
		user.Name = req.Name
	}
	if req.Email != nil {
		user.Email = req.Email
	}
	if user.Profile != nil {
		if req.NIS != nil {
			user.Profile.NIS = req.NIS
		}
		if req.NIP != nil {
			user.Profile.NIP = req.NIP
		}
		if req.Class != nil {
			user.Profile.Class = req.Class
		}
		if req.Major != nil {
			user.Profile.Major = req.Major
		}
		if req.Phone != nil {
			user.Profile.Phone = req.Phone
		}
	}

	if err := h.userRepo.Update(user); err != nil {
		response.InternalError(c, "Gagal memperbarui profil")
		return
	}
	response.Success(c, user)
}

// POST /profile/avatar
func (h *ProfileHandler) UploadAvatar(c *gin.Context) {
	userID := c.GetUint("user_id")

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "File avatar wajib diupload")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format file tidak didukung (jpg, png, webp)")
		return
	}

	uploadDir := "uploads/avatars"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.InternalError(c, "Gagal membuat direktori upload")
		return
	}

	filename := fmt.Sprintf("%d_%d%s", userID, time.Now().Unix(), ext)
	savePath := filepath.Join(uploadDir, filename)

	// Validate path traversal: ensure the resolved path stays within uploadDir
	cleanSavePath := filepath.Clean(savePath)
	cleanUploadDir := filepath.Clean(uploadDir)
	if !strings.HasPrefix(cleanSavePath, cleanUploadDir+string(os.PathSeparator)) {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Nama file tidak valid")
		return
	}

	if err := c.SaveUploadedFile(header, savePath); err != nil {
		response.InternalError(c, "Gagal menyimpan file")
		return
	}

	avatarURL := "/" + savePath
	if err := h.userRepo.UpdateAvatar(userID, avatarURL); err != nil {
		response.InternalError(c, "Gagal memperbarui avatar")
		return
	}

	response.Success(c, gin.H{"avatar_path": avatarURL})
}

// PUT /profile/password
func (h *ProfileHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.userRepo.FindByID(userID)
	if err != nil || user == nil {
		response.NotFound(c, "User tidak ditemukan")
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	if !pkgbcrypt.CheckPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"success": false,
			"error":   "Password lama tidak sesuai",
		})
		return
	}

	hashed, err := pkgbcrypt.HashPassword(req.NewPassword)
	if err != nil {
		response.InternalError(c, "Gagal hash password")
		return
	}
	user.Password = hashed

	if err := h.userRepo.Update(user); err != nil {
		response.InternalError(c, "Gagal mengubah password")
		return
	}

	// Invalidate all existing sessions by generating a new login_token
	newLoginToken := uuid.New().String()
	if err := h.userRepo.UpdateLoginToken(userID, newLoginToken); err != nil {
		// Password already changed, but session invalidation failed — log but don't fail the request
		response.Success(c, gin.H{"message": "Password berhasil diubah, silakan login ulang"})
		return
	}

	response.Success(c, gin.H{"message": "Password berhasil diubah, silakan login ulang"})
}
