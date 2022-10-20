package dbtoolset

import (
	"strings"
	"sync"

	// make sure mysql engine
	_ "github.com/go-sql-driver/mysql"

	"github.com/go-redis/redis/v8"
	"github.com/sgostarter/i/l"
	"github.com/sgostarter/libstg/mysqlgorm"
	"github.com/sgostarter/libstg/mysqlxorm"
	"github.com/sgostarter/libstg/redisv8"
	"gorm.io/gorm"
	"xorm.io/xorm"
)

const (
	DefaultName = "default"
)

type Config struct {
	RedisDSN     string            `yaml:"redis_dsn" json:"redis_dsn"`
	RedisDSNList map[string]string `yaml:"redis_dsn_list" json:"redis_dsn_list"`

	MysqlDSN     string            `yaml:"mysql_dsn" json:"mysql_dsn"`
	MysqlDSNList map[string]string `yaml:"mysql_dsn_list" json:"mysql_dsn_list"`
}

type Toolset struct {
	cfg    *Config
	logger l.Wrapper

	redisOnce sync.Once
	redisCli  *redis.Client

	redisListOnce sync.Once
	redisList     map[string]*redis.Client

	xOrmOnce   sync.Once
	xOrmEngine *xorm.Engine

	xOrmListOnce sync.Once
	xOrmList     map[string]*xorm.Engine

	gOrmOnce sync.Once
	gOrmDB   *gorm.DB

	gOrmListOnce sync.Once
	gOrmList     map[string]*gorm.DB
}

func NewToolset(cfg *Config, logger l.Wrapper) *Toolset {
	if logger == nil {
		logger = l.NewNopLoggerWrapper()
	}

	return &Toolset{
		cfg:       cfg,
		logger:    logger.WithFields(l.StringField(l.ClsKey, "Toolset")),
		redisList: make(map[string]*redis.Client),
		xOrmList:  make(map[string]*xorm.Engine),
		gOrmList:  make(map[string]*gorm.DB),
	}
}

func (toolset *Toolset) GetRedis() *redis.Client {
	toolset.redisOnce.Do(func() {
		redisCli, err := redisv8.InitRedis(toolset.cfg.RedisDSN)
		if err != nil {
			toolset.logger.WithFields(l.ErrorField(err), l.StringField("dsn", toolset.cfg.RedisDSN)).Error("initRedisFailed")

			return
		}

		toolset.redisCli = redisCli
	})

	return toolset.redisCli
}

func (toolset *Toolset) GetRedisByName(name string) *redis.Client {
	toolset.redisListOnce.Do(func() {
		for name, dsn := range toolset.cfg.RedisDSNList {
			redisCli, err := redisv8.InitRedis(dsn)
			if err != nil {
				toolset.logger.WithFields(l.ErrorField(err), l.StringField("name", name),
					l.StringField("dsn", toolset.cfg.RedisDSN)).Error("initRedisFailed")

				return
			}

			toolset.redisList[name] = redisCli
		}
	})

	if redisCli, ok := toolset.redisList[name]; ok {
		return redisCli
	}

	if strings.EqualFold(name, DefaultName) {
		return toolset.GetRedis()
	}

	return nil
}

func (toolset *Toolset) GetXOrm() *xorm.Engine {
	toolset.xOrmOnce.Do(func() {
		db, err := mysqlxorm.InitXorm(toolset.cfg.MysqlDSN)
		if err != nil {
			toolset.logger.WithFields(l.ErrorField(err), l.StringField("dsn", toolset.cfg.MysqlDSN)).
				Error("initXOrmFailed")

			return
		}

		toolset.xOrmEngine = db
	})

	return toolset.xOrmEngine
}

func (toolset *Toolset) GetXOrmByName(name string) *xorm.Engine {
	toolset.xOrmListOnce.Do(func() {
		for name, dsn := range toolset.cfg.MysqlDSNList {
			db, err := mysqlxorm.InitXorm(dsn)
			if err != nil {
				toolset.logger.WithFields(l.ErrorField(err), l.StringField("dsn", toolset.cfg.MysqlDSN),
					l.StringField("name", name)).Error("initXOrmFailed")

				return
			}

			toolset.xOrmList[name] = db
		}
	})

	if db, ok := toolset.xOrmList[name]; ok {
		return db
	}

	if strings.EqualFold(name, DefaultName) {
		return toolset.GetXOrm()
	}

	return nil
}

func (toolset *Toolset) GetGOrm() *gorm.DB {
	toolset.gOrmOnce.Do(func() {
		db, err := mysqlgorm.InitGorm(toolset.cfg.MysqlDSN)
		if err != nil {
			toolset.logger.WithFields(l.ErrorField(err), l.StringField("dsn", toolset.cfg.MysqlDSN)).
				Error("initGOrmFailed")

			return
		}

		toolset.gOrmDB = db
	})

	return toolset.gOrmDB
}

func (toolset *Toolset) GetGOrmByName(name string) *gorm.DB {
	toolset.gOrmListOnce.Do(func() {
		for name, dsn := range toolset.cfg.MysqlDSNList {
			db, err := mysqlgorm.InitGorm(dsn)
			if err != nil {
				toolset.logger.WithFields(l.ErrorField(err), l.StringField("dsn", dsn),
					l.StringField("name", name)).Error("initGOrmFailed")

				return
			}

			toolset.gOrmList[name] = db
		}
	})

	if db, ok := toolset.gOrmList[name]; ok {
		return db
	}

	if strings.EqualFold(name, DefaultName) {
		return toolset.GetGOrm()
	}

	return nil
}
