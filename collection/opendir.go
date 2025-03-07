// OpenDir is like Open(), but the Readdir() os.FileInfo
// results are lazy-loaded.
package collection

import (
	"errors"
	"os"
	"path"
	"time"
)

type Dir struct {
	name string
	file *os.File
}

type FileInfo struct {
	dir        *Dir
	name       string
	size       int64
	mode       os.FileMode
	modtime    time.Time
	createtime time.Time
	sys        interface{}
	isdir      bool
	didstat    bool
}

var ErrNotDirectory = errors.New("not a directory")

func OpenDir(name string) (dir *Dir, err error) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	fi, _ := f.Stat()
	if !fi.IsDir() {
		err = &os.PathError{
			Op:   "Opendir",
			Path: name,
			Err:  ErrNotDirectory,
		}
		return
	}
	dir = &Dir{
		name: name,
		file: f,
	}
	return
}

func (dir *Dir) Close() error {
	return dir.file.Close()
}

func (dir *Dir) Stat() (os.FileInfo, error) {
	return dir.file.Stat()
}

func (dir *Dir) Readdirnames(n int) (names []string, err error) {
	return dir.file.Readdirnames(n)
}

func (dir *Dir) Readdir(n int) (fi []FileInfo, err error) {
	names, err := dir.Readdirnames(n)
	if err != nil {
		return
	}
	fi = make([]FileInfo, len(names))
	for i := range names {
		fi[i].dir = dir
		fi[i].name = names[i]
	}
	return
}

func (fi *FileInfo) Name() string {
	return fi.name
}

func (fi *FileInfo) Size() int64 {
	fi.stat()
	return fi.size
}

func (fi *FileInfo) Mode() os.FileMode {
	fi.stat()
	return fi.mode
}

func (fi *FileInfo) Modtime() time.Time {
	fi.stat()
	return fi.modtime
}

func (fi *FileInfo) Createtime() time.Time {
	fi.stat()
	return fi.createtime
}

func (fi *FileInfo) CreatetimeMS() int64 {
	fi.stat()
	return fi.createtime.UnixNano() / 1000000
}

func (fi *FileInfo) IsDir() bool {
	fi.stat()
	return fi.isdir
}

func (fi *FileInfo) stat() {
	if fi.didstat {
		return
	}
	p := path.Join(fi.dir.name, fi.name)
	s, err := os.Stat(p)
	if err != nil {
		return
	}
	fi.set(s)
}

func (fi *FileInfo) set(s os.FileInfo) {
	fi.size = s.Size()
	fi.mode = s.Mode()
	fi.modtime = s.ModTime()
	fi.isdir = s.IsDir()
	fi.sys = s.Sys()
	fi.setCreatetime()
	fi.didstat = true
}

func (fi *FileInfo) Sys() interface{} {
	fi.stat()
	return fi.sys
}
