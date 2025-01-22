package utils

import (
	"testing"

	"bytes"
	"database/sql"
	_ "modernc.org/sqlite" // The underscore registers the driver without directly using it
	"os"
)

func copyFile(src string, dst string) error {
	input, _ := os.ReadFile(src)
	err := os.WriteFile(dst, input, 0644)
	return err
}

func TestGetRpmSQLiteDBPackageData(t *testing.T) {
	dbCopyPath := "test_data/rpmdb_test.sqlite"
	err := copyFile("test_data/rpmdb.sqlite", dbCopyPath)
	if err != nil {
		t.Fatalf("failed copying rpm db: %v", err)
	}
	db, _ := sql.Open("sqlite", dbCopyPath)
	defer db.Close()
	defer os.Remove(dbCopyPath)

	hnum, blob, err := getRpmSQLiteDBPackageData(db, "test")
	if err != nil {
		t.Fatalf("failed getting rpm db package data: %v", err)
	}

	if hnum != 1 {
		t.Fatalf("unexpected hnum: %d", hnum)
	}

	header := createHeaderBlob(blob)
	if header == nil {
		t.Fatalf("failed to create header blob")
	}

	if len(header.entryMapping.Keys()) != 3 {
		t.Fatalf("unexpected entry count: %d", len(header.entryMapping.Keys()))
	}

	entry := header.getEntry(1000)
	if entry.Tag != 1000 || entry.Type != 6 || entry.Offset != 0 || entry.Count != 1 || string(entry.Content) != "test\x00" {
		t.Fatalf("unexpected entry: %v", entry)
	}
}

func TestUpdatePackageSQLite(t *testing.T) {
	dbCopyPath := "test_data/rpmdb_test.sqlite"
	err := copyFile("test_data/rpmdb.sqlite", dbCopyPath)
	if err != nil {
		t.Fatalf("failed copying rpm db: %v", err)
	}
	db, _ := sql.Open("sqlite", dbCopyPath)
	defer db.Close()
	defer os.Remove(dbCopyPath)

	newBlob := []byte{
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x0b, // entry count + content len
		0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x06, // tag1 + type1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // offset1 + count1
		0x00, 0x00, 0x03, 0xe9, 0x00, 0x00, 0x00, 0x06, // tag2 + type2
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, // offset2 + count2
		0x74, 0x65, 0x73, 0x74, 0x00, 0x31, 0x2e, 0x32, // data1 + data2
		0x2e, 0x33, 0x00,
	}

	err = updatePackageSQLite(db, 1, newBlob, "seal-test")
	if err != nil {
		t.Fatalf("failed updating package: %v", err)
	}

	hnum, blob, err := getRpmSQLiteDBPackageData(db, "seal-test")
	if err != nil {
		t.Fatalf("failed getting rpm db package data: %v", err)
	}

	if !bytes.Equal(blob, newBlob) {
		t.Fatalf("unexpected blob: %v", blob)
	}

	var sha1 string
	err = db.QueryRow("SELECT key FROM Sha1header WHERE hnum = ?", hnum).Scan(&sha1)
	if err == nil || err.Error() != "sql: no rows in result set" {
		t.Fatalf("failed querying rpm db: %v", err)
	}
}
