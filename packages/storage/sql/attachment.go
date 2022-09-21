package sql

import (
	"os"
)

var UploadDir = "./upload/"

func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func Savefile(file string, buf []byte) error {
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, 066)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err1 := f.Write(buf)
	return err1
}

func GetFileNameByHash(hash string) (string, int64, error) {
	var d, b Binary
	f, err := d.GetByHash(hash)
	if err != nil {
		return "", 0, err
	}
	if f {
		if d.MimeType != "application/octet-stream" {
			file := d.GetByJpeg()
			return file, d.ID, nil
		} else {
			f, err := b.GetByPng(&d)
			if err != nil {
				return "", 0, err
			}
			if f {
				if b.MimeType != "application/octet-stream" {
					file := b.GetByJpeg()
					return file, d.ID, nil
				}
			}
		}
	}
	return "", 0, nil
}

func GetFileHash(id int64) (string, error) {
	if id == 0 {
		return "", nil
	}
	var d Binary
	f, err := d.GetByIdHash(id)
	if err != nil {
		return "", err
	}
	if !f {
		return "", nil
	}
	return d.Hash, nil
}

func LoadFile(id int64) (string, error) {
	var d, b Binary
	f, err := d.GetByID(id)
	if err != nil {
		return "", err
	}
	if f {
		if d.MimeType != "application/octet-stream" {
			file := d.GetByJpeg()
			if !IsExist(UploadDir + file) {
				//fmt.Printf("save file:%s\n", UploadDir+file)
				err := Savefile(UploadDir+file, d.Data)
				if err != nil {
					return "", err
				}
				return file, nil
			}
			return file, nil
		} else {
			f, err := b.GetByPng(&d)
			if err != nil {
				return "", err
			}
			if f {
				if b.MimeType != "application/octet-stream" {
					file := b.GetByJpeg()
					if !IsExist(UploadDir + file) {
						//fmt.Printf("save file:%s\n", UploadDir+file)
						err := Savefile(UploadDir+file, b.Data)
						if err != nil {
							return "", err
						}
						return file, nil
					}
					return file, nil
				}
			}
		}
	}
	return "", nil
}
