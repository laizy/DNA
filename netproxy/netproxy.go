package main

import (
	"DNA/common/log"
	"DNA/net/message"
	"DNA/net/protocol"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type Config struct {
	Port int
}

// PeerChaincodeStream interface for stream between Peer and chaincode instance.
type inProcStream struct {
	recv <-chan []byte
	send chan<- []byte
}

func newInProcStream(recv <-chan []byte, send chan<- []byte) *inProcStream {
	return &inProcStream{recv, send}
}

func (s *inProcStream) Send(msg []byte) error {
	s.send <- msg
	return nil
}

func (s *inProcStream) Recv() ([]byte, error) {
	msg := <-s.recv
	return msg, nil
}

func (s *inProcStream) CloseSend() error {
	return nil
}

const MAXBUFLEN = 1024 * 1024 * 5
const CHANNELBUFLEN = 100

type Peer struct {
	buf  []byte
	conn net.Conn
	recv chan []byte
	send chan []byte
}

func NewPeer(conn net.Conn) *Peer {
	peer := new(Peer)
	peer.buf = make([]byte, 0)
	peer.conn = conn
	peer.recv = make(chan []byte, CHANNELBUFLEN)
	peer.send = make(chan []byte, CHANNELBUFLEN)

	return peer
}

func (self *Peer) Send(msg []byte) error {
	go func() {
		self.send <- msg
	}()

	return nil
}

func (self *Peer) Start() {
	go self.recvpack()
	go self.sendpack()
	go self.loop()
}

func (self *Peer) loop() {
	for {
		pack, isopen := <-self.recv
		if isopen == false {
			log.Info("recv channel closed")
			close(self.send)
			return
		}

		self.handleMsg(pack)
	}
}

func (self *Peer) sendpack() error {
	for {
		buf, isopen := <-self.send
		if isopen {
			return nil
		}

		for len(buf) > 0 {
			n, err := self.conn.Write(buf)
			if err != nil {
				fmt.Println("send error")
				return err
			}
			buf = buf[n:]
		}
	}
}

func (self *Peer) recvpack() error {
	conn := self.conn
	defer conn.Close()
	defer close(self.recv)

	log.Info(conn.RemoteAddr().String(), "receive data string")

	buf := make([]byte, MAXBUFLEN)
	for {
		len, err := conn.Read(buf[0:(MAXBUFLEN - 1)])
		buf[MAXBUFLEN-1] = 0 //Prevent overflow

		self.depack(buf[0:len])

		if err != nil {
			fmt.Println("Read connection error ", err)
			return err
		}
	}

	return nil
}

func (self *Peer) depack(buf []byte) {

	MSGHDRLEN := protocol.MSGHDRLEN

	hlen := len(self.buf)
	blen := len(buf)
	tlen := hlen + blen

	for blen > 0 {

		if tlen < MSGHDRLEN {
			self.buf = append(self.buf, buf...)
			return
		}

		//  guarentee msgHdr is in self.buf
		if hlen < MSGHDRLEN {
			self.buf = append(self.buf, buf[0:MSGHDRLEN-hlen]...)
			buf = buf[MSGHDRLEN-hlen:]
			hlen = MSGHDRLEN
			blen = len(buf)
		}

		packLen := message.PayloadLen(self.buf) + MSGHDRLEN

		if tlen < packLen {
			self.buf = append(self.buf, buf...)
			return
		}

		pack := append(self.buf, buf[0:packLen-hlen]...)
		// self.Send(pack)
		self.recv <- pack

		self.buf = make([]byte, 0)
		buf = buf[packLen-hlen:]

		hlen = 0
		blen -= (packLen - hlen)
		tlen -= packLen
	}

}

//*/

func main() {
	var path string = "./Log/"
	log.CreatePrintLog(path)

	var config Config
	config.Port = 9090

	listener, err := net.Listen("tcp", ":"+strconv.Itoa(config.Port))
	if err != nil {
		fmt.Println("Error net listen")
		return
	}
	peers := make([]*Peer, 0)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error net listen")
			// handle error
			continue
		}
		log.Info(conn.RemoteAddr().String(), " tcp connect success")
		peer := NewPeer(conn)
		peers = append(peers, peer)
		fmt.Println("peers len", len(peers))
		go peer.Start()
	}

}

// FIXME the length exceed int32 case?
func (self *Peer) handleMsg(buf []byte) error {

	str := hex.EncodeToString(buf)
	fmt.Println("Received data len: ", len(buf), "\n", str)

	s, err := message.MsgType(buf)
	if err != nil {
		fmt.Println("Message type parsing error")
		return err
	}

	msg := message.AllocMsg(s, len(buf))
	if msg == nil {
		fmt.Println(fmt.Sprintf("Allocation message %s failed", s))
		return errors.New("Allocation message failed")
	}
	// Todo attach a ndoe pointer to each message
	// Todo drop the message when verify/deseria packet error
	msg.Deserialization(buf)
	msg.Verify(buf[protocol.MSGHDRLEN:])

	switch t, _ := message.MsgType(buf); t {
	case "version":
		vs := message.NewMessageVersion()
		vs.Deserialization(buf)
		fmt.Println("version message:", vs.P)

	}

	fmt.Println("meessage", msg)

	return nil
	// return msg.Handle(node)
}
