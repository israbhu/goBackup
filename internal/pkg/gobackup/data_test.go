package gobackup

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type DataSuite struct{}

var _ = Suite(&DataSuite{})

func (s *DataSuite) TestGetMetadata(c *C) {
	want := "foobar.txt:42:foobar notes:1626394466"
	md := Metadata{
		FileName: "foobar.txt",
		FileNum:  42,
		Notes:    "foobar notes",
		Mtime:    time.Unix(1626394466, 0),
	}
	got := GetMetadata(md)
	c.Check(got, Equals, want)
}
