package sql

import (
	"encoding/json"
	"errors"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"jutkey-server/packages/consts"
	"jutkey-server/packages/storage/geoip"
	"jutkey-server/packages/storage/kv"
	"jutkey-server/packages/storage/locator"
	"net"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type NodeInfo struct {
	ApiAddress string `json:"api_address"`
	PublicKey  string `json:"public_key"`
	TcpAddress string `json:"tcp_address"`
}

type HonorNodeInfo struct {
	ID        int64   `gorm:"primary_key;not null"`
	Value     string  `json:"value,omitempty" gorm:"not null;type:jsonb"`
	Address   string  `json:"address,omitempty" gorm:"not null"`
	Latitude  float64 `json:"latitude,omitempty" gorm:"not null"`
	Longitude float64 `json:"longitude,omitempty" gorm:"not null"`
	Display   bool    `json:"display,omitempty" gorm:"not null"`
}

type NodeValue struct {
	Id            int64  `json:"id"`
	NodeName      string `json:"node_name"`
	ApiAddress    string `json:"api_address"`
	Vote          int64  `json:"vote"`
	ConsensusMode int32  `json:"consensus_mode"`
	IsHonor       bool   `json:"is_honor"`
}

type ipInfo struct {
	CityName  string
	Latitude  float64
	Longitude float64
}

type nodePkg struct {
	NodePosition  int64           `gorm:"column:node_position"`
	Count         int64           `gorm:"column:count"`
	PkgFor        decimal.Decimal `gorm:"column:pkg_for"`
	KeyId         int64           `gorm:"column:key_id"`
	ConsensusMode int32           `gorm:"column:consensus_mode"`
	ReplyRate     decimal.Decimal `gorm:"column:reply_rate"`
}

type HonorNodeModel struct {
	NodeName     string `json:"node_name"`
	TCPAddress   string `json:"tcp_address,omitempty"`
	APIAddress   string `json:"api_address"`
	City         string `json:"city"`
	Icon         string `json:"icon"`
	IconUrl      string `json:"icon_url"`
	NodePosition int64  `json:"node_position"`
	KeyID        string `json:"key_id"`
	Display      bool   `json:"display"`
	//PublicKey       string          `json:"public_key"`
	Latitude        string          `json:"latitude,omitempty"`
	Longitude       string          `json:"longitude,omitempty"`
	NodeBlock       int64           `json:"node_block"`
	PkgAccountedFor decimal.Decimal `json:"pkg_accounted_for"`
	ReplyRate       string          `json:"reply_rate"`
	ConsensusMode   int32           `json:"consensus_mode"`

	NodeStatusTime time.Time `json:"node_status_time,omitempty"`
}

var (
	HonorNodes []HonorNodeModel
)

func (p *HonorNodeInfo) TableName() string {
	return "honor_node_info"
}

func (p *HonorNodeInfo) CreateTable() (err error) {
	err = nil
	if !GetDB(nil).Migrator().HasTable(p) {
		if err = GetDB(nil).Migrator().CreateTable(p); err != nil {
			return err
		}
	}
	return err
}

func GetHonorNode() {
	var (
		err      error
		list2    []NodeValue
		list1    []NodeValue
		p        HonorNodeInfo
		sp       sqldb.PlatformParameter
		nodeInfo string
		nodes    []NodeInfo
		list     []NodeValue
	)
	if err = GetDB(nil).Table(sp.TableName()).Where("name = ?", "honor_nodes").Select("value").Take(&nodeInfo).Error; err != nil {
		if err != gorm.ErrRecordNotFound {
			log.WithFields(log.Fields{"error": err}).Error("get Honor Node Info Find DB Failed")
		}
	}
	if nodeInfo != "" {
		err = json.Unmarshal([]byte(nodeInfo), &nodes)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("get Honor Node Info json marshal failed")
			return
		}
		for key, value := range nodes {
			list1 = append(list1, NodeValue{Id: int64(key), ApiAddress: value.ApiAddress, Vote: 0, ConsensusMode: 1, IsHonor: true})
		}
	} else {
		var app Applications
		f, err := app.GetByName("Basic", 1)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("get First Node App Basic Failed")
		}
		if f {
			info, err := getAppValue(app.ID, "first_node", 1)
			if err == nil {
				var firstNode NodeInfo
				err = json.Unmarshal([]byte(info), &firstNode)
				if err == nil {
					if firstNode.ApiAddress != "" {
						list1 = append(list1, NodeValue{ApiAddress: firstNode.ApiAddress, Vote: 0, ConsensusMode: 1, IsHonor: true})
					}
				} else {
					log.WithFields(log.Fields{"error": err}).Error("get First Node value json unmarshal Failed")
				}
			} else {
				log.WithFields(log.Fields{"error": err}).Error("get First Node value Failed")
			}
		}
	}

	if NodeReady {
		err = GetDB(nil).Raw(`
SELECT cs.id,cs.node_name,cs.api_address,
	CASE WHEN coalesce(ds.earnest,0) > 0 THEN 
		round(coalesce(ds.earnest,0) / 1e12,0)
	ELSE
		0
	END as vote,
	CASE WHEN coalesce(cs.earnest_total,0) >= ? THEN
		TRUE
	ELSE
		FALSE
	END AS is_honor
FROM (
	SELECT id,node_name,api_address,reply_count,earnest_total FROM "1_candidate_node_requests" WHERE deleted = 0
)AS cs
LEFT JOIN (
	SELECT sum(earnest)earnest,request_id FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3 GROUP BY request_id
)AS ds ON (ds.request_id = cs.id)
`, PledgeAmount).Find(&list2).Error
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Get Honor Node List Failed")
			return
		}
	}
	if len(list1) == 0 && len(list2) == 0 {
		syncNodeDisplayStatus(nil)
		p.InsertRedis()
		return
	}

	if len(list1) > 0 {
		if err = GetNodeListInfo(list1, 1); err != nil {
			log.WithFields(log.Fields{"error": err, "list": list1}).Error("Get Node List Info Failed")
			return
		}
		list = append(list, list1...)
	}
	if len(list2) > 0 {
		for key, _ := range list2 {
			list2[key].ConsensusMode = 2
		}
		if err = GetNodeListInfo(list2, 2); err != nil {
			log.WithFields(log.Fields{"error": err, "list": list2}).Error("Get Node List Info Failed")
			return
		}
		list = append(list, list2...)
	}
	syncNodeDisplayStatus(list)

	p.InsertRedis()
}

