package remote

import (
	"bytes"
	"errors"
	crypt "github.com/bketelsen/crypt/config"
	"github.com/shima-park/agollo"
	"github.com/spf13/viper"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrUnsupportedProvider = errors.New("This configuration manager is not supported")

	_ viperConfigManager = apolloConfigManager{}
	// getConfigManager方法每次返回新对象导致缓存无效，
	// 这里通过endpoint作为key复一个对象
	// key: endpoint+appid value: agollo.Agollo
	agolloMap sync.Map

	//存储多个appId
	providers sync.Map
)

var (
	// 默认为properties，apollo默认配置文件格式
	defaultConfigType = "properties"
	// 默认创建Agollo的Option
	defaultAgolloOptions = []agollo.Option{
		agollo.AutoFetchOnCacheMiss(),
		agollo.FailTolerantOnBackupExists(),
	}
)

type viperConfigManager interface {
	Get(key string) ([]byte, error)
	Watch(key string, stop chan bool) <-chan *viper.RemoteResponse
}

type apolloConfigManager struct {
	agollo agollo.Agollo
}

func newApolloConfigManager(rp viper.RemoteProvider) (*apolloConfigManager, error) {
	//读取provider相关配置
	providerConf, ok := providers.Load(rp.Provider())
	if !ok {
		return nil, ErrUnsupportedProvider
	}

	p := providerConf.(conf)

	if p.appId == "" {
		return nil, errors.New("The appid is not set")
	}

	opts := defaultAgolloOptions
	if len(p.opts) > 0 {
		opts = p.opts
	}

	ag, err := newAgollo(p.appId, rp.Endpoint(), opts)
	if err != nil {
		return nil, err
	}

	return &apolloConfigManager{
		agollo: ag,
	}, nil

}

func newAgollo(appid, endpoint string, opts []agollo.Option) (agollo.Agollo, error) {
	i, found := agolloMap.Load(endpoint + "/" + appid)
	if !found {
		ag, err := agollo.New(
			endpoint,
			appid,
			opts...,
		)
		if err != nil {
			return nil, err
		}

		// 监听并同步apollo配置
		ag.Start()

		agolloMap.Store(endpoint+"/"+appid, ag)

		return ag, nil
	}
	return i.(agollo.Agollo), nil
}

func (cm apolloConfigManager) Get(namespace string) ([]byte, error) {
	configs := cm.agollo.GetNameSpace(namespace)
	return marshalConfigs(getConfigType(namespace), configs)
}

func marshalConfigs(configType string, configs map[string]interface{}) ([]byte, error) {
	var bts []byte
	var err error
	switch configType {
	case "json", "yml", "yaml", "xml":
		content := configs["content"]
		if content != nil {
			bts = []byte(content.(string))
		}
	case "properties":
		bts, err = marshalProperties(configs)
	}
	return bts, err
}

func (cm apolloConfigManager) Watch(namespace string, stop chan bool) <-chan *viper.RemoteResponse {
	resp := make(chan *viper.RemoteResponse)
	backendResp := cm.agollo.WatchNamespace(namespace, stop)
	go func() {
		for {
			select {
			case <-stop:
				return
			case r := <-backendResp:
				if r.Error != nil {
					resp <- &viper.RemoteResponse{
						Value: nil,
						Error: r.Error,
					}
					continue
				}

				configType := getConfigType(namespace)
				value, err := marshalConfigs(configType, r.NewValue)

				resp <- &viper.RemoteResponse{Value: value, Error: err}
			}
		}
	}()
	return resp
}

type configProvider struct {
}

func (rc configProvider) Get(rp viper.RemoteProvider) (io.Reader, error) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}

	var b []byte
	switch cm := cmt.(type) {
	case viperConfigManager:
		b, err = cm.Get(rp.Path())
	case crypt.ConfigManager:
		b, err = cm.Get(rp.Path())
	}

	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

func (rc configProvider) Watch(rp viper.RemoteProvider) (io.Reader, error) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, err
	}

	var resp []byte
	switch cm := cmt.(type) {
	case viperConfigManager:
		resp, err = cm.Get(rp.Path())
	case crypt.ConfigManager:
		resp, err = cm.Get(rp.Path())
	}

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(resp), nil
}

