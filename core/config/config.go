package config

import (
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"

	"github.com/saveblush/reraw-relay/core/utils/logger"
)

var (
	CF = &Configs{}
)

var (
	filePath       = "./configs"
	fileExtension  = "yml"
	fileNameConfig = "config"
)

// Environment environment
type Environment string

const (
	Develop    Environment = "develop"
	Production Environment = "prod"
)

// Production check is production
func (e Environment) Production() bool {
	return e == Production
}

type DatabaseConfig struct {
	Host         string        `mapstructure:"HOST"`
	Port         int           `mapstructure:"PORT"`
	Username     string        `mapstructure:"USERNAME"`
	Password     string        `mapstructure:"PASSWORD"`
	DatabaseName string        `mapstructure:"DATABASE_NAME"`
	MaxIdleConns int           `mapstructure:"MAX_IDLE_CONNS"`
	MaxOpenConns int           `mapstructure:"MAX_OPEN_CONNS"`
	MaxLifetime  time.Duration `mapstructure:"MAX_LIFE_TIME"`
}

type InfoLimitation struct {
	MaxMessageLength int  `mapstructure:"MAX_MESSAGE_LENGTH"`
	MaxSubscriptions int  `mapstructure:"MAX_SUBSCRIPTIONS"`
	MaxFilters       int  `mapstructure:"MAX_FILTERS"`
	MaxLimit         int  `mapstructure:"MAX_LIMIT"`
	MaxSubidLength   int  `mapstructure:"MAX_SUBID_LENGTH"`
	MaxEventTags     int  `mapstructure:"MAX_EVENT_TAGS"`
	MaxContentLength int  `mapstructure:"MAX_CONTENT_LENGTH"`
	MinPowDifficulty int  `mapstructure:"MIN_POW_DIFFICULTY"`
	AuthRequired     bool `mapstructure:"AUTH_REQUIRED"`
	PaymentRequired  bool `mapstructure:"PAYMENT_REQUIRED"`
	RestrictedWrites bool `mapstructure:"RESTRICTED_WRITES"`
}

type Configs struct {
	Info struct {
		Name          string          `mapstructure:"NAME"`
		Description   string          `mapstructure:"DESCRIPTION"`
		Pubkey        string          `mapstructure:"PUBKEY"`
		Contact       string          `mapstructure:"CONTACT"`
		SupportedNIPs []int           `mapstructure:"SUPPORTED_NIPS" json:"supported_nips"`
		Software      string          `mapstructure:"SOFTWARE"`
		Version       string          `mapstructure:"VERSION"`
		Icon          string          `mapstructure:"ICON"`
		Limitation    *InfoLimitation `mapstructure:"LIMITATION"`
	} `mapstructure:"INFO"`

	App struct {
		AvailableStatus string      // สถานะปิด/เปิดระบบ [on/off]
		Port            int         `mapstructure:"PORT"`
		Environment     Environment `mapstructure:"ENVIRONMENT"`
	} `mapstructure:"APP"`

	Database struct {
		RelaySQL DatabaseConfig `mapstructure:"RELAY_SQL"`
	} `mapstructure:"DATABASE"`
}

// InitConfig init config
func InitConfig() error {
	v := viper.New()
	v.AddConfigPath(filePath)
	v.SetConfigName(fileNameConfig)
	v.SetConfigType(fileExtension)
	v.AutomaticEnv()

	// แปลง _ underscore เป็น . dot
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		logger.Log.Errorf("read config file error: %s", err)
		return err
	}

	if err := v.Unmarshal(CF); err != nil {
		logger.Log.Errorf("binding config error: %s", err)
		return err
	}

	v.OnConfigChange(func(e fsnotify.Event) {
		logger.Log.Infof("config file changed: %s", e.Name)
		if err := v.Unmarshal(CF); err != nil {
			logger.Log.Errorf("binding config error: %s", err)
		}
	})
	v.WatchConfig()

	return nil
}
