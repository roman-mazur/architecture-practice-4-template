package datastore

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const segFilePrefix = "segment-"

var ErrNotFound = fmt.Errorf("record does not exist")

type keyValuePosition struct {
	position int64
	segment  int
}

type hashIndex map[string]keyValuePosition

type Db struct {
	out       *os.File
	outPath   string
	outOffset int64

	segSizeLimit int64
	segments     []*os.File

	index hashIndex
}

func NewDb(dir string, segSizeLimit int64) (*Db, error) {
	db := &Db{
		segSizeLimit: segSizeLimit,
		outPath:      dir,
		index:        make(hashIndex),
	}
	// read directory to find existing segment files
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var segFiles []string
	for _, f := range files {
		// collect segment files
		if strings.HasPrefix(f.Name(), segFilePrefix) {
			segFiles = append(segFiles, f.Name())
		}
	}

	// sort segFiles, assuming they have format 'segment-N'
	sort.Slice(segFiles, func(i, j int) bool {
		// converting the file name back into segment number for sorting
		numi, _ := strconv.Atoi(strings.TrimPrefix(segFiles[i], segFilePrefix))
		numj, _ := strconv.Atoi(strings.TrimPrefix(segFiles[j], segFilePrefix))
		return numi < numj
	})
	for _, segFile := range segFiles {
		// Open each segment file and append to segments slice
		f, err := os.OpenFile(filepath.Join(dir, segFile), os.O_APPEND|os.O_RDWR, 0o600)
		if err != nil {
			return nil, err
		}
		db.segments = append(db.segments, f)
	}

	// take the last file as the current output file
	if len(db.segments) > 0 {
		db.out = db.segments[len(db.segments)-1]
		db.segments = db.segments[:len(db.segments)-1]
		stat, _ := db.out.Stat()
		db.outOffset = stat.Size()
	}

	// if no file is found, create new output file
	if db.out == nil {
		outputPath := filepath.Join(dir, fmt.Sprintf("%s%d", segFilePrefix, len(db.segments)))
		f, err := os.OpenFile(outputPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			return nil, err
		}
		db.out = f
	}

	err = db.recover()
	if err != nil && err != io.EOF {
		return nil, err
	}
	return db, nil
}

const bufSize = 8192

func (db *Db) recover() error {
	// start recovery from oldest segment
	for i := range db.segments {
		if err := db.recoverSegment(db.segments[i], i); err != nil {
			return err
		}
	}
	// recover the latest segment
	return db.recoverSegment(db.out, len(db.segments))
}

func (db *Db) recoverSegment(f *os.File, segmentIndex int) error {
	stat, _ := f.Stat()
	var buf [bufSize]byte
	in := bufio.NewReaderSize(f, bufSize)

	var offset int64 = 0 // maintain the offset in the segment
	for offset < stat.Size() {
		header, err := in.Peek(bufSize)
		if err != nil && err != io.EOF {
			return err
		}
		size := binary.LittleEndian.Uint32(header)

		var data []byte
		if size < bufSize {
			data = buf[:size]
		} else {
			data = make([]byte, size)
		}

		n, err := in.Read(data)
		if err != nil {
			return err
		}

		if n != int(size) {
			return fmt.Errorf("corrupted file")
		}

		// Stores or updates the offset of the most recent instance of key
		var e entry
		e.Decode(data)
		// store both the offset and the segment index in the hashIndex
		db.index[e.key] = keyValuePosition{
			position: offset,
			segment:  segmentIndex,
		}

		offset += int64(n)
	}
	return nil
}

func (db *Db) Close() error {
	return db.out.Close()
}

func (db *Db) Get(key string) (string, error) {
	kvp, ok := db.index[key]
	if !ok {
		return "", ErrNotFound
	}

	position := kvp.position
	segmentIndex := kvp.segment
	// Try to find the segment where the key is located
	var file *os.File

	// Check if the key resides in the current segment.
	if segmentIndex == len(db.segments) {
		file = db.out
	} else if segmentIndex < len(db.segments) {
		var err error
		file, err = os.OpenFile(db.segments[segmentIndex].Name(), os.O_RDONLY, 0o600)
		if err != nil {
			return "", err
		}
		defer file.Close()
	} else {
		return "", ErrNotFound
	}

	_, err := file.Seek(position, 0)
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

func (db *Db) Put(key, value string) error {
	e := entry{
		key:   key,
		value: value,
	}
	data := e.Encode()
	n, err := db.out.Write(data)
	if err != nil {
		return err
	}

	// Update index and current file offset
	db.index[key] = keyValuePosition{
		position: db.outOffset,
		segment:  len(db.segments),
	}
	db.outOffset += int64(n)

	// If the current segment file size limit is reached, create a new segment
	if db.outOffset >= db.segSizeLimit {
		db.out.Close()
		db.segments = append(db.segments, db.out)

		newOutPath := filepath.Join(db.outPath, fmt.Sprintf("%s%d", segFilePrefix, len(db.segments)))
		f, err := os.OpenFile(newOutPath, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			return err
		}
		db.out = f
		db.outOffset = 0
	}

	return nil
}
