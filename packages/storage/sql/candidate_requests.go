package sql

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/converter"
	"github.com/IBAX-io/go-ibax/packages/smart"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"jutkey-server/packages/params"
	"reflect"
	"time"
)

var (
	PledgeAmount int64 = 1e18
	NodeReady    bool
)

type CandidateNodeRequests struct {
	Id           int64  `gorm:"primary_key;not_null"`
	TcpAddress   string `gorm:"not_null"`
	ApiAddress   string `gorm:"not_null"`
	NodePubKey   string `gorm:"not_null"`
	DateCreated  int64  `gorm:"not_null"`
	Deleted      int    `gorm:"not_null"`
	DateDeleted  int64  `gorm:"not_null"`
	Website      string `gorm:"not_null"`
	ReplyCount   int64  `gorm:"not_null"`
	DateReply    int64  `gorm:"not_null"`
	EarnestTotal string `gorm:"not_null"`
	NodeName     string `gorm:"not_null"`
}

type nodeDetailInfo struct {
	Id           int64
	NodeName     string
	Website      string
	ApiAddress   string
	Address      string
	Vote         decimal.Decimal
	VoteTrend    int
	EarnestTotal decimal.Decimal
	Committee    bool
	NodePubKey   string
	Ranking      int64
	Packed       int64
	PackedRate   string
	MyVote       decimal.Decimal
	MyStaking    string
	Decision     int
}

func (p *CandidateNodeRequests) TableName() string {
	return "1_candidate_node_requests"
}

func (p *CandidateNodeRequests) GetPubKeyById(id int64) (bool, error) {
	return isFound(GetDB(nil).Select("node_pub_key").Where("id = ?", id).First(&p))
}

func InitPledgeAmount() {
	if NodeReady {
		pledgeAmount, err := sqldb.GetPledgeAmount()
		if err != nil {
			log.WithFields(log.Fields{"error": err}).Error("init Pledge Amount Failed")
			return
		}

		PledgeAmount = pledgeAmount
	}
}

func NodeListSearch(page, limit int, wallet string) (*GeneralResponse, error) {
	var (
		list []NodeListResponse
		rets GeneralResponse
	)

	var info []nodeDetailInfo

	rets.Page = page
	rets.Limit = limit

	err := GetDB(nil).Raw(`
SELECT cs.node_name,cs.id,cs.website,cs.api_address,hr.address,cs.vote,RANK() OVER (ORDER BY vote DESC) AS ranking,cs.packed,
	CASE WHEN cs.packed > 0 THEN
		round(cs.packed*100 / cast( (SELECT max(id) FROM block_chain)  as numeric),2) 
	ELSE
		0
	END packed_rate,
	CASE WHEN cs.vote > cast(coalesce(hr.value->>'vote','0') as numeric) THEN
		1
	WHEN cs.vote < cast(coalesce(hr.value->>'vote','0') as numeric) THEN
		2
	ELSE
		3
	END vote_trend,cs.earnest_total,
	CASE WHEN cs.earnest_total >= ? AND row_number() OVER (ORDER BY vote DESC,date_updated_referendum asc) < 102 THEN 
		TRUE
	ELSE
		FALSE
	END committee,cs.node_pub_key,
	COALESCE((SELECT case WHEN coalesce(earnest,0) > 0 THEN 
		round(coalesce(earnest,0) / 1e12,12)
	ELSE
	 0
	END
	 FROM "1_candidate_node_decisions" WHERE request_id = cs.id AND account = ? AND decision_type = 1),'0')AS my_vote
 FROM (
	SELECT id,api_address,(SELECT count(1) FROM block_chain WHERE node_position = c1.id AND consensus_mode = 2)packed,website,node_name,earnest_total,node_pub_key,
		CASE WHEN coalesce(referendum_total,0)>0 THEN
			round(coalesce(referendum_total,0) / 1e12,12)
		ELSE
			0
		END
		as vote,date_updated_referendum
	FROM "1_candidate_node_requests" AS c1 WHERE deleted = 0
) AS cs
LEFT JOIN(
	SELECT value,address FROM honor_node_info AS he
)AS hr ON (cs.id = CAST(hr.value->>'id' AS numeric) AND CAST(hr.value->>'consensus_mode' AS numeric) = 2)
ORDER BY vote desc,date_updated_referendum asc OFFSET ? LIMIT ?
`, PledgeAmount, wallet, (page-1)*limit, limit).Find(&info).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Node List Search Failed")
		return &rets, err
	}

	type totalInfo struct {
		Total     int64
		VoteTotal decimal.Decimal
	}
	var ti totalInfo
	err = GetDB(nil).Raw(`
SELECT (SELECT count(1) FROM "1_candidate_node_requests" WHERE deleted = 0) total,case WHEN coalesce(sum(earnest),0) > 0 THEN 
	round(coalesce(sum(earnest),0) / 1e12,12)
ELSE
	0
END as vote_total FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3
AND request_id IN (SELECT id FROM "1_candidate_node_requests" WHERE deleted = 0)
`).Take(&ti).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Node List Search Total Info Failed")
		return &rets, err
	}
	rets.Total = ti.Total
	for i := 0; i < len(info); i++ {
		account := converter.IDToAddress(smart.PubToID(info[i].NodePubKey))
		if account == "invalid" {
			log.WithFields(log.Fields{"pub_key": info[i].NodePubKey}).Error("Node List Search Pub Key Failed")
			return &rets, errors.New("candidate requests pub_key invalid")
		}
		rts, err := info[i].getNodeDetailInfo(ti.VoteTotal, account)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "info": info[i]}).Error("Get Node Detail Info Failed")
			return &rets, err
		}

		list = append(list, rts)
	}
	rets.List = list
	return &rets, nil
}

