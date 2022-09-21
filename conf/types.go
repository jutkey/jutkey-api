package conf

import (
	"context"
	"fmt"
	"github.com/centrifugal/gocent"
	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

var (
	publisher *gocent.Client
	rc        *redis.Client
	ctx       = context.Background()
)

type serverModel struct {
	Mode        string `yaml:"mode"`         // run mode
	Host        string `yaml:"host"`         // server host
	Port        int    `yaml:"port"`         // server port
	EnableHttps bool   `yaml:"enable_https"` // enable https
	CertFile    string `yaml:"cert_file"`    // cert file path
	KeyFile     string `yaml:"key_file"`     // key file path
	DocsApi     string `yaml:"docs_api"`     // api docs request address
	BaseUrl     string `yaml:"base_url"`
}

type crontab struct {
	RealTime string `yaml:"real_time"`
	Delay    string `yaml:"delay"`
}

type databaseModel struct {
	Enable  bool   `yaml:"enable"`
	DBType  string `yaml:"type"`
	Connect string `yaml:"connect"`
	Name    string `yaml:"name"`
	Ver     string `yaml:"ver"`
	MaxIdle int    `yaml:"max_idle"`
	MaxOpen int    `yaml:"max_open"`
}

type cryptoSettings struct {
	Cryptoer string `yaml:"cryptoer"`
	Hasher   string `yaml:"hasher"`
}

type centrifugoConfig struct {
	Enable bool   `yaml:"enable"`
	Secret string `yaml:"secret"`
	URL    string `yaml:"url"`
	Socket string `yaml:"socket"`
	Key    string `yaml:"key"`
}

type redisModel struct {
	Address  string `yaml:"address"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	Db       int    `yaml:"db"`
}

func (r *redisModel) Str() string {
	return fmt.Sprintf("%s:%d", r.Address, r.Port)
}

func (r *redisModel) Initer() error {
	rc = redis.NewClient(&redis.Options{
		Addr:     r.Str(),
		Password: r.Password,
		DB:       r.Db,
	})
	_, err := rc.Ping(ctx).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r *redisModel) Conn() *redis.Client {
	return rc
}

func (l *redisModel) Close() error {
	return rc.Close()
}

func (r *serverModel) Str() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

func (d *databaseModel) Close() error {
	if pgdb != nil {
		sqlDB, err := pgdb.DB()
		if err != nil {
			return err
		}
		if err = sqlDB.Close(); err != nil {
			return err
		}
		pgdb = nil
	}
	return nil
}

func (d *databaseModel) Conn() *gorm.DB {
	return pgdb
}

func (c *centrifugoConfig) Initer() error {
	if c.Enable {
		publisher = gocent.New(gocent.Config{
			Addr: c.URL,
			Key:  c.Key,
		})
	}
	return nil
}

func (c *centrifugoConfig) Conn() *gocent.Client {
	return publisher
}

func (l *centrifugoConfig) Close() error {
	return nil
}
