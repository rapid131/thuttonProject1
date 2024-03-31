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
	Blocksize      = 1024
	Inodesize      = 256
	Numberofinodes = 120
)

type Inode struct {
	IsValid      bool
	IsDirectory  bool
	Datablocks   [4]int
	Filecreated  time.Time
	Filemodified time.Time
	Inodenumber  int
}

type DirectoryEntry struct {
	Filename [8]byte
	Filetype [4]byte
	Inode    int
}

type SuperBlock struct {
	Inodeoffset       int
	Blockbitmapoffset int
	Inodebitmapoffset int
}

var VirtualDisk [6044][1024]byte
var BlockBitmap [6000]bool
var InodeBitmap [120]bool
var Inodes [120]Inode
var EndBlockBitmap int
var EndInodeBitmap int

func InitializeDisk() {
	var encoder bytes.Buffer
	enc := gob.NewEncoder(&encoder)
	var inode Inode
	for i := range Inodes {
		inode.Datablocks = [4]int{0, 0, 0, 0}
		inode.IsDirectory = false
		inode.IsValid = false
		inode.Filecreated = time.Now()
		inode.Filemodified = time.Now()
		inode.Inodenumber = i
		Inodes[i] = inode
	}
	for i := range BlockBitmap {
		BlockBitmap[i] = false
	}
	for i := range InodeBitmap {
		InodeBitmap[i] = false
	}

	var superblock SuperBlock
	superblock.Inodeoffset = 3
	superblock.Blockbitmapoffset = 2
	superblock.Inodebitmapoffset = 1

	err := enc.Encode(superblock)
	if err != nil {
		log.Fatal(err)
	}
	for i := range encoder.Bytes() {
		VirtualDisk[0][i] = encoder.Bytes()[i]
	}
	encoder.Reset()

	bitmapBytesInode := boolsToBytes(InodeBitmap[:])
	EndInodeBitmap = len(bitmapBytesInode)
	for i := range bitmapBytesInode {
		VirtualDisk[1][i] = bitmapBytesInode[i]
	}

	bitmapBytesBlocks := boolsToBytes(BlockBitmap[:])
	EndBlockBitmap = len(bitmapBytesBlocks)
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}

	for i, inode := range Inodes {
		var buf bytes.Buffer
		gob.NewEncoder(&buf).Encode(inode)
		copy(VirtualDisk[3+i][:], buf.Bytes())
	}
} //end of initialize disk
func boolsToBytes(t []bool) []byte {
	b := make([]byte, (len(t)+7)/8)
	for i, x := range t {
		if x {
			b[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return b
}

func bytesToBools(b []byte) []bool {
	t := make([]bool, 8*len(b))
	for i, x := range b {
		for j := 0; j < 8; j++ {
			if (x<<uint(j))&0x80 == 0x80 {
				t[8*i+j] = true
			}
		}
	}
	return t
}
func ReadSuperblock() SuperBlock {
	var superblock SuperBlock
	decoder := gob.NewDecoder(bytes.NewReader(VirtualDisk[0][:]))
	if err := decoder.Decode(&superblock); err != nil {
		return superblock
	}
	return superblock
}
func ReadInodesFromDisk() []Inode {
	superblock := ReadSuperblock()
	inodeOffset := int(superblock.Inodeoffset)
	var inodes []Inode
	for i := inodeOffset; i < inodeOffset+len(Inodes); i++ {
		var inode Inode
		buf := bytes.NewBuffer(VirtualDisk[i][:])
		gob.NewDecoder(buf).Decode(&inode)
		inodes = append(inodes, inode)
	}
	return inodes
}
