package datastore

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEntry_Encode(t *testing.T) {
	e := Entry{"tK", "tV"}
	data := e.Encode()
	e.Decode(data)
	if e.GetLength() != 16 {
		t.Error("Incorrect length")
	}
	if e.key != "tK" {
		t.Error("Incorrect key")
	}
	if e.value != "tV" {
		t.Error("Incorrect value")
	}
}

func TestReadValue(t *testing.T) {
	e := Entry{"tK", "tV"}
	data := e.Encode()
	readData := bytes.NewReader(data)
	bReadData := bufio.NewReader(readData)
	value, err := readValue(bReadData)
	if err != nil {
		t.Fatal(err)
	}
	if value != "tV" {
		t.Errorf("Wrong value: [%s]", value)
	}
}