func (p *nodeDetailInfo) getNodeDetailInfo(voteTotal decimal.Decimal, account string) (NodeListResponse, error) {
	var rets NodeListResponse
	zeroDec := decimal.New(0, 0)

	type committee struct {
		FrontCommittee bool
	}
	var rlt committee
	f, _ := isFound(GetDB(nil).Raw(`
	SELECT CASE WHEN (SELECT control_mode FROM "1_ecosystems" WHERE id = 1) = 2 THEN
		CASE WHEN (SELECT count(1) FROM "1_votings_participants" WHERE
				voting_id = (SELECT id FROM "1_votings" WHERE deleted = 0 AND voting->>'name' like '%%voting_for_control_mode_template%%' AND ecosystem = 1 ORDER BY id DESC LIMIT 1)
				AND member->>'account'=?) > 0 THEN
			TRUE
		ELSE
			FALSE
		END
	ELSE
		FALSE
	END AS front_committee
	`, account).Take(&rlt))
	if f && rlt.FrontCommittee {
		rets.FrontCommittee = true
	}
	rets.Committee = p.Committee
	rets.ApiAddress = p.ApiAddress
	rets.Name = p.NodeName
	rets.Vote = p.Vote.String()
	voteDec := p.Vote.Mul(decimal.NewFromInt(100))
	if voteDec.GreaterThan(zeroDec) && voteTotal.GreaterThan(zeroDec) {
		rets.VoteRate = voteDec.DivRound(voteTotal, 2).String()
	} else {
		rets.VoteRate = zeroDec.String()
	}
	rets.VoteTrend = p.VoteTrend
	rets.IconUrl = getIconNationalFlag(getCountry(p.Address))
	rets.Ranking = p.Ranking
	rets.Website = p.Website
	rets.Address = p.Address
	rets.Staking = p.EarnestTotal.String()
	rets.Id = p.Id
	rets.Packed = p.Packed
	rets.PackedRate = p.PackedRate
	rets.MyVote, _ = p.MyVote.Float64()
	rets.MyStaking = p.MyStaking
	if p.Decision == 2 || p.Decision == 4 {
		rets.Pending = true
	}
	return rets, nil
}

func CandidateTableExist() bool {
	var p CandidateNodeRequests
	if !HasTableOrView(p.TableName()) {
		return false
	}
	return true
}

