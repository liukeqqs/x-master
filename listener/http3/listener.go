package http3

import (
	"net"

	"github.com/go-gost/gost/v3/pkg/common/metrics"
	"github.com/go-gost/gost/v3/pkg/listener"
	"github.com/go-gost/gost/v3/pkg/logger"
	md "github.com/go-gost/gost/v3/pkg/metadata"
	"github.com/go-gost/gost/v3/pkg/registry"
	pht_util "github.com/go-gost/x/internal/util/pht"
	"github.com/lucas-clemente/quic-go"
)

func init() {
	registry.ListenerRegistry().Register("http3", NewListener)
	registry.ListenerRegistry().Register("h3", NewListener)
}

type http3Listener struct {
	addr    net.Addr
	server  *pht_util.Server
	logger  logger.Logger
	md      metadata
	options listener.Options
}

func NewListener(opts ...listener.Option) listener.Listener {
	options := listener.Options{}
	for _, opt := range opts {
		opt(&options)
	}
	return &http3Listener{
		logger:  options.Logger,
		options: options,
	}
}

func (l *http3Listener) Init(md md.Metadata) (err error) {
	if err = l.parseMetadata(md); err != nil {
		return
	}

	l.addr, err = net.ResolveUDPAddr("udp", l.options.Addr)
	if err != nil {
		return
	}

	l.server = pht_util.NewHTTP3Server(
		l.options.Addr,
		&quic.Config{},
		pht_util.TLSConfigServerOption(l.options.TLSConfig),
		pht_util.BacklogServerOption(l.md.backlog),
		pht_util.PathServerOption(l.md.authorizePath, l.md.pushPath, l.md.pullPath),
		pht_util.LoggerServerOption(l.options.Logger),
	)

	go func() {
		if err := l.server.ListenAndServe(); err != nil {
			l.logger.Error(err)
		}
	}()

	return
}

func (l *http3Listener) Accept() (conn net.Conn, err error) {
	conn, err = l.server.Accept()
	if err != nil {
		return
	}
	return metrics.WrapConn(l.options.Service, conn), nil
}

func (l *http3Listener) Addr() net.Addr {
	return l.addr
}

func (l *http3Listener) Close() (err error) {
	return l.server.Close()
}
