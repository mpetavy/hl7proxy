package main

import (
	"bytes"
	"fmt"
	"github.com/mpetavy/common"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
)

const (
	SB = 0x0b
	EB = 0x1c
	CR = 0x0d
)

type HL7Filter struct {
	reader  io.Reader
	encoder string

	started     bool
	buffer      bytes.Buffer
	msg         bytes.Buffer
	msgComplete bool
}

func NewHL7Filter(reader io.Reader, encoder string) *HL7Filter {
	return &HL7Filter{reader: reader, encoder: encoder}
}

func (this *HL7Filter) drain(p []byte) int {
	if this.msgComplete {
		l := common.Min(len(p), this.msg.Len())

		ba := this.msg.Next(l)

		copy(p, ba)

		this.msgComplete = this.msg.Len() > 0

		return l
	}

	return 0
}

func (this *HL7Filter) encode(ba []byte) []byte {
	srcFile, err := common.CreateTempFile()
	common.Fatal(err)

	common.Debug("srcFile: %s", srcFile.Name())

	defer func() {
		common.Fatal(os.Remove(srcFile.Name()))
	}()

	destFile, err := common.CreateTempFile()
	common.Fatal(err)

	common.Debug("destFile: %s", destFile.Name())

	defer func() {
		common.Fatal(os.Remove(destFile.Name()))
	}()

	common.Fatal(ioutil.WriteFile(srcFile.Name(), ba, common.DefaultFileMode))

	cmd := exec.Command("cmd.exe", "/c", this.encoder, srcFile.Name(), destFile.Name())
	common.Debug(common.CmdToString(cmd))

	common.Fatal(cmd.Run())

	ba, err = ioutil.ReadFile(destFile.Name())
	common.Fatal(err)

	return ba
}

func (this *HL7Filter) Read(p []byte) (int, error) {
	n := this.drain(p)
	if n > 0 {
		return n, nil
	}

	var err error
	ba := make([]byte, 32768)

	for !this.msgComplete && err == nil {
		if this.buffer.Len() == 0 {
			this.buffer.Reset()

			n, err = this.reader.Read(ba)

			if n > 0 {
				this.buffer.Write(ba[:n])
			}
		}

		if this.buffer.Len() > 0 {
			i := 0
			for !this.msgComplete && this.buffer.Len() > 0 {
				b := this.buffer.Next(1)

				switch b[0] {
				case SB:
					common.Debug("%5d: <SB> received", i)

					if this.msg.Len() > 0 {
						common.Debug(fmt.Sprintf("drop %d orphaned bytes", this.msg.Len()))
						this.msg.Reset()
					}

					this.msg.Write(b)
				case CR:
					common.Debug("%5d: <CR> received", i)

					this.msg.Write(b)

					if this.msg.Len() > 1 {
						ba := this.msg.Bytes()

						this.msgComplete = ba[len(ba)-2] == 0x1c && ba[len(ba)-1] == 0x0d

						if this.msgComplete {
							this.msgComplete = len(ba) > 3

							if this.msgComplete {
								common.Debug("msg complete!")

								if this.encoder != "" {
									ba = this.encode(ba[1 : len(ba)-2])

									ba = append([]byte{SB}, ba...)
									ba = append(ba, []byte{EB, CR}...)

									this.msg.Reset()
									this.msg.Write(ba)
								}
							} else {
								common.Debug("drop msg with no content!")
								this.msg.Reset()
							}
						}
					}
				case EB:
					common.Debug("%5d: <EB> received", i)
					this.msg.Write(b)
				default:
					this.msg.Write(b)
				}

				i++
			}
		}
	}

	n = this.drain(p)
	if n > 0 {
		return n, nil
	}

	return 0, err

}
