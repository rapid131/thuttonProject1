package filesystem

import (
	"bytes"
	"encoding/gob"
	"log"
	"time"
)

// 30 blocks for inodes
// 1 block for inode bitmap
const (
	blocksize      = 1024
	inodesize      = 256
	numberofinodes = 120
)

type Inode struct {
	IsValid      bool
	IsDirectory  bool
	Datablocks   [4]int
	Filecreated  time.Time
	Filemodified time.Time
}

type DirectoryEntry struct {
	Filename [8]byte
	Filetype [4]byte
	Inode    int
}
type FileSystem struct {
	BlockBitmap [6000]bool
	InodeBitmap [120]bool
	Inodes      [120]Inode
}
type SuperBlock struct {
	Inodeoffset       byte
	Blockbitmapoffset byte
	Inodebitmapoffset byte
}

var VirtualDisk [6044][1024]byte

func InitializeDisk() {
	var encoder bytes.Buffer
	enc := gob.NewEncoder(&encoder)
	var fs FileSystem
	for i := range fs.BlockBitmap {
		fs.BlockBitmap[i] = false
	}
	for i := range fs.InodeBitmap {
		fs.InodeBitmap[i] = false
	}

	var superblock SuperBlock
	superblock.Inodeoffset = 14
	superblock.Blockbitmapoffset = 13
	superblock.Inodebitmapoffset = 1

	err := enc.Encode(superblock)
	if err != nil {
		log.Fatal(err)
	}
	for i := range encoder.Bytes() {
		VirtualDisk[0][i] = encoder.Bytes()[i]
	}
}
