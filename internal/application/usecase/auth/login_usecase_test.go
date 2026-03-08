package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/omanjaya/patra/config"
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/omanjaya/patra/pkg/logger"
	"github.com/omanjaya/patra/pkg/pagination"
)

func init() {
	logger.Init("test")
}

// ─── Mock User Repository ──────────────────────────────────────

type mockUserRepo struct {
	findByUsernameOrEmailFn func(login string) (*entity.User, error)
	updateLastLoginFn       func(id uint) error
	updateLoginTokenFn      func(id uint, token string) error
}

func (m *mockUserRepo) FindByUsernameOrEmail(login string) (*entity.User, error) {
	if m.findByUsernameOrEmailFn != nil {
		return m.findByUsernameOrEmailFn(login)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepo) UpdateLastLogin(id uint) error {
	if m.updateLastLoginFn != nil {
		return m.updateLastLoginFn(id)
	}
	return nil
}

func (m *mockUserRepo) UpdateLoginToken(id uint, token string) error {
	if m.updateLoginTokenFn != nil {
		return m.updateLoginTokenFn(id, token)
	}
	return nil
}

// Stubs for remaining interface methods — not used in login tests.
func (m *mockUserRepo) Create(_ *entity.User) error                              { return nil }
func (m *mockUserRepo) CreateInTx(_ interface{}, _ *entity.User) error           { return nil }
func (m *mockUserRepo) BeginTx() (interface{}, error)                            { return nil, nil }
func (m *mockUserRepo) CommitTx(_ interface{}) error                             { return nil }
func (m *mockUserRepo) RollbackTx(_ interface{})                                 {}
func (m *mockUserRepo) FindByID(_ uint) (*entity.User, error)                    { return nil, nil }
func (m *mockUserRepo) FindByUsername(_ string) (*entity.User, error)            { return nil, nil }
func (m *mockUserRepo) Update(_ *entity.User) error                              { return nil }
func (m *mockUserRepo) Delete(_ uint) error                                      { return nil }
func (m *mockUserRepo) Restore(_ uint) error                                     { return nil }
func (m *mockUserRepo) ForceDelete(_ uint) error                                 { return nil }
func (m *mockUserRepo) UpdateAvatar(_ uint, _ string) error                      { return nil }
func (m *mockUserRepo) List(_ repository.UserListFilter, _ pagination.Params) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) ListTrashed(_ repository.UserListFilter, _ pagination.Params) ([]*entity.User, int64, error) {
	return nil, 0, nil
}
func (m *mockUserRepo) BulkCreate(_ []*entity.User) error               { return nil }
func (m *mockUserRepo) FindByEmail(_ string) (*entity.User, error)      { return nil, nil }
func (m *mockUserRepo) FindExistingUsernames(_ []string) ([]string, error) { return nil, nil }
func (m *mockUserRepo) FindExistingEmails(_ []string) ([]string, error)    { return nil, nil }
func (m *mockUserRepo) FindExistingNIS(_ []string) ([]string, error)       { return nil, nil }
func (m *mockUserRepo) FindExistingNIP(_ []string) ([]string, error)       { return nil, nil }
func (m *mockUserRepo) BulkDelete(_ []uint) error                         { return nil }
func (m *mockUserRepo) BulkRestore(_ []uint) error                        { return nil }
func (m *mockUserRepo) BulkForceDelete(_ []uint) error                    { return nil }

var _ repository.UserRepository = (*mockUserRepo)(nil)

// ─── Mock Exam Session Repository (minimal for login) ──────────

type mockExamSessionRepo struct {
	findOngoingByUserFn func(userID uint) (*entity.ExamSession, error)
}

func (m *mockExamSessionRepo) FindOngoingByUser(userID uint) (*entity.ExamSession, error) {
	if m.findOngoingByUserFn != nil {
		return m.findOngoingByUserFn(userID)
	}
	return nil, nil
}

