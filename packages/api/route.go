package api

import (
	"context"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/utils"
	"github.com/gin-gonic/gin/binding"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"golang.org/x/net/http2"
	"jutkey-server/packages/consts"
	"net/http"
	_ "net/http/pprof"
	"strings"

	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth_gin"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"jutkey-server/conf"
	"jutkey-server/docs"
)

func init() {
	if err := utils.MakeDirectory("./logo"); err != nil {
		log.WithFields(log.Fields{"error": err, "type": consts.IOError}).Error("can't create temporary logo directory")
	}

	if err := utils.MakeDirectory("./upload"); err != nil {
		log.WithFields(log.Fields{"error": err, "type": consts.IOError}).Error("can't create temporary upload directory")
	}

	binding.EnableDecoderUseNumber = true
}

var server *http.Server

func Cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		method := c.Request.Method

		origin := c.Request.Header.Get("Origin")
		var headerKeys []string
		for k := range c.Request.Header {
			headerKeys = append(headerKeys, k)
		}
		headerStr := strings.Join(headerKeys, ", ")
		if headerStr != "" {
			headerStr = fmt.Sprintf("access-control-allow-origin, access-control-allow-headers, %s", headerStr)
		} else {
			headerStr = "access-control-allow-origin, access-control-allow-headers"
		}
		if origin != "" {
			//c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Origin", "*")
			//c.Header("Access-Control-Allow-Headers", headerStr)
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length, X-CSRF-Token, Accept, Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, Keep-Alive, User-Agent, X-Requested-With, If-Modified-Since, Cache-Control, Content-Type, Pragma")
			c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Content-Type")
			// c.Header("Access-Control-Max-Age", "172800")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Set("content-type", "application/json")
		}

		//OPTIONS
		if method == "OPTIONS" {
			c.JSON(http.StatusOK, "Options Request!")
		}
		c.Next()
	}
}
func prefix(s string) string {
	return "/api/v2/" + s
}

func Run(host string) (err error) {
	r := gin.Default()
	//Ten requests per second
	limiter := tollbooth.NewLimiter(10, nil)
	r.Use(Cors(), tollbooth_gin.LimitHandler(limiter))
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": consts.Version(),
		})
	})

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	rte := r.Group(consts.ApiPath)

	// programatically set swagger info
	docs.SwaggerInfo.Title = "jutkey API"
	docs.SwaggerInfo.Description = "jutkey wallet api server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = conf.GetEnvConf().ServerInfo.DocsApi
	//docs.SwaggerInfo.BasePath = ""
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
	//use ginSwagger middleware to serve the API docs
	rte.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	rte.GET("/websocket_token", getWebsocketToken)

	//ecoLibs
	rte.POST("/eco_libs", getAllEcosystemList)
	rte.POST("/ecosystem_search", ecosystemSearchHandler)

	//dashboard
	rte.GET("/statistics", getStatisticsHandler)
	rte.POST("/ecosystem_key_totals", getEcosystemThroughKey)
	rte.POST("/month_history_detail", monthHistoryDetailHandler)
	rte.POST("/month_history_total", monthHistoryTotalHandler)
	rte.POST("/user_nft_miner_summary", userNftMinerSummaryHandler)
	rte.POST("/key_income_day", nftMinerDayRewardHandler)
	rte.POST("/key_amount", getKeyAmountHandler)

	//nft-miner
	rte.POST("/nft_miner_key_infos", getNftMinerKeyInfosHandler)
	rte.POST("/nft_miner_reward_history", getNftMinerRewardHistoryHandler)
	rte.POST("/nft_miner_detail", getNftMinerDetailHandler)
	rte.POST("/nft_miner_staking", getNftMinerStakingHandler)
	rte.POST("/nft_miner_reward", getNftMinerRewardHandler)
	rte.GET("/nft_miner_file/:id", getNftMinerFileHandler)

	//user-center
	rte.POST("/history", getHistoryHandler)
	rte.POST("/key_total", getKeyTotalHandler)
	rte.GET("/assign_balance/:wallet", getMyAssignBalanceHandler)
	rte.GET("/key_info/:account", getKeyInfoHandler)
	rte.POST("/get_utxo_input", getUtxoInputHandler)

	//honor-node
	rte.GET("/node_statistics", getNodeStatisticsHandler)
	rte.POST("/node_list", getHonorNodeListHandler)
	rte.POST("/node_detail", nodeDetailHandler)
	rte.POST("/node_dao_list", getNodeDaoVoteListHandler)
	rte.POST("/node_block_list", getNodeBlockListHandler)
	rte.POST("/node_vote_history", getNodeVoteHistoryHandler)
	rte.POST("/node_substitute_history", getNodeSubstituteHistoryHandler)

	//other
	rte.GET(`/get_attachment/:hash`, getAttachmentHandler)
	rte.GET("/get_locator", getLocatorHandler)

	rte.StaticFS("/logo", http.Dir("./logo"))

	server = &http.Server{
		Addr:    host,
		Handler: r,
	}
	err = http2.ConfigureServer(server, &http2.Server{})
	if err != nil {
		log.Errorf("http2 configure Server failed :%s", err.Error())
		return err
	}
	if conf.GetEnvConf().ServerInfo.EnableHttps {
		err = server.ListenAndServeTLS(conf.GetEnvConf().ServerInfo.CertFile, conf.GetEnvConf().ServerInfo.KeyFile)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil {
		log.Errorf("server http/https start failed :%s", err.Error())
		return err
	}

	return nil
}

func SeverShutdown() {
	if server != nil {
		if err := server.Shutdown(context.Background()); err != nil {
			log.WithFields(log.Fields{"error": err}).Error("sever shutdown failed")
		}
	}
}
