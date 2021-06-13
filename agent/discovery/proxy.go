package discovery

import (
	"context"
	"io"
	"log"
	"net"

	"github.com/james-lawrence/bw/internal/x/errorsx"
	"github.com/james-lawrence/bw/proxy"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type auth interface {
	Authorization([]byte) error
}

func Proxy(l net.Listener, d dialer, a auth) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("proxy failed", err)
			return err
		}

		if err = (upgrader{}).Inbound(context.Background(), a, conn, d); err != nil {
			conn.Close()
			log.Println("failed to accept proxy connection", err)
			continue
		}
	}
}

type upgrader struct{}

func (t upgrader) writemsg(w io.Writer, m proto.Message) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = proto.Marshal(m); err != nil {
		return err
	}

	if _, err = w.Write(proxy.WireformatEncode(encoded)); err != nil {
		return err
	}

	return nil
}

func (t upgrader) readmsg(r io.Reader, m proto.Message) (err error) {
	var (
		encoded []byte
	)

	if encoded, err = proxy.WireformatDecode(r); err != nil {
		return err
	}

	return proto.Unmarshal(encoded, m)
}

func (t upgrader) Inbound(ctx context.Context, a auth, c1 net.Conn, d dialer) (err error) {
	var (
		c2  net.Conn
		req ProxyRequest
	)

	if err = t.readmsg(c1, &req); err != nil {
		return errorsx.Compact(err, t.writemsg(c1, &ProxyResponse{
			Version: 1,
			Code:    ProxyResponse_ClientError,
		}))
	}

	if err = a.Authorization(req.Token); err != nil {
		return errorsx.Compact(err, t.writemsg(c1, &ProxyResponse{
			Version: 1,
			Code:    ProxyResponse_ClientError,
		}))
	}

	if c2, err = d.DialContext(ctx, c1.LocalAddr().Network(), req.Connect); err != nil {
		return errorsx.Compact(err, t.writemsg(c1, &ProxyResponse{
			Version: 1,
			Code:    ProxyResponse_ClientError,
		}))
	}

	err = t.writemsg(c1, &ProxyResponse{
		Version: 1,
		Code:    ProxyResponse_None,
	})
	if err != nil {
		return err
	}

	go proxy.Proxy(ctx, c1, c2, nil)

	return nil
}

func (t upgrader) Outbound(c net.Conn, to string, auth []byte) (err error) {
	var (
		resp ProxyResponse
	)

	req := ProxyRequest{
		Token:   auth,
		Connect: to,
	}

	if err = t.writemsg(c, &req); err != nil {
		return err
	}

	if err = t.readmsg(c, &resp); err != nil {
		return err
	}

	if resp.Code != ProxyResponse_None {
		return errorsx.Compact(errors.Errorf("proxy request failed: %s", resp.Code), c.Close())
	}

	return nil
}

type dialer interface {
	DialContext(ctx context.Context, network string, address string) (net.Conn, error)
}

type signer interface {
	Token() (encoded string, err error)
}

type ProxyDialer struct {
	Proxy  string
	Signer signer
	Dialer dialer
}

func (t ProxyDialer) DialContext(ctx context.Context, network, address string) (conn net.Conn, err error) {
	type handshaker interface {
		Handshake() error
	}

	var (
		token string
	)

	if conn, err = t.Dialer.DialContext(ctx, network, t.Proxy); err != nil {
		return nil, err
	}

	defer func() {
		if err != nil && conn != nil {
			conn.Close()
		}
	}()

	if c, ok := conn.(handshaker); ok {
		if err = c.Handshake(); err != nil {
			return nil, err
		}
	}

	if token, err = t.Signer.Token(); err != nil {
		return nil, err
	}

	if err = (upgrader{}).Outbound(conn, address, []byte(token)); err != nil {
		return nil, errors.Wrap(err, "handshake failed")
	}

	return conn, nil
}
