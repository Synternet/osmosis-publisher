package osmosis

import (
	"encoding/json"
	"fmt"
	"github.com/synternet/osmosis-publisher/pkg/types"
	"log"
	"net"
)

func (p *Publisher) initUnixSocket() error {
	var err error
	if p.UnixSocketConn, err = net.Dial("unix", p.SocketAddr()); err != nil {
		return fmt.Errorf("failed to open Unix socket: %w", err)
	}
	return nil
}

func (p *Publisher) closeUnixSocket() {
	if p.UnixSocketConn != nil {
		p.UnixSocketConn.Close()
		p.UnixSocketConn = nil
	}
}

func (p *Publisher) publishToUnixSocket(tx *types.Transaction) {
	if p.UnixSocketConn == nil {
		if err := p.initUnixSocket(); err != nil {
			log.Println("Could not reconnect Unix socket: %v", err)
			return
		}
	}

	data, err := json.Marshal(tx)
	if err != nil {
		log.Println("Error serializing transaction: %v", err)
		return
	}

	if _, err = p.UnixSocketConn.Write(data); err != nil {
		log.Println("Error writing to Unix socket: %v", err)
		p.closeUnixSocket()
		p.initUnixSocket()
	}
}