func NodeDetail(search any, wallet string) (NodeDetailResponse, error) {
	var (
		nodeInfo  nodeDetailInfo
		rets      NodeDetailResponse
		voteTotal decimal.Decimal
		id        int64
		err       error
	)
	switch reflect.TypeOf(search).String() {
	case "json.Number":
		id, err = search.(json.Number).Int64()
		if err != nil {
			return rets, err
		}
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Node Detail Failed")
		return rets, errors.New("request params invalid")
	}

	err = GetDB(nil).Raw(`
SELECT case WHEN coalesce(sum(earnest),0) > 0 THEN
	round(coalesce(sum(earnest),0) / 1e12,12)
ELSE
	0
END as vote_total FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3
AND request_id IN (SELECT id FROM "1_candidate_node_requests" WHERE deleted = 0)
`).Take(&voteTotal).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err, "node id": id}).Error("Get Node Detail Vote Total Failed")
		return rets, err
	}

	err = GetDB(nil).Raw(`
SELECT cs.node_name,cs.id,cs.website,cs.api_address,hr.address,cs.vote,cs.packed,
	CASE WHEN cs.packed > 0 THEN
		round(cs.packed*100 / cast( (SELECT max(id) FROM block_chain)  as numeric),2)
	ELSE
		0
	END packed_rate,
(SELECT CASE WHEN (SELECT count(1) FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3 AND request_id = cs.id)  > 0 THEN
	(SELECT ct.ranking
	 FROM(
		SELECT coalesce(RANK() OVER (ORDER BY coalesce(sum(earnest),0) DESC),1)AS ranking,request_id FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3  GROUP BY request_id
	 )AS ct WHERE ct.request_id = cs.id)
ELSE
	(SELECT coalesce(count(1),0) + 1 FROM (SELECT request_id FROM "1_candidate_node_decisions" WHERE decision_type = 1 AND decision <> 3  GROUP BY request_id)AS te)
END) ranking,
CASE WHEN cs.vote > cast(coalesce(hr.value->>'vote','0') as numeric) THEN
	1
WHEN cs.vote < cast(coalesce(hr.value->>'vote','0') as numeric) THEN
	2
ELSE
	3
END vote_trend,cs.earnest_total,CASE WHEN cs.earnest_total >= ? AND row_number() OVER (ORDER BY vote DESC,date_updated_referendum asc) < 102 THEN
	TRUE
ELSE
	FALSE
END committee,cs.node_pub_key,
	COALESCE((SELECT case WHEN coalesce(earnest,0) > 0 THEN
		round(coalesce(earnest,0) / 1e12,12)
	ELSE
	 0
	END
	 FROM "1_candidate_node_decisions" WHERE request_id = cs.id AND account = ? AND decision_type = 1),'0')AS my_vote,
	COALESCE((SELECT earnest
	 FROM "1_candidate_node_decisions" WHERE request_id = cs.id AND account = ? AND decision_type = 2),'0')AS my_staking,
	COALESCE((SELECT decision
	 FROM "1_candidate_node_decisions" WHERE request_id = cs.id AND account = ? AND decision_type = 2),'0')AS decision
 FROM (
	SELECT id,api_address,website,node_name,earnest_total,(SELECT count(1) FROM block_chain WHERE node_position = cs.id AND consensus_mode = 2)packed,node_pub_key,
		CASE WHEN coalesce(referendum_total,0)>0 THEN
			round(coalesce(referendum_total,0) / 1e12,12)
		ELSE
			0
		END
		as vote,date_updated_referendum
	FROM "1_candidate_node_requests" AS cs WHERE deleted = 0 AND id = ?
) AS cs
LEFT JOIN(
	SELECT value,address FROM honor_node_info AS he
)AS hr ON (cs.id = CAST(hr.value->>'id' AS numeric) AND CAST(hr.value->>'consensus_mode' AS numeric) = 2)
`, PledgeAmount, wallet, wallet, wallet, id).Take(&nodeInfo).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err, "node id": id}).Error("Get Node Detail Failed")
		return rets, err
	}
	account := converter.IDToAddress(smart.PubToID(nodeInfo.NodePubKey))
	if account == "invalid" {
		log.WithFields(log.Fields{"pub_key": nodeInfo.NodePubKey}).Error("Get Node Detail Pub Key Failed")
		return rets, errors.New("candidate requests pub_key invalid")
	}

	info, err := nodeInfo.getNodeDetailInfo(voteTotal, account)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "node id": id}).Error("Get Node Detail Info Failed")
		return rets, err
	}
	rets.NodeListResponse = info
	zeroDec := decimal.New(0, 0)
	eligibleDecl := decimal.NewFromInt(PledgeAmount)
	stakeDec, _ := decimal.NewFromString(info.Staking)
	if stakeDec.GreaterThan(zeroDec) {
		rets.StakeRate = stakeDec.Mul(decimal.NewFromInt(100)).DivRound(eligibleDecl, 2).String()
	} else {
		rets.StakeRate = "0"
	}
	rets.Account = account

	return rets, nil
}

