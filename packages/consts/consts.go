package consts

import (
	"fmt"
	"strings"
)

const (

	// VERSION is current version
	VERSION = "1.0.0"

	// ApiPath is the beginning of the api url
	ApiPath = `/api/v1/`

	LogoRoad = ApiPath + "logo/"
)

const ()

var (
	buildBranch = ""
	buildDate   = ""
	commitHash  = ""
)

var BuildInfo string

func Version() string {
	status := `scan server status is running`
	fmt.Printf("BuildInfo:%s\n", BuildInfo)
	return strings.TrimSpace(strings.Join([]string{status, VERSION, BuildInfo}, " "))
}

//InitBuildInfo
//go build -ldflags "-X 'jutkey-server/cmd.buildBranch=main' -X 'jutkey-server/cmd.buildDate=2022-06-17' -X 'jutkey-server/cmd.commitHash=2141saf'"
func InitBuildInfo() {
	BuildInfo = func() string {
		if buildBranch == "" {
			return fmt.Sprintf("branch.%s commit.%s time.%s", "unknown", "unknown", "unknown")
		}
		return fmt.Sprintf("branch.%s commit.%s time.%s", buildBranch, commitHash, buildDate)
	}()
}
