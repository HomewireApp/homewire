package context

import (
	"context"
	"log"

	"github.com/HomewireApp/homewire/internal/database"
	"github.com/HomewireApp/homewire/logger"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
)

type Context struct {
	Context context.Context
	Host    host.Host
	DB      *database.Database
	Logger  logger.Logger
	Pubsub  *pubsub.PubSub
}

func (ctx *Context) Destroy() {
	if err := ctx.Host.Close(); err != nil {
		ctx.Logger.Warn("[homewire:context] Failed to dispose host connection %v", err)
	}

	if err := ctx.DB.Close(); err != nil {
		ctx.Logger.Warn("[homewire:context] Failed to close database connection %v", err)
	}

	if err := ctx.Logger.Close(); err != nil {
		log.Printf("Failed to close logger due to %v", err)
	}
}
