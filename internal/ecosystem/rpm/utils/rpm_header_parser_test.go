package utils

import (
	"testing"

	"github.com/iancoleman/orderedmap"
)

func TestInitialize(t *testing.T) {
	blob := []byte{
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x0b, // entry count + content len
		0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x06, // tag1 + type1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // offset1 + count1
		0x00, 0x00, 0x03, 0xe9, 0x00, 0x00, 0x00, 0x06, // tag2 + type2
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, // offset2 + count2
		0x74, 0x65, 0x73, 0x74, 0x00, 0x31, 0x2e, 0x32, // data1 + data2
		0x2e, 0x33, 0x00,
	}
	header := createHeaderBlob(blob)
	if header == nil {
		t.Fatalf("failed to create header blob")
	}

	if len(header.entryMapping.Keys()) != 2 {
		t.Fatalf("unexpected entry count: %d", len(header.entryMapping.Keys()))
	}

	entry := header.getEntry(1000)
	expected := &entryInfo{
		Tag:     1000,
		Type:    6,
		Offset:  0,
		Count:   1,
		Content: []byte("test\x00"),
	}
	if entry == expected {
		t.Fatalf("unexpected entry: %v", entry)
	}

	entry = header.getEntry(1001)
	expected = &entryInfo{
		Tag:     1001,
		Type:    6,
		Offset:  5,
		Count:   1,
		Content: []byte("1.2.3\x00"),
	}
	if entry == expected {
		t.Fatalf("unexpected entry: %v", entry)
	}
}

func TestRemoveEntry(t *testing.T) {
	blob := []byte{
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x0b, // entry count + content len
		0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x06, // tag1 + type1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // offset1 + count1
		0x00, 0x00, 0x03, 0xe9, 0x00, 0x00, 0x00, 0x06, // tag2 + type2
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, // offset2 + count2
		0x74, 0x65, 0x73, 0x74, 0x00, 0x31, 0x2e, 0x32, // data1 + data2
		0x2e, 0x33, 0x00,
	}
	header := createHeaderBlob(blob)
	if header == nil {
		t.Fatalf("failed to create header blob")
	}
	header.removeEntry(1000)
	entry := header.getEntry(1001)
	expected := &entryInfo{
		Tag:     1001,
		Type:    6,
		Offset:  0,
		Count:   1,
		Content: []byte("1.2.3\x00"),
	}
	if entry == expected {
		t.Fatalf("unexpected entry: %v", entry)
	}
}

func TestModifyEntry(t *testing.T) {
	blob := []byte{
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x09, // entry count + content len
		0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x06, // tag1 + type1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // offset1 + count1
		0x00, 0x00, 0x03, 0xed, 0x00, 0x00, 0x00, 0x04, // tag2 + type2
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, // offset2 + count2
		0x74, 0x65, 0x73, 0x74, 0x00, 0x00, 0x00, 0x00, // data1 + data2
		0x01,
	}
	header := createHeaderBlob(blob)
	if header == nil {
		t.Fatalf("failed to create header blob")
	}
	header.modifyTagContent(1000, []byte("seal-test\x00"))
	strEntry := header.getEntry(1000)
	expected := &entryInfo{
		Tag:     1000,
		Type:    6,
		Offset:  0,
		Count:   1,
		Content: []byte("seal-test\x00\x00\x00"),
	}
	if strEntry == expected {
		t.Fatalf("unexpected entry: %v", strEntry)
	}

	intEntry := header.getEntry(1005)
	expected = &entryInfo{
		Tag:     1005,
		Type:    4,  // Integer type
		Offset:  12, // Must be aligned to 4 bytes
		Count:   1,
		Content: []byte{0x00, 0x00, 0x00, 0x01},
	}
	if intEntry == expected {
		t.Fatalf("unexpected entry: %v", intEntry)
	}
}

func TestDumpBytes(t *testing.T) {
	header := &headerBlob{
		entryMapping: *orderedmap.New(),
	}
	header.entryMapping.Set("1000", &entryInfo{
		Tag:     1000,
		Type:    6,
		Offset:  0,
		Count:   1,
		Content: []byte("test\x00"),
	})
	header.entryMapping.Set("1001", &entryInfo{
		Tag:     1001,
		Type:    6,
		Offset:  5,
		Count:   1,
		Content: []byte("1.2.3\x00"),
	})
	blob := header.dumpBytes()
	expected := []byte{
		0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x0b, // entry count + content len
		0x00, 0x00, 0x03, 0xe8, 0x00, 0x00, 0x00, 0x06, // tag1 + type1
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, // offset1 + count1
		0x00, 0x00, 0x03, 0xe9, 0x00, 0x00, 0x00, 0x06, // tag2 + type2
		0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x01, // offset2 + count2
		0x74, 0x65, 0x73, 0x74, 0x00, 0x31, 0x2e, 0x32, // data1 + data2
		0x2e, 0x33, 0x00,
	}
	if len(blob) != len(expected) {
		t.Fatalf("unexpected blob length: %d", len(blob))
	}
	for i := 0; i < len(blob); i++ {
		if blob[i] != expected[i] {
			t.Fatalf("unexpected blob: %v", blob)
		}
	}
}

func TestHeaderIterateValues(t *testing.T) {
	header := &headerBlob{
		entryMapping: *orderedmap.New(),
	}
	entry1 := &entryInfo{
		Tag:     1000,
		Type:    6,
		Offset:  0,
		Count:   1,
		Content: []byte("test\x00"),
	}
	entry2 := &entryInfo{
		Tag:     200,
		Type:    6,
		Offset:  5,
		Count:   1,
		Content: []byte("1.2.3\x00"),
	}

	header.entryMapping.Set("1000", entry1)
	// The order of the keys "1000" and "200" would reverse on values iteration
	header.entryMapping.Set("200", entry2)
	entryList := header.iterateValues()
	if len(entryList) != 2 {
		t.Fatalf("unexpected entry count: %d", len(entryList))
	}
	if entryList[0] != entry1 || entryList[1] != entry2 {
		t.Fatalf("Order of entries is not preserved")
	}
}
