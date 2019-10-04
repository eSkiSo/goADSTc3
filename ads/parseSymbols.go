package ads

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"time"
	"unsafe"
)

type adsSymbolUploadDataType struct {
	DatatypeEntry AdsDatatypeEntry
	Name          string
	DataType      string
	Comment       string

	Childs map[string]*adsSymbolUploadDataType
}

type adsSymbolUploadSymbol struct {
	SymbolEntry adsSymbolEntry
	Name        string
	DataType    string
	Comment     string
	Childs      map[string]*adsSymbolUploadDataType
}

func getSymbolUploadInfo() (uploadInfo adsSymbolUploadInfo2, err error) {
	data, err := syncReadReqEx2(
		ADSIGRP_SYM_UPLOADINFO2,
		0x0,
		uint32(unsafe.Sizeof(uploadInfo)))
	buff := bytes.NewBuffer(data)
	binary.Read(buff, binary.LittleEndian, &uploadInfo)
	return
}

func uploadSymbolInfoSymbols(length uint32) error {
	res, err := syncReadReqEx2(ADSIGRP_SYM_UPLOAD, 0, length)
	if err != nil {
		return err
	}

	var buff = bytes.NewBuffer(res)

	for buff.Len() > 0 {
		begBuff := buff.Len()
		result := adsSymbolEntry{}
		binary.Read(buff, binary.LittleEndian, &result)

		name := make([]byte, result.NameLength)
		dt := make([]byte, result.TypeLength)
		comment := make([]byte, result.CommentLength)

		binary.Read(buff, binary.LittleEndian, name)
		buff.Next(1)
		binary.Read(buff, binary.LittleEndian, dt)
		buff.Next(1)
		binary.Read(buff, binary.LittleEndian, comment)
		buff.Next(1)

		item := adsSymbolUploadSymbol{}
		item.Name = string(name)
		item.DataType = string(dt)
		item.Comment = string(comment)
		item.SymbolEntry = result
		if len(item.DataType) > 6 {
			if item.DataType[:6] == "STRING" {
				item.DataType = "STRING"
			}
		}
		endBuff := buff.Len()

		addSymbol(&item)

		buff.Next(int(item.SymbolEntry.EntryLength) - (begBuff - endBuff))

	}
	return err
}

func addSymbol(symbol *adsSymbolUploadSymbol) {
	sym := &Symbol{}

	// sym.Connection = conn
	// sym.Self = sym
	sym.Name = symbol.Name
	sym.LastUpdateTime = time.Now()
	sym.MinUpdateInterval = time.Millisecond * 50
	sym.FullName = symbol.Name
	sym.DataType = symbol.DataType
	sym.Comment = symbol.Comment
	sym.Length = symbol.SymbolEntry.Size

	sym.Group = symbol.SymbolEntry.IGroup
	sym.Offset = symbol.SymbolEntry.IOffs

	dt, ok := connection.datatypes[symbol.DataType]
	if ok {
		sym.Childs = dt.addOffset(sym, symbol.SymbolEntry.IGroup, symbol.SymbolEntry.IOffs)
	}
	connection.symbols[symbol.Name] = sym
	return
}

func (data *adsSymbolUploadDataType) addOffset(parent *Symbol, group uint32, offset uint32) (childs map[string]*Symbol) {
	childs = map[string]*Symbol{}

	var path string

	for key, segment := range data.Childs {

		if segment.Name[0:1] != "[" {
			path = fmt.Sprint(parent.FullName, ".", segment.Name)
		} else {
			path = fmt.Sprint(parent.Name, segment.Name)
		}

		child := Symbol{}
		// child.Self = &child
		// child.Connection = parent.Connection
		child.Name = segment.Name
		child.LastUpdateTime = time.Now()
		child.MinUpdateInterval = time.Millisecond * 50
		child.FullName = path
		child.DataType = segment.DataType
		child.Comment = segment.Comment
		child.Length = segment.DatatypeEntry.Size

		// Uppdate with area and offset
		child.Group = group
		child.Offset = segment.DatatypeEntry.Offs

		child.Parent = parent

		// parent.Connection.Symbols[child.FullName] = child

		// Check if subitems exist
		dt, ok := connection.datatypes[segment.DataType]
		if ok {
			//log.Warn("Found sub ",segment.DataType);
			child.Childs = dt.addOffset(&child, child.Group, child.Offset)

		}

		childs[key] = &child
		connection.symbols[child.FullName] = &child
	}

	return
}

