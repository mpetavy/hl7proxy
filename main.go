package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"io"
	"net"
	"os"
	"time"
)

var (
	source        *string
	sourceEncoder *string
	dest          *string
	destEncoder   *string
	file          *string
	listener      *net.TCPListener
	emrCon        *net.TCPConn
	forumCon      *net.TCPConn
	hl7filter     *bool
)

const (
	proxyToForum = "Proxy    ->Forum"
	emrToProxy   = "Emr->Proxy"
)

//go:embed go.mod
var resources embed.FS

func init() {
	common.Init("", "", "", "", "HL7 connection proxy", "", "", "", &resources, start, stop, nil, 0)

	source = flag.String("s", "", "proxy host address:port (:5000)")
	sourceEncoder = flag.String("senc", "", "encoder to convert incoming HL7 messages")
	dest = flag.String("d", "", "destination host address (forumserver:7000)")
	destEncoder = flag.String("denc", "", "encoder to convert outgoing HL7 messages")
	file = flag.String("f", "", "filename to log all transferred HL7 data of stream")
	hl7filter = flag.Bool("hl7", true, "trim data to valid HL7 message blocks in MLLP")
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
		defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

		for common.AppLifecycle().IsSet() {
			err := startProxy()
			if err != nil {
				common.Error(stopProxy())

				continue
			}

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

				common.Error(stopProxy())

				continue
			}

			common.Debug("Start data transfer")

			var f *os.File
			var teeReader io.Reader
			var emrReader io.Reader
			var forumReader io.Reader

			emrReader = emrCon
			forumReader = forumCon

			if *hl7filter {
				emrReader = NewHL7Filter(emrReader, *sourceEncoder)
				forumReader = NewHL7Filter(forumReader, *destEncoder)
			}

			if *file != "" {
				common.Debug("open file %s ...", *file)
				f, err = os.OpenFile(*file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, common.DefaultFileMode)
				if err != nil {
					common.Error(err)
				}

				teeReader = io.TeeReader(emrReader, f)
			} else {
				teeReader = emrReader
			}

			ctxDelayer, cancelDelayer := context.WithCancel(context.Background())
			ctxConnection, cancelConnection := context.WithCancel(context.Background())

			defer func() {
				cancelDelayer()
				cancelConnection()
			}()

			go func() {
				defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

				_, err := common.CopyBuffer(ctxDelayer, cancelDelayer, emrToProxy, forumCon, teeReader, -1)
				common.Error(err)
			}()
			go func() {
				defer common.UnregisterGoRoutine(common.RegisterGoRoutine(1))

				_, err := common.CopyBuffer(ctxDelayer, cancelDelayer, proxyToForum, emrCon, forumCon, -1)
				common.Error(err)
			}()

			outsideDelay := common.NewNotice(true, nil)

		Delayer:
			for {
				select {
				case <-ctxDelayer.Done():
					if outsideDelay.IsSet() {
						outsideDelay.Unset()

						common.Debug("Delayer received Done()")
						common.Sleep(time.Second)

						cancelConnection()
					}
				case <-ctxConnection.Done():
					if f != nil {
						common.Error(f.Close())
					}

					common.Error(stopForumConnection())
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
	common.Run([]string{"s", "d"})
}
