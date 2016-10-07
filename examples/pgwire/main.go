// +build gofuzz

package pgwire

import (
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach/base"
	"github.com/cockroachdb/cockroach/server"
	"github.com/cockroachdb/cockroach/testutils/serverutils"
)

var expected = []string{
	"fewer than half the known nodes are within the maximum offset",
}

// panicExpected returns whether a given panic value is was expected.
func panicExpected(r interface{}) bool {
	var errStr string
	switch t := r.(type) {
	case string:
		errStr = t
	case error:
		errStr = t.Error()
	}
	if errStr != "" {
		for _, exp := range expected {
			if strings.Contains(errStr, exp) {
				return true
			}
		}
	}
	return false
}

func Fuzz(data []byte) int {
	defer func() {
		if r := recover(); r != nil {
			if !panicExpected(r) {
				panic(r)
			}
		}
	}()

	serverutils.InitTestServerFactory(server.TestServerFactory)
	s, err := serverutils.StartServerRaw(base.TestServerArgs{
		Insecure: true,
	})
	if err != nil {
		panic(err)
	}
	defer s.Stopper().Stop()

	conn, err := net.Dial("tcp", s.ServingAddr())
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	go io.Copy(ioutil.Discard, conn)
	io.Copy(conn, bytes.NewReader(data))
	time.Sleep(1 * time.Millisecond)
	return 0
}
