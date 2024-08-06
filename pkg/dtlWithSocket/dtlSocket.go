package dtlWithSocket

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/synternet/data-layer-sdk/pkg/options"
	"github.com/synternet/data-layer-sdk/pkg/service"

	"google.golang.org/protobuf/proto"
)

type Service struct {
	*service.Service
	Socket     *net.Conn
	SocketAddr string
}

// NewNatsAndSocketConn initializes a new Service with default settings
func NewNatsAndSocketConn() *Service {
	return &Service{
		Service: &service.Service{},
	}
}

// Close closes both NATS and socket connections
func (n *Service) Close() {
	if n.Service != nil {
		n.Service.Close()
	}
	n.closeUnixSocket()
}

// InitUnixSocket initializes a Unix socket connection
func (n *Service) InitUnixSocket() error {
	// Ensure previous socket connection is closed
	n.closeUnixSocket()

	if n.SocketAddr == "" {
		return fmt.Errorf("socket address is empty")
	}

	conn, err := net.Dial("unix", n.SocketAddr)
	if err != nil {
		return fmt.Errorf("failed to open Unix socket: %w", err)
	}
	n.Socket = &conn

	return nil
}

// closeUnixSocket closes the Unix socket connection
func (n *Service) closeUnixSocket() {
	if n.Socket != nil && *n.Socket != nil {
		(*n.Socket).Close()
		n.Socket = nil
	}
}

// PublishToUnixSocket publishes a message to the Unix socket
func (n *Service) PublishToUnixSocket(msg any) {
	if n.Socket == nil || *n.Socket == nil {
		if err := n.InitUnixSocket(); err != nil {
			log.Printf("Could not initialize Unix socket: %v", err)
			return
		}
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Error serializing message: %v", err)
		return
	}

	// Prefix with the length
	lengthPrefix := fmt.Sprintf("%010d", len(data))
	fullMessage := append([]byte(lengthPrefix), data...)

	if _, err = (*n.Socket).Write(fullMessage); err != nil {
		log.Printf("Error writing to Unix socket: %v", err)
		n.closeUnixSocket()
	}
}

// WithPubSocket sets up a socket connection for publishing
func WithPubSocket(socketAddr string) options.Option {
	return func(o *options.Options) {
		fmt.Println("SOCKET_ADDR")
		o.Params["SocketAddr"] = socketAddr
	}
}

// Publish will sign the message and publish it to a subject constructed from "{prefix}.{name}.{suffixes}".
// Publish will use PubNats connection.
func (n *Service) Publish(msg proto.Message, suffixes ...string) error {
	for _, suffix := range suffixes {
		if suffix == "tx" {
			n.PublishToUnixSocket(msg)
		}
	}
	return n.Service.PublishTo(msg, n.Service.Subject(suffixes...))
}

// Configure overrides the service's Configure method to initialize the socket if specified
func (n *Service) Configure(opts ...options.Option) error {
	if err := n.Service.Configure(opts...); err != nil {
		return err
	}
	if socketAddr, ok := n.Service.Params["SocketAddr"]; ok {
		if addr, ok := socketAddr.(string); ok && addr != "" {
			n.SocketAddr = addr
			if err := n.InitUnixSocket(); err != nil {
				return fmt.Errorf("failed to initialize Unix socket: %w", err)
			}
			n.Service.Logger.Info("Unix socket initialized", "address", addr)
		}
	}

	return nil
}