func GetDaoVoteList(search any, page, limit int) (GeneralResponse, error) {
	var (
		can     CandidateNodeRequests
		list    []NodeVoteResponse
		rets    GeneralResponse
		total   CountInt64
		err     error
		nodeId  int64
		account string
	)

	rets.Page = page
	rets.Limit = limit

	switch reflect.TypeOf(search).String() {
	case "json.Number":
		nodeId, err = search.(json.Number).Int64()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Get Node Dao Vote List Json Number Failed")
			return rets, err
		}
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Node Dao Vote List Search Failed")
		return rets, errors.New("request params invalid")
	}

	f, err := can.GetPubKeyById(nodeId)
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Get Node Dao Voting Pub Key Failed")
		return rets, err
	}
	if !f {
		return rets, errors.New("get Node Pub Key Failed")
	}

	account = converter.IDToAddress(smart.PubToID(can.NodePubKey))
	if account == "invalid" {
		log.WithFields(log.Fields{"pub_key": can.NodePubKey}).Error("Node Pub Key Invalid")
		return rets, errors.New("candidate requests pub_key invalid")
	}

	err = GetDB(nil).Raw(fmt.Sprintf(`
SELECT count(1) FROM "1_votings_participants" WHERE 
	voting_id IN(SELECT id FROM "1_votings" WHERE deleted = 0 AND voting->>'name' like '%%voting_for_control_mode_template%%' AND ecosystem = 1) 
	AND member->>'account'='%s'
`, account)).Take(&total).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Get Node Dao Vote List Total Failed")
		return rets, err
	}
	rets.Total = total.Count

	err = GetDB(nil).Raw(`
SELECT vs.title,vs.created,vs.voted_rate,sub.result_rate,sub.rejected_rate,vs.id,vs.result,coalesce((
	SELECT CASE WHEN decision = 1 THEN
		1
	WHEN decision = -1 THEN
		2
	ELSE
		3
	END 
	 FROM "1_votings_participants" WHERE member->>'account'=? AND voting_id = vs.id),0)AS status FROM(
	select id,coalesce(voting->>'name','') AS title,date_started AS created,
	CAST(coalesce(progress->>'percent_voters','0') as numeric)as voted_rate,
	CASE WHEN cast(flags->>'success' AS numeric) = 1 THEN
		case WHEN cast(flags->>'decision' AS numeric) = 1 THEN
		 1
		ELSE
		 2
		END
	ELSE
		3
	END result
	FROM "1_votings" WHERE deleted = 0 AND voting->>'name' like '%voting_for_control_mode_template%' AND id IN(
		SELECT voting_id FROM "1_votings_participants" WHERE voting_id IN(
			SELECT id FROM "1_votings" WHERE deleted = 0 AND voting->>'name' like '%voting_for_control_mode_template%' AND ecosystem = 1
		) AND member->>'account'=?
	)
)AS vs
LEFT JOIN (
	SELECT round(CAST(coalesce(results->>'percent_accepted','0') as numeric),2) AS result_rate,
			round(CAST(coalesce(results->>'percent_rejected','0') as numeric),2) AS rejected_rate,voting_id
	FROM "1_votings_subject"
)AS sub ON(sub.voting_id = vs.id)
	ORDER BY id desc OFFSET ? LIMIT ?
`, account, account, (page-1)*limit, limit).Find(&list).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Error("Get Node Dao Vote List Failed")
		return rets, err
	}
	rets.List = list

	return rets, nil
}

