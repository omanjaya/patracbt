package postgres

import (
	"github.com/omanjaya/patra/internal/domain/entity"
	"github.com/omanjaya/patra/internal/domain/repository"
	"github.com/omanjaya/patra/pkg/pagination"
	"gorm.io/gorm"
)

type PermissionRepo struct{ db *gorm.DB }

func NewPermissionRepository(db *gorm.DB) *PermissionRepo {
	return &PermissionRepo{db: db}
}

func (r *PermissionRepo) Create(p *entity.Permission) error {
	return r.db.Create(p).Error
}

func (r *PermissionRepo) FindByID(id uint) (*entity.Permission, error) {
	var p entity.Permission
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).First(&p).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &p, err
}

func (r *PermissionRepo) Update(p *entity.Permission) error {
	return r.db.Save(p).Error
}

func (r *PermissionRepo) Delete(id uint) error {
	return r.db.Model(&entity.Permission{}).Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error
}

func (r *PermissionRepo) List(filter repository.PermissionListFilter, p pagination.Params) ([]*entity.Permission, int64, error) {
	var perms []*entity.Permission
	var total int64

	q := r.db.Model(&entity.Permission{}).Where("deleted_at IS NULL")
	if filter.Search != "" {
		q = q.Where("name ILIKE ? OR group_name ILIKE ? OR description ILIKE ?",
			"%"+filter.Search+"%", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}
	if filter.GroupName != "" {
		q = q.Where("group_name = ?", filter.GroupName)
	}

	q.Count(&total)
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("group_name ASC, name ASC").Find(&perms).Error
	return perms, total, err
}

func (r *PermissionRepo) ListAll() ([]*entity.Permission, error) {
	var perms []*entity.Permission
	err := r.db.Where("deleted_at IS NULL").Order("group_name ASC, name ASC").Find(&perms).Error
	return perms, err
}

func (r *PermissionRepo) ListGroups() ([]string, error) {
	var groups []string
	err := r.db.Model(&entity.Permission{}).
		Where("deleted_at IS NULL").
		Distinct("group_name").
		Order("group_name ASC").
		Pluck("group_name", &groups).Error
	return groups, err
}

func (r *PermissionRepo) AssignToUsers(permissionID uint, userIDs []uint) error {
	perm := entity.Permission{ID: permissionID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&perm).Association("Users").Append(users)
}

func (r *PermissionRepo) RemoveFromUsers(permissionID uint, userIDs []uint) error {
	perm := entity.Permission{ID: permissionID}
	users := make([]entity.User, len(userIDs))
	for i, id := range userIDs {
		users[i] = entity.User{ID: id}
	}
	return r.db.Model(&perm).Association("Users").Delete(users)
}

type userPermRow struct {
	ID       uint
	Name     string
	Username string
	NIS      *string
	Rombel   *string
}

func (r *PermissionRepo) ListUsersWithPermissions(
	filter repository.UserPermissionListFilter,
	p pagination.Params,
) ([]*repository.UserWithPermissions, int64, error) {
	var total int64

	q := r.db.Table("users u").
		Select("u.id, u.name, u.username, up2.nis, r.name as rombel").
		Joins("LEFT JOIN user_profiles up2 ON up2.user_id = u.id").
		Joins("LEFT JOIN user_rombels ur ON ur.user_id = u.id").
		Joins("LEFT JOIN rombels r ON r.id = ur.rombel_id").
		Where("u.deleted_at IS NULL AND u.role = 'peserta'")

	if filter.Search != "" {
		q = q.Where("(u.name ILIKE ? OR u.username ILIKE ? OR up2.nis ILIKE ?)",
			"%"+filter.Search+"%", "%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	if filter.PermissionID != nil {
		q = q.Joins("JOIN user_permissions uperm ON uperm.user_id = u.id AND uperm.permission_id = ?", *filter.PermissionID)
	}

	if filter.NoPermissionID != nil {
		q = q.Where("u.id NOT IN (SELECT user_id FROM user_permissions WHERE permission_id = ?)", *filter.NoPermissionID)
	}

	q.Count(&total)

	var rows []userPermRow
	err := q.Offset(p.Offset()).Limit(p.PerPage).Order("u.name ASC").Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	// Batch-load permissions for all users in a single query — avoids N+1
	userIDs := make([]uint, len(rows))
	for i, row := range rows {
		userIDs[i] = row.ID
	}

	type userPerm struct {
		UserID uint `gorm:"column:user_id"`
		entity.Permission
	}
	var allPerms []userPerm
	if len(userIDs) > 0 {
		r.db.Table("user_permissions up").
			Select("up.user_id, permissions.*").
			Joins("JOIN permissions ON permissions.id = up.permission_id").
			Where("up.user_id IN ? AND permissions.deleted_at IS NULL", userIDs).
			Scan(&allPerms)
	}

	permMap := make(map[uint][]entity.Permission)
	for _, ap := range allPerms {
		permMap[ap.UserID] = append(permMap[ap.UserID], ap.Permission)
	}

	result := make([]*repository.UserWithPermissions, len(rows))
	for i, row := range rows {
		result[i] = &repository.UserWithPermissions{
			ID:          row.ID,
			Name:        row.Name,
			Username:    row.Username,
			NIS:         row.NIS,
			Rombel:      row.Rombel,
			Permissions: permMap[row.ID],
		}
	}

	return result, total, nil
}
