package datastore

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDb_Put(t *testing.T) {
	dir, err := ioutil.TempDir("", "test-db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDb(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	pairs := [][]string {
		{"key1", "value1"},
		{"key2", "value2"},
		{"key3", "value3"},
	}

	outFile, err := os.Open(filepath.Join(dir, outFileName))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("put/get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot get %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

	outInfo, err := outFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	size1 := outInfo.Size()

	t.Run("file growth", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
		}
		outInfo, err := outFile.Stat()
		if err != nil {
			t.Fatal(err)
		}
		if size1 * 2 != outInfo.Size() {
			t.Errorf("Unexpected size (%d vs %d)", size1, outInfo.Size())
		}
	})

	t.Run("new db process", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = NewDb(dir)
		if err != nil {
			t.Fatal(err)
		}

		for _, pair := range pairs {
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

}

func TestDb_Segments_Merge(t *testing.T) {
	saveDirectory, err := ioutil.TempDir("", "testDir")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(saveDirectory)

	db, err := NewDb(saveDirectory, 45) // size as 2nd argument
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	t.Run("check creation of new segment", func(t *testing.T) {
		db.Put("key1", "v1")
		db.Put("key2", "v2")
		db.Put("key3", "v3")
		db.Put("key1", "V1")
		db.Put("key4", "v4")
		actual := len(db.segments)
		expected := 2
		if actual != expected {
			t.Errorf("Expected %d files, but received %d.", expected, actual)
		}
	})

	t.Run("check starting segmentation", func(t *testing.T) {
		db.Put("key2", "V2")
		actual := len(db.segments)
		expected := 3
		if actual != expected {
			t.Errorf("Expected %d files before merge, but received %d.", expected, actual)
		}

		time.Sleep(2 * time.Second)

		actualAfterMerge := len(db.segments)
		expectedAfterMerge := 2
		if actual != expected {
			t.Errorf("Expected %d files after merge, but received %d.", expectedAfterMerge, actualAfterMerge)
		}
	})

	t.Run("check not storing new values of duplicate keys", func(t *testing.T) {
		actual, err := db.Get("key2")
		if err != nil {
			t.Error(err)
		}
		expected := "V2"
		if actual != expected {
			t.Errorf("An error occurred during segmentation. Expected value: %s, Actual one: %s", expected, actual)
		}
	})

	t.Run("check size", func(t *testing.T) {
		file, err := os.Open(db.segments[0].filePath)
		defer file.Close()
		if err != nil {
			t.Error(err)
		}
		
		inf, _ := file.Stat()
		actual := inf.Size()
		expected := int64(45)
		if actual != expected {
			t.Errorf("An error occurred during segmentation. Expected size %d, Actual one: %d", expected, actual)
		}
	})
}
