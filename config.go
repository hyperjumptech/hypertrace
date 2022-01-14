package hypertrace

import (
	configLog "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

var (
	defCfg map[string]string
)

// initialize this configuration
func init() {
	viper.SetEnvPrefix("trace")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	defCfg = make(map[string]string)

	defCfg["adminpassword"] = "admin password is a secret"

	defCfg["server.host"] = "0.0.0.0"
	defCfg["server.port"] = "8080"

	defCfg["mongo.database"] = "hypertrace"
	defCfg["mongo.host"] = "localhost"
	defCfg["mongo.port"] = "27017"
	defCfg["mongo.user"] = "root"
	defCfg["mongo.password"] = "root"

	defCfg["tempid.valid.period.hour"] = "1"
	defCfg["tempid.count"] = "100"
	defCfg["tempid.crypt.key"] = "tH1Sis4nEncryPt10nKeydOn0tsHar3!"

	for k := range defCfg {
		err := viper.BindEnv(k)
		if err != nil {
			configLog.Errorf("Failed to bind env \"%s\" into configuration. Got %s", k, err)
		}
	}

}

// SetConfig put configuration key value
func SetConfig(key, value string) {
	viper.Set(key, value)
}

// ConfigGet fetch configuration as string value
func ConfigGet(key string) string {
	ret := viper.GetString(key)
	if len(ret) == 0 {
		if ret, ok := defCfg[key]; ok {
			return ret
		}
		configLog.Debugf("%s config key not found", key)
	}
	return ret
}

// ConfigGetBoolean fetch configuration as boolean value
func ConfigGetBoolean(key string) bool {
	if len(ConfigGet(key)) == 0 {
		return false
	}
	b, err := strconv.ParseBool(ConfigGet(key))
	if err != nil {
		panic(err)
	}
	return b
}

// ConfigGetInt fetch configuration as integer value
func ConfigGetInt(key string) int {
	if len(ConfigGet(key)) == 0 {
		return 0
	}
	i, err := strconv.ParseInt(ConfigGet(key), 10, 64)
	if err != nil {
		panic(err)
	}
	return int(i)
}

// ConfigGetFloat fetch configuration as float value
func ConfigGetFloat(key string) float64 {
	if len(ConfigGet(key)) == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(ConfigGet(key), 64)
	if err != nil {
		panic(err)
	}
	return f
}
