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
	EndBlockBitmap = len(bitmapBytesBlocks)
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}
}
func AddInodeBitmapToDisk(x []bool) {
	bitmapBytesInode := boolsToBytes(x[:])
	EndInodeBitmap = len(bitmapBytesInode)
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
func EncodeDirectoryEntryToDisk(entry DirectoryEntry, inode Inode) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(entry)
	if err != nil {
		log.Fatal(err)
	}
	data := buf.Bytes()

	if len(data) > (Blocksize * 4) {
		log.Fatal("Directory entry size exceeds maximum supported size")
	}

	numBlocks := (len(data) + Blocksize - 1) / Blocksize

	superblock := ReadSuperblock()
	blockBitmap := bytesToBools(VirtualDisk[superblock.Blockbitmapoffset][:EndBlockBitmap])

	freeBlocks := make([]int, 0)
	for i := range blockBitmap {
		if blockBitmap[i] {
			freeBlocks = append(freeBlocks, i)
			if len(freeBlocks) == numBlocks {
				break
			}
		}
	}

	if len(freeBlocks) < numBlocks {
		log.Fatal("Not enough free blocks available for data")
	}

	for _, idx := range freeBlocks {
		blockBitmap[idx] = true
	}
	AddBlockBitmapToDisk(blockBitmap)

	// Update the inode with the new data blocks
	for i := 0; i < numBlocks; i++ {
		inode.Datablocks[i] = freeBlocks[i] + superblock.Datablocksoffset
	}

	// Pull the inodes from the disk and put this inode onto the inode array in the right place
	inodes := ReadInodesFromDisk()
	inodes[inode.Inodenumber] = inode

	// Put the inodes back into the disk
	WriteInodesToDisk(inodes)

	// Copy the data to the virtual disk
	start := 0
	for i := 0; i < numBlocks; i++ {
		end := start + Blocksize
		if end > len(data) {
			end = len(data)
		}
		copy(VirtualDisk[inode.Datablocks[i]][:], data[start:end])
		start = end
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
			//create a file
			var newfile DirectoryEntry
			blockbitmap := bytesToBools(VirtualDisk[superblock.Blockbitmapoffset][:EndBlockBitmap])
			inodebitmap := bytesToBools(VirtualDisk[superblock.Inodebitmapoffset][:EndInodeBitmap])
			newfile.Filename = filename
			//get the first free inode
			for i := range inodebitmap {
				if inodebitmap[i] == false {
					//set the inode features
					inodebitmap[i] = true
					newfile.Inode = i
					inodes[i].Filecreated = time.Now()
					inodes[i].Filemodified = time.Now()
					inodes[i].IsDirectory = false
					inodes[i].IsValid = true
					//get the first free block
					for j := range blockbitmap {
						if blockbitmap[j] == false {
							blockbitmap[j] = true
							inodes[i].Datablocks = [4]int{j + superblock.Datablocksoffset, 0, 0, 0}
							EncodeDirectoryEntryToDisk(newfile, inodes[i])
							break
						}
					}
					//update the working directory
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
func Unlink(filename string, searchnode int) {
	//read inodes and search for correct inode
	inodes := ReadInodesFromDisk()
	superblock := ReadSuperblock()
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
	var workinginode int
	var found bool
	blockbitmap := bytesToBools(VirtualDisk[superblock.Blockbitmapoffset][:EndBlockBitmap])
	inodebitmap := bytesToBools(VirtualDisk[superblock.Inodebitmapoffset][:EndInodeBitmap])
	workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
	for i := range workingdirectory.Filenames {
		if workingdirectory.Filenames[i] == filename {
			found = true
			break
		}
	}
	if found == true {
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workingdirectory.Filenames[i] = ""
				workinginode = workingdirectory.Files[i]
				workingdirectory.Files[i] = 0
			}
		}
		blockbitmap[inodes[workinginode].Datablocks[0]] = false
		blockbitmap[inodes[workinginode].Datablocks[1]] = false
		blockbitmap[inodes[workinginode].Datablocks[2]] = false
		blockbitmap[inodes[workinginode].Datablocks[3]] = false
		var emptyarray [1024]byte
		copy(VirtualDisk[inodes[workinginode].Datablocks[0]][:], emptyarray[:])
		copy(VirtualDisk[inodes[workinginode].Datablocks[1]][:], emptyarray[:])
		copy(VirtualDisk[inodes[workinginode].Datablocks[2]][:], emptyarray[:])
		copy(VirtualDisk[inodes[workinginode].Datablocks[3]][:], emptyarray[:])
		inodebitmap[workinginode] = false
		inodes[workinginode].Datablocks = [4]int{0, 0, 0, 0}
		inodes[workinginode].IsDirectory = false
		inodes[workinginode].IsValid = false
		AddBlockBitmapToDisk(blockbitmap)
		AddInodeBitmapToDisk(inodebitmap)
		WriteInodesToDisk(inodes)
		AddWorkingDirectoryToDisk(workingdirectory, datablocks)
	} else {
		fmt.Println("Could not find file")
	}
}
