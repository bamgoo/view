package view

import (
	"github.com/infrago/base"
	"github.com/infrago/infra"
)

func (m *Module) Ready() bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.started && m.connected && m.instance != nil && m.instance.conn != nil
}

func (m *Module) Health() infra.ModuleHealth {
	m.mutex.Lock()
	ready := m.started && m.connected && m.instance != nil && m.instance.conn != nil
	connected := m.instance != nil && m.instance.conn != nil
	helpers := len(m.helpers)
	var conn Connection
	if connected {
		conn = m.instance.conn
	}
	m.mutex.Unlock()

	workload := int64(0)
	var err error
	if conn != nil {
		var health Health
		health, err = conn.Health()
		workload = health.Workload
		if err != nil {
			ready = false
		}
	}
	return infra.NewModuleHealth("view", ready, err, base.Map{
		"connected": connected,
		"helpers":   helpers,
		"workload":  workload,
	})
}

func (m *Module) Stats() infra.ModuleStats {
	m.mutex.Lock()
	ready := m.started && m.connected && m.instance != nil && m.instance.conn != nil
	connected := m.instance != nil && m.instance.conn != nil
	helpers := len(m.helpers)
	m.mutex.Unlock()
	return infra.NewModuleStats("view", ready, base.Map{"connected": connected, "helpers": helpers})
}
