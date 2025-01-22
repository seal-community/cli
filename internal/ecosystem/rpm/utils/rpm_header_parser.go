package utils

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log/slog"

	"strconv"

	"github.com/iancoleman/orderedmap"
)

const integerType = 4
const entryOffset = 8
const entrySize = 16

// Header Structure is as follows:
// int32 - entry count, int32 - content len, entryInfo1, entryInfo2, ..., entryInfoN, content
// content includes all the data for the entries one after the other
type entryInfo struct {
	Tag     int
	Type    int
	Offset  int
	Count   int
	Content []byte
}

type headerBlob struct {
	entryMapping orderedmap.OrderedMap
}

func readInt(blob []byte, offset int) int {
	return int(binary.BigEndian.Uint32(blob[offset : offset+4]))
}

func intToBytes(i int) []byte {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, uint32(i))
	return buf
}

func readEntryInfo(blob []byte, offset int) *entryInfo {
	return &entryInfo{
		Tag:     readInt(blob, offset),
		Type:    readInt(blob, offset+4),
		Offset:  readInt(blob, offset+8),
		Count:   readInt(blob, offset+12),
		Content: []byte{},
	}
}

func createHeaderBlob(blob []byte) *headerBlob {
	header := &headerBlob{
		entryMapping: *orderedmap.New(),
	}
	header.initialize(blob)
	slog.Debug("header blob created successfully", "header", header)
	return header
}

func (h *headerBlob) initialize(blob []byte) {
	tagCount := readInt(blob, 0)
	contentStart := entryOffset + tagCount*entrySize
	var prevEntry, entry *entryInfo

	slog.Debug("initializing header blob", "tagCount", tagCount, "contentStart", contentStart)
	blobContentSegment := blob[contentStart:]
	for i := 0; i < int(tagCount); i++ {
		entry = readEntryInfo(blob, entryOffset+i*entrySize)
		if prevEntry != nil && prevEntry.Offset < entry.Offset {
			prevEntry.Content = blobContentSegment[prevEntry.Offset:entry.Offset]
		}
		h.entryMapping.Set(fmt.Sprintf("%d", entry.Tag), entry)
		prevEntry = entry
	}

	if entry != nil {
		entry.Content = blobContentSegment[entry.Offset:]
		h.entryMapping.Set(fmt.Sprintf("%d", entry.Tag), entry)
	}
}

func (h *headerBlob) getEntry(tag_key int) *entryInfo {
	tag, _ := h.entryMapping.Get(fmt.Sprintf("%d", tag_key))
	return tag.(*entryInfo)
}

// Iterate over the ordered map and return the values
// When iterating over the map, only the order of the keys is preserved
func (h *headerBlob) iterateValues() []*entryInfo {
	var entryList []*entryInfo
	for _, entryKey := range h.entryMapping.Keys() {
		entryKeyInt, _ := strconv.Atoi(entryKey)
		entryList = append(entryList, h.getEntry(entryKeyInt))
	}
	return entryList
}

func (h *headerBlob) hasEntry(tag_key int) bool {
	_, exists := h.entryMapping.Get(fmt.Sprintf("%d", tag_key))
	return exists
}

func (h *headerBlob) removeEntry(entry_key int) {
	slog.Debug("removing entry", "entry_key", entry_key)
	removedEntry := h.getEntry(entry_key)
	for _, entry := range h.iterateValues() {
		if entry.Offset > removedEntry.Offset {
			entry.Offset -= len(removedEntry.Content)
		}
	}
	h.entryMapping.Delete(fmt.Sprintf("%d", entry_key))
	slog.Debug("entry removed successfully", "entry_key", entry_key)
}

func (h *headerBlob) alignContents() {
	var prevEntry *entryInfo
	for _, entry := range h.iterateValues() {
		// Align int content to 4 bytes
		if entry.Type == integerType && entry.Offset%4 != 0 {
			slog.Debug("aligning content before integer", "entry tag", entry.Tag)
			padding := make([]byte, (4 - entry.Offset%4))
			newPrevContent := append(prevEntry.Content, padding...)
			if bytes.Count(newPrevContent, []byte{0}) > 4 {
				newPrevContent = bytes.ReplaceAll(newPrevContent, []byte{0, 0, 0, 0}, []byte{})
			}
			h.modifyTagContent(prevEntry.Tag, newPrevContent)
		}
		prevEntry = entry
	}
}

func (h *headerBlob) modifyTagContent(tag_key int, newContent []byte) {
	ModifiedTag := h.getEntry(tag_key)
	lenDiff := len(newContent) - len(ModifiedTag.Content)
	slog.Debug("modifying tag content", "tag_key", tag_key, "lenDiff", lenDiff)

	for _, entry := range h.iterateValues() {
		if entry.Offset > ModifiedTag.Offset {
			entry.Offset += lenDiff
		}
	}

	ModifiedTag.Content = newContent
	h.alignContents()
}

func (h *headerBlob) contentLen() int {
	contentLen := 0
	for _, entry := range h.iterateValues() {
		contentLen += len(entry.Content)
	}
	return contentLen
}

func (e *entryInfo) dumpBytes() []byte {
	buf := make([]byte, 16)
	binary.BigEndian.PutUint32(buf[0:4], uint32(e.Tag))
	binary.BigEndian.PutUint32(buf[4:8], uint32(e.Type))
	binary.BigEndian.PutUint32(buf[8:12], uint32(e.Offset))
	binary.BigEndian.PutUint32(buf[12:16], uint32(e.Count))
	return buf
}

func (h *headerBlob) dumpBytes() []byte {
	contentBlob := intToBytes(len(h.entryMapping.Keys()))
	contentBlob = append(contentBlob, intToBytes(h.contentLen())...)

	for _, entry := range h.iterateValues() {
		contentBlob = append(contentBlob, entry.dumpBytes()...)
	}
	for _, entry := range h.iterateValues() {
		contentBlob = append(contentBlob, entry.Content...)
	}
	return contentBlob
}