func (rc configProvider) WatchChannel(rp viper.RemoteProvider) (<-chan *viper.RemoteResponse, chan bool) {
	cmt, err := getConfigManager(rp)
	if err != nil {
		return nil, nil
	}

	switch cm := cmt.(type) {
	case viperConfigManager:
		quitwc := make(chan bool)
		viperResponsCh := cm.Watch(rp.Path(), quitwc)
		return viperResponsCh, quitwc
	default:
		ccm := cm.(crypt.ConfigManager)
		quit := make(chan bool)
		quitwc := make(chan bool)
		viperResponsCh := make(chan *viper.RemoteResponse)
		cryptoResponseCh := ccm.Watch(rp.Path(), quit)
		// need this function to convert the Channel response form crypt.Response to viper.Response
		go func(cr <-chan *crypt.Response, vr chan<- *viper.RemoteResponse, quitwc <-chan bool, quit chan<- bool) {
			for {
				select {
				case <-quitwc:
					quit <- true
					return
				case resp := <-cr:
					vr <- &viper.RemoteResponse{
						Error: resp.Error,
						Value: resp.Value,
					}

				}

			}
		}(cryptoResponseCh, viperResponsCh, quitwc, quit)

		return viperResponsCh, quitwc
	}
}

func getConfigManager(rp viper.RemoteProvider) (interface{}, error) {
	provider := rp.Provider()
	if strings.Index(rp.Provider(), "apollo:") != -1 {
		provider = "apollo"
	}

	if rp.SecretKeyring() != "" {
		kr, err := os.Open(rp.SecretKeyring())
		if err != nil {
			return nil, err
		}
		defer kr.Close()

		switch provider {
		case "etcd":
			return crypt.NewEtcdConfigManager([]string{rp.Endpoint()}, kr)
		case "consul":
			return crypt.NewConsulConfigManager([]string{rp.Endpoint()}, kr)
		case "apollo":
			return nil, errors.New("The Apollo configuration manager is not encrypted")
		default:
			return nil, ErrUnsupportedProvider
		}
	} else {
		switch provider {
		case "etcd":
			return crypt.NewStandardEtcdConfigManager([]string{rp.Endpoint()})
		case "consul":
			return crypt.NewStandardConsulConfigManager([]string{rp.Endpoint()})
		case "apollo":
			return newApolloConfigManager(rp)
		default:
			return nil, ErrUnsupportedProvider
		}
	}
}

// 配置文件有多种格式，例如：properties、xml、yml、yaml、json等。同样Namespace也具有这些格式。在Portal UI中可以看到“application”的Namespace上有一个“properties”标签，表明“application”是properties格式的。
// 如果使用Http接口直接调用时，对应的namespace参数需要传入namespace的名字加上后缀名，如datasources.json。
func getConfigType(namespace string) string {
	ext := filepath.Ext(namespace)

	if len(ext) > 1 {
		fileExt := ext[1:]
		// 还是要判断一下碰到，TEST.Namespace1
		// 会把Namespace1作为文件扩展名
		for _, e := range viper.SupportedExts {
			if e == fileExt {
				return fileExt
			}
		}
	}

	return defaultConfigType
}

func init() {
	//这里append废弃，因为这么写只能支持一个appId
	//viper.SupportedRemoteProviders = append(
	//	viper.SupportedRemoteProviders,
	//	"apollo",
	//)
	viper.RemoteConfig = &configProvider{} //目的：重写viper.RemoteConfig的相关方法
}

type conf struct {
	appId string
	opts  []agollo.Option
}

//【重要】这里是实现支持多个appId的核心操作
func AddProviders(appId string, opts ...agollo.Option) string {
	provider := "apollo:" + appId
	_, loaded := providers.LoadOrStore(provider, conf{
		appId: appId,
		opts:  opts,
	})

	//之前未存储过，则向viper新增一个provider，让viper认识这个新提供器
	if !loaded {
		viper.SupportedRemoteProviders = append(
			viper.SupportedRemoteProviders,
			provider,
		)
	}

	return provider
}
