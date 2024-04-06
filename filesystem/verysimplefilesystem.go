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

// this is the inode struct
type Inode struct {
	IsValid      bool
	IsDirectory  bool
	Datablocks   [4]int
	Filecreated  time.Time
	Filemodified time.Time
	Inodenumber  int
}

// this is a folder struct
type Directory struct {
	Filename  string
	Inode     int
	Files     []int
	Filenames []string
}

// this is a file struct
type DirectoryEntry struct {
	Filename string
	Inode    int
	Fileinfo string
}

// this is the superblock
type SuperBlock struct {
	Inodeoffset       int
	Blockbitmapoffset int
	Inodebitmapoffset int
	Datablocksoffset  int
}

// these are my globals
var VirtualDisk [6010][1024]byte
var BlockBitmap [6000]bool
var InodeBitmap [120]bool
var Inodes [120]Inode
var EndBlockBitmap int
var EndInodeBitmap int
var EndInodes int
var LastInodeBlock int
var Nextopeninode int

// this is the function that initializes the disk with a root directory, bitmaps, and 120 inodes
func InitializeDisk() {
	//create a new encoder
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
	//add the root directory to the disk
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

	//encode and push the superblock onto block 0
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

	bitmapBytesBlocks := boolsToBytes(BlockBitmap[:])
	EndBlockBitmap = len(bitmapBytesBlocks) - 1
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}

	// encode and push the inodes onto the disk
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
} // end of initialize disk
// this function converts a boolean array to bytes
// boolstobytes and bytestobools comes from https://stackoverflow.com/questions/53924984/bool-array-to-byte-array
func boolsToBytes(t []bool) []byte {
	b := make([]byte, (len(t)+7)/8)
	for i, x := range t {
		if x {
			b[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return b
}

// this function converts a byte array to booleans
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

// this function reads the superblock by decoding it from VirtualDisk[0]
func ReadSuperblock() SuperBlock {
	var superblock SuperBlock
	decoder := gob.NewDecoder(bytes.NewReader(VirtualDisk[0][:]))
	if err := decoder.Decode(&superblock); err != nil {
		return superblock
	}
	return superblock
}

// this function reads a directory by decoding it from up to 4 data blocks
func ReadFolder(w, x, y, z int) Directory {
	var directory Directory
	var blockData []byte
	// write relevant blocks to blockdata
	blockData = append(blockData, VirtualDisk[w][:]...)
	blockData = append(blockData, VirtualDisk[x][:]...)
	blockData = append(blockData, VirtualDisk[y][:]...)
	blockData = append(blockData, VirtualDisk[z][:]...)
	// decode blockdata
	decoder := gob.NewDecoder(bytes.NewReader(blockData))
	if err := decoder.Decode(&directory); err != nil {
		fmt.Println("Error decoding directory:", err)
	}

	return directory
}

// this function reads the inodes from the disk
func ReadInodesFromDisk() [120]Inode {
	var inodes [120]Inode
	var blockData []byte
	superblock := ReadSuperblock()
	//outer loop goes from superblock offset to last inode block, inner loop goes from start to end of block
	for i := superblock.Inodeoffset; i < superblock.Inodeoffset+LastInodeBlock; i++ {
		for j := 0; j < Blocksize; j++ {
			blockData = append(blockData, VirtualDisk[i][j])
			if len(blockData) > EndInodes {
				break
			}
		}
	}
	//decode blockData into inodes
	buf := bytes.NewBuffer(blockData[:])
	gob.NewDecoder(buf).Decode(&inodes)
	return inodes
}

// this function takes an inode struct array and writes it to the appropriate disk space
func WriteInodesToDisk(x [120]Inode) {
	var buf bytes.Buffer
	j := 0
	//encode the array
	superblock := ReadSuperblock()
	gob.NewEncoder(&buf).Encode(x)
	data := buf.Bytes()
	EndInodes = len(data)
	blockSize := Blocksize
	inodeOffset := int(superblock.Inodeoffset)
	//write the encoded array to proper disk space
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

// this function adds the blockbitmap to the disk
func AddBlockBitmapToDisk(x []bool) {
	bitmapBytesBlocks := boolsToBytes(x[:])
	EndBlockBitmap = len(bitmapBytesBlocks)
	for i := range bitmapBytesBlocks {
		VirtualDisk[2][i] = bitmapBytesBlocks[i]
	}
}

// this function adds the inode bitmap to the disk
func AddInodeBitmapToDisk(x []bool) {
	bitmapBytesInode := boolsToBytes(x[:])
	EndInodeBitmap = len(bitmapBytesInode)
	for i := range bitmapBytesInode {
		VirtualDisk[1][i] = bitmapBytesInode[i]
	}
}

// this function adds an updated working directory to the disk with 4 datablocks for index
func AddWorkingDirectoryToDisk(directory Directory, datablocks [4]int) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(directory); err != nil {
		fmt.Println("Error encoding directory:", err)
		return
	}
	data := buf.Bytes()
	blockIndex := 0
	//copy to all data blocks to the end of len(data)
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

// this function decodes a directory entry at the inode datablocks
func DecodeDirectoryEntryFromDisk(inode Inode) DirectoryEntry {
	var entry DirectoryEntry
	var data []byte

	// iterate through the data blocks of the inode
	for i := 0; i < len(inode.Datablocks); i++ {
		// if the data block is not zero
		if inode.Datablocks[i] != 0 {
			// Read the data from the data block
			data = append(data, VirtualDisk[inode.Datablocks[i]][:]...)
			// Decode the data into DirectoryEntry
		}
	}
	//decode data
	decoder := gob.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&entry); err != nil {
		log.Fatal("Error decoding directory entry:", err)
	}
	return entry
}

// this function encodes a directory entry to the disk, allocates more blocks if needed
func EncodeDirectoryEntryToDisk(entry DirectoryEntry, inode Inode) Inode {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(entry)
	if err != nil {
		log.Fatal(err)
	}
	data := buf.Bytes()

	superblock := ReadSuperblock()
	blockBitmap := bytesToBools(VirtualDisk[superblock.Blockbitmapoffset][:EndBlockBitmap])

	// Calculate the number of blocks needed for the data
	numBlocksNeeded := (len(data) + Blocksize - 1) / Blocksize

	// Check if any blocks are already allocated for the inode
	allocatedBlocks := 0
	for _, block := range inode.Datablocks {
		if block != 0 {
			allocatedBlocks++
		}
	}

	// Check if the existing allocated blocks are sufficient
	if allocatedBlocks >= numBlocksNeeded {
		// Write data to the existing blocks
		start := 0
		for i := 0; i < numBlocksNeeded; i++ {
			end := start + Blocksize
			if end > len(data) {
				end = len(data)
			}
			copy(VirtualDisk[inode.Datablocks[i]][:], data[start:end])
			start = end
		}
		return inode
	}

	// Allocate new blocks for the remaining data
	freeBlocks := make([]int, 0)
	for i := range blockBitmap {
		if !blockBitmap[i] {
			blockBitmap[i] = true
			freeBlocks = append(freeBlocks, i)
			if len(freeBlocks) == numBlocksNeeded-allocatedBlocks {
				break
			}
		}
	}

	// Check if enough free blocks are available
	if len(freeBlocks) < numBlocksNeeded-allocatedBlocks {
		log.Fatal("Not enough free blocks available for data")
	}

	// Update the inode with the new data blocks
	for i := allocatedBlocks; i < numBlocksNeeded; i++ {
		inode.Datablocks[i] = freeBlocks[i-allocatedBlocks] + superblock.Datablocksoffset
	}

	// Write data to the allocated blocks
	start := 0
	for i := 0; i < numBlocksNeeded; i++ {
		end := start + Blocksize
		if end > len(data) {
			end = len(data)
		}
		copy(VirtualDisk[inode.Datablocks[i]][:], data[start:end])
		start = end
	}

	// Update block bitmap on disk
	AddBlockBitmapToDisk(blockBitmap)

	return inode
}

// this is the Open function with open, write, read, and append options. Takes mode, filename, and inode of
// parent directory as arguments
func Open(mode string, filename string, searchnode int) {
	superblock := ReadSuperblock()
	switch mode {
	case "open":
		//read inodes and search for correct inode
		inodes := ReadInodesFromDisk()
		var disknode Inode
		var inode Inode
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
			if len(filename) > 12 {
				fmt.Println("Filename exceeds maximum")
			} else {
				fmt.Println("Creating new file ", filename, " in working directory ", workingdirectory.Filename)
				//create a file
				var newfile DirectoryEntry
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
						inodes[i] = EncodeDirectoryEntryToDisk(newfile, inodes[i])
						//update the working directory
						workingdirectory.Filenames = append(workingdirectory.Filenames, filename)
						workingdirectory.Files = append(workingdirectory.Files, i)
						break
					}
				}
				AddInodeBitmapToDisk(inodebitmap)
				AddWorkingDirectoryToDisk(workingdirectory, datablocks)
				inodes[inode.Inodenumber] = inode
				WriteInodesToDisk(inodes)
			}
		}
	case "write":
		//read inodes and search for correct inode
		inodes := ReadInodesFromDisk()
		var disknode Inode
		var inode Inode
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
		var workingfile DirectoryEntry
		var workinginode int
		var found bool
		workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workinginode = workingdirectory.Files[i]
				found = true
				break
			}
		}
		if found == true {
			fmt.Println("Writing to file: ", filename)
			var info string
			fmt.Println("Please enter a string to write to disk")
			fmt.Scanln(&info)
			inode = inodes[workinginode]
			workingfile = DecodeDirectoryEntryFromDisk(inode)
			workingfile.Fileinfo = info
			inode = EncodeDirectoryEntryToDisk(workingfile, inode)
			inode.Filemodified = time.Now()
			inodes[inode.Inodenumber] = inode
		} else {
			fmt.Println("Could not find file")
		}
		WriteInodesToDisk(inodes)
	case "read":
		//read inodes and search for correct inode
		inodes := ReadInodesFromDisk()
		var disknode Inode
		var inode Inode
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
		var workingfile DirectoryEntry
		var workinginode int
		var found bool
		workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workinginode = workingdirectory.Files[i]
				found = true
				break
			}
		}
		if found == true {
			inode = inodes[workinginode]
			workingfile = DecodeDirectoryEntryFromDisk(inode)
			fmt.Println("File ", workingfile.Filename, " contains info: ", workingfile.Fileinfo)
			inode = EncodeDirectoryEntryToDisk(workingfile, inode)
			inodes[inode.Inodenumber] = inode
		} else {
			fmt.Println("Could not find file")
		}
		WriteInodesToDisk(inodes)
	case "append":
		//read inodes and search for correct inode
		fmt.Println("appending to file: ", filename)
		inodes := ReadInodesFromDisk()
		var disknode Inode
		var inode Inode
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
		var workingfile DirectoryEntry
		var workinginode int
		var found bool
		workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workinginode = workingdirectory.Files[i]
				found = true
				break
			}
		}
		if found == true {
			var info string
			fmt.Println("Please enter a string to append to disk")
			fmt.Scanln(&info)
			inode = inodes[workinginode]
			workingfile = DecodeDirectoryEntryFromDisk(inode)
			workingfile.Fileinfo = workingfile.Fileinfo + info
			inode = EncodeDirectoryEntryToDisk(workingfile, inode)
			inode.Filemodified = time.Now()
			inodes[inode.Inodenumber] = inode
		} else {
			fmt.Println("Could not find file")
		}
		WriteInodesToDisk(inodes)
	}
}

