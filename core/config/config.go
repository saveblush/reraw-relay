package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/goccy/go-json"
	"github.com/spf13/viper"

	"github.com/saveblush/reraw-relay/core/utils/logger"
)

var (
	CF = &Configs{}
)

var (
	filePath                           = "./configs"
	fileExtension                      = "yml"
	fileNameConfig                     = "config"
	fileNameConfigAvailableStatus      = "config_available_status.yml"
	fileNameConfigAvailableDescription = "config_available_description.yml"
	AvailableStatusOnline              = "online"
	AvailableStatusOffline             = "offline"
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

type AvailableConfig struct {
	Status string `json:"status"`
}

type DatabaseConfig struct {
	Host         string        `mapstructure:"HOST"`
	Port         int           `mapstructure:"PORT"`
	Username     string        `mapstructure:"USERNAME"`
	Password     string        `mapstructure:"PASSWORD"`
	DatabaseName string        `mapstructure:"DATABASE_NAME"`
	Timeout      string        `mapstructure:"TIMEOUT"`
	MaxIdleConns int           `mapstructure:"MAX_IDLE_CONNS"`
	MaxOpenConns int           `mapstructure:"MAX_OPEN_CONNS"`
	MaxLifetime  time.Duration `mapstructure:"MAX_LIFE_TIME"`
}

type infoLimitation struct {
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
		Name          string `mapstructure:"NAME"`
		Description   string `mapstructure:"DESCRIPTION"`
		Pubkey        string `mapstructure:"PUBKEY"`
		Contact       string `mapstructure:"CONTACT"`
		SupportedNIPs []int  `mapstructure:"SUPPORTED_NIPS" json:"supported_nips"`
		Software      string `mapstructure:"SOFTWARE"`
		Version       string `mapstructure:"VERSION"`
		Icon          string `mapstructure:"ICON"`
		/*Limitation    struct {
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
		} `mapstructure:"LIMITATION"`*/
		Limitation *infoLimitation `mapstructure:"LIMITATION"`
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

	// set config ปิด/เปิด ระบบ
	if err := initConfigAvailable(); err != nil {
		logger.Log.Errorf("init config available error: %s", err)
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

// initConfigAvailable init config available
// init config ปิด/เปิด ระบบ
func initConfigAvailable() error {
	// create file config
	if err := CF.SetConfigAvailableStatus(AvailableStatusOnline); err != nil {
		logger.Log.Errorf("creating file available status error: %s", err)
		return err
	}

	if err := CF.SetConfigAvailableDescription(""); err != nil {
		logger.Log.Errorf("creating file available description error: %s", err)
		return err
	}

	// read config
	v := viper.New()
	v.AddConfigPath(filePath)
	v.SetConfigName(fileNameConfigAvailableStatus)
	v.SetConfigType(fileExtension)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		logger.Log.Errorf("read config file error: %s", err)
		return err
	}

	cf := &AvailableConfig{}
	if err := v.Unmarshal(cf); err != nil {
		logger.Log.Errorf("binding config error: %s", err)
		return err
	}
	CF.App.AvailableStatus = cf.Status

	v.OnConfigChange(func(e fsnotify.Event) {
		logger.Log.Infof("config file changed: %s", e.Name)
		if err := v.Unmarshal(cf); err != nil {
			logger.Log.Errorf("binding config error: %s", err)
		}
		CF.App.AvailableStatus = cf.Status
	})
	v.WatchConfig()

	return nil
}

// SetConfigAvailableStatus set config available status
// สร้าง config สถานะ ปิด/เปิด ระบบ
func (cf *Configs) SetConfigAvailableStatus(status string) error {
	d, _ := json.Marshal(&AvailableConfig{
		Status: status,
	})
	p := fmt.Sprintf("%s/%s", filePath, fileNameConfigAvailableStatus)
	err := os.WriteFile(p, d, 0644)
	if err != nil {
		return err
	}

	return nil
}

// SetConfigAvailableDescription set config available description
// สร้าง config html ใช้แสดงเมื่อปิดระบบ
func (cf *Configs) SetConfigAvailableDescription(body string) error {
	d := []byte(body)
	p := fmt.Sprintf("%s/%s", filePath, fileNameConfigAvailableDescription)
	err := os.WriteFile(p, d, 0644)
	if err != nil {
		return err
	}

	return nil
}

// ReadConfigAvailableDescription read config available description
// อ่าน config html ใช้แสดงเมื่อปิดระบบ
func (cf *Configs) ReadConfigAvailableDescription() (string, error) {
	d, err := os.ReadFile(fmt.Sprintf("./%s/%s", filePath, fileNameConfigAvailableDescription))
	if err != nil {
		return "", err
	}

	return string(d), nil
}
