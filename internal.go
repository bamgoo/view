package view

import . "github.com/bamgoo/base"

func (m *Module) Parse(body Body) (string, error) {
	if m.instance == nil || m.instance.conn == nil {
		return "", ErrInvalidConnection
	}

	if body.Helpers == nil {
		body.Helpers = Map{}
	}
	for key, helper := range m.helperActions {
		if _, ok := body.Helpers[key]; !ok {
			body.Helpers[key] = helper
		}
	}

	return m.instance.conn.Parse(body)
}