func GetNodeBlockList(search any, page, limit int, order string) (GeneralResponse, error) {
	var (
		list   []NodeBlockListResponse
		rets   GeneralResponse
		err    error
		id     int64
		bk     Block
		bkList []Block
		txList []LogTransaction
	)

	if order == "" {
		order = "id desc"
	}
	rets.Page = page
	rets.Limit = limit

	switch reflect.TypeOf(search).String() {
	case "json.Number":
		id, err = search.(json.Number).Int64()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Get Node Block List Json Number Failed")
			return rets, err
		}
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(search).String()}).Warn("Get Node Block List Search Failed")
		return rets, errors.New("request params invalid")
	}
	if id <= 0 {
		return rets, errors.New("unknown node id 0")
	}
	err = GetDB(nil).Table(bk.TableName()).Where("node_position = ? AND consensus_mode = 2", id).Count(&rets.Total).Error
	if err != nil {
		log.WithFields(log.Fields{"err": err}).Warn("Get Node Block List Total Failed")
		return rets, err
	}
	if rets.Total > 0 {
		err = GetDB(nil).Select("id,tx,time").Where("node_position = ? AND consensus_mode = 2", id).Offset((page - 1) * limit).Limit(limit).Order(order).Find(&bkList).Error
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Get Node Block Block List Failed")
			return rets, err
		}

		for _, value := range bkList {
			var rts NodeBlockListResponse
			rts.BlockId = value.ID

			err = GetDB(nil).Select("hash").Where("block = ?", value.ID).Find(&txList).Error
			if err != nil {
				log.WithFields(log.Fields{"err": err}).Warn("Get Node Block Tx List Failed")
				return rets, err
			}
			var hashList [][]byte
			for _, vue := range txList {
				hashList = append(hashList, vue.Hash)
			}
			type txGasFee struct {
				Amount      string
				Ecosystem   int64
				TokenSymbol string
			}
			var gasFee []txGasFee
			err = GetDB(nil).Raw(`
				SELECT h1.ecosystem,h1.amount,es.token_symbol FROM(
					SELECT ecosystem,sum(amount)amount FROM "1_history" WHERE txhash IN(?) AND type IN(1,2) GROUP BY ecosystem
				)AS h1
				LEFT JOIN(
					SELECT coalesce(token_symbol,'') token_symbol,id FROM "1_ecosystems"
				)AS es ON(es.id = h1.ecosystem)`, hashList).Find(&gasFee).Error
			if err != nil {
				log.WithFields(log.Fields{"err": err, "block_id": value.ID}).Warn("Get Node Block Tx List Failed")
				return rets, err
			}
			gasFeeCursor := 1
			rts.EcoNumber = len(gasFee)
			rts.Time = value.Time
			rts.Tx = value.Tx
			for _, vue := range gasFee {
				if vue.Ecosystem == 1 {
					rts.GasFee1.Amount = vue.Amount
					rts.GasFee1.TokenSymbol = vue.TokenSymbol
				} else {
					gasFeeCursor += 1
					if gasFeeCursor > 5 {
						break
					}
					switch gasFeeCursor {
					case 2:
						rts.GasFee2.Amount = vue.Amount
						rts.GasFee2.TokenSymbol = vue.TokenSymbol
					case 3:
						rts.GasFee3.Amount = vue.Amount
						rts.GasFee3.TokenSymbol = vue.TokenSymbol
					case 4:
						rts.GasFee4.Amount = vue.Amount
						rts.GasFee4.TokenSymbol = vue.Amount
					case 5:
						rts.GasFee5.Amount = vue.Amount
						rts.GasFee5.TokenSymbol = vue.TokenSymbol
					}
				}
			}

			list = append(list, rts)
		}
		rets.List = list
	}

	return rets, nil
}

