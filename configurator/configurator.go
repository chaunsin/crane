package configurator

import (
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// IConfig Configurator interface
type IConfig interface {
	Node() string
	OnChange(viper *viper.Viper)
}

// Configurator configurator, based on viper integration
type Configurator struct {
	// config file path
	path  string
	viper *viper.Viper

	mu    sync.Mutex
	nodes []IConfig

	configChangeInterval time.Time
}

func (config *Configurator) watch() {
	config.viper.OnConfigChange(func(in fsnotify.Event) {
		switch in.Op {
		case fsnotify.Write:
			if time.Since(config.configChangeInterval) < time.Second {
				return
			}
			config.configChangeInterval = time.Now()
			for _, component := range config.nodes {
				component.OnChange(config.viper.Sub(component.Node()))
			}
		}
	})
	config.viper.WatchConfig()
}

// Add add a configuration node, and for each additional node, a top-level node with the same name as the node is required in the configuration file
func (config *Configurator) Add(impl IConfig) {
	config.mu.Lock()
	defer config.mu.Unlock()
	config.nodes = append(config.nodes, impl)
	impl.OnChange(config.viper.Sub(impl.Node()))
}

// NewConfigurator new a configurator
func NewConfigurator(filename string) (*Configurator, error) {
	var ext = filepath.Ext(filename)
	var configuration = new(Configurator)
	file, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	configuration.path = filepath.Dir(file)

	configuration.viper = viper.New()
	configuration.viper.AddConfigPath(configuration.path)
	configuration.viper.SetConfigType(ext[1:])
	info, err := os.Stat(file)
	if err != nil {
		return nil, err
	}
	configuration.viper.SetConfigName(strings.Replace(info.Name(), ext, "", -1))
	err = configuration.viper.ReadInConfig()
	if err != nil {
		return nil, err
	}
	configuration.nodes = make([]IConfig, 0)
	configuration.watch()
	return configuration, nil
}
