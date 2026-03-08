package dto

// ===== ROMBEL =====

type CreateRombelRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	GradeLevel  *string `json:"grade_level" binding:"omitempty,max=50"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

type UpdateRombelRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=255"`
	GradeLevel  *string `json:"grade_level" binding:"omitempty,max=50"`
	Description *string `json:"description" binding:"omitempty,max=500"`
}

type AssignUsersRequest struct {
	UserIDs []uint `json:"user_ids" binding:"required"`
}

// ===== SUBJECT =====

type CreateSubjectRequest struct {
	Name string  `json:"name" binding:"required,min=1,max=255"`
	Code *string `json:"code" binding:"omitempty,max=50"`
}

type UpdateSubjectRequest struct {
	Name string  `json:"name" binding:"required,min=1,max=255"`
	Code *string `json:"code" binding:"omitempty,max=50"`
}

// ===== TAG =====

type CreateTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=100"`
	Color string `json:"color" binding:"omitempty,max=20"`
}

type UpdateTagRequest struct {
	Name  string `json:"name" binding:"required,min=1,max=100"`
	Color string `json:"color" binding:"omitempty,max=20"`
}

// ===== ROOM =====

type CreateRoomRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Capacity int    `json:"capacity" binding:"omitempty,min=0,max=1000"`
}

type UpdateRoomRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Capacity int    `json:"capacity" binding:"omitempty,min=0,max=1000"`
}

// ===== SETTING =====

type UpdateSettingsRequest struct {
	Settings map[string]string `json:"settings" binding:"required"`
}