func honorNodeDbIsExist(nodeId int64, consensusMode int32, list []NodeValue) (bool, int) {
	for i := 0; i < len(list); i++ {
		if list[i].Id == nodeId && list[i].ConsensusMode == consensusMode {
			return true, i
		}
	}
	return false, 0
}

func syncNodeDisplayStatus(nodeValue []NodeValue) {
	var (
		p    HonorNodeInfo
		list []HonorNodeInfo
	)
	if !HasTableOrView(nil, p.TableName()) {
		return
	}
	if err := GetDB(nil).Table(p.TableName()).Select("id,value,display").Order("id desc").Find(&list).Error; err != nil {
		log.WithFields(log.Fields{"error": err}).Error("sync display db failed")
		return
	}

	oldVal := make([]NodeValue, len(list))
	for i := 0; i < len(list); i++ {
		if err := json.Unmarshal([]byte(list[i].Value), &oldVal[i]); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("syn display json failed")
			continue
		}

	}
	if len(nodeValue) > 0 {
		for i := 0; i < len(list); i++ {
			if list[i].Display == false {
				for j := 0; j < len(nodeValue); j++ {
					if nodeValue[j].IsHonor && oldVal[i].Id == nodeValue[j].Id && oldVal[i].ConsensusMode == nodeValue[j].ConsensusMode {
						list[i].Display = true
						if err := GetDB(nil).Table(p.TableName()).Where("id = ?", list[i].ID).Update("display", list[i].Display).Error; err != nil {
							log.WithFields(log.Fields{"error": err}).Error("sync display status update1 err")
							continue
						}
					}
				}
			} else {
				statusIsTrue := false
				for j := 0; j < len(nodeValue); j++ {
					if oldVal[i].Id == nodeValue[j].Id && oldVal[i].ConsensusMode == nodeValue[j].ConsensusMode {
						if !nodeValue[j].IsHonor {
							statusIsTrue = false
							break
						}
						statusIsTrue = true
						break
					}
				}
				if statusIsTrue == false {
					list[i].Display = false
					if err := GetDB(nil).Table(p.TableName()).Where("id = ?", list[i].ID).Update("display", list[i].Display).Error; err != nil {
						log.WithFields(log.Fields{"error": err}).Error("sync display status update1 err")
						continue
					}
				}
			}
		}
	} else {
		if err := GetDB(nil).Table(p.TableName()).Where("display = ?", true).Updates(map[string]interface{}{"display": false}).Error; err != nil {
			log.WithFields(log.Fields{"error": err}).Error("sync all node display false failed")
			return
		}

	}
}

