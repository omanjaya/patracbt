package types

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// JSON is a JSONB-compatible type for PostgreSQL via GORM.
type JSON json.RawMessage

func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

func (j *JSON) Scan(src interface{}) error {
	switch v := src.(type) {
	case nil:
		*j = nil
	case string:
		*j = JSON(v)
	case []byte:
		*j = JSON(v)
	default:
		return fmt.Errorf("JSON.Scan: unsupported type %T", src)
	}
	return nil
}

func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

func (j *JSON) UnmarshalJSON(data []byte) error {
	*j = JSON(data)
	return nil
}
