package cmd

import (
	"context"
	"fmt"
	"github.com/IBAX-io/go-ibax/packages/storage/sqldb"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"jutkey-server/conf"
	"jutkey-server/packages/api"
	"jutkey-server/packages/consts"
	"jutkey-server/packages/daemons"
	"jutkey-server/packages/storage/geoip"
	"jutkey-server/packages/storage/sql"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "jutkey-server",
	Short: "jutkey application",
}

func init() {
	rootCmd.AddCommand(
		initDatabaseCmd,
		startCmd,
		versionCmd,
	)
	consts.InitBuildInfo()
	// This flags are visible for all child commands
	rFlag := rootCmd.PersistentFlags()
	rFlag.StringVar(&conf.GetEnvConf().ConfigPath, "config", defaultConfigPath(), "filepath to config.yml")
}

func defaultConfigPath() string {
	p, err := os.Getwd()
	if err != nil {
		log.WithError(err).Fatal("getting cur wd")
	}
	return filepath.Join(p, "conf")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.WithError(err).Fatal("Executing root command")
	}
}

func loadStartRun() error {
	defer func() {
		if r := recover(); r != nil {
			log.WithFields(log.Fields{"panic": r, "type": consts.PanicRecoveredError}).Error("recovered panic")
			panic(r)
		}
	}()
	daemons.ExitCh = make(chan error)
	conf.Initer()

	exitErr := func() {
		api.SeverShutdown()
		err := sqldb.GormClose()
		if err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("sql db gorm close failed")
		}
		err = sql.GormClose()
		if err != nil {
			log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("pg db gorm close failed")
		}

		geoip.CloseGeoIp()

		os.Exit(1)
	}

	daemons.StartDaemons(context.Background())

	go func() {
		err := api.Run(conf.GetEnvConf().ServerInfo.Str())
		if err != nil {
			daemons.ExitCh <- fmt.Errorf("route run err:%s\n", err.Error())
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	select {
	case err := <-daemons.ExitCh:
		log.WithFields(log.Fields{"err:": err}).Error("Start Daemons Failed")
		exitErr()
		return err
	case <-sigChan:
		exitErr()
		return nil
	}
}

func loadInitDatabase() error {
	return conf.InitDatabase()
}

func loadConfigWKey(cmd *cobra.Command, args []string) {
	conf.LoadConfig(conf.GetEnvConf().ConfigPath)
}
