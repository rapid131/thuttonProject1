package filesystem

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"log"
	"time"
)

const (
	Blocksize      = 1024
	Numberofinodes = 120
	Filenamelength = 12
)

type Inode struct {
	IsValid      bool
	IsDirectory  bool
	Datablocks   [4]int
	Filecreated  time.Time
	Filemodified time.Time
	Inodenumber  int
}
type Pointerblock struct {
	Datablocks [4]int
}
type Directory struct {
	Filename  string
	Inode     int
	Files     []int
	Filenames []string
}
type DirectoryEntry struct {
	Filename string
	Inode    int
	Fileinfo string
}

type SuperBlock struct {
	Inodeoffset       int
	Blockbitmapoffset int
	Inodebitmapoffset int
	Datablocksoffset  int
}

var VirtualDisk [6010][1024]byte
var BlockBitmap [6000]bool
var InodeBitmap [120]bool
var Inodes [120]Inode
var EndBlockBitmap int
var EndInodeBitmap int
var EndInodes int
var LastInodeBlock int
var Nextopeninode int

func InitializeDisk() {
	//create a enw encoder
	var encoder bytes.Buffer
	enc := gob.NewEncoder(&encoder)
	var inode Inode

	//prepare empty Inode array of size 120
	for i := range Inodes {
		inode.Datablocks = [4]int{0, 0, 0, 0}
		inode.IsDirectory = false
		inode.IsValid = false
		inode.Filecreated = time.Now()
		inode.Filemodified = time.Now()
		inode.Inodenumber = i
		Inodes[i] = inode
	}

	//inititate second to first inode with root directory
	Inodes[1].IsDirectory = true
	Inodes[1].IsValid = true
	Inodes[1].Datablocks = [4]int{9, 0, 0, 0}

	//create a root directory
	var rootdirectory Directory
	rootdirectory.Filename = "root.dir"
	rootdirectory.Inode = 1

	//encode root directory and push it onto disk
	err := enc.Encode(rootdirectory)
	if err != nil {
		log.Fatal(err)
	}
	for i := range encoder.Bytes() {
		VirtualDisk[9][i] = encoder.Bytes()[i]
	}
	encoder.Reset()

	//set all bitmaps to false
	for i := range BlockBitmap {
		BlockBitmap[i] = false
	}
	for i := range InodeBitmap {
		InodeBitmap[i] = false
	}

	//set bitmaps for root inode and root directory
	BlockBitmap[0] = true
	InodeBitmap[0] = true
	InodeBitmap[1] = true

	//create the superblock
	var superblock SuperBlock
	superblock.Inodeoffset = 3
	superblock.Blockbitmapoffset = 2
	superblock.Inodebitmapoffset = 1
	superblock.Datablocksoffset = 9

	//encode and push the superblock onto block 1
	err = enc.Encode(superblock)
	if err != nil {
		log.Fatal(err)
	}
	for i := range encoder.Bytes() {
		VirtualDisk[0][i] = encoder.Bytes()[i]
	}
	encoder.Reset()

	//change bools to bytes of both bitmaps and put them on disk
	bitmapBytesInode := boolsToBytes(InodeBitmap[:])
	EndInodeBitmap = len(bitmapBytesInode) - 1
	for i := range bitmapBytesInode {
		VirtualDisk[1][i] = bitmapBytesInode[i]
	}
	BlockBitmap[0] = true
	bitmapBytesBlocks := boolsToBytes(BlockBitmap[:])
	EndBlockBitmap = len(bitmapBytesBlocks) - 1
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}

	//encode and push the inodes onto the disk
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
func ReadFolder(w, x, y, z int) Directory {
	var directory Directory
	var blockData []byte
	//write relevant blocks to blockdata
	blockData = append(blockData, VirtualDisk[w][:]...)
	blockData = append(blockData, VirtualDisk[x][:]...)
	blockData = append(blockData, VirtualDisk[y][:]...)
	//check for indirect block pointers
	if z != 0 {
		var pointerBlock Pointerblock
		decoder := gob.NewDecoder(bytes.NewReader(VirtualDisk[z][:]))
		if err := decoder.Decode(&pointerBlock); err != nil {
			fmt.Println("Error decoding pointer block:", err)
			return directory
		}
		// Append additional data blocks to blockData
		for _, blockNum := range pointerBlock.Datablocks {
			if blockNum != 0 {
				blockData = append(blockData, VirtualDisk[blockNum][:]...)
			}
		}
	}
	//decode blockdata
	decoder := gob.NewDecoder(bytes.NewReader(blockData))
	if err := decoder.Decode(&directory); err != nil {
		fmt.Println("Error decoding directory:", err)
	}

	return directory
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
func AddBlockBitmapToDisk(x []bool) {
	bitmapBytesBlocks := boolsToBytes(x[:])
	EndBlockBitmap = len(bitmapBytesBlocks) - 1
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}
}
func AddInodeBitmapToDisk(x []bool) {
	bitmapBytesInode := boolsToBytes(x[:])
	EndInodeBitmap = len(bitmapBytesInode) - 1
	for i := range bitmapBytesInode {
		VirtualDisk[1][i] = bitmapBytesInode[i]
	}
}
func AddWorkingDirectoryToDisk(directory Directory, datablocks [4]int) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(directory); err != nil {
		fmt.Println("Error encoding directory:", err)
		return
	}
	data := buf.Bytes()
	blockIndex := 0
	for i := 0; i < len(data); i += Blocksize {
		start := i
		end := start + Blocksize
		if end > len(data) {
			end = len(data)
		}
		copy(VirtualDisk[datablocks[blockIndex]][:], data[start:end])
		blockIndex++
		if blockIndex >= len(datablocks) {
			break
		}
	}
}
func Open(mode string, filename string, searchnode int) {
	superblock := ReadSuperblock()
	switch mode {
	case "open":
		//read inodes and search for correct inode
		inodes := ReadInodesFromDisk()
		var disknode Inode
		for i := range inodes {
			if inodes[i].Inodenumber == searchnode {
				disknode = Inodes[i]
				break
			}
		}
		//get the datablocks for the inode
		var datablocks [4]int
		for i := range disknode.Datablocks {
			datablocks[i] = disknode.Datablocks[i]
		}
		if datablocks[0] == 0 {
			fmt.Println("No directory present at inode ", searchnode)
		}

		//get the workingdirectory from the inodes
		var workingfile string
		var workinginode int
		var found bool
		workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workingfile = workingdirectory.Filenames[i]
				workinginode = workingdirectory.Files[i]
				found = true
				break
			}
		}
		//if file found, print it out
		if found == true {
			fmt.Println("Found file ", workingfile, " at Inode ", workinginode)
			//file not found, create it
		} else {
			fmt.Println("Creating new file ", filename, " in working directory ", workingdirectory.Filename)
			var newfile DirectoryEntry
			blockbitmap := bytesToBools(VirtualDisk[superblock.Blockbitmapoffset][:EndBlockBitmap])
			inodebitmap := bytesToBools(VirtualDisk[superblock.Inodebitmapoffset][:EndInodeBitmap])
			newfile.Filename = filename
			for i := range inodebitmap {
				if inodebitmap[i] == false {
					inodebitmap[i] = true
					newfile.Inode = i
					inodes[i].Filecreated = time.Now()
					inodes[i].Filemodified = time.Now()
					inodes[i].IsDirectory = false
					inodes[i].IsValid = true
					for j := range blockbitmap {
						if blockbitmap[j] == false {
							blockbitmap[j] = true
							inodes[i].Datablocks = [4]int{j + superblock.Datablocksoffset, 0, 0, 0}
							break
						}
					}
					workingdirectory.Filenames = append(workingdirectory.Filenames, filename)
					workingdirectory.Files = append(workingdirectory.Files, i)
					break
				}
			}
			AddBlockBitmapToDisk(blockbitmap)
			AddInodeBitmapToDisk(inodebitmap)
			AddWorkingDirectoryToDisk(workingdirectory, datablocks)
			WriteInodesToDisk(inodes)
		}
	}
}
