// +build gofuzz

package pgwire

import (
	"encoding/binary"
	"net"
	"strings"
	"time"

	"github.com/cockroachdb/cockroach/server"
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

	// Run in insecure mode to avoid dealing with TLS.
	ctx := server.MakeTestContext()
	ctx.Insecure = true

	s := server.StartTestServerWithContext(nil, &ctx)
	defer s.Stop()

	conn, err := net.Dial("tcp", s.ServingAddr())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	// Connect to the PGWire v3 server. This connection message includes information
	// to match with pgwire.Match and bypass sql/pgwire.parseOptions:
	// - 4 byte prefix for the buffer's length
	// - 4 bytes for the version30 value
	// - 1 byte for a null terminator to skip session args
	const version30 = 196608
	var connectBuf [9]byte
	binary.BigEndian.PutUint32(connectBuf[:4], uint32(len(connectBuf)))
	binary.BigEndian.PutUint32(connectBuf[4:8], version30)
	if err != nil {
		panic(err)
	}
	conn.Write(connectBuf[:])
	time.Sleep(1 * time.Millisecond)

	// Send a pgwire "typed message" (see sql/pgwire.readBuffer.{readTypedMsg, readUntypedMsg}).
	// The message includes:
	// - 1 byte for the type
	// - 4 bytes for the message's length (minus the type prefix)
	// - the rest of message
	//
	// TODO(nvanbenschoten) investigate sending multiple messages.
	sendBuf := data
	if len(data) > 1 {
		newLen := len(data) + 4
		sendBuf = make([]byte, newLen)
		sendBuf[0] = data[0]
		binary.BigEndian.PutUint32(sendBuf[1:5], uint32(newLen-1))
		copy(sendBuf[5:], data[1:])
	}
	conn.Write(sendBuf)
	time.Sleep(1 * time.Millisecond)
	return 0
}
