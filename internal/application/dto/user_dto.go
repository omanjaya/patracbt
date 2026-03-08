package dto

type CreateUserRequest struct {
	Name       string  `json:"name" binding:"required,min=1,max=255"`
	Username   string  `json:"username" binding:"required,min=3,max=100"`
	Email      *string `json:"email" binding:"omitempty,email"`
	Password   string  `json:"password" binding:"required,min=8"`
	Role       string  `json:"role" binding:"required,oneof=admin guru pengawas peserta"`
	RombelIDs  []uint  `json:"rombel_ids"`
	Profile    *CreateUserProfileRequest `json:"profile"`
}

type CreateUserProfileRequest struct {
	NIS   *string `json:"nis" binding:"omitempty,max=50"`
	NIP   *string `json:"nip" binding:"omitempty,max=50"`
	Class *string `json:"class" binding:"omitempty,max=50"`
	Major *string `json:"major" binding:"omitempty,max=100"`
	Year  *int16  `json:"year" binding:"omitempty,min=2000,max=2100"`
	Phone *string `json:"phone" binding:"omitempty,max=20"`
}

type UpdateUserRequest struct {
	Name      string  `json:"name" binding:"required,min=1,max=255"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Password  *string `json:"password" binding:"omitempty,min=8"`
	Role      string  `json:"role" binding:"required,oneof=admin guru pengawas peserta"`
	RombelIDs []uint  `json:"rombel_ids"`
	Profile   *CreateUserProfileRequest `json:"profile"`
}

type ImportUserRow struct {
	Name       string
	Username   string
	Password   string
	Role       string
	NIS        string
	Class      string
	RombelName string
}
