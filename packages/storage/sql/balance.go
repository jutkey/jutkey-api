package sql

import (
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
)

type MyAssignBalanceResult struct {
	Amount  string `json:"amount"`
	Balance string `json:"balance"`
	Show    bool   `json:"show"`
}

type KeyEcosystemInfo struct {
	Ecosystem     string       `json:"ecosystem"`
	Name          string       `json:"name"`
	Roles         []RoleInfo   `json:"roles,omitempty"`
	Notifications []NotifyInfo `json:"notifications,omitempty"`
}

type KeyInfoResult struct {
	Account    string              `json:"account"`
	Ecosystems []*KeyEcosystemInfo `json:"ecosystems"`
}

type RoleInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type NotifyInfo struct {
	RoleID string `json:"role_id"`
	Count  int64  `json:"count"`
}

func GetNotifications(ecosystemID int64, key *sqldb.Key) ([]NotifyInfo, error) {
	nfys, err := sqldb.GetNotificationsCount(ecosystemID, []string{key.AccountID})
	if err != nil {
		return nil, err
	}

	list := make([]NotifyInfo, 0)
	for _, n := range nfys {
		if n.RecipientID != key.ID {
			continue
		}

		list = append(list, NotifyInfo{
			RoleID: converter.Int64ToStr(n.RoleID),
			Count:  n.Count,
		})
	}
	return list, nil
}
