package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

type DBFField struct {
	Name         string
	Type         byte
	Length       int
	DecimalCount int
	Offset       int
}

type DBFFile struct {
	NumRecords int
	RecordSize int
	Fields     []DBFField
	DataOffset int64
	file       *os.File
}

func OpenDBF(path string) (*DBFFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	header := make([]byte, 32)
	if _, err := io.ReadFull(f, header); err != nil {
		f.Close()
		return nil, fmt.Errorf("error reading header: %w", err)
	}

	numRecords := int(binary.LittleEndian.Uint32(header[4:8]))
	headerLen := int(binary.LittleEndian.Uint16(header[8:10]))
	recordSize := int(binary.LittleEndian.Uint16(header[10:12]))

	dbf := &DBFFile{
		NumRecords: numRecords,
		RecordSize: recordSize,
		file:       f,
	}

	offset := 0
	for i := 32; i < headerLen-1; i += 32 {
		fieldBytes := make([]byte, 32)
		if _, err := io.ReadFull(f, fieldBytes); err != nil {
			f.Close()
			return nil, fmt.Errorf("error reading field: %w", err)
		}
		if fieldBytes[0] == 0x0D {
			break
		}
		name := strings.TrimRight(string(fieldBytes[0:11]), "\x00 ")
		ftype := fieldBytes[11]
		flen := int(fieldBytes[16])
		fdec := int(fieldBytes[17])

		dbf.Fields = append(dbf.Fields, DBFField{
			Name:         name,
			Type:         ftype,
			Length:       flen,
			DecimalCount: fdec,
			Offset:       offset + 1,
		})
		offset += flen
	}

	dbf.DataOffset = int64(headerLen)
	return dbf, nil
}

func (d *DBFFile) Close() error {
	return d.file.Close()
}

func (d *DBFFile) ReadRecord(idx int) ([]byte, error) {
	offset := d.DataOffset + int64(idx)*int64(d.RecordSize)
	record := make([]byte, d.RecordSize)
	_, err := d.file.ReadAt(record, offset)
	if err != nil {
		return nil, err
	}
	return record, nil
}

func (d *DBFFile) GetString(record []byte, field DBFField) string {
	start := field.Offset
	end := start + field.Length
	if end > len(record) {
		end = len(record)
	}
	return strings.TrimSpace(string(record[start:end]))
}

func (d *DBFFile) GetFloat(record []byte, field DBFField) float64 {
	s := d.GetString(record, field)
	if s == "" {
		return 0
	}
	var val float64
	fmt.Sscanf(s, "%f", &val)
	return val
}

func (d *DBFFile) GetDate(record []byte, field DBFField) (time.Time, bool) {
	s := d.GetString(record, field)
	if len(s) < 8 {
		return time.Time{}, false
	}
	s = strings.ReplaceAll(s, " ", "0")
	var yy, mm, dd int
	fmt.Sscanf(s, "%4d%2d%2d", &yy, &mm, &dd)
	if yy < 1900 || mm < 1 || mm > 12 || dd < 1 || dd > 31 {
		return time.Time{}, false
	}
	return time.Date(yy, time.Month(mm), dd, 0, 0, 0, 0, time.UTC), true
}

func (d *DBFFile) GetBool(record []byte, field DBFField) bool {
	s := d.GetString(record, field)
	return strings.ToUpper(s) == "T" || strings.ToUpper(s) == "Y" || s == "1"
}

func (d *DBFFile) GetTime(record []byte, field DBFField) (int, bool) {
	s := d.GetString(record, field)
	if len(s) < 2 {
		return 0, false
	}
	var h int
	fmt.Sscanf(s[:2], "%d", &h)
	if h < 0 || h > 23 {
		return 0, false
	}
	return h, true
}

func (d *DBFFile) FieldIndex(name string) int {
	for i, f := range d.Fields {
		if f.Name == name {
			return i
		}
	}
	return -1
}
