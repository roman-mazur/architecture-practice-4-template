package datastore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDb_Put(t *testing.T) {
	saveDirectory, err := ioutil.TempDir("", "temp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(saveDirectory)

	db, err := NewDb(saveDirectory, 48)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pairs := [][]string{
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
	}
	finalPath := filepath.Join(saveDirectory, outFileName+"0")
	outFile, err := os.Open(finalPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put & get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[0], err)
			}
			actual, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Unable to retrieve %s: %s", pair[0], err)
			}
			if actual != pair[1] {
				t.Errorf("Invalid value returned: expected: %s, actual: %s.", pair[1], actual)
			}
		}
	})

	outInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	expectedStateSize := outInfo.Size()

	t.Run("increase file size", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[0], err)
			}
		}
		t.Log(db)
		outInfo, err := outFile.Stat()
		actualStateSize := outInfo.Size()
		if err != nil {
			t.Fatal(err)
		}
		if expectedStateSize != actualStateSize {
			t.Errorf("Size mismatch: expected: %d, actual: %d.", expectedStateSize, actualStateSize)
		}
	})

	t.Run("new process is created", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = NewDb(saveDirectory, 48)
		if err != nil {
			t.Fatal(err)
		}

		for _, pair := range pairs {
			actual, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Unable to place %s: %s.", pair[1], err)
			}
			expected := pair[1]
			if actual != expected {
				t.Errorf("Invalid value returned: expected: %s, actual: %s.", expected, actual)
			}
		}
	})
}

func TestDb_Segmentation(t *testing.T) {
	saveDirectory, err := ioutil.TempDir("", "temp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(saveDirectory)

	db, err := NewDb(saveDirectory, 35)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("create new file", func(t *testing.T) {
		db.Put("1", "v1")
		db.Put("2", "v2")
		db.Put("3", "v3")
		db.Put("2", "v5")
		actualTwoFiles := len(db.segments)
		expected2Files := 2
		if actualTwoFiles != expected2Files {
			t.Errorf("Segmentation error: expected 2 files, but received %d.", len(db.segments))
		}
	})

	t.Run("segmentation start", func(t *testing.T) {
		db.Put("4", "v4")
		actualTreeFiles := len(db.segments)
		expected3Files := 3
		if actualTreeFiles != expected3Files {
			t.Errorf("Segmentation error: expected 3 files, but received %d.", len(db.segments))
		}

		time.Sleep(2 * time.Second)

		actualTwoFiles := len(db.segments)
		expected2Files := 2
		if actualTwoFiles != expected2Files {
			t.Errorf("Segmentation error: expected 2 files, but received %d.", len(db.segments))
		}
	})

	t.Run("store new values of duplicate keys", func(t *testing.T) {
		actual, _ := db.Get("2")
		expected := "v5"
		if actual != expected {
			t.Errorf("Segmentation error: expected value: %s, actual: %s", expected, actual)
		}
	})

	t.Run("size check", func(t *testing.T) {
		file, err := os.Open(db.segments[0].filePath)
		defer file.Close()

		if err != nil {
			t.Error(err)
		}
		inf, _ := file.Stat()
		actual := inf.Size()
		expected := int64(45)
		if actual != expected {
			t.Errorf("Segmentation error: expected size: %d, actual: %d", expected, actual)
		}
	})
}
