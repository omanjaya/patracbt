package setting

import (
	"github.com/omanjaya/patra/internal/application/dto"
	"github.com/omanjaya/patra/internal/domain/repository"
)

type SettingUseCase struct {
	repo repository.SettingRepository
}

func NewSettingUseCase(repo repository.SettingRepository) *SettingUseCase {
	return &SettingUseCase{repo: repo}
}

func (uc *SettingUseCase) GetAll() (map[string]string, error) {
	settings, err := uc.repo.GetAll()
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, s := range settings {
		if s.Value != nil {
			result[s.Key] = *s.Value
		} else {
			result[s.Key] = ""
		}
	}
	return result, nil
}

func (uc *SettingUseCase) Update(req dto.UpdateSettingsRequest) error {
	return uc.repo.SetMultiple(req.Settings)
}
