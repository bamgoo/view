package view

import (
	"fmt"
	"io/fs"
	"sync"
	"time"

	"github.com/infrago/infra"
	. "github.com/infrago/base"
)

func init() {
	infra.Mount(module)
	module.RegisterDriver(infra.DEFAULT, &defaultDriver{})
}

var module = &Module{
	config: Config{
		Driver: infra.DEFAULT,
		Root:   "asset/views",
		Shared: "shared",
		Left:   "{%",
		Right:  "%}",
	},
	drivers: make(map[string]Driver, 0),
	helpers: make(map[string]Helper, 0),
}

func SetFS(fsys fs.FS) {
	infra.AssetFS(fsys)
}

type (
	Module struct {
		mutex sync.Mutex

		initialized bool
		connected   bool
		started     bool

		drivers map[string]Driver
		helpers map[string]Helper

		helperActions Map
		config        Config
		instance      *Instance
	}

	Config struct {
		Driver  string
		Root    string
		Shared  string
		Left    string
		Right   string
		Setting Map
	}

	Body struct {
		View     string
		Site     string
		Language string
		Timezone *time.Location
		Data     Map
		Model    Map
		Helpers  Map
	}

	Instance struct {
		conn    Connection
		Config  Config
		Setting Map
	}
)

func (m *Module) Register(name string, value Any) {
	switch v := value.(type) {
	case Driver:
		m.RegisterDriver(name, v)
	case Helper:
		m.RegisterHelper(name, v)
	}
}

func (m *Module) RegisterDriver(name string, driver Driver) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if driver == nil {
		panic("Invalid view driver: " + name)
	}
	if name == "" {
		name = infra.DEFAULT
	}

	if infra.Override() {
		m.drivers[name] = driver
	} else {
		if _, ok := m.drivers[name]; !ok {
			m.drivers[name] = driver
		}
	}
}

func (m *Module) RegisterHelper(name string, helper Helper) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	aliases := make([]string, 0)
	if name != "" {
		aliases = append(aliases, name)
	}
	if helper.Alias != nil {
		aliases = append(aliases, helper.Alias...)
	}

	for _, key := range aliases {
		if infra.Override() {
			m.helpers[key] = helper
		} else {
			if _, ok := m.helpers[key]; !ok {
				m.helpers[key] = helper
			}
		}
	}
}

func (m *Module) RegisterConfig(config Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.initialized || m.connected || m.started {
		return
	}
	m.config = config
}

func (m *Module) Config(global Map) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.initialized || m.connected || m.started {
		return
	}

	cfgAny, ok := global["view"]
	if !ok {
		return
	}
	cfg, ok := cfgAny.(Map)
	if !ok || cfg == nil {
		return
	}

	if v, ok := cfg["driver"].(string); ok && v != "" {
		m.config.Driver = v
	}
	if v, ok := cfg["root"].(string); ok {
		m.config.Root = v
	}
	if v, ok := cfg["shared"].(string); ok {
		m.config.Shared = v
	}
	if v, ok := cfg["left"].(string); ok {
		m.config.Left = v
	}
	if v, ok := cfg["right"].(string); ok {
		m.config.Right = v
	}
	if v, ok := cfg["setting"].(Map); ok {
		m.config.Setting = v
	}
}

func (m *Module) Setup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.initialized {
		return
	}

	if m.config.Driver == "" {
		m.config.Driver = infra.DEFAULT
	}
	if m.config.Root == "" {
		m.config.Root = "asset/views"
	}
	if m.config.Shared == "" {
		m.config.Shared = "shared"
	}
	if m.config.Left == "" {
		m.config.Left = "{%"
	}
	if m.config.Right == "" {
		m.config.Right = "%}"
	}

	m.helperActions = Map{}
	for key, helper := range m.helpers {
		if helper.Action != nil {
			m.helperActions[key] = helper.Action
		}
	}

	m.initialized = true
}

func (m *Module) Open() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.connected {
		return
	}

	driver, ok := m.drivers[m.config.Driver]
	if !ok || driver == nil {
		panic("Invalid view driver: " + m.config.Driver)
	}

	inst := &Instance{Config: m.config, Setting: m.config.Setting}
	conn, err := driver.Connect(inst)
	if err != nil {
		panic("Failed to connect to view: " + err.Error())
	}
	if err := conn.Open(); err != nil {
		panic("Failed to open view: " + err.Error())
	}

	inst.conn = conn
	m.instance = inst
	m.connected = true
}

func (m *Module) Start() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.started {
		return
	}
	m.started = true
	connCount := 0
	if m.instance != nil && m.instance.conn != nil {
		connCount = 1
	}
	fmt.Printf("infrago view module is running with %d connections, %d helpers.\n", connCount, len(m.helpers))
}

func (m *Module) Stop() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if !m.started {
		return
	}
	m.started = false
}

func (m *Module) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.instance != nil && m.instance.conn != nil {
		_ = m.instance.conn.Close()
	}

	m.instance = nil
	m.connected = false
	m.initialized = false
}
