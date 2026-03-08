package main

import (
	"fmt"

	"github.com/omanjaya/patra/internal/domain/entity"
	pkgbcrypt "github.com/omanjaya/patra/pkg/bcrypt"
	"github.com/omanjaya/patra/pkg/logger"
	"gorm.io/gorm"
)

// seedMasterData seeds all initial master data matching ExamPatra's seeders.
// Safe to call multiple times — uses firstOrCreate pattern.
func seedMasterData(db *gorm.DB) {
	seedRoles(db)
	seedRombels(db)
	seedSubjects(db)
	seedRooms(db)
	seedTags(db)
	seedDummyUsers(db)
}

// seedRoles creates the default roles (Super Admin, Operator, Guru, Peserta, Pengawas).
func seedRoles(db *gorm.DB) {
	roles := []entity.Role{
		{Name: "Super Admin", GuardName: "web"},
		{Name: "Operator", GuardName: "web"},
		{Name: "Guru", GuardName: "web"},
		{Name: "Peserta", GuardName: "web"},
		{Name: "Pengawas", GuardName: "web"},
	}

	var created int
	for _, r := range roles {
		var count int64
		db.Model(&entity.Role{}).Where("name = ?", r.Name).Count(&count)
		if count == 0 {
			if err := db.Create(&r).Error; err != nil {
				logger.Log.Errorf("Gagal seed role %s: %v", r.Name, err)
			} else {
				created++
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new roles", created)
	}
}

// seedRombels creates initial rombel (class groups) X-1..XII-2.
func seedRombels(db *gorm.DB) {
	type rombelDef struct {
		Name       string
		GradeLevel string
	}

	rombels := []rombelDef{
		{"X-1", "10"},
		{"X-2", "10"},
		{"XI-1", "11"},
		{"XI-2", "11"},
		{"XII-1", "12"},
		{"XII-2", "12"},
	}

	var created int
	for _, r := range rombels {
		var count int64
		db.Model(&entity.Rombel{}).Where("name = ?", r.Name).Count(&count)
		if count == 0 {
			gl := r.GradeLevel
			rombel := entity.Rombel{
				Name:       r.Name,
				GradeLevel: &gl,
			}
			if err := db.Create(&rombel).Error; err != nil {
				logger.Log.Errorf("Gagal seed rombel %s: %v", r.Name, err)
			} else {
				created++
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new rombels", created)
	}
}

// seedSubjects creates initial subjects (Bahasa Indonesia, Inggris, MTK, IPA, IPS).
func seedSubjects(db *gorm.DB) {
	type subjectDef struct {
		Name string
		Code string
	}

	subjects := []subjectDef{
		{"Bahasa Indonesia", "B-INDO"},
		{"Bahasa Inggris", "B-INGG"},
		{"Matematika", "MTK"},
		{"Ilmu Pengetahuan Alam", "IPA"},
		{"Ilmu Pengetahuan Sosial", "IPS"},
	}

	var created int
	for _, s := range subjects {
		var count int64
		db.Model(&entity.Subject{}).Where("code = ?", s.Code).Count(&count)
		if count == 0 {
			code := s.Code
			subj := entity.Subject{
				Name: s.Name,
				Code: &code,
			}
			if err := db.Create(&subj).Error; err != nil {
				logger.Log.Errorf("Gagal seed subject %s: %v", s.Name, err)
			} else {
				created++
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new subjects", created)
	}
}

// seedRooms creates initial rooms (6 rooms for X, XI, XII × 2).
func seedRooms(db *gorm.DB) {
	grades := []string{"X", "XI", "XII"}

	var created int
	for _, grade := range grades {
		for i := 1; i <= 2; i++ {
			name := fmt.Sprintf("Ruang Kelas %s-%d", grade, i)
			var count int64
			db.Model(&entity.Room{}).Where("name = ?", name).Count(&count)
			if count == 0 {
				room := entity.Room{
					Name:     name,
					Capacity: 40,
				}
				if err := db.Create(&room).Error; err != nil {
					logger.Log.Errorf("Gagal seed room %s: %v", name, err)
				} else {
					created++
				}
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new rooms", created)
	}
}

// seedTags creates tags for religion×grade and administration.
func seedTags(db *gorm.DB) {
	type tagDef struct {
		Name  string
		Color string
	}

	// Religion colors
	religionColors := map[string]string{
		"hindu":   "#F59E0B",
		"islam":   "#10B981",
		"kristen": "#3B82F6",
		"katolik": "#8B5CF6",
		"budha":   "#EC4899",
	}

	var tags []tagDef

	// Religion × Grade tags
	religions := []string{"hindu", "islam", "kristen", "katolik", "budha"}
	grades := []string{"kelas_X", "kelas_XI", "kelas_XII"}
	for _, religion := range religions {
		for _, grade := range grades {
			tags = append(tags, tagDef{
				Name:  fmt.Sprintf("%s_%s", religion, grade),
				Color: religionColors[religion],
			})
		}
	}

	// Administration tags
	tags = append(tags,
		tagDef{Name: "spp", Color: "#6366F1"},
		tagDef{Name: "raport", Color: "#14B8A6"},
		tagDef{Name: "ijazah", Color: "#F97316"},
	)

	var created int
	for _, t := range tags {
		var count int64
		db.Model(&entity.Tag{}).Where("name = ?", t.Name).Count(&count)
		if count == 0 {
			tag := entity.Tag{
				Name:  t.Name,
				Color: t.Color,
			}
			if err := db.Create(&tag).Error; err != nil {
				logger.Log.Errorf("Gagal seed tag %s: %v", t.Name, err)
			} else {
				created++
			}
		}
	}
	if created > 0 {
		logger.Log.Infof("Seeded %d new tags", created)
	}
}

// seedDummyUsers creates test users (operator, guru, peserta, pengawas) for development.
func seedDummyUsers(db *gorm.DB) {
	// Check if dummy users already exist
	var count int64
	db.Model(&entity.User{}).Where("username = ?", "operator1").Count(&count)
	if count > 0 {
		return // Already seeded
	}

	hashedPw, _ := pkgbcrypt.HashPassword("password")

	// Lookup master data
	var rombelX1, rombelX2 entity.Rombel
	db.Where("name = ?", "X-1").First(&rombelX1)
	db.Where("name = ?", "X-2").First(&rombelX2)

	var roomX1, roomX2 entity.Room
	db.Where("name = ?", "Ruang Kelas X-1").First(&roomX1)
	db.Where("name = ?", "Ruang Kelas X-2").First(&roomX2)

	var tagHinduX, tagIslamX entity.Tag
	db.Where("name = ?", "hindu_kelas_X").First(&tagHinduX)
	db.Where("name = ?", "islam_kelas_X").First(&tagIslamX)

	// === Operator ===
	op1 := entity.User{
		Name:     "Operator Satu",
		Username: "operator1",
		Password: hashedPw,
		Role:     "admin",
		IsActive: true,
	}
	if err := db.Create(&op1).Error; err != nil {
		logger.Log.Errorf("Gagal seed operator1: %v", err)
	} else {
		nip := "OP-123456"
		db.Create(&entity.UserProfile{UserID: op1.ID, NIP: &nip})
	}

	// === Guru 1 ===
	g1 := entity.User{
		Name:     "Guru Pertama",
		Username: "guru1",
		Password: hashedPw,
		Role:     "guru",
		IsActive: true,
	}
	if err := db.Create(&g1).Error; err != nil {
		logger.Log.Errorf("Gagal seed guru1: %v", err)
	} else {
		nip := "G-111111"
		db.Create(&entity.UserProfile{UserID: g1.ID, NIP: &nip})
		if rombelX1.ID > 0 {
			db.Create(&entity.UserRombel{UserID: g1.ID, RombelID: rombelX1.ID})
		}
	}

	// === Guru 2 ===
	g2 := entity.User{
		Name:     "Guru Kedua",
		Username: "guru2",
		Password: hashedPw,
		Role:     "guru",
		IsActive: true,
	}
	if err := db.Create(&g2).Error; err != nil {
		logger.Log.Errorf("Gagal seed guru2: %v", err)
	} else {
		nip := "G-222222"
		db.Create(&entity.UserProfile{UserID: g2.ID, NIP: &nip})
		if rombelX2.ID > 0 {
			db.Create(&entity.UserRombel{UserID: g2.ID, RombelID: rombelX2.ID})
		}
	}

	// === Pengawas ===
	pw1 := entity.User{
		Name:     "Pengawas Satu",
		Username: "pengawas1",
		Password: hashedPw,
		Role:     "pengawas",
		IsActive: true,
	}
	if err := db.Create(&pw1).Error; err != nil {
		logger.Log.Errorf("Gagal seed pengawas1: %v", err)
	} else {
		nip := "PW-111111"
		db.Create(&entity.UserProfile{UserID: pw1.ID, NIP: &nip})
	}

	// === Peserta 1 ===
	s1 := entity.User{
		Name:     "Siswa Satu",
		Username: "siswa1",
		Password: hashedPw,
		Role:     "peserta",
		IsActive: true,
	}
	if err := db.Create(&s1).Error; err != nil {
		logger.Log.Errorf("Gagal seed siswa1: %v", err)
	} else {
		nis := "NIS-1000"
		profile := entity.UserProfile{UserID: s1.ID, NIS: &nis}
		if rombelX1.ID > 0 {
			profile.RombelID = &rombelX1.ID
		}
		if roomX1.ID > 0 {
			profile.RoomID = &roomX1.ID
		}
		db.Create(&profile)
		if tagHinduX.ID > 0 {
			db.Create(&entity.UserTag{UserID: s1.ID, TagID: tagHinduX.ID})
		}
	}

	// === Peserta 2 ===
	s2 := entity.User{
		Name:     "Siswa Dua",
		Username: "siswa2",
		Password: hashedPw,
		Role:     "peserta",
		IsActive: true,
	}
	if err := db.Create(&s2).Error; err != nil {
		logger.Log.Errorf("Gagal seed siswa2: %v", err)
	} else {
		nis := "NIS-1001"
		profile := entity.UserProfile{UserID: s2.ID, NIS: &nis}
		if rombelX2.ID > 0 {
			profile.RombelID = &rombelX2.ID
		}
		if roomX2.ID > 0 {
			profile.RoomID = &roomX2.ID
		}
		db.Create(&profile)
		if tagIslamX.ID > 0 {
			db.Create(&entity.UserTag{UserID: s2.ID, TagID: tagIslamX.ID})
		}
	}

	logger.Log.Info("Seeded dummy users: operator1, guru1, guru2, pengawas1, siswa1, siswa2 (password: 'password')")
}

// seedRolePermissions assigns permissions to roles matching ExamPatra's RolesAndPermissionsSeeder.
func seedRolePermissions(db *gorm.DB) {
	// Check if already done (Super Admin role has permissions)
	var superAdmin entity.Role
	if err := db.Where("name = ?", "Super Admin").First(&superAdmin).Error; err != nil {
		return // No roles seeded yet
	}
	var existingCount int64
	db.Table("role_permissions").Where("role_id = ?", superAdmin.ID).Count(&existingCount)
	if existingCount > 0 {
		return // Already assigned
	}

	// Load all permissions
	var allPerms []entity.Permission
	db.Find(&allPerms)
	if len(allPerms) == 0 {
		return
	}

	// Build permission map by name
	permMap := make(map[string]entity.Permission)
	for _, p := range allPerms {
		permMap[p.Name] = p
	}

	// Helper to resolve permission names to entities
	resolvePerms := func(names []string) []entity.Permission {
		var result []entity.Permission
		for _, name := range names {
			if p, ok := permMap[name]; ok {
				result = append(result, p)
			}
		}
		return result
	}

	// Super Admin gets all permissions
	if err := db.Model(&superAdmin).Association("Permissions").Replace(allPerms); err != nil {
		logger.Log.Errorf("Gagal assign permissions ke Super Admin: %v", err)
	}

	// Operator permissions
	var operator entity.Role
	if db.Where("name = ?", "Operator").First(&operator).Error == nil {
		operatorPerms := resolvePerms([]string{
			"user-list", "user-create", "user-edit", "user-delete",
			"user-view-trash", "user-restore",
			"tag-list", "tag-create", "tag-edit", "tag-delete",
			"rombel-list", "rombel-create", "rombel-edit", "rombel-delete",
			"subject-list", "subject-create", "subject-edit", "subject-delete",
			"room-list", "room-create", "room-edit", "room-delete",
			"question-bank-list", "question-bank-create", "question-bank-edit", "question-bank-delete",
			"exam-schedule-list", "exam-schedule-create", "exam-schedule-edit", "exam-schedule-delete",
			"supervision-view",
			"report-view", "report-export",
		})
		if err := db.Model(&operator).Association("Permissions").Replace(operatorPerms); err != nil {
			logger.Log.Errorf("Gagal assign permissions ke Operator: %v", err)
		}
	}

	// Guru permissions
	var guru entity.Role
	if db.Where("name = ?", "Guru").First(&guru).Error == nil {
		guruPerms := resolvePerms([]string{
			"question-bank-list", "question-bank-create", "question-bank-edit", "question-bank-delete",
			"exam-schedule-list", "exam-schedule-create", "exam-schedule-edit", "exam-schedule-delete",
			"supervision-view",
			"report-view",
		})
		if err := db.Model(&guru).Association("Permissions").Replace(guruPerms); err != nil {
			logger.Log.Errorf("Gagal assign permissions ke Guru: %v", err)
		}
	}

	// Pengawas permissions
	var pengawas entity.Role
	if db.Where("name = ?", "Pengawas").First(&pengawas).Error == nil {
		pengawasPerms := resolvePerms([]string{
			"supervision-view", "supervision-action",
		})
		if err := db.Model(&pengawas).Association("Permissions").Replace(pengawasPerms); err != nil {
			logger.Log.Errorf("Gagal assign permissions ke Pengawas: %v", err)
		}
	}

	logger.Log.Info("Seeded role-permission assignments")
}
