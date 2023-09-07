package main

import (
	"os"
	"testing"
)

func TestDbPath(t *testing.T) {
	p, err := dbPath("fromCmd")
	if p != "fromCmd" || err != nil {
		t.Errorf("Got %s, want fromCmd", p)
	}

	os.Setenv("TICKTOCK_DB", "test/mydb")
	p, err = dbPath("")
	want := "test/mydb"
	if p != want || err != nil {
		t.Errorf("TICKTOCK_DB exists, got %s, want %s", p, want)
	}
	os.Unsetenv("TICKTOCK_DB")

	os.Setenv("XDG_DATA_HOME", "test/myXdgHome")
	os.Setenv("HOME", "test/myHome")
	p, err = dbPath("")
	want = "test/myXdgHome/ticktock/db"
	if p != want || err != nil {
		t.Errorf("XDG_DATA_HOME exists, got %s, want %s", p, want)
	}

	os.Unsetenv("XDG_DATA_HOME")
	p, err = dbPath("")
	want = "test/myHome/.local/share/ticktock/db"
	if p != want || err != nil {
		t.Errorf("XDG_DATA_HOME not exists, HOME exists,  got %s, want %s", p, want)
	}

	os.Unsetenv("HOME")
	_, err = dbPath("")
	if err == nil {
		t.Errorf("Both XDG_DATA_HOME and HOME does not exist, expect error.")
	}
}