// this function takes a filename and the inode number of a parent directory and
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
		log.Fatal("No directory present at inode ", searchnode)
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
	//if found start unlinking and deleting data
	if found == true {
		fmt.Println("unlinking file: ", filename)
		for i := range workingdirectory.Filenames {
			if workingdirectory.Filenames[i] == filename {
				workingdirectory.Filenames[i] = ""
				workinginode = workingdirectory.Files[i]
				workingdirectory.Files[i] = 0
			}
		}
		var emptyarray [1024]byte
		//adjust the blockbitmap
		for i := range inodes[workinginode].Datablocks {
			if inodes[workinginode].Datablocks[i] != 0 {
				blockbitmap[inodes[workinginode].Datablocks[i]-superblock.Datablocksoffset] = false
				copy(VirtualDisk[inodes[workinginode].Datablocks[i]][:], emptyarray[:])
			}
		}
		//adjust the inodes
		inodebitmap[workinginode] = false
		inodes[workinginode].Datablocks = [4]int{0, 0, 0, 0}
		inodes[workinginode].IsDirectory = false
		inodes[workinginode].IsValid = false
		AddBlockBitmapToDisk(blockbitmap)
		AddInodeBitmapToDisk(inodebitmap)
		WriteInodesToDisk(inodes)
		AddWorkingDirectoryToDisk(workingdirectory, datablocks)
	} else {
		//file not found
		fmt.Println("Could not find file")
	}
}
func Read(filename string, searchnode int) {
	//read inodes and search for correct inode
	inodes := ReadInodesFromDisk()
	var disknode Inode
	var inode Inode
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
	var workingfile DirectoryEntry
	var workinginode int
	var found bool
	workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
	for i := range workingdirectory.Filenames {
		if workingdirectory.Filenames[i] == filename {
			workinginode = workingdirectory.Files[i]
			found = true
			break
		}
	}
	if found == true {
		inode = inodes[workinginode]
		workingfile = DecodeDirectoryEntryFromDisk(inode)
		fmt.Println("File ", workingfile.Filename, " contains info: ", workingfile.Fileinfo)
		inode = EncodeDirectoryEntryToDisk(workingfile, inode)
		inodes[inode.Inodenumber] = inode
	} else {
		fmt.Println("Could not find file")
	}
	WriteInodesToDisk(inodes)
}
func Write(filename string, searchnode int) {
	//read inodes and search for correct inode
	inodes := ReadInodesFromDisk()
	var disknode Inode
	var inode Inode
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
	var workingfile DirectoryEntry
	var workinginode int
	var found bool
	workingdirectory := ReadFolder(datablocks[0], datablocks[1], datablocks[2], datablocks[3])
	for i := range workingdirectory.Filenames {
		if workingdirectory.Filenames[i] == filename {
			workinginode = workingdirectory.Files[i]
			found = true
			break
		}
	}
	if found == true {
		fmt.Println("Writing to file: ", filename)
		var info string
		fmt.Println("Please enter a string to write to disk")
		fmt.Scanln(&info)
		inode = inodes[workinginode]
		workingfile = DecodeDirectoryEntryFromDisk(inode)
		workingfile.Fileinfo = info
		inode = EncodeDirectoryEntryToDisk(workingfile, inode)
		inode.Filemodified = time.Now()
		inodes[inode.Inodenumber] = inode
	} else {
		fmt.Println("Could not find file")
	}
	WriteInodesToDisk(inodes)
}
