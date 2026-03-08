package handler

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/application/usecase/auth"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/infrastructure/persistence/postgres"
	"github.com/omanjaya/patra/pkg/jwt"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/response"
)

type AuthHandler struct {
	loginUC        *auth.LoginUseCase
	refreshTokenUC *auth.RefreshTokenUseCase
	userRepo       repository.UserRepository
	cfg            *config.Config
	auditRepo      *postgres.AuditLogRepo
}

func NewAuthHandler(
	loginUC *auth.LoginUseCase,
	refreshTokenUC *auth.RefreshTokenUseCase,
	userRepo repository.UserRepository,
	cfg *config.Config,
	auditRepo ...*postgres.AuditLogRepo,
) *AuthHandler {
	h := &AuthHandler{
		loginUC:        loginUC,
		refreshTokenUC: refreshTokenUC,
		userRepo:       userRepo,
		cfg:            cfg,
	}
	if len(auditRepo) > 0 {
		h.auditRepo = auditRepo[0]
	}
	return h
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, gin.H{"login": "Login wajib diisi", "password": "Password wajib diisi"})
		return
	}

	result, err := h.loginUC.Execute(req)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			response.Unauthorized(c, "Username atau password salah")
			return
		}
		if errors.Is(err, auth.ErrUserInactive) {
			response.Error(c, http.StatusForbidden, "USER_INACTIVE", "Akun Anda tidak aktif, hubungi administrator")
			return
		}
		if errors.Is(err, auth.ErrSessionExists) {
			response.Error(c, http.StatusConflict, "SESSION_EXISTS", "User sudah login di perangkat lain. Kirim ulang dengan force_login=true untuk melanjutkan.")
			return
		}
		if errors.Is(err, auth.ErrExamInProgress) {
			response.Error(c, http.StatusConflict, "EXAM_IN_PROGRESS", "User sedang mengerjakan ujian. Tidak dapat login paksa.")
			return
		}
		response.InternalError(c, "Gagal melakukan login")
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	response.Success(c, gin.H{"message": "Logout berhasil"})
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.ValidationError(c, gin.H{"refresh_token": "Refresh token wajib diisi"})
		return
	}

	result, err := h.refreshTokenUC.Execute(req)
	if err != nil {
		response.Unauthorized(c, "Refresh token tidak valid")
		return
	}

	response.Success(c, result)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, _ := c.Get("user_id")
	id, ok := userID.(uint)
	if !ok {
		response.Unauthorized(c, "Token tidak valid")
		return
	}

	user, err := h.userRepo.FindByID(id)
	if err != nil || user == nil {
		response.NotFound(c, "User tidak ditemukan")
		return
	}

	resp := dto.UserResponse{
		ID:        user.ID,
		Name:      user.Name,
		Username:  user.Username,
		Role:      user.Role,
		AvatarURL: user.AvatarPath,
	}

	if user.Profile != nil {
		resp.Profile = &dto.ProfileResponse{
			NIS:   user.Profile.NIS,
			Class: user.Profile.Class,
			Major: user.Profile.Major,
		}
	}

	response.Success(c, resp)
}

// PreviewAsPeserta generates a short-lived JWT for admin to preview as a peserta user.
// POST /api/v1/admin/preview-as-peserta
func (h *AuthHandler) PreviewAsPeserta(c *gin.Context) {
	var body struct {
		UserID uint `json:"user_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	target, err := h.userRepo.FindByID(body.UserID)
	if err != nil || target == nil {
		response.NotFound(c, "User tidak ditemukan")
		return
	}
	if target.Role != entity.RolePeserta {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "User bukan peserta")
		return
	}

	// Generate a short-lived access token (1 hour) for the target peserta
	previewTTL := 1 * time.Hour
	tokenPair, err := jwt.GenerateTokenPair(
		target.ID, target.Username, target.Role,
		h.cfg.JWT.AccessSecret, h.cfg.JWT.RefreshSecret,
		previewTTL, previewTTL,
	)
	if err != nil {
		response.InternalError(c, "Gagal membuat token preview")
		return
	}

	adminUserID := c.GetUint("user_id")
	logger.Log.Infow("admin preview-as-peserta",
		"admin_id", adminUserID,
		"target_user_id", target.ID,
		"ip", c.ClientIP(),
	)

	if h.auditRepo != nil {
		if err := h.auditRepo.Create(&entity.AuditLog{
			UserID:     adminUserID,
			Action:     "preview_as_peserta",
			TargetID:   target.ID,
			TargetType: "user",
			IPAddress:  c.ClientIP(),
			Details:    fmt.Sprintf(`{"target_username":"%s"}`, target.Username),
		}); err != nil {
			logger.Log.Warnf("Failed to create audit log: %v", err)
		}
	}

	response.Success(c, gin.H{
		"preview_token":   tokenPair.AccessToken,
		"peserta_user_id": target.ID,
		"expires_in":      int64(previewTTL.Seconds()),
	})
}

// PreviewBack allows admin to return to their original session after previewing as peserta.
// POST /api/v1/admin/preview-back
func (h *AuthHandler) PreviewBack(c *gin.Context) {
	var body struct {
		AdminToken string `json:"admin_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "admin_token wajib diisi")
		return
	}

	// Quick format check: a valid JWT has 3 dot-separated parts and is
	// never shorter than ~36 characters. Reject obviously invalid strings
	// early to avoid unnecessary JWT parsing overhead.
	if len(body.AdminToken) < 10 {
		response.Error(c, http.StatusBadRequest, "BAD_REQUEST", "Format admin_token tidak valid")
		return
	}

	// Validate the original admin token
	claims, err := jwt.ValidateToken(body.AdminToken, h.cfg.JWT.AccessSecret)
	if err != nil {
		response.Unauthorized(c, "Token admin tidak valid atau expired")
		return
	}

	// Verify it's an admin token
	if claims.Role != entity.RoleAdmin {
		response.Forbidden(c, "Token bukan milik admin")
		return
	}

	// Verify admin user still exists and has admin role
	admin, err := h.userRepo.FindByID(claims.UserID)
	if err != nil || admin == nil {
		response.NotFound(c, "User admin tidak ditemukan")
		return
	}
	if admin.Role != entity.RoleAdmin {
		response.Forbidden(c, "User bukan admin")
		return
	}

	// Verify admin's LoginToken matches the token in JWT claims
	if claims.LoginToken != "" {
		if admin.LoginToken == nil || *admin.LoginToken != claims.LoginToken {
			response.Unauthorized(c, "Sesi admin telah berakhir, silakan login kembali")
			return
		}
	}

	// Audit log
	if h.auditRepo != nil {
		if err := h.auditRepo.Create(&entity.AuditLog{
			UserID:     claims.UserID,
			Action:     "preview_back",
			TargetID:   claims.UserID,
			TargetType: "user",
			IPAddress:  c.ClientIP(),
			Details:    fmt.Sprintf(`{"admin_username":"%s"}`, admin.Username),
		}); err != nil {
			logger.Log.Warnf("Failed to create audit log: %v", err)
		}
	}

	response.Success(c, gin.H{
		"access_token": body.AdminToken,
		"user_id":      admin.ID,
		"role":         admin.Role,
	})
}
