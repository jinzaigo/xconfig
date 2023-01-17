package xconfig

type option struct {
	configFile string
	configType string
}

type OptionFunc func(*option)

func WithFile(configFile string) OptionFunc {
	return func(o *option) {
		o.configFile = configFile
	}
}

func WithConfigType(confType string) OptionFunc {
	return func(o *option) {
		o.configType = confType
	}
}