// Stubs for remaining interface methods.
func (m *mockExamSessionRepo) Create(_ *entity.ExamSession) error               { return nil }
func (m *mockExamSessionRepo) FindByID(_ uint) (*entity.ExamSession, error)     { return nil, nil }
func (m *mockExamSessionRepo) FindByIDBasic(_ uint) (*entity.ExamSession, error) { return nil, nil }
func (m *mockExamSessionRepo) FindByUserAndSchedule(_, _ uint) (*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockExamSessionRepo) Update(_ *entity.ExamSession) error                     { return nil }
func (m *mockExamSessionRepo) AtomicFinish(_ uint) (int64, error)                     { return 0, nil }
func (m *mockExamSessionRepo) UpdateStatus(_ uint, _ string, _ *time.Time) error      { return nil }
func (m *mockExamSessionRepo) UpdateScore(_ uint, _, _ float64) error                 { return nil }
func (m *mockExamSessionRepo) IncrementViolation(_ uint) error                        { return nil }
func (m *mockExamSessionRepo) UpdateExtraTime(_ uint, _ int) error                    { return nil }
func (m *mockExamSessionRepo) ListBySchedule(_ uint, _ pagination.Params) ([]*entity.ExamSession, int64, error) {
	return nil, 0, nil
}
func (m *mockExamSessionRepo) ListByUser(_ uint) ([]*entity.ExamSession, error) { return nil, nil }
func (m *mockExamSessionRepo) UpsertAnswer(_ *entity.ExamAnswer) error          { return nil }
func (m *mockExamSessionRepo) BatchUpsertAnswers(_ []*entity.ExamAnswer) error  { return nil }
func (m *mockExamSessionRepo) GetAnswer(_, _ uint) (*entity.ExamAnswer, error)  { return nil, nil }
func (m *mockExamSessionRepo) GetAllAnswers(_ uint) ([]entity.ExamAnswer, error) { return nil, nil }
func (m *mockExamSessionRepo) GetAllAnswersBySchedule(_ uint) (map[uint][]entity.ExamAnswer, error) {
	return nil, nil
}
func (m *mockExamSessionRepo) DeleteAnswersBySession(_ uint) error                    { return nil }
func (m *mockExamSessionRepo) CountNonEmptyAnswers(_ uint) (int, error)               { return 0, nil }
func (m *mockExamSessionRepo) LogViolation(_ *entity.ViolationLog) error              { return nil }
func (m *mockExamSessionRepo) CountViolations(_ uint) (int, error)                    { return 0, nil }
func (m *mockExamSessionRepo) UserInRombels(_ uint, _ []uint) (bool, error)           { return false, nil }
func (m *mockExamSessionRepo) UserHasTags(_ uint, _ []uint) (bool, error)             { return false, nil }
func (m *mockExamSessionRepo) GetUserRombelIDs(_ uint) ([]uint, error)                { return nil, nil }
func (m *mockExamSessionRepo) GetUserTagIDs(_ uint) ([]uint, error)                   { return nil, nil }
func (m *mockExamSessionRepo) FindExpiredOngoing() ([]*entity.ExamSession, error)     { return nil, nil }
func (m *mockExamSessionRepo) ListFinishedBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockExamSessionRepo) ListOngoingBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockExamSessionRepo) ListNotStartedBySchedule(_ uint) ([]*entity.ExamSession, error) {
	return nil, nil
}
func (m *mockExamSessionRepo) Delete(_ uint) error                              { return nil }
func (m *mockExamSessionRepo) CountByScheduleAndStatus(_ uint, _ string) (int64, error) { return 0, nil }
func (m *mockExamSessionRepo) CreateRegradeLog(_ *entity.RegradeLog) error      { return nil }
func (m *mockExamSessionRepo) ListRegradeLogs(_ uint) ([]entity.RegradeLog, error) { return nil, nil }
func (m *mockExamSessionRepo) GetUserRombelNames(_ []uint) (map[uint][]string, error) {
	return nil, nil
}

var _ repository.ExamSessionRepository = (*mockExamSessionRepo)(nil)

// ─── Test Helpers ──────────────────────────────────────────────

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{
			AccessSecret:  "test-access-secret-that-is-long-enough-for-jwt-signing-purposes-64chars!!",
			RefreshSecret: "test-refresh-secret-that-is-long-enough-for-jwt-signing-purposes-64chars!",
			AccessTTL:     15 * time.Minute,
			RefreshTTL:    7 * 24 * time.Hour,
		},
	}
}

func hashedPassword(plain string) string {
	h, _ := bcrypt.HashPassword(plain)
	return h
}

