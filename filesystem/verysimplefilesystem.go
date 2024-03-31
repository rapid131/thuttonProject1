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
	Filename string
	Filetype string
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
var EndInodes int
var LastInodeBlock int
var Diskinitialized bool

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
	var buf bytes.Buffer
	gob.NewEncoder(&buf).Encode(Inodes)
	data := buf.Bytes()
	EndInodes = len(data)
	blockSize := Blocksize
	inodeOffset := int(superblock.Inodeoffset)
	j := 0
	for i := 0; i < len(data); i += blockSize {
		blockIndex := inodeOffset + i/blockSize
		end := i + blockSize
		if end > len(data) {
			end = len(data)
		}
		copy(VirtualDisk[blockIndex][:], data[i:end])
		j++
	}
	Diskinitialized = true
	LastInodeBlock = j
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
func ReadInodesFromDisk() [120]Inode {
	var inodes [120]Inode
	var blockData []byte
	superblock := ReadSuperblock()
	for i := superblock.Inodeoffset; i < superblock.Inodeoffset+LastInodeBlock; i++ {
		for j := 0; j < Blocksize; j++ {
			blockData = append(blockData, VirtualDisk[i][j])
			if len(blockData) > EndInodes {
				break
			}
		}
	}
	buf := bytes.NewBuffer(blockData[:])
	gob.NewDecoder(buf).Decode(&inodes)
	return inodes
}
func WriteInodesToDisk(x [120]Inode) {
	var buf bytes.Buffer
	j := 0
	superblock := ReadSuperblock()
	gob.NewEncoder(&buf).Encode(x)
	data := buf.Bytes()
	EndInodes = len(data)
	blockSize := Blocksize
	inodeOffset := int(superblock.Inodeoffset)
	for i := 0; i < len(data); i += blockSize {
		blockIndex := inodeOffset + i/blockSize
		end := i + blockSize
		if end > len(data) {
			end = len(data)
		}
		copy(VirtualDisk[blockIndex][:], data[i:end])
		j++
	}
	LastInodeBlock = j
}
