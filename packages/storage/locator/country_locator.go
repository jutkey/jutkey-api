package locator

import (
	"encoding/json"
	"jutkey-server/conf"
	"os"
	"path/filepath"
)

type CountriesGeoJson struct {
	Type     string           `json:"type"`
	Features []CountryFeature `json:"features"`
}

type CountryFeature struct {
	Type       string      `json:"type"`
	Properties CountryInfo `json:"properties"`
	Geometry   Geometry    `json:"geometry"`
}

type CountryInfo struct {
	ADMIN     string `json:"ADMIN"`
	ISOA2     string `json:"ISO_A2"`
	Continent string `json:"continent"`
}

type Geometry struct {
	Type        string          `json:"type"`
	Coordinates [][][][]float64 `json:"coordinates"`
}

var (
	CountriesGeo CountriesGeoJson
)

func InitCountryLocator() error {
	dbFile := filepath.Join(conf.GetEnvConf().ConfigPath, "locator_db", "countries_geo.json")
	file, err := os.ReadFile(dbFile)
	if err != nil {
		return err
	}
	//fmt.Println(CountriesGeo.Type)
	err = json.Unmarshal(file, &CountriesGeo)
	if err != nil {
		return err
	}
	return nil
}

func isPointInCountry(feature CountryFeature, point Point) bool {
	flag := false
ok:
	for _, coordinates1 := range feature.Geometry.Coordinates {
		for _, coordinates2 := range coordinates1 {
			countryPolygon := &Polygon{}
			for _, point := range coordinates2 {
				if len(point) >= 2 {
					countryPolygon.Add(&Point{Lng: point[0], Lat: point[1]})
				}
			}
			if countryPolygon.Contains(&point) {
				//fmt.Println(feature.Properties)
				flag = true
				break ok
			}
		}
	}
	return flag
}

func FindCountryByCoordinate(lat, lng float64) CountryInfo {
	point := Point{Lat: lat, Lng: lng}
	var country CountryInfo
	for _, feature := range CountriesGeo.Features {
		//fmt.Println(index)
		//fmt.Println(feature)

		if isPointInCountry(feature, point) {
			country.ADMIN = feature.Properties.ADMIN
			country.ISOA2 = feature.Properties.ISOA2
			country.Continent = feature.Properties.Continent
			break
		}
	}
	if country.ADMIN == "" {
		country.ADMIN = "Global"
		country.ISOA2 = "GL"
		country.Continent = "Global"
	}

	return country
}
