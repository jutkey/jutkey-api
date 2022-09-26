package conf

import (
	"encoding/json"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/common/crypto"
	"github.com/IBAX-io/go-ibax/packages/conf/syspar"
	"github.com/IBAX-io/go-ibax/packages/smart"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"path"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	configInfo EnvConf // all server config information
	pgdb       *gorm.DB
)

type EnvConf struct {
	ConfigPath     string
	ServerInfo     *serverModel      `yaml:"server"`
	Centrifugo     *centrifugoConfig `yaml:"centrifugo"`
	DatabaseInfo   *databaseModel    `yaml:"database"`
	RedisInfo      *redisModel       `yaml:"redis"`
	Crontab        *crontab          `yaml:"crontab"`
	CryptoSettings cryptoSettings    `yaml:"crypto_settings"`
}

func GetEnvConf() *EnvConf {
	return &configInfo
}

func GetDbConn() *databaseModel {
	return GetEnvConf().DatabaseInfo
}

func GetRedisDbConn() *redisModel {
	return GetEnvConf().RedisInfo
}

func GetCentrifugoConn() *centrifugoConfig {
	return GetEnvConf().Centrifugo
}

func LoadConfig(configPath string) {
	filePath := path.Join(configPath, "config.yml")
	configData, err := os.ReadFile(filePath)
	if err != nil {
		logrus.WithError(err).Fatal("config file read failed")
	}
	configData = []byte(os.ExpandEnv(string(configData)))
	err = yaml.Unmarshal(configData, &configInfo)
	data, _ := json.Marshal(&configInfo)
	fmt.Printf("config: %v\n", string(data))
	if err != nil {
		logrus.WithError(err).Fatal("config parse failed")
	}
	registerCrypto(GetEnvConf().CryptoSettings)
}

func Initer() {
	redis := GetEnvConf().RedisInfo
	centrifugo := GetEnvConf().Centrifugo
	err := initLogs()
	if err != nil {
		logrus.WithError(err).Fatal("init log file")
	}
	if err := redis.Initer(); err != nil {
		logrus.WithError(err).Fatal("redis database config information: %v", redis)
	}
	if err := centrifugo.Initer(); err != nil {
		logrus.WithError(err).Fatal("centrifugo config information: %v", centrifugo)
	}
	err = InitDatabase()
	if err != nil {
		logrus.WithError(err).Fatal("postgres sql database connect failed")
	}
}

func InitTimeLocal() {
	time.Local = time.UTC
}

func initLogs() error {
	InitTimeLocal()
	fileName := path.Join(GetEnvConf().ConfigPath, "logrus.log")
	openMode := os.O_APPEND
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		openMode = os.O_CREATE
	}
	f, err := os.OpenFile(fileName, os.O_WRONLY|openMode, 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Can't open log file: ", fileName)
		return err
	}
	logrus.SetOutput(f)
	return nil
}

func registerCrypto(c cryptoSettings) {
	crypto.InitAsymAlgo(c.Cryptoer)
	crypto.InitHashAlgo(c.Hasher)
}

func InitDatabase() (err error) {
	dbn := configInfo.DatabaseInfo

	dsn := fmt.Sprintf("%s TimeZone=UTC", dbn.Connect)
	pgdb, err = gorm.Open(postgres.New(postgres.Config{
		DSN: dsn,
	}), &gorm.Config{
		//AllowGlobalUpdate: true,                      //allow global update
		Logger: logger.Default.LogMode(logger.Silent), // start Logger,show detail log
	})

	if err != nil {
		return err
	}
	dbSql, err := pgdb.DB()
	if err != nil {
		return err
	}
	dbSql.SetConnMaxLifetime(time.Minute * 10)
	dbSql.SetMaxIdleConns(5)
	dbSql.SetMaxOpenConns(20)
	sqldb.DBConn = pgdb

	if err = syspar.SysUpdate(nil); err != nil {
		return err
	}
	smart.InitVM()
	if err := smart.LoadContracts(); err != nil {
		return err
	}
	return nil
}
