package db

import (
	"context"

	"github.com/davecgh/go-spew/spew"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"go.uber.org/zap"
)

// RegisterListener creates a connection dedicated to listening
func (s *DB) RegisterListener(channel string, cb func(*pgconn.Notification) error) error {
	conn, err := s.conn.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	return conn.Raw(func(connInt any) error {
		stdconn, ok := connInt.(*stdlib.Conn)
		if !ok {
			zap.S().Error("Bad database connection type, expected pgx stdlib connection")
			return nil
		}
		conn := stdconn.Conn()
		_ = conn
		conn.Config()
		return nil
	})
}

type NotifyListener struct {
	conn *pgx.Conn
}

func (l *NotifyListener) Close(ctx context.Context) error {
	return l.conn.Close(ctx)
}
func (l *NotifyListener) HandleNotification(_ *pgconn.PgConn, n *pgconn.Notification) {
	spew.Dump(n)
}

func NewListener(ctx context.Context, originalConf *pgx.ConnConfig) (*NotifyListener, error) {
	// Add the notification handler
	l := &NotifyListener{}
	config := originalConf.Copy()
	config.OnNotification = l.HandleNotification

	conn, err := pgx.ConnectConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	l.conn = conn
	return &NotifyListener{conn}, nil
}
