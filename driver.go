package view

import (
	. "github.com/infrago/base"
)

type (
	Driver interface {
		Connect(*Instance) (Connection, error)
	}

	Connection interface {
		Open() error
		Health() (Health, error)
		Close() error

		Parse(Body) (string, error)
	}

	Health struct {
		Workload int64
	}

	Helper struct {
		Name   string   `json:"name"`
		Desc   string   `json:"desc"`
		Alias  []string `json:"alias"`
		Action Any      `json:"-"`
	}
)
