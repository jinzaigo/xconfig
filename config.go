package xconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jinzaigo/xconfig/remote"
	"github.com/shima-park/agollo"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"log"
	"os"
	"path/filepath"
	"sync"
)

type Config struct {
	configType string
	viper      *viper.Viper
	viperLock  sync.RWMutex //viper并发读写不安全 https://github.com/spf13/viper#is-it-safe-to-concurrently-read-and-write-to-a-viper
}

func New(opts ...OptionFunc) *Config {
	o := option{}
	for _, opt := range opts {
		opt(&o)
	}

	//检查配置文件是否合法
	if o.configFile != "" {
		statInfo, err := os.Stat(o.configFile)
		if err != nil {
			log.Panic(o.configFile + " config file path error:" + err.Error())
		}
		if statInfo.IsDir() {
			log.Panic(o.configFile + " config file path is dir")
		}
	}

	v := viper.New()
	if o.configType != "" {
		v.SetConfigType(o.configType)
	}
	if o.configFile != "" {
		v.SetConfigFile(o.configFile)
		err := v.ReadInConfig()
		if err != nil {
			log.Panic(o.configFile + " ReadInConfig error: " + err.Error())
		}
		//热加载
		v.WatchConfig()
	}

	return &Config{
		configType: getConfigType(o.configType, o.configFile),
		viper:      v,
	}
}

func getConfigType(configType, configFile string) string {
	if configType != "" {
		return configType
	}

	ext := filepath.Ext(configFile)
	if ext != "" {
		return ext[1:]
	}
	return ""
}

func (c *Config) AddApolloRemoteConfig(endpoint, appId, namespace, backupFile string) error {
	if endpoint == "" || appId == "" || namespace == "" || backupFile == "" {
		return errors.New("AddApolloRemoteConfig params error")
	}
	if c.viper == nil || c.configType == "" {
		return errors.New("viper is not init or configType is empty")
	}

	c.viper.SetConfigType(c.configType)
	//namespace默认类型不用加后缀，非默认类型需要加后缀（备注：这里会涉及到apollo变更通知后的热加载操作 Start->longPoll）
	//详见接口文档https://www.apolloconfig.com/#/zh/usage/other-language-client-user-guide?id=_14-%e5%ba%94%e7%94%a8%e6%84%9f%e7%9f%a5%e9%85%8d%e7%bd%ae%e6%9b%b4%e6%96%b0 notifications/v2 notifications字段说明
	if c.configType != "properties" {
		namespace = namespace + "." + c.configType
	}

	//用appId换取provider，让viper认识它
	provider := remote.AddProviders(appId, agollo.AutoFetchOnCacheMiss(), agollo.BackupFile(backupFile), agollo.FailTolerantOnBackupExists())

	err := c.viper.AddRemoteProvider(provider, endpoint, namespace)
	if err != nil {
		return errors.New("viper AddRemoteProvider error:" + err.Error())
	}
	err = c.viper.ReadRemoteConfig()
	if err != nil {
		return errors.New("viper ReadRemoteConfig error:" + err.Error())
	}

	//热加载
	//用viper自带的方法有并发读写不安全问题，下面采用重写，起协程并加读写锁来解决
	//_ = c.viper.WatchRemoteConfigOnChannel()
	respc, _ := viper.RemoteConfig.WatchChannel(remote.NewProviderSt(provider, endpoint, namespace, ""))
	go func(rc <-chan *viper.RemoteResponse) {
		for {
			<-rc
			c.viperLock.Lock()
			err = c.viper.ReadRemoteConfig()
			c.viperLock.Unlock()
		}
	}(respc)

	return nil
}

// IsSet 判断配置项是否存在
func (c *Config) IsSet(key string) bool {
	c.viperLock.RLock()
	defer c.viperLock.RUnlock()
	return c.viper.IsSet(key)
}

func (c *Config) Get(key string) interface{} {
	c.viperLock.RLock()
	defer c.viperLock.RUnlock()
	return c.viper.Get(key)
}

// AllSettings 获取所有的配置信息
func (c *Config) AllSettings() map[string]interface{} {
	c.viperLock.RLock()
	defer c.viperLock.RUnlock()
	return c.viper.AllSettings()
}

// GetStringMap 根据key获取配置信息
func (c *Config) GetStringMap(key string) map[string]interface{} {
	return cast.ToStringMap(c.Get(key))
}

// GetStringMapString 根据key获取配置信息
func (c *Config) GetStringMapString(key string) map[string]string {
	return cast.ToStringMapString(c.Get(key))
}

// GetStringSlice 根据key获取配置信息
func (c *Config) GetStringSlice(key string) []string {
	return cast.ToStringSlice(c.Get(key))
}

// GetIntSlice 根据key获取配置信息
func (c *Config) GetIntSlice(key string) []int {
	return cast.ToIntSlice(c.Get(key))
}

// GetString 根据key获取配置项的值
func (c *Config) GetString(key string) string {
	return cast.ToString(c.Get(key))
}

// GetInt 根据key获取配置项的整数值
func (c *Config) GetInt(key string) int {
	return cast.ToInt(c.Get(key))
}

// GetInt32 根据key获取配置项的整数值
func (c *Config) GetInt32(key string) int32 {
	return cast.ToInt32(c.Get(key))
}

// GetInt64 根据key获取配置项的整数值
func (c *Config) GetInt64(key string) int64 {
	return cast.ToInt64(c.Get(key))
}

// GetUint 根据key获取配置项的无符号整数值
func (c *Config) GetUint(key string) uint {
	return cast.ToUint(c.Get(key))
}

// GetUint32 根据key获取配置项的无符号整数值
func (c *Config) GetUint32(key string) uint32 {
	return cast.ToUint32(c.Get(key))
}

// GetUint64 根据key获取配置项的无符号整数值
func (c *Config) GetUint64(key string) uint64 {
	return cast.ToUint64(c.Get(key))
}

// GetFloat 根据key获取配置项的小数值
// Deprecated
func (c *Config) GetFloat(key string) float64 {
	return cast.ToFloat64(c.Get(key))
}

// GetFloat64 根据key获取配置项的小数值
func (c *Config) GetFloat64(key string) float64 {
	return cast.ToFloat64(c.Get(key))
}

// GetFloat32 根据key获取配置项的小数值
func (c *Config) GetFloat32(key string) float32 {
	return cast.ToFloat32(c.Get(key))
}

// GetBool 根据key获取配置项的布尔值
func (c *Config) GetBool(key string) bool {
	return cast.ToBool(c.Get(key))
}

// SubAndUnmarshal 根据key提取子树并反序列化
func (c *Config) SubAndUnmarshal(key string, i interface{}) error {
	if c.configType == "properties" {
		return json.Unmarshal([]byte(c.GetString(key)), &i)
	} else {
		keySub := c.viper.Sub(key)
		if keySub == nil {
			return errors.New(fmt.Sprintf("%s config is not foud", key))
		}
		return keySub.Unmarshal(i)
	}
}