func GetNodeVoteHistory(req *params.HistoryFindForm, getType int) (*NodeStakingHistoryResponse, error) {
	var (
		nodeId             int64
		err                error
		h1                 History
		rets               NodeStakingHistoryResponse
		vh                 []NodeStakingHistory
		getNowStakingQuery *gorm.DB
	)

	keyId := converter.StringToAddress(req.Wallet)
	type voteHistory struct {
		Txhash    []byte
		Vote      decimal.Decimal
		Amount    string
		CreatedAt int64
	}
	type voteInfo struct {
		MyVote       decimal.Decimal
		Staking      string
		Decision     int
		DateWithdraw int64
	}
	var list []voteHistory
	var info voteInfo
	if keyId == 0 {
		return nil, errors.New("request params wallet invalid")
	}
	switch reflect.TypeOf(req.Search).String() {
	case "json.Number":
		nodeId, err = req.Search.(json.Number).Int64()
		if err != nil {
			log.WithFields(log.Fields{"err": err}).Warn("Get Node Vote History json number invalid")
			return nil, err
		}
		if nodeId <= 0 {
			return nil, errors.New("request params node id invalid")
		}
	default:
		log.WithFields(log.Fields{"search type": reflect.TypeOf(req.Search).String()}).Warn("Get Node Vote History Search Failed")
		return nil, errors.New("request params Node Id invalid")
	}
	rets.Page = req.Page
	rets.Limit = req.Limit
	inType := []int{18, 19}
	outType := 21
	inComment := []string{fmt.Sprintf("Candidate Node Earnest #%d", nodeId), fmt.Sprintf("Candidate Node Substitute #%d", nodeId)}
	outComment := fmt.Sprintf("Candidate Node Withdraw Substitute #%d", nodeId)
	if getType == 1 {
		inType = []int{20}
		outType = 22
		inComment = []string{fmt.Sprintf("Candidate Node Referendum #%d", nodeId)}
		outComment = fmt.Sprintf("Candidate Node Withdraw Referendum #%d", nodeId)
		rets.TotalVote = "0" //default
	}

	if getType == 1 {
		getNowStakingQuery = GetDB(nil).Raw(`
SELECT earnest as staking,CASE WHEN COALESCE(earnest,0) > 0 THEN
	round(coalesce(earnest,0) / 1e12,12)
ELSE
	0
END AS my_vote FROM "1_candidate_node_decisions" WHERE request_id = ? AND account = ? AND decision_type = 1 AND decision <> 3
`, nodeId, req.Wallet)
	} else {
		getNowStakingQuery = GetDB(nil).Raw(`
SELECT earnest as staking,date_withdraw,decision FROM "1_candidate_node_decisions" WHERE request_id = ? AND account = ? AND decision_type = 2 AND decision <> 3
`, nodeId, req.Wallet)
	}

	f, err := isFound(getNowStakingQuery.Take(&info))
	if err != nil {
		log.WithFields(log.Fields{"error": err, "wallet": req.Wallet}).Warn("Get Node Vote History Info Failed")
		return nil, errors.New("request params wallet invalid")
	}
	if !f {
		rets.Staking = "0" //default
		return &rets, nil
	}

	_, err = isFound(GetDB(nil).Select("block_id,created_at").
		Where("recipient_id = ? AND type IN(?) AND comment IN(?)", keyId, outType, outComment).
		Order("block_id desc").First(&h1))
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("Get Node Vote History Withdraw referendum Failed")
		return nil, err
	}

	err = GetDB(nil).Raw(`
SELECT count(1) FROM "1_history" WHERE sender_id = ? AND TYPE IN(?) AND
			comment IN(?) AND block_id >= ? AND created_at > ?
`, keyId, inType, inComment, h1.BlockId, h1.CreatedAt).Count(&rets.Total).Error
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("Get Node Vote History Total Failed")
		return nil, err
	}
	err = GetDB(nil).Raw(`
SELECT txhash,created_at,amount,CASE WHEN coalesce(amount,0)>0 THEN
			round(coalesce(amount,0) / 1e12,12)
		ELSE
			0
		END
		AS vote FROM "1_history" WHERE sender_id = ? AND TYPE IN(?) AND
			comment IN(?) AND block_id >= ? AND created_at > ? ORDER BY block_id DESC
OFFSET ? Limit ?
`, keyId, inType, inComment, h1.BlockId, h1.CreatedAt, (req.Page-1)*req.Limit, req.Limit).Find(&list).Error
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Warn("Get Node Vote History Failed")
		return nil, err
	}

	for _, v := range list {
		var his NodeStakingHistory
		his.Hash = hex.EncodeToString(v.Txhash)
		if getType == 1 {
			his.Vote = v.Vote.String()
		}
		his.Amount = v.Amount
		his.Time = MsToSeconds(v.CreatedAt)
		vh = append(vh, his)
	}
	if getType == 1 {
		rets.TotalVote = info.MyVote.String()
	} else {
		if info.Decision == 2 || info.Decision == 4 {
			rets.GetStatus = 1
			if time.Now().Unix() >= info.DateWithdraw {
				rets.GetStatus = 2
			}
		}
		rets.DateWithdraw = info.DateWithdraw
	}
	rets.Staking = info.Staking
	rets.List = vh

	return &rets, nil
}

func GetNodeStatistics() (*NodeStatisticsResponse, error) {
	var (
		rets         NodeStatisticsResponse
		eligibleNode int64
		p            CandidateNodeRequests
	)
	err := GetDB(nil).Table(p.TableName()).Where("deleted = 0 AND earnest_total >= ?", PledgeAmount).Count(&eligibleNode).Error
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Get Node Map Eligible Node Failed")
		return nil, err
	}

	err = GetDB(nil).Table(p.TableName()).Where("deleted = 0 AND earnest_total < ?", PledgeAmount).Count(&rets.CandidateTotal).Error
	if err != nil {
		log.WithFields(log.Fields{"error": err}).Error("Get Node Map Candidate Total Failed")
		return nil, err
	}

	if eligibleNode > 101 {
		rets.HonorTotal = 101
		rets.CandidateTotal += eligibleNode - 101
	} else {
		rets.HonorTotal = eligibleNode
	}

	return &rets, nil

}
