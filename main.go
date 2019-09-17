package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"io"
	"net"
	"os"
	"time"
)

var (
	source   *string
	dest     *string
	file     *string
	listener *net.TCPListener
	emrCon   *net.TCPConn
	forumCon *net.TCPConn
)

const (
	proxyToForum = "Proxy->Forum"
	emrToProxy   = "Emr->Proxy"
)

func init() {
	common.Init("hl7proxy", "1.1.0", "2018", "Persistent connection proxy", "mpetavy", common.APACHE, "https://github.com/mpetavy/hl7proxy", true, start, stop, nil, 0)

	source = flag.String("s", "", "proxy host address:port (:5000)")
	dest = flag.String("d", "", "destination host address (forumserver:7000)")
	file = flag.String("f", "", "file to save the network stream")
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

	if *file != "" {
		*file = common.CleanPath(*file)

		err := common.FileDelete(*file)
		if err != nil {
			return err
		}
	}

	go func() {
		for !common.AppStopped() {
			err := startProxy()
			if err != nil {
				common.DebugError(stopProxy())

				continue
			}

			//err = startForumConnection()
			//if err != nil {
			//	common.Warn(fmt.Sprintf("connection to client %s not possible, retry on next request ...", *dest))
			//}

			common.Debug("listener.AcceptTCP() ...")
			emrCon, err = listener.AcceptTCP()
			if err != nil {
				if listener != nil {
					common.Error(err)
				}

				break
			}

			common.Debug("listener.AcceptTCP() from %s", emrCon.RemoteAddr())

			err = startForumConnection()
			if err != nil {
				common.Error(fmt.Errorf("connection to client %s not possible, reset server connection", *dest))

				common.DebugError(stopProxy())

				continue
			}

			common.Debug("Start data transfer")

			var f *os.File
			var teeReader io.Reader

			if *file != "" {
				common.Debug("open file %s ...", *file)
				f, err = os.OpenFile(*file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, os.ModePerm)
				if err != nil {
					common.Error(err)
				}

				teeReader = io.TeeReader(emrCon, f)
			} else {
				teeReader = emrCon
			}

			ctxDelayer, cancelDelayer := context.WithCancel(context.Background())
			ctxConnection, cancelConnection := context.WithCancel(context.Background())

			go common.CopyWithContext(ctxConnection, cancelDelayer, emrToProxy, forumCon, teeReader)
			go common.CopyWithContext(ctxConnection, cancelDelayer, proxyToForum, emrCon, forumCon)

			var inDelay common.Sign

		Delayer:
			for {
				select {
				case <-ctxDelayer.Done():
					if !inDelay.IsSet() {
						inDelay.Set()

						common.Debug("Delayer received Done()")
						common.Debug("Sleep 1 sec ...")
						time.Sleep(time.Second)
						common.Debug("1 sec slept")

						cancelConnection()
					}
				case <-ctxConnection.Done():
					if f != nil {
						common.DebugError(f.Close())
					}

					common.DebugError(stopForumConnection())
					break Delayer
				}
			}
		}
	}()

	return nil
}

func stop() error {
	return stopProxy()
}

func main() {
	defer common.Done()

	common.Run([]string{"s", "d"})
}
