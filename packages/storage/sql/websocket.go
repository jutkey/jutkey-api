package sql

import (
	"context"
	"encoding/json"
	"errors"
	"jutkey-server/conf"
	"time"
)

type channelRouter interface {
	SendWebsocket(string, string) error
}

type WebsocketDataTitle struct {
	Cmd  string `json:"cmd"`
	Info any    `json:"info"`
}

var centrifugoTimeout = time.Second * 5

const (
	ChannelDashboard = "dashboard"

	CmdStatistical = "statistical"
)

func ParseChannel(channel string, cmd string, p channelRouter) error {
	if channel == "" || cmd == "" {
		return errors.New("channel or cmd invalid")
	}
	return p.SendWebsocket(channel, cmd)
}

func SendDataToWebsocket(channel, cmd string, data any) error {
	dat := WebsocketDataTitle{}
	dat.Cmd = cmd
	dat.Info = data

	return sendChannelDashboardData(channel, dat)
}

func sendChannelDashboardData(channel string, data WebsocketDataTitle) error {
	ds, err := json.Marshal(data)
	if err != nil {
		return err
	}
	err = writeChannelByte(channel, ds)
	if err != nil {
		return err
	}
	return nil
}

func writeChannelByte(channel string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), centrifugoTimeout)
	defer cancel()
	return conf.GetCentrifugoConn().Conn().Publish(ctx, channel, data)
}
