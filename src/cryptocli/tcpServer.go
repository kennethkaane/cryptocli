package main

import (
	"sync"
	"time"
	"log"
	"net"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
)

func init() {
	MODULELIST.Register("tcp-server", "Listens TCP and wait for a single connection to complete", NewTCPServer)
}

type TCPServer struct {
	addr string
	connectTimeout time.Duration
	readTimeout time.Duration
}

type TCPServerRelayer struct {
	Callback MessageChannelFunc
	MessageChannel *MessageChannel
	Wg *sync.WaitGroup
}

func tcpServerHandler(conn net.Conn, m *TCPServer, relay *TCPServerRelayer) {
	mc, cb, wg := relay.MessageChannel, relay.Callback, relay.Wg
	defer wg.Done()
	defer conn.Close()

	mc.Start(map[string]interface{}{
		"local-addr": conn.RemoteAddr().String(),
		"remote-addr": conn.RemoteAddr().String(),
		"addr": m.addr,
	})

	_, inc := cb()
	outc := mc.Channel
	defer close(outc)

	log.Printf("Client %q is connected\n", conn.LocalAddr().String())
	go func(conn net.Conn, inc chan []byte) {
		defer conn.Close()

		for payload := range inc {
			_, err := conn.Write(payload)
			if err != nil {
				err = errors.Wrap(err, "Error writing to tcp connection")
				log.Println(err.Error())
				break
			}
		}

		DrainChannel(inc, nil)
	}(conn, inc)

	defer conn.Close()
	conn.SetReadDeadline(time.Now().Add(m.readTimeout))

	err := ReadBytesStep(conn, func(payload []byte) bool {
		outc <- payload
		conn.SetReadDeadline(time.Now().Add(m.readTimeout))

		return true
	})
	if err != nil {
		err = errors.Wrap(err, "Error reading from tcp socket")
		log.Println(err.Error())
		return
	}
}

func tcpServerServe(conn net.Conn, m *TCPServer, relayer chan *TCPServerRelayer, connc, donec, cancel chan struct{}) {
	donec <- struct{}{}
	defer func(donec chan struct{}) {
		<- donec
	}(donec)

	select {
		case relay, opened := <- relayer:
			if ! opened {
				return
			}

			tcpServerHandler(conn, m, relay)
			return
		case <- cancel:
			return
		default:
	}

	select {
		case connc <- struct{}{}:
		case <- cancel:
			return
	}

	select {
		case relay, opened := <- relayer:
			if ! opened {
				return
			}

			tcpServerHandler(conn, m, relay)
			return
		case <- cancel:
			return
	}
}

func (m *TCPServer) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.addr, "listen", "", "Listen on addr:port. If port is 0, random port will be assigned")
	fs.DurationVar(&m.connectTimeout, "connect-timeout", 30 * time.Second, "Max amount of time to wait for a potential connection when pipeline is closing")
	fs.DurationVar(&m.readTimeout, "read-timeout", 15 * time.Second, "Amout of time to wait reading from the connection")
}

func NewTCPServer() (Module) {
	return &TCPServer{}
}

func (m *TCPServer) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.readTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--read-timeout")
	}

	if m.connectTimeout < 1 {
		return errors.Errorf("Flag %q cannot be negative or zero", "--connect-timeout")
	}

	addr, err := net.ResolveTCPAddr("tcp", m.addr)
	if err != nil {
		return errors.Wrap(err, "Unable to resolve tcp address")
	}

	var listener net.Listener
	listener, err = net.ListenTCP("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "Unable to listen on tcp address")
	}

	log.Printf("Tcp-server listening on %s\n", listener.Addr().String())

	go func() {
		wg := &sync.WaitGroup{}
		relayer := make(chan *TCPServerRelayer)
		connc := make(chan struct{})
		cancel := make(chan struct{})

		donec := make(chan struct{}, global.MaxConcurrentStreams)

		go func(m *TCPServer, l net.Listener, relayer chan *TCPServerRelayer, connc, done, cancel chan struct{}) {
			for {
				conn, err := l.Accept()
				if err != nil {
					err = errors.Wrap(err, "Error accepting tcp connection")
					log.Println(err.Error())
					return
				}

				go tcpServerServe(conn, m, relayer, connc, donec, cancel)
			}
		}(m, listener, relayer, connc, donec, cancel)

		ticker := time.NewTicker(m.connectTimeout)

		cbs := make([]MessageChannelFunc, 0)
		mcs := make([]*MessageChannel, 0)

		LOOP: for {
			select {
				case <- ticker.C:
					ticker.Stop()
					close(cancel)
					wg.Wait()
					log.Println("Connect timeout reached, nobody connected and no messages from inputs were received")
					out <- &Message{
						Type: MessageTypeTerminate,
					}

					break LOOP
				case _, opened := <- connc:
					if ! opened {
						break LOOP
					}

					mc := NewMessageChannel()

					out <- &Message{
						Type: MessageTypeChannel,
						Interface: mc.Callback,
					}

					if len(cbs) == 0 {
						mcs = append(mcs, mc)
						continue
					}

					wg.Add(1)

					cb := cbs[0]
					cbs = cbs[1:]

					relayer <- &TCPServerRelayer{
						Callback: cb,
						MessageChannel: mc,
						Wg: wg,
					}

					if ! global.MultiStreams {
						close(cancel)
						wg.Wait()
						out <- &Message{Type: MessageTypeTerminate,}
						break LOOP
					}

				case message, opened := <- in:
					ticker.Stop()
					if ! opened {
						close(cancel)
						wg.Wait()
						out <- &Message{
							Type: MessageTypeTerminate,
						}
						break LOOP
					}

					switch message.Type {
						case MessageTypeTerminate:
							close(cancel)
							wg.Wait()
							out <- message
							break LOOP
						case MessageTypeChannel:
							cb, ok := message.Interface.(MessageChannelFunc)
							if ok {
								if len(mcs) == 0 {
									cbs = append(cbs, cb)
									continue
								}

								wg.Add(1)
								mc := mcs[0]
								mcs = mcs[1:]

								relayer <- &TCPServerRelayer{
									Callback: cb,
									MessageChannel: mc,
									Wg: wg,
								}

								if ! global.MultiStreams {
									close(cancel)
									wg.Wait()
									out <- &Message{Type: MessageTypeTerminate,}
									break LOOP
								}
							}
					}
			}
		}

		listener.Close()
		close(connc)

		for _, mc := range mcs {
			close(mc.Channel)
		}

		for _, cb := range cbs {
			_, inc := cb()
			DrainChannel(inc, nil)
		}

		wg.Wait()
		close(relayer)
		close(donec)

		<- in
		close(out)
	}()

	return nil
}