func (p *HonorNodeInfo) GetNodeList() (node []HonorNodeModel, err error) {
	var info []HonorNodeInfo
	f, date := p.GetRedis()
	if !f {
		if date == "" {
			return nil, errors.New("get redis honor-node doesn't not exist")
		}
		return nil, errors.New(date)
	}

	if err = json.Unmarshal([]byte(date), &info); err != nil {
		log.WithFields(log.Fields{"warn": err}).Warn("Get Node List json err")
		return nil, err
	}
	node = make([]HonorNodeModel, len(info))

	var nodePkgList []nodePkg
	if err := GetDB(nil).Raw(`
SELECT case when cast (count(1)*100 as numeric)=0 OR cast((SELECT max(id) FROM block_chain)  as numeric)=0 THEN
	0
ELSE
	round(cast (count(1)*100 as numeric)/ cast( (SELECT max(id) FROM block_chain)  as numeric),2) 
end as pkg_for,node_position AS node_position,count(*),key_id,consensus_mode
FROM block_chain AS bk GROUP BY node_position,key_id,consensus_mode
`).Find(&nodePkgList).Error; err != nil {
		return nil, err
	}

	for i := 0; i < len(info); i++ {
		nodeValue := NodeValue{}
		err := json.Unmarshal([]byte(info[i].Value), &nodeValue)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("GetNodeList Failed")
			continue
		}
		if nodeValue.NodeName != "" {
			node[i].NodeName = nodeValue.NodeName
		} else {
			node[i].NodeName = "HONOR_NODE" + strconv.FormatInt(info[i].ID, 10)
		}
		//node[i].TCPAddress = nodeValue.TcpAddress
		node[i].APIAddress = nodeValue.ApiAddress
		node[i].City = getCity(info[i].Address)
		node[i].Icon = getCountry(info[i].Address)
		node[i].NodePosition = nodeValue.Id
		//node[i].NodeStatusTime = time.Now()
		//node[i].KeyID = converter.AddressToString(crypto2.Address([]byte(nodeValue.PublicKey)))
		//node[i].PublicKey = nodeValue.PublicKey
		node[i].Latitude = strconv.FormatFloat(info[i].Latitude, 'f', 5, 64)
		node[i].Longitude = strconv.FormatFloat(info[i].Longitude, 'f', 5, 64)
		node[i].IconUrl = getIconNationalFlag(node[i].Icon)
		node[i].Display = info[i].Display

		v1, v2, v3 := getNodePkgInfo(node[i].NodePosition, nodeValue.ConsensusMode, nodePkgList)
		node[i].PkgAccountedFor, node[i].NodeBlock, node[i].KeyID = v1, v2, v3
		node[i].ConsensusMode = nodeValue.ConsensusMode
	}
	return node, nil
}

func (p *HonorNodeInfo) GetRedis() (bool, string) {
	rd := kv.RedisParams{
		Key:   "honor-node",
		Value: "",
	}
	err := rd.Get()
	if err != nil {
		if err.Error() == "redis: nil" || err.Error() == "EOF" {
			return false, ""
		}
		return false, err.Error()
	}
	return true, rd.Value
}

func (p *HonorNodeInfo) InsertRedis() {
	var node []HonorNodeInfo
	var value []byte
	f, err := isFound(GetDB(nil).Table(p.TableName()).Order("id asc").Find(&node))
	if f {
		value, err = json.Marshal(node)
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("insert honerNode redis json failed")
			return
		}
	}
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("get honer node db failed")
		return
	}
	rd := kv.RedisParams{
		Key:   "honor-node",
		Value: string(value),
	}
	if err := rd.Set(); err != nil {
		log.WithFields(log.Fields{"error": err}).Error("insert honerNode redis failed")
		return
	}
}

func (p *HonorNodeInfo) DelRedis() {
	rd := kv.RedisParams{
		Key:   "honor-node",
		Value: "",
	}
	if err := rd.Del(); err != nil {
		log.WithFields(log.Fields{"err": err}).Warn("DelRedis failed:", " key:", rd.Key)
	}
}

func UpdateHonorNodeInfo() {
	HonorNodes = getHonorNodeInfo()
}

