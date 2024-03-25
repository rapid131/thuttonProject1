package filesystem

type Inode struct {
	isValid     bool
	isDirectory bool
}

var VirtualDisk [6000][1024]byte