func makeActiveUser(id uint, username, password, role string) *entity.User {
	return &entity.User{
		ID:       id,
		Name:     "Test User",
		Username: username,
		Password: hashedPassword(password),
		Role:     role,
		IsActive: true,
	}
}

// ─── Tests ─────────────────────────────────────────────────────

func TestLogin_Success(t *testing.T) {
	user := makeActiveUser(1, "admin", "password123", entity.RoleAdmin)

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			if login == "admin" {
				return user, nil
			}
			return nil, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	resp, err := uc.Execute(dto.LoginRequest{
		Login:    "admin",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected non-empty refresh token")
	}
	if resp.User.Username != "admin" {
		t.Errorf("expected username 'admin', got '%s'", resp.User.Username)
	}
	if resp.User.Role != entity.RoleAdmin {
		t.Errorf("expected role '%s', got '%s'", entity.RoleAdmin, resp.User.Role)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	user := makeActiveUser(1, "admin", "correct-password", entity.RoleAdmin)

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "admin",
		Password: "wrong-password",
	})
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return nil, nil // user not found
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "nonexistent",
		Password: "any-password",
	})
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got: %v", err)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	user := makeActiveUser(1, "inactive_user", "password123", entity.RolePeserta)
	user.IsActive = false

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "inactive_user",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error for inactive user")
	}
	if !errors.Is(err, ErrUserInactive) {
		t.Errorf("expected ErrUserInactive, got: %v", err)
	}
}

func TestLogin_PesertaWithOngoingExam(t *testing.T) {
	token := "existing-token"
	user := makeActiveUser(1, "peserta1", "password123", entity.RolePeserta)
	user.LoginToken = &token

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{
		findOngoingByUserFn: func(userID uint) (*entity.ExamSession, error) {
			return &entity.ExamSession{ID: 99, UserID: userID, Status: entity.SessionStatusOngoing}, nil
		},
	}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "peserta1",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error for peserta with ongoing exam")
	}
	if !errors.Is(err, ErrExamInProgress) {
		t.Errorf("expected ErrExamInProgress, got: %v", err)
	}
}

func TestLogin_PesertaDuplicateSession_NoExam(t *testing.T) {
	token := "existing-token"
	user := makeActiveUser(1, "peserta1", "password123", entity.RolePeserta)
	user.LoginToken = &token

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{
		findOngoingByUserFn: func(userID uint) (*entity.ExamSession, error) {
			return nil, nil // no ongoing exam
		},
	}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "peserta1",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected ErrSessionExists for duplicate session without ongoing exam")
	}
	if !errors.Is(err, ErrSessionExists) {
		t.Errorf("expected ErrSessionExists, got: %v", err)
	}
}

func TestLogin_PesertaForceLogin(t *testing.T) {
	token := "existing-token"
	user := makeActiveUser(1, "peserta1", "password123", entity.RolePeserta)
	user.LoginToken = &token

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{
		findOngoingByUserFn: func(userID uint) (*entity.ExamSession, error) {
			return nil, nil
		},
	}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	// With ForceLogin=true, should succeed even with existing session token
	resp, err := uc.Execute(dto.LoginRequest{
		Login:      "peserta1",
		Password:   "password123",
		ForceLogin: true,
	})
	if err != nil {
		t.Fatalf("expected no error with force login, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if resp.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
}

func TestLogin_RepoError(t *testing.T) {
	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return nil, errors.New("database connection error")
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	_, err := uc.Execute(dto.LoginRequest{
		Login:    "admin",
		Password: "password123",
	})
	if err == nil {
		t.Fatal("expected error for repository failure")
	}
	if err.Error() != "database connection error" {
		t.Errorf("expected database error, got: %v", err)
	}
}

func TestLogin_AdminBypassesSessionCheck(t *testing.T) {
	// Admin users should not be subject to duplicate session or ongoing exam checks
	token := "existing-token"
	user := makeActiveUser(1, "admin", "password123", entity.RoleAdmin)
	user.LoginToken = &token

	userRepo := &mockUserRepo{
		findByUsernameOrEmailFn: func(login string) (*entity.User, error) {
			return user, nil
		},
	}
	sessionRepo := &mockExamSessionRepo{}

	uc := NewLoginUseCase(userRepo, sessionRepo, testConfig(), nil)

	resp, err := uc.Execute(dto.LoginRequest{
		Login:    "admin",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("admin should bypass session check, got: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
}
