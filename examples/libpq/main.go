// +build gofuzz

package libpq

import "github.com/cockroachdb/pq"

func FuzzParseTimestamp(data []byte) int {
	_, err := pq.ParseTimestamp(nil, string(data))
	if err != nil {
		return 0
	}
	return 1
}