func uploadSymbolInfoDataTypes(length uint32) (err error) {
	data, errInt := syncReadReqEx2(
		ADSIGRP_SYM_DT_UPLOAD,
		0x0,
		length)
	if errInt != nil {
		err = fmt.Errorf("error doing DT UPLOAD %d", err)
	}
	buff := bytes.NewBuffer(data)

	if connection.datatypes == nil {
		connection.datatypes = map[string]adsSymbolUploadDataType{}
	}

	for buff.Len() > 0 {
		header, _ := decodeSymbolUploadDataType(buff, "")
		connection.datatypes[header.Name] = header
	}
	return
}

func decodeSymbolUploadDataType(data *bytes.Buffer, parent string) (header adsSymbolUploadDataType, err error) {

	result := AdsDatatypeEntry{}
	header = adsSymbolUploadDataType{}

	totalSize := data.Len()

	if totalSize < 48 {
		err = fmt.Errorf(parent, " - Wrong size <48 byte")
		fmt.Printf(hex.Dump(data.Bytes()))
	}

	binary.Read(data, binary.LittleEndian, &result)

	name := make([]byte, result.NameLength)
	dt := make([]byte, result.TypeLength)
	comment := make([]byte, result.CommentLength)

	binary.Read(data, binary.LittleEndian, name)
	data.Next(1)
	binary.Read(data, binary.LittleEndian, dt)
	data.Next(1)
	binary.Read(data, binary.LittleEndian, comment)
	data.Next(1)

	header.Name = string(name)
	header.DataType = string(dt)
	header.Comment = string(comment)

	header.DatatypeEntry = result

	if len(header.DataType) > 6 {
		if header.DataType[:6] == "STRING" {
			header.DataType = "STRING"
		}
	}

	childLen := int(result.EntryLength) - (totalSize - data.Len())
	if childLen <= 0 {
		return
	}

	childs := make([]byte, childLen)
	data.Read(childs)

	if len(childs) == 0 {
		return
	}

	buff := bytes.NewBuffer(childs)
	if header.Childs == nil {
		header.Childs = map[string]*adsSymbolUploadDataType{}
	}
	if header.DatatypeEntry.ArrayDim > 0 {
		// Childs is an array
		var result AdsDatatypeArrayInfo
		arrayLevels := []AdsDatatypeArrayInfo{}

		for i := 0; i < int(header.DatatypeEntry.ArrayDim); i++ {
			binary.Read(buff, binary.LittleEndian, &result)

			arrayLevels = append(arrayLevels, result)
		}

		header.Childs = makeArrayChilds(arrayLevels, header.DataType, header.DatatypeEntry.Size)

	} else {
		// Childs is standard variables
		for j := 0; j < (int)(result.SubItems); j++ {

			child, _ := decodeSymbolUploadDataType(buff, header.Name)

			header.Childs[child.Name] = &child
		}
	}

	return
}

func makeArrayChilds(levels []AdsDatatypeArrayInfo, dt string, size uint32) (childs map[string]*adsSymbolUploadDataType) {
	childs = map[string]*adsSymbolUploadDataType{}

	if len(levels) < 1 {
		return
	}

	level := levels[:1][0]
	subChilds := makeArrayChilds(levels[1:], dt, size)

	var offset uint32

	for i := level.LBound; i < level.LBound+level.Elements; i++ {
		name := fmt.Sprint("[", i, "]")

		child := adsSymbolUploadDataType{}
		child.Name = name
		child.DataType = dt
		child.DatatypeEntry.Offs = offset
		child.DatatypeEntry.Size = size / level.Elements
		child.Childs = subChilds

		//child.Walk("")

		childs[name] = &child
		offset += size / level.Elements
	}

	return
}
