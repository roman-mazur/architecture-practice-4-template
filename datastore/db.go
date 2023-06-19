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

type IndexOp struct {
	isWrite bool
	key     string
	index   int64
}

type KeyPosition struct {
	segment  *Segment
	position int64
}

type Db struct {
	out       *os.File
	outOffset int64
	dir       string
	segSize   int64
	segIndex  int
	segments  []*Segment

	indexOps     chan IndexOp
	keyPositions chan *KeyPosition
	putOps       chan entry
	putDone      chan error
}

type Segment struct {
	index    hashIndex
	filePath string
}

func NewDb(dir string, segmentSize int64) (*Db, error) {
	db := &Db{
		segments:     make([]*Segment, 0),
		dir:          dir,
		segSize:      segmentSize,
		indexOps:     make(chan IndexOp),
		keyPositions: make(chan *KeyPosition),
		putOps:       make(chan entry),
		putDone:      make(chan error),
	}
	err := db.makeSegment()
	if err != nil {
		return nil, err
	}
	err = db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}

	db.startIndexRoutine()
	db.startPutRoutine()

	return db, nil
}

func (db *Db) startIndexRoutine() {
	go func() {
		for {
			op := <-db.indexOps
			if op.isWrite {
				db.setKey(op.key, op.index)
			} else {
				s, p, err := db.getSegmentAndPosition(op.key)
				if err != nil {
					db.keyPositions <- nil
				} else {
					db.keyPositions <- &KeyPosition{
						segment:  s,
						position: p,
					}
				}
			}
		}
	}()
}

func (db *Db) setKey(key string, index int64) {
	db.segments[len(db.segments)-1].index[key] = db.outOffset
	db.outOffset += index
}

func (db *Db) getSegmentAndPosition(key string) (*Segment, int64, error) {
	for i := len(db.segments) - 1; i >= 0; i-- {
		segment := db.segments[i]
		position, ok := segment.index[key]
		if ok {
			return segment, position, nil
		}
	}
	return nil, 0, ErrNotFound
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
	for _, segment := range db.segments {
		file, err := os.Open(segment.filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		reader := bufio.NewReaderSize(file, bufSize)
		for err == nil {
			var (
				header, data []byte
				n            int
			)
			header, err = reader.Peek(bufSize)
			if err == io.EOF {
				if len(header) == 0 {
					break
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
			n, err = io.ReadFull(reader, data)

			if err == nil {
				if n != int(size) {
					return fmt.Errorf("corrupted file")
				}

				var e entry
				e.Decode(data)
				db.setKey(e.key, int64(n))
			}
		}
	}

	return err
}

func (db *Db) Close() error {
	return db.out.Close()
}

func (db *Db) Get(key string) (string, error) {
	keyPos := db.getPos(key)
	if keyPos == nil {
		return "", ErrNotFound
	}
	mark, err := keyPos.segment.getMark(keyPos.position)
	if err != nil {
		return "", err
	}
	if mark == 1 {
		return "", ErrNotFound
	}
	value, err := keyPos.segment.getFromSegment(keyPos.position)
	if err != nil {
		return "", err
	}
	return value, nil
}

func (db *Db) getPos(key string) *KeyPosition {
	op := IndexOp{
		isWrite: false,
		key:     key,
	}
	db.indexOps <- op
	keyPos := <-db.keyPositions
	return keyPos
}

func (seg *Segment) getMark(position int64) (int, error) {
	file, err := os.Open(seg.filePath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	_, err = file.Seek(position, 0)
	if err != nil {
		return -1, err
	}

	reader := bufio.NewReader(file)
	mark, err := readMark(reader)
	if err != nil {
		return -1, err
	}
	return mark, nil
}

func (seg *Segment) getFromSegment(position int64) (string, error) {
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

func (db *Db) startPutRoutine() {
	go func() {
		for {
			e := <-db.putOps
			length := e.Length()
			stat, err := db.out.Stat()
			if err != nil {
				db.putDone <- err
				continue
			}
			if stat.Size()+length > db.segSize {
				err := db.makeSegment()
				if err != nil {
					db.putDone <- err
					continue
				}
			}
			n, err := db.out.Write(e.Encode())
			if err == nil {
				db.indexOps <- IndexOp{
					isWrite: true,
					key:     e.key,
					index:   int64(n),
				}
			}
			db.putDone <- nil
		}
	}()
}

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
		mark:  0,
	}
	db.putOps <- e
	return <-db.putDone
}

func (db *Db) Delete(key string) error {
	e := entry{
		key:  key,
		mark: 1,
	}
	db.putOps <- e
	return <-db.putDone
}
