package node

import (
	"DNA/common/log"
	"DNA/crypto"
	"DNA/net/message"
	"DNA/net/protocol"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Peer struct {
	state     uint32 // node state
	id        uint64 // The nodes's id
	cap       uint32 // The node capability set
	version   uint32 // The network protocol the node used
	services  uint64 // The services the node supplied
	relay     bool   // The relay capability of the node (merge into capbility flag)
	height    uint64 // The node latest block height
	txnCnt    uint64 // The transactions be transmit by this node
	rxTxnCnt  uint64 // The transaction received by this node
	publicKey *crypto.PubKey

	local *node // The pointer to local node
	/*
	 * |--|--|--|--|--|--|isSyncFailed|isSyncHeaders|
	 */
	syncFlag      uint8
	flightHeights []uint32
	lastTime2     time.Time
	lastTime      int64 // UnixNano time

	pendings [2]chan []byte
	quit     chan interface{}
	sendquit chan interface{}
	wg       sync.WaitGroup
	buf      []byte
	conn     net.Conn // The link status and infomation
}

func NewPeer(conn net.Conn) *Peer {
	peer := &Peer{
		conn:     conn,
		quit:     make(chan interface{}),
		sendquit: make(chan interface{}),
	}
	peer.pendings[0] = make(chan []byte, 2000)
	peer.pendings[1] = make(chan []byte, 2000)

	peer.start()

	return peer
}

func (self *Peer) start() {
	peer.UpdateRXTime(time.Now())

	peer.startupRecvWorker()
	peer.startupSendWorker()
}

func (self *Peer) Close() {
	close(self.quit)
	self.wg.Wait()
}

const BUFLEN = 1024 * 1024 * 5

func (self *Peer) UpdateRXTime(t time.Time) {
	self.lastTime2 = t
}

func (self *Peer) setLastTime(t time.Time) {
	atomic.StoreInt64(&self.lastTime, t.UnixNano())
}

func (self *Peer) startupRecvWorker() {
	conn := self.conn

	var buf [BUFLEN]byte
	self.wg.Add(1)
	go func() {
		defer self.wg.Done()
		for {
			select {
			case <-self.quit:
				return
			default:
			}

			len, err := conn.Read(buf[0:(BUFLEN - 1)])
			buf[BUFLEN-1] = 0 //Prevent overflow

			self.depack(buf[0:len])

			switch err {
			case nil:
				t := time.Now()
				self.UpdateRXTime(t)
			case io.EOF:
				log.Error("Rx io.EOF ", err)
				return
			default:
				log.Error("Read connection error ", err)
				return
			}
		}
	}()
}

func (self *Peer) startupSendWorker() {
	self.wg.Add(1)
	go func() {
		defer self.wg.Done()
		defer close(self.sendquit)
		for {
			var stats [2]int
			// block wait for any buf
			var buf []byte
			select {
			case buf = <-self.pendings[0]:
				stats[0]++
			case buf = <-self.pendings[1]:
				stats[1]++
			case <-self.quit:
				self.conn.Close()
				return
			}

			_, err := self.conn.Write(buf)
			if err != nil {
				log.Error("Error sending messge to peer node ", err.Error())
				return
			}

			// handle the rest tasks with priority

			for prior := 0; prior <= 1; {
				select {
				case buf := <-self.pendings[prior]:
					stats[prior]++
					_, err := self.conn.Write(buf)
					if err != nil {
						log.Error("Error sending messge to peer node ", err.Error())
						return
					}
					if stats[prior] > 200 {
						prior += 1
					}
				default:
					prior += 1
				}
			}

			if stats[0] > 5 {
				log.Error("send stats: high=", stats[0], ", low=", stats[1])
			}
		}

	}()

}

func (self *Peer) Send(buf []byte, isprior bool) error {
	index := 1

	if isprior {
		index = 0
	}

	select {
	case self.pendings[index] <- buf:
		return nil
	case <-self.quit:
		return errors.New("connection to peer has been closed")
	case <-self.sendquit:
		return errors.New("send connection to peer has been closed")
	}
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
		// self.recv <- pack
		self.handleMsg(pack)

		self.buf = make([]byte, 0)
		buf = buf[packLen-hlen:]

		hlen = 0
		blen -= (packLen - hlen)
		tlen -= packLen
	}

}

// FIXME the length exceed int32 case?
func (self *Peer) handleMsg(buf []byte) error {

	s, err := message.MsgType(buf)
	if err != nil {
		log.Error("Message type parsing error")
		return err
	}

	msg := message.AllocMsg(s, len(buf))
	if msg == nil {
		log.Error(fmt.Sprintf("Allocation message %s failed", s))
		return errors.New("Allocation message failed")
	}
	// Todo attach a ndoe pointer to each message
	// Todo drop the message when verify/deseria packet error
	msg.Deserialization(buf)
	msg.Verify(buf[protocol.MSGHDRLEN:])

	switch t, _ := message.MsgType(buf); t {
	case "version":
		// vs := message.NewMessageVersion()
		// vs.Deserialization(buf)
		// fmt.Println("version message:", vs.P)
	}

	log.Info("meessage", msg)

	return nil
	// return msg.Handle(node)
}
