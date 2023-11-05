package context

import (
	"context"

	"github.com/HomewireApp/homewire/internal/database"
	"github.com/HomewireApp/homewire/internal/logger"
	"github.com/libp2p/go-libp2p/core/host"
)

type Context struct {
	Context context.Context
	Host    host.Host
	DB      *database.Database
}

func (ctx *Context) Destroy() {
	if err := ctx.Host.Close(); err != nil {
		logger.Warn("[homewire:context] Failed to dispose host connection %v", err)
	}

	if err := ctx.DB.Close(); err != nil {
		logger.Warn("[homewire:context] Failed to close database connection %v", err)
	}
}