func FindNodeLocatedSave(list []NodeValue) (err error) {
	var (
		ip         string
		validNodes []HonorNodeInfo
	)
	for i := 0; i < len(list); i++ {
		var node HonorNodeInfo
		ip = getIPAddress(list[i].ApiAddress)
		if isNotIp := net.ParseIP(list[i].ApiAddress); isNotIp == nil {
			addr, err := net.ResolveIPAddr("ip", ip)
			if err != nil {
				log.WithFields(log.Fields{"info": err, "api address": list[i].ApiAddress}).Info("node resolve ip Failed")
				continue
			} else {
				ip = addr.String()
			}
		} else {
			log.WithFields(log.Fields{"info": err, "api_address": list[i].ApiAddress}).Info("node Parse IP Failed")
			continue
		}
		if ip == "" {
			continue
		} else if ip == "127.0.0.1" {
			node.Address = "Singapore-Singapore"
			node.Longitude = 103.854200
			node.Latitude = 1.340914
		} else {
			//node address format: country-city || country-continent
			info, result := findAddressFromIp(ip)
			if result == 3 {
				node.Address = info.CityName
			} else if result == 2 {
				if info.Longitude == 0 && info.Latitude == 0 {
					log.WithFields(log.Fields{"api address": list[i].ApiAddress}).Info("find Address From Ip Failed")
					continue
				}
				countryInfo := locator.FindCountryByCoordinate(info.Latitude, info.Longitude)
				if countryInfo.ADMIN != "Global" {
					node.Address = countryInfo.ADMIN + "-" + countryInfo.Continent
				}
			} else {
				continue
			}
			node.Latitude = info.Latitude
			node.Longitude = info.Longitude
		}
		node.Display = list[i].IsHonor

		value, err := json.Marshal(list[i])
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("FindNodeAddress json marshal Failed")
			continue
		}

		node.Value = string(value)

		//exist update
		if !node.nodeStatusUpdate() {
			validNodes = append(validNodes, node)
		}

	}

	//not exist insert
	if len(validNodes) > 0 {
		for i := 0; i < len(validNodes); i++ {
			//Before writing to the database, judge whether this value exists, if it exists, then judge whether the address of the ip is changed and update it, otherwise it will not be written
			if err := validNodes[i].insertData(); err != nil {
				log.WithFields(log.Fields{"warn": err}).Warn("honor node info insert db failed")
			}
		}
	}
	return nil
}

func getIconNationalFlag(icon string) string {
	IconFileName := strings.ToLower(icon) + ".png"
	var pictureName = consts.LogoRoad + "default.png"
	road, _ := os.Getwd()
	road = path.Join(road, "logo", IconFileName)
	_, err := os.Stat(road)
	if !os.IsNotExist(err) {
		pictureName = consts.LogoRoad + IconFileName
	}
	return pictureName
}

func getCity(city string) string {
	if strings.Contains(city, "-") {
		if index := strings.Index(city, "-"); index != -1 {
			return city[index+1:]
		}
	}
	return city
}

func getCountry(addr string) string {
	if strings.Contains(addr, "-") {
		if index := strings.Index(addr, "-"); index != -1 {
			return addr[:index]
		}
	}
	return addr
}

func getIPAddress(addressName string) (ip string) {
	ip = addressName
	if strings.Contains(addressName, "http") {
		total := `://`
		if index1 := strings.Index(addressName, total); index1 != -1 {
			if strings.Contains(addressName[index1+len(total):], ":") {
				if index2 := strings.Index(addressName[index1+len(total):], ":"); index2 != -1 {
					ip = addressName[index1+len(total) : index1+len(total)+index2]
				}
			} else {
				ip = addressName[index1+len(total):]
			}
		}
	}
	return ip
}

func findAddressFromIp(ipStr string) (info ipInfo, findResult int) {
	findResult = 1
	//findResult 1:invalid 2:notFind 3:findOut   2 or 3:valid
	info = ipInfo{}
	defer func() {
		if e := recover(); e != nil {
			panic(e)
		}
	}()
	if geoip.DB == nil {
		log.WithFields(log.Fields{"info": "geo ip db doesn't not init"}).Info("find Address From Ip failed")
		return
	}
	// If you are using strings that may be invalid, check that ip is not nil
	ip := net.ParseIP(ipStr)
	record, err1 := geoip.DB.City(ip)
	if err1 != nil {
		log.WithFields(log.Fields{"error": err1}).Error("findAddressFromIp db city err")
		return info, findResult
	}

	info.Latitude = record.Location.Latitude
	info.Longitude = record.Location.Longitude
	if len(record.Subdivisions) > 0 {
		cityName := record.Subdivisions[0].Names["en"]
		country := record.Country.Names["en"]
		if cityName != "" {
			findResult = 3
			info.CityName = strings.Replace(country, " ", "", -1) + "-" + cityName
		}
	}

	if info.CityName == "" {
		findResult = 2
		return info, findResult
	}
	return info, findResult
}

