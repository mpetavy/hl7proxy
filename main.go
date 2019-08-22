package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"net"
	"sync"
	"time"
)

var (
	source      *string
	dest        *string
	listener    *net.TCPListener
	commCon     *net.TCPConn
	forumCon    *net.TCPConn
	readTimeout *int64
)

const (
	proxyToForum = "Proxy->Forum"
	commToProxy  = "Comm->Proxy"
)

func init() {
	source = flag.String("s", "", "server socket host address")
	dest = flag.String("d", "", "destination socket host address")
	readTimeout = flag.Int64("rt", int64(100), "destination socket read timeout in ms")
}

func copier(name string, wg *sync.WaitGroup, ctx *context.Context, cancel *context.CancelFunc, dest *net.TCPConn, source *net.TCPConn) {

	var c int64
	serr := ""
	b := make([]byte, 8192)

	defer func() {
		common.Debug("%s quit, copied %d bytes", name, c)
		wg.Done()
	}()

	for {
		select {
		case <-(*ctx).Done():
			return
		default:
		}

		source.SetReadDeadline(time.Now().Add(time.Duration(*readTimeout) * time.Millisecond))

		r, err := source.Read(b)

		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				continue
			}

			serr = err.Error()
		}

		if r > 0 {
			s := 0
			for s < r {
				w, err := dest.Write(b[s:r])
				if err != nil {
					serr = err.Error()
					break
				}

				s = s + w
				c = c + int64(w)
			}
		}

		if serr != "" {
			common.Debug("%s cancels! err: %s", name, serr)
			(*cancel)()
			return
		}
	}
}

func startProxy() error {
	if listener == nil {
		common.DebugFunc()

		common.Debug("resolve %s", *source)
		addr, err := net.ResolveTCPAddr("tcp", *source)
		if err != nil {
			return err
		}

		common.Debug("listenTCP %s ...", addr)
		listener, err = net.ListenTCP("tcp", addr)
		if err != nil {
			return err
		}
	}

	return nil
}

func stopProxy() error {
	var err error

	if listener != nil {
		common.DebugFunc()

		err = listener.Close()

		listener = nil
	}

	return err
}

func startForumConnection() error {
	if forumCon == nil {
		common.DebugFunc()

		common.Debug("resolve %s", *dest)
		addr, err := net.ResolveTCPAddr("tcp", *dest)
		if err != nil {
			return err
		}

		common.Debug("dialTCP %s ...", addr)
		forumCon, err = net.DialTCP("tcp", nil, addr)
		if err != nil {
			return err
		}

		common.Debug("dialTCP connection established")
	}

	return nil
}

func stopForumConnection() error {
	var err error

	if forumCon != nil {
		common.DebugFunc()

		err = forumCon.Close()

		forumCon = nil
	}

	return err
}

func start() error {
	err := startProxy()
	if err != nil {
		return err
	}

	go func() {
		for {
			err := startProxy()
			if err != nil {
				stopProxy()

				continue
			}

			err = startForumConnection()
			if err != nil {
				common.Warn(fmt.Sprintf("connection to client %s not possible, retry on next request ...", *dest))
			}

			common.Debug("listener.AcceptTCP() ...")
			commCon, err = listener.AcceptTCP()
			if err != nil {
				if listener != nil {
					common.Error(err)
				}

				break
			}

			common.Debug("listener.AcceptTCP() from %s", commCon.RemoteAddr())

			err = startForumConnection()
			if err != nil {
				common.Error(fmt.Errorf("connection to client %s not possible, reset server connection", *dest))

				stopProxy()

				continue
			}

			common.Debug("Start data transfer")

			ctx, cancel := context.WithCancel(context.Background())
			wg := sync.WaitGroup{}
			wg.Add(2)

			go copier(commToProxy, &wg, &ctx, &cancel, forumCon, commCon)
			go copier(proxyToForum, &wg, &ctx, &cancel, commCon, forumCon)

			wg.Wait()

			common.Debug("Stop data transfer")

			stopForumConnection()
		}
	}()

	return nil
}

func stop() error {
	return stopProxy()
}

func main() {
	defer common.Cleanup()

	common.New(&common.App{"hl7proxy", "1.0.3", "2018", "Persistent connection proxy", "mpetavy", common.APACHE, "https://github.com/mpetavy/hl7proxy", true, start, stop, nil, 0}, []string{"s", "d"})
	common.Run()
}
