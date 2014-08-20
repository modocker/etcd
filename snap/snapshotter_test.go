package snap

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/coreos/etcd/raft"
)

var testSnap = &raft.Snapshot{
	ClusterId: 0xBEEF,
	Data:      []byte("some snapshot"),
	Nodes:     []int64{1, 2, 3},
	Index:     1,
	Term:      1,
}

func TestSaveAndLoad(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)
	ss := New(dir)
	err := ss.Save(testSnap)
	if err != nil {
		t.Fatal(err)
	}

	g, err := ss.Load()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if !reflect.DeepEqual(g, testSnap) {
		t.Errorf("snap = %#v, want %#v", g, testSnap)
	}
}

func TestBadCRC(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)
	ss := New(dir)
	err := ss.Save(testSnap)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { crcTable = crc32.MakeTable(crc32.Castagnoli) }()
	// switch to use another crc table
	// fake a crc mismatch
	crcTable = crc32.MakeTable(crc32.Koopman)

	_, err = ss.Load()
	if err == nil || err != ErrCRCMismatch {
		t.Errorf("err = %v, want %v", err, ErrCRCMismatch)
	}
}

func TestFailback(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)

	large := fmt.Sprintf("%016x-%016x-%016x.snap", 0xFFFF, 0xFFFF, 0xFFFF)
	err := ioutil.WriteFile(path.Join(dir, large), []byte("bad data"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	ss := New(dir)
	err = ss.Save(testSnap)
	if err != nil {
		t.Fatal(err)
	}

	g, err := ss.Load()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if !reflect.DeepEqual(g, testSnap) {
		t.Errorf("snap = %#v, want %#v", g, testSnap)
	}
}

func TestSnapNames(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)
	for i := 1; i <= 5; i++ {
		if f, err := os.Create(path.Join(dir, fmt.Sprintf("%d", i))); err != nil {
			t.Fatal(err)
		} else {
			f.Close()
		}
	}
	ss := New(dir)
	names, err := ss.snapNames()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if len(names) != 5 {
		t.Errorf("len = %d, want 10", len(names))
	}
	w := []string{"5", "4", "3", "2", "1"}
	if !reflect.DeepEqual(names, w) {
		t.Errorf("names = %v, want %v", names, w)
	}
}

func TestLoadNewestSnap(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)
	ss := New(dir)
	err := ss.Save(testSnap)
	if err != nil {
		t.Fatal(err)
	}

	newSnap := *testSnap
	newSnap.Index = 5
	err = ss.Save(&newSnap)
	if err != nil {
		t.Fatal(err)
	}

	g, err := ss.Load()
	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
	if !reflect.DeepEqual(g, &newSnap) {
		t.Errorf("snap = %#v, want %#v", g, &newSnap)
	}
}

func TestNoSnapshot(t *testing.T) {
	dir := path.Join(os.TempDir(), "snapshot")
	os.Mkdir(dir, 0700)
	defer os.RemoveAll(dir)
	ss := New(dir)
	_, err := ss.Load()
	if err == nil || err != ErrNoSnapshot {
		t.Errorf("err = %v, want %v", err, ErrNoSnapshot)
	}
}
