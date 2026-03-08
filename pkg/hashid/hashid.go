package hashid

import (
	"errors"

	goHashids "github.com/speps/go-hashids/v2"
)

var h *goHashids.HashID

func Init(salt string, minLength int) {
	hd := goHashids.NewData()
	hd.Salt = salt
	hd.MinLength = minLength
	var err error
	h, err = goHashids.NewWithData(hd)
	if err != nil {
		panic("hashid init failed: " + err.Error())
	}
}

func Encode(id uint) string {
	encoded, _ := h.Encode([]int{int(id)})
	return encoded
}

func Decode(hash string) (uint, error) {
	ids, err := h.DecodeWithError(hash)
	if err != nil || len(ids) == 0 {
		return 0, errors.New("invalid hash ID")
	}
	return uint(ids[0]), nil
}
