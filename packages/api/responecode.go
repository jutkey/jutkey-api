package api

import (
	"fmt"
	"net/http"
)

var (
	defaultStatus           = http.StatusOK
	CodeSystembusy          = CodeType{-1, "System is busy", defaultStatus, ""}
	CodeSuccess             = CodeType{0, "Success", defaultStatus, "OK"}
	CodeIlgmediafiletype    = CodeType{40003, "illegal media file type  ", defaultStatus, ""}
	CodeIlgfiletype         = CodeType{40004, "illegal file type  ", defaultStatus, ""}
	CodeFilesize            = CodeType{40005, "illegal file size  ", defaultStatus, ""}
	CodeImagesize           = CodeType{40006, "illegal image file size  ", defaultStatus, ""}
	CodeVoicesize           = CodeType{40007, "illegal voice file size  ", defaultStatus, ""}
	CodeVideosize           = CodeType{40008, "illegal video file size  ", defaultStatus, ""}
	CodeRequestformat       = CodeType{40009, "illegal request format  ", defaultStatus, ""}
	CodeThumbnailfilesize   = CodeType{400010, "illegal thumbnail file size  ", defaultStatus, ""}
	CodeUrllength           = CodeType{400011, "illegal URL length  ", defaultStatus, ""}
	CodeMultimediafileempty = CodeType{400012, "The multimedia file is empty  ", defaultStatus, ""}
	CodePostpacketempty     = CodeType{400013, "POST packet is empty ", defaultStatus, ""}
	CodeContentempty        = CodeType{400014, "The content of the graphic message is empty. ", defaultStatus, ""}
	CodeTextcmpty           = CodeType{400015, "text message content is empty ", defaultStatus, ""}
	CodeMultimediasizelimit = CodeType{400016, "multimedia file size exceeds limit ", defaultStatus, ""}
	CodeParamNotNull        = CodeType{400017, "Param  message content exceeds limit ", defaultStatus, ""}
	CodeParamOutRange       = CodeType{400018, "Param out of range ", defaultStatus, ""}
	CodeParam               = CodeType{400019, "Param error ", defaultStatus, ""}
	CodeParamNotExists      = CodeType{400020, "Param is exists  ", defaultStatus, ""}
	CodeParamType           = CodeType{400021, "Param type error ", defaultStatus, ""}
	CodeParamKeyConflict    = CodeType{400022, "Param Keyword conflict error ", defaultStatus, ""}
	CodeRecordExists        = CodeType{400023, "Record already exists  ", defaultStatus, ""}
	CodeRecordNotExists     = CodeType{400024, "Record not exists error  ", defaultStatus, ""}
	CodeNewRecordNotRelease = CodeType{400025, "New Record not Release error ", defaultStatus, ""}
	CodeReleaseRule         = CodeType{400026, "Release rule error  ", defaultStatus, ""}
	CodeDeleteRule          = CodeType{400027, "Delete Record  delete rule error  ", defaultStatus, ""}
	CodeHelpDirNotExists    = CodeType{400028, "Help parentdir  not exists error  ", defaultStatus, ""}

	CodeDBfinderr     = CodeType{400029, "DB find error   ", defaultStatus, ""}
	CodeDBcreateerr   = CodeType{400030, "DB create error  ", defaultStatus, ""}
	CodeDBupdateerr   = CodeType{400031, "DB update error  ", defaultStatus, ""}
	CodeDBdeleteerr   = CodeType{400032, "DB delete error  ", defaultStatus, ""}
	CodeDBopertionerr = CodeType{400033, "DB opertion error  ", defaultStatus, ""}
	CodeJsonformaterr = CodeType{400034, "Json format error  ", defaultStatus, ""}
	CodeBodyformaterr = CodeType{400035, "Body format error  ", defaultStatus, ""}

	CodeFileNotExists         = CodeType{400036, "File not exists", defaultStatus, ""}
	CodeFileExists            = CodeType{400037, "File already exists", defaultStatus, ""}
	CodeFileFormatNotSupports = CodeType{400038, "File format is not supported", defaultStatus, ""}
	CodeFileCreated           = CodeType{400039, "Create File is not supported ", defaultStatus, ""}
	CodeFileOpen              = CodeType{400039, "Open File is not supported", defaultStatus, ""}
	CodeCheckParam            = CodeType{400040, "Param error: ", defaultStatus, ""}
	CodeGenerateMine          = CodeType{400041, "new miner generate faile ", defaultStatus, ""}
	CodeImportMine            = CodeType{400042, "import miner faile   ", defaultStatus, ""}
	CodeBooltype              = CodeType{400043, "bool type error  ", defaultStatus, ""}

	CodeUpdateRule               = CodeType{400044, "rule error  ", defaultStatus, ""}
	CodePermissionDenied         = CodeType{400045, "Permission denied  ", defaultStatus, ""}
	CodeNotMineDevidBindActiveid = CodeType{400046, "not mine devid boind Activeid  ", defaultStatus, ""}
	CodeSignError                = CodeType{400047, "sign err ", defaultStatus, ""}
)

type CodeType struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"status"`
	Msg     string `json:"msg"`
}

type errType struct {
	Err     string `json:"error"`
	Message string `json:"msg"`
	Status  int    `json:"-"`
}

func (et errType) Error() string {
	return et.Err
}

func (et errType) Errorf(v ...any) errType {
	et.Message = fmt.Sprintf(et.Message, v...)
	return et
}

func (ct CodeType) Errorf(err error) CodeType {
	et, ok := err.(errType)
	if !ok {
		et.Message = err.Error()
	}
	ct.Message = fmt.Sprintln(ct.Message, et.Message)
	ct.Msg = http.StatusText(ct.Status)
	return ct
}

func (ct CodeType) String(dat string) CodeType {
	ct.Message += " " + dat
	ct.Msg = http.StatusText(ct.Status)
	return ct
}

func (ct CodeType) Success() CodeType {
	ct.Msg = http.StatusText(ct.Status)
	return ct
}
