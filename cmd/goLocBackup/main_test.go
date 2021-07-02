package main

import (
	"testing"
	"time"

	. "gopkg.in/check.v1"

	"github.com/israbhu/goBackup/internal/pkg/gobackup"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type MainSuite struct{}

var _ = Suite(&MainSuite{})

func (s *MainSuite) TestPopulatePayloadAndMeta(c *C) {
	var dat gobackup.Data1
	meta := gobackup.Metadata{
		Permissions: "-rw-rw-rw-",
		Filepath:    "foo/bar/kau/aux.txt",
		Hash:        "d41d8cd98f00b204e9800998ecf8427e",
		ForeignKey:  "",
		Mtime:       time.Time{},
		Size:        0,
	}

	metaFK := meta
	metaFK.Hash = "68b329da9893e34099c7d8ad5cb9c940" // TODO This is arbitrary
	metaFK.ForeignKey = meta.Hash

	populatePayloadAndMeta(&dat, &meta, metaFK.Hash)

	c.Assert(dat.TheMetadata, HasLen, 2)
	// Sorted by Hash value
	c.Check(metaFK, DeepEquals, dat.TheMetadata[0])
	c.Check(meta, DeepEquals, dat.TheMetadata[1])
}
