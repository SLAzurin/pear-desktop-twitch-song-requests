package appservices

import (
	"context"
	"log"
)

type appService[T any, T2 any] interface {
	Log() *log.Logger
	MsgChan() chan T
	RcvChan() chan T2
	StartCtx(ctx context.Context) error
	Stop() error
}
