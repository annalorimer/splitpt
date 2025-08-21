/*
Package splitpt_client implements the functionality necessary for a client to establish a connection with
a server using splitpt.
*/
package splitpt_client

import (
	split "anticensorshiptrafficsplitting/splitpt/common/split"
	tt "anticensorshiptrafficsplitting/splitpt/common/turbotunnel"
	"errors"
	"log"
	"net"
	"time"

	"github.com/txthinking/socks5"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
)

type dummyAddr struct{}

func (addr dummyAddr) Network() string { return "dummy" }
func (addr dummyAddr) String() string  { return "dummy" }

const (
	HOST = "217.162.72.192"
	PORT = "8888"
	TYPE = "tcp"
)

type SplitPTClient struct {
	SplitPTConfig
}

func NewSplitPTClient(config SplitPTConfig) (SplitPTClient, error) {
	return SplitPTClient{SplitPTConfig: config}, nil
}

func (t *SplitPTClient) Dial() (*smux.Stream, error) {
	log.Printf("Dialing")
	var cleanup []func()
	defer func() {
		for i := len(cleanup) - 1; i >= 0; i-- {
			cleanup[i]()
		}
	}()

	log.Printf("Starting new session")

	connList, err := t.GetPTConnections()
	if err != nil {
		log.Printf("Error connecting to pts: %s", err.Error())
		return nil, err
	}

	log.Printf("Setting up turbotunnel")
	// TurboTunnel
	sessionID := tt.NewSessionID()
	// TODO make this the correct type of redial packat conn for the splitting algorithm
	/*var pconn
	switch sptConfig.SplittingAlg {
		case "round-robin":
			pconn = split.NewRoundRobinPacketConn(sessionID, ptconn)
	}*/
	log.Printf("Getting splitting packet conn")

	var pconn split.SplittingPacketConn
	switch t.SplittingAlg {
	case "round-robin":
		pconn = split.NewRoundRobinPacketConn(sessionID, connList, dummyAddr{})
	case "random":
		pconn = split.NewRandomPacketConn(sessionID, connList, dummyAddr{})

	}
	log.Printf("Got splitting packet conn")
	conn, err := kcp.NewConn2(pconn.RemoteAddr(), nil, 0, 0, pconn)
	if err != nil {
		return nil, err
	}
	log.Printf("SessionID: %v", sessionID)

	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 2
	smuxConfig.KeepAliveTimeout = 1 * time.Minute
	smuxConfig.MaxReceiveBuffer = 4 * 1024 * 1024 // default is 4 * 1024 * 1024
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024  // default is 65536
	sess, err := smux.Client(conn, smuxConfig)
	if err != nil {
		return nil, err
	}

	stream, err := sess.OpenStream()
	if err != nil {
		return nil, err
	}
	log.Printf("Finished dialing")
	return stream, nil
	// AHL simple tcp server for now
	//tcpServer, err := net.ResolveTCPAddr(TYPE, HOST+":"+PORT)
	//if err != nil {
	//	os.Exit(1)
	//}
	//return net.DialTCP(TYPE, nil, tcpServer)

}

func (t *SplitPTClient) GetPTConnections() ([]net.Conn, error) {
	log.Printf("Launching PT connections")
	var connList []net.Conn
	for _, conn := range t.Connections["connections"] {
		var client *socks5.Client
		//log.Printf("conn.Transport: %s", conn.Transport)
		tcpaddr, err := net.ResolveTCPAddr("tcp", conn.Bridge)
		if err != nil {
			log.Printf("Error resolving TCP address: %s", err.Error())
			return nil, err
		}
		if conn.Transport == "obfs4" || conn.Transport == "webtunnel" {
			log.Printf("Launching Lyrebird connection for %s", conn.Transport)
			client, err = LyrebirdConnect(conn.BinPath, conn.Transport, conn.Args, conn.Cert)
			if err != nil {
				log.Printf("Error connecting to lyrebird: %s", err.Error())
				return nil, err
			}
		} else {
			err := errors.New("Unrecognized PT")
			return nil, err
		}
		ptconn, err := client.DialWithLocalAddr("tcp", "", conn.Bridge, tcpaddr)
		if err != nil {
			log.Printf("Error dialing: %s", err.Error())
			return nil, err
		}
		connList = append(connList, ptconn)
	}
	log.Printf("Connections launched: %v", len(connList))
	return connList, nil
}
