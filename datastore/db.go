package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const outFileName = "current-data"

var ErrNotFound = fmt.Errorf("record does not exist")

type hashIndex map[string]int64

type Db struct {
	out       *os.File
	outOffset int64
	dir       string
	segSize   int64
	segIndex  int
	segments  []*Segment
}

type Segment struct {
	index    hashIndex
	filePath string
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments: make([]*Segment, 0),
		dir:      dir,
		segSize:  segmentSize,
	}
	err := db.makeSegment()
	if err != nil {
		return nil, err
	}
	err = db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return db, nil
}

func (db *Db) makeSegment() error {
	filePath := filepath.Join(db.dir, fmt.Sprintf("%s%d", outFileName, db.segIndex))
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
	db.segIndex++
	if err != nil {
		return err
	}
	seg := &Segment{
		filePath: filePath,
		index:    make(hashIndex),
	}
	db.out = f
	db.outOffset = 0
	db.segments = append(db.segments, seg)
	if len(db.segments) >= 3 {
		db.segCompact()
	}
	return err
}

func (db *Db) segCompact() {
	go func() {
		filePath := filepath.Join(db.dir, fmt.Sprintf("%s%d", outFileName, db.segIndex))
		f, err := os.OpenFile(filePath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
		db.segIndex++
		if err != nil {
			return
		}
		seg := &Segment{
			filePath: filePath,
			index:    make(hashIndex),
		}
		segIndex := len(db.segments) - 2
		var offset int64
		for i := 0; i <= segIndex; i++ {
			s := db.segments[i]
			for key := range s.index {
				if i < segIndex {
					if check(db.segments[i+1:segIndex+1], key) {
						continue
					}
				}
				value, _ := db.Get(key)
				e := entry{
					key:   key,
					value: value,
				}
				n, err := f.Write(e.Encode())
				if err == nil {
					seg.index[key] = offset
					offset += int64(n)
				}
			}
		}
		db.segments = []*Segment{seg, db.segments[len(db.segments)-1]}
	}()
}

func check(segments []*Segment, key string) bool {
	for _, segment := range segments {
		if _, ok := segment.index[key]; ok {
			return true
		}
	}
	return false
}

const bufSize = 8192

func (db *Db) recover() error {
	var err error
	var buf [bufSize]byte
	in := bufio.NewReaderSize(db.out, bufSize)
	for err == nil {
		var (
			header, data []byte
			n            int
		)
		header, err = in.Peek(bufSize)
		if err == io.EOF {
			if len(header) == 0 {
				return err
			}
		} else if err != nil {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}
		n, err = in.Read(data)

		if err == nil {
			if n != int(size) {
				return fmt.Errorf("corrupted file")
			}

			var e entry
			e.Decode(data)
			db.segments[len(db.segments)-1].index[e.key] = db.outOffset
			db.outOffset += int64(n)
		}
	}
	return err
}

func (db *Db) Close() error {
	return db.out.Close()
}

func (db *Db) Get(key string) (string, error) {
	for i := range db.segments {
		seg := db.segments[len(db.segments)-i-1]
		position, ok := seg.index[key]
		if ok {
			file, err := os.Open(seg.filePath)
			if err != nil {
				return "", err
			}
			defer file.Close()

			_, err = file.Seek(position, 0)
			if err != nil {
				return "", err
			}

			reader := bufio.NewReader(file)
			value, err := readValue(reader)
			if err != nil {
				return "", err
			}
			return value, nil
		}
	}
	return "", ErrNotFound
}

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
	}
	length := e.Length()
	stat, err := db.out.Stat()
	if err != nil {
		return err
	}
	if stat.Size()+int64(length) > db.segSize {
		if err := db.makeSegment(); err != nil {
			return err
		}
	}
	n, err := db.out.Write(e.Encode())
	if err == nil {
		db.segments[len(db.segments)-1].index[e.key] = db.outOffset
		db.outOffset += int64(n)
	}
	return err
}