func (p *HonorNodeInfo) nodeStatusUpdate() bool {
	var (
		node  HonorNodeInfo
		value NodeValue
	)
	err := json.Unmarshal([]byte(p.Value), &value)
	if err != nil {
		log.WithFields(log.Fields{"error": err, "value": p.Value}).Error("Node Status Update value1 json unmarshal failed")
		return false
	}

	f, err := isFound(GetDB(nil).Table(p.TableName()).
		Where("CAST(value->>'id' as numeric) = ? AND CAST(value->>'consensus_mode' as numeric) = ?", value.Id, value.ConsensusMode).Take(&node))
	if err != nil {
		log.WithFields(log.Fields{"error": err, "node_id": value.Id, "consensus_mode": value.ConsensusMode}).Error("Node Status Update DB Failed ")
		return false
	}
	if !f {
		return false
	} else {
		if node.Address == p.Address && node.Display == p.Display &&
			node.Latitude == p.Latitude && node.Longitude == p.Longitude {
			return true
		}

		if err := GetDB(nil).Where("id = ?", node.ID).Model(HonorNodeInfo{}).Updates(map[string]interface{}{"address": p.Address,
			"latitude": p.Latitude, "longitude": p.Latitude, "display": p.Display,
		}).Error; err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Node Status Update err")
			return true
		}

		return true
	}
}

func (p *HonorNodeInfo) insertData() (err error) {
	err = nil
	if err = GetDB(nil).Model(HonorNodeInfo{}).Create(&p).Error; err != nil {
		return err
	}
	return err
}

func getHonorNodeInfo() []HonorNodeModel {
	var p HonorNodeInfo
	var nodeInfo []HonorNodeModel

	node, err := p.GetNodeList()
	if err == nil {
		nodeInfo = make([]HonorNodeModel, len(node))
		for i := 0; i < len(node); i++ {
			nodeInfo[i] = node[i]
		}
	} else {
		return nil
	}

	return nodeInfo
}

func redisOrderNode(cd []HonorNodeInfo, order string) (rd []HonorNodeInfo) {
	if strings.Contains(order, "id desc") {
		sort.SliceStable(cd, func(i, j int) bool {
			return cd[i].ID > cd[j].ID
		})
	} else if strings.Contains(order, "id asc") {
		sort.SliceStable(cd, func(i, j int) bool {
			return cd[i].ID < cd[j].ID
		})
	} else {
		log.WithFields(log.Fields{"warn": order}).Warn("redisOrderNode not find warn")
	}
	rd = cd
	return
}

func GetNodeListInfo(list []NodeValue, consensusMode int32) (err error) {
	var (
		valueList []string
		validNode []NodeValue
		p         HonorNodeInfo
		oldValue  []NodeValue
	)

	if err := GetDB(nil).Table(p.TableName()).Select("value").Order("id desc").Find(&valueList).Error; err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Get Node List Info api address list failed")
		return err
	}
	for _, val := range valueList {
		var info NodeValue
		if err := json.Unmarshal([]byte(val), &info); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("Get Node List Info json unmarshal failed")
			continue
		}
		oldValue = append(oldValue, info)
	}
	for i := 0; i < len(list); i++ {
		exist, index := honorNodeDbIsExist(list[i].Id, consensusMode, oldValue)
		if exist {
			if oldValue[index] != list[i] {
				if err := GetDB(nil).
					Where("CAST(value->>'id' as numeric) = ? AND CAST(value->>'consensus_mode' as numeric) = ?", list[i].Id, consensusMode).
					Model(HonorNodeInfo{}).Updates(map[string]interface{}{
					"value": list[i],
				}).Error; err != nil {
					log.WithFields(log.Fields{"error": err}).Error("Honor Node Value Status Update Err")
				}
			}
		}
		validNode = append(validNode, list[i])
	}
	if len(validNode) > 0 {
		err = FindNodeLocatedSave(validNode)
	}
	return err
}
