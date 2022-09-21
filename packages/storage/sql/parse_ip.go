package sql

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"jutkey-server/packages/storage/locator"
	"net"
	"strings"
)

type LocatorInfo struct {
	Continent string `json:"continent"`
	Nation    string `json:"nation"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

// ClientIP tries its best to implement the algorithm for obtaining ClientIP.
// Resolve the x-real-IP to X-Forwarded-For so that the reverse proxy (Nginx or HaProxy) works properly.
func ClientIP(r *gin.Context) string {
	xForwardedFor := r.Request.Header.Get("X-Forwarded-For")
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if ip != "" {
		return ip
	}

	ip = strings.TrimSpace(r.Request.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}

	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.Request.RemoteAddr)); err == nil {
		return ip
	}

	return ""
}

func GetLocator(ip string) (*LocatorInfo, error) {
	var (
		rets LocatorInfo
	)
	gotInfo, gotFindResult := findAddressFromIp(ip)
	if gotFindResult == 1 {
		return nil, errors.New("error obtaining the IP address")
	}
	if !PointValid(gotInfo.Latitude, gotInfo.Longitude) {
		return nil, errors.New("parse ip locator invalid")
	}
	lat := ReservedDecimal(gotInfo.Latitude, 4)
	lng := ReservedDecimal(gotInfo.Longitude, 4)

	info := locator.FindCountryByCoordinate(gotInfo.Latitude, gotInfo.Longitude)
	if info.Continent == "" || info.ADMIN == "" {
		return nil, errors.New("get locator invalid")
	}
	rets.Continent = info.Continent
	rets.Nation = info.ADMIN
	rets.Latitude = lat
	rets.Longitude = lng

	return &rets, nil
}

func ReservedDecimal(val float64, place int) string {
	format := "%." + fmt.Sprintf("%d", place) + "f"
	return fmt.Sprintf(format, val)
}

func PointValid(latitude, longitude float64) bool {
	lat := decimal.NewFromFloat(latitude)
	lng := decimal.NewFromFloat(longitude)
	minLat := decimal.NewFromFloat(-90)
	maxLat := decimal.NewFromFloat(90)
	minLng := decimal.NewFromFloat(-180)
	maxLng := decimal.NewFromFloat(180)
	if lat.LessThanOrEqual(minLat) || lat.GreaterThanOrEqual(maxLat) {
		return false
	}
	if lng.LessThanOrEqual(minLng) || lng.GreaterThanOrEqual(maxLng) {
		return false
	}

	return true
}
