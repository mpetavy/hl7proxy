package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/mpetavy/common"
	"github.com/stretchr/testify/assert"
	"io"
	"strings"
	"testing"
)

var (
	msg1 = fmt.Sprintf("%caaa%c%c", SB, EB, CR)
	msg2 = fmt.Sprintf("%cbbb%c%c", SB, EB, CR)
)

func TestMain(m *testing.M) {
	flag.Parse()
	common.Exit(m.Run())
}

func TestStream(t *testing.T) {
	common.SetTesting(t)

	for _, orphaned := range []string{
		fmt.Sprintf("%c%c%c", CR, SB, CR),
		fmt.Sprintf("%c%c%c", SB, EB, CR),
		fmt.Sprintf("%c%c%c", SB, SB, CR),
		fmt.Sprintf("%c%c%c", EB, CR, SB),
		fmt.Sprintf("x%cx%cx%c", SB, EB, CR),
		fmt.Sprintf("%cxx%cxx%cxx", SB, EB, CR),
		fmt.Sprintf("xx%cxx%cxx%cxx", SB, EB, CR),
	} {
		var tests = []struct {
			name string
			in   string
			out  string
		}{
			{"NULL stream", "", ""},
			{"orphaned", orphaned, ""},
			{"orphaned+orphaned", orphaned + orphaned, ""},
			{"msg1", msg1, msg1},
			{"orphaned + msg1", orphaned + msg1, msg1},
			{"msg1 + orphaned", msg1 + orphaned, msg1},
			{"msg1 + msg2", msg1 + msg2, msg1 + msg2},
			{"orphaned + msg1 + orphaned", orphaned + msg1 + orphaned, msg1},
			{"orphaned + msg1 + msg2", orphaned + msg1 + msg2, msg1 + msg2},
			{"orphaned + msg1 + msg2 + orphaned", orphaned + msg1 + msg2 + orphaned, msg1 + msg2},
			{"orphaned + msg1 + msg2 + orphaned", orphaned + msg1 + msg2 + orphaned, msg1 + msg2},
			{"orphaned + msg1 + orphaned + msg2 + orphaned", orphaned + msg1 + orphaned + msg2 + orphaned, msg1 + msg2},
		}

		t.Logf("orphaned: %+q\n", orphaned)

		for _, test := range tests {
			t.Run(test.name, func(t *testing.T) {
				r := strings.NewReader(test.in)
				w := bytes.Buffer{}

				n, err := io.Copy(&w, NewHL7Filter(r, ""))

				t.Logf("output: %+q\n", string(w.Bytes()))

				assert.True(t, err == nil, "EOF returned")
				assert.Equal(t, len(test.out), int(n), "Length of expected output")
			})
		}
	}
}
