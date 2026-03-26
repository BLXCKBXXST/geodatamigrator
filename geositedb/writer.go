// Package geositedb implements the sing-box geosite.db binary format writer.
//
// Format specification (reverse-engineered from sing-box source):
//
//	[version: 1 byte = 0x00]
//	[entry_count: uvarint]
//	for each entry (sorted by code):
//	    [code: vstring]       // length-prefixed UTF-8 string (uvarint length + bytes)
//	    [data_offset: uvarint] // offset into the data section
//	    [item_count: uvarint]  // number of domain items
//	[data section]:
//	for each entry (sorted by code):
//	    for each item:
//	        [type: 1 byte]     // 0=Domain, 1=DomainSuffix, 2=DomainKeyword, 3=DomainRegex
//	        [value: vstring]
package geositedb

import (
	"bytes"
	"encoding/binary"
	"io"
	"sort"
)

// ItemType matches sing-box geosite item types.
const (
	RuleTypeDomain        byte = 0
	RuleTypeDomainSuffix  byte = 1
	RuleTypeDomainKeyword byte = 2
	RuleTypeDomainRegex   byte = 3
)

// Item represents a single domain rule in sing-box format.
type Item struct {
	Type  byte
	Value string
}

// writeUvarint writes a uvarint to w.
func writeUvarint(w io.Writer, x uint64) error {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], x)
	_, err := w.Write(buf[:n])
	return err
}

// writeVString writes a length-prefixed string (uvarint length + bytes).
func writeVString(w io.Writer, s string) error {
	if err := writeUvarint(w, uint64(len(s))); err != nil {
		return err
	}
	_, err := io.WriteString(w, s)
	return err
}

// Write writes a geosite.db file in sing-box format.
// domains is a map of code → []Item.
func Write(w io.Writer, domains map[string][]Item) error {
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(domains))
	for code := range domains {
		keys = append(keys, code)
	}
	sort.Strings(keys)

	// Build data section in a buffer to calculate offsets.
	var dataBuf bytes.Buffer
	offsets := make(map[string]int)
	for _, code := range keys {
		offsets[code] = dataBuf.Len()
		for _, item := range domains[code] {
			dataBuf.WriteByte(item.Type)
			if err := writeVString(&dataBuf, item.Value); err != nil {
				return err
			}
		}
	}

	// Write header: version byte.
	if _, err := w.Write([]byte{0}); err != nil {
		return err
	}

	// Write entry count.
	if err := writeUvarint(w, uint64(len(keys))); err != nil {
		return err
	}

	// Write index entries.
	for _, code := range keys {
		if err := writeVString(w, code); err != nil {
			return err
		}
		if err := writeUvarint(w, uint64(offsets[code])); err != nil {
			return err
		}
		if err := writeUvarint(w, uint64(len(domains[code]))); err != nil {
			return err
		}
	}

	// Write data section.
	_, err := w.Write(dataBuf.Bytes())
	return err
}
