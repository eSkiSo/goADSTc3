package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

type ADSSymbol struct {
	Connection         *Connection
	Self               *ADSSymbol
	FullName           string
	Name               string
	DataType           string
	Comment            string
	Handle             *uint32
	NotificationHandle *uint32

	Group  uint32
	Offset uint32
	Length uint32

	Value   string
	Valid   bool
	Changed bool

	Parent *ADSSymbol
	Childs map[string]*ADSSymbol
}

type ADSSymbolUploadDataType struct {
	DatatypeEntry AdsDatatypeEntry
	Name          string
	DataType      string
	Comment       string

	Childs map[string]ADSSymbolUploadDataType
}

type ADSSymbolUploadSymbol struct {
	SymbolEntry AdsSymbolEntry
	Name        string
	DataType    string
	Comment     string
	Childs      map[string]ADSSymbolUploadDataType
}

func main() {
	version := adsGetDllVersion()
	log.Println(version.Version, version.Revision, version.Build)

	fmt.Println()

	port := adsPortOpen()
	log.Println(port)

	err := adsGetLocalAddress()
	if err != nil {
		log.Println(err)
	}
	log.Println(address.addr.netId, address.addr.port)
	address.addr.port = 851

	uploadInfo, err := getSymbolUploadInfo()
	log.Println(uploadInfo.NSymSize, uploadInfo.NDatatypeSize)

	UploadSymbolInfoDataTypes(uploadInfo.NDatatypeSize)

	// for _, value := range address.datatypes {
	// 	showComments(value)
	// }

	UploadSymbolInfoSymbols(uploadInfo.NSymSize)

	// handle, err := getHandleByName("Main.i")
	// if err != nil {
	// 	log.Println(err)
	// }
	// log.Println("handle:", handle)
	// symbol := address.symbols["Main.i"]
	// symbol.Handle = &handle
	address.handles = map[uint32]*ADSSymbol{}
	// address.handles[handle] = &symbol

	// fmt.Println(symbol.FullName)
	// fmt.Println(address.handles[handle].FullName)

	// for _, value := range address.symbols {
	// 	showInfoComments(value)
	// }
	variable := address.symbols["ALARMS.WorkingAlarms"]
	val, err := variable.getStringValue()

	fmt.Println("error", err)
	fmt.Println("value", val)

	notificationVariable := address.symbols["ALARMS.WorkingAlarms"]
	notificationVariable.AdsSyncAddDeviceNotificationReq(0, 0, 0)

	//fmt.Println(address.symbols["MAIN.i"].FullName)
	//fmt.Println(address.symbols["MAIN.Yikes.Blargh"].FullName)
	//fmt.Println(address.symbols["MAIN.Yikes.Blargh"].Name)

	// data, err := getValueByHandle(handle, size)
	// if err != nil {
	// 	fmt.Println(binary.LittleEndian.Uint16(data))
	// }
	// log.Println(err)

	val, err = variable.getStringValue()

	fmt.Println("error", err)
	fmt.Println("value", val)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)

	go func() {
		<-c
		closeEverything()
		// sig is a ^C, handle it
		os.Exit(1)
	}()

	for {

	}

}

func closeEverything() {
	for k := range address.handles {
		err := releaseHandle(k)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println()
		}
	}
	err := adsPortClose()
	if err != nil {
		log.Println(err)
	}
}

func showComments(info ADSSymbolUploadDataType) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showComments(value)
	}
}

func showInfoComments(info ADSSymbol) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showInfoComments(*value)
	}

}

func getSymbolUploadInfo() (uploadInfo AdsSymbolUploadInfo2, err error) {
	data, err := adsSyncReadReq(
		ADSIGRP_SYM_UPLOADINFO2,
		0x0,
		uint32(unsafe.Sizeof(uploadInfo)))
	buff := bytes.NewBuffer(data)
	binary.Read(buff, binary.LittleEndian, &uploadInfo)
	if err != nil {
		err = fmt.Errorf("new error")
	}
	return
}

func UploadSymbolInfoSymbols(length uint32) {
	res, e := adsSyncReadReq(ADSIGRP_SYM_UPLOAD, 0, length)
	if e != nil {
		log.Fatal(e)
		return
	}

	if address.symbols == nil {
		address.symbols = map[string]ADSSymbol{}
	}

	var buff = bytes.NewBuffer(res)

	for buff.Len() > 0 {
		begBuff := buff.Len()
		result := AdsSymbolEntry{}
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

		var item ADSSymbolUploadSymbol
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

		addSymbol(item)

		buff.Next(int(item.SymbolEntry.EntryLength) - (begBuff - endBuff))

	}
}

func addSymbol(symbol ADSSymbolUploadSymbol) {
	sym := ADSSymbol{}

	sym.Self = &sym
	sym.Name = symbol.Name
	sym.FullName = symbol.Name
	sym.DataType = symbol.DataType
	sym.Comment = symbol.Comment
	sym.Length = symbol.SymbolEntry.Size

	sym.Group = symbol.SymbolEntry.IGroup
	sym.Offset = symbol.SymbolEntry.IOffs

	dt, ok := address.datatypes[symbol.DataType]
	if ok {
		//sym.Childs = dt.addOffset(sym.Name, symbol.SymbolEntry.IGroup, symbol.SymbolEntry.IOffs)
		sym.Childs = dt.addOffset(&sym, symbol.SymbolEntry.IGroup, symbol.SymbolEntry.IOffs)
	}

	address.symbols[symbol.Name] = sym

	return
}

func (data *ADSSymbolUploadDataType) addOffset(parent *ADSSymbol, group uint32, offset uint32) (childs map[string]*ADSSymbol) {
	childs = map[string]*ADSSymbol{}

	var path string

	for key, segment := range data.Childs {

		if segment.Name[0:1] != "[" {
			path = fmt.Sprint(parent.FullName, ".", segment.Name)
		} else {
			path = fmt.Sprint(parent.Name, segment.Name)
		}

		child := ADSSymbol{}
		child.Self = &child

		child.Name = segment.Name
		child.FullName = path
		child.DataType = segment.DataType
		child.Comment = segment.Comment
		child.Length = segment.DatatypeEntry.Size

		// Uppdate with area and offset
		child.Group = group
		child.Offset = segment.DatatypeEntry.Offs

		child.Parent = parent

		address.symbols[child.FullName] = child

		// Check if subitems exist
		dt, ok := address.datatypes[segment.DataType]
		if ok {
			//log.Warn("Found sub ",segment.DataType);
			child.Childs = dt.addOffset(&child, child.Group, child.Offset)
		}

		childs[key] = &child
	}

	return
}

func UploadSymbolInfoDataTypes(length uint32) (err error) {
	data, errInt := adsSyncReadReq(
		ADSIGRP_SYM_DT_UPLOAD,
		0x0,
		length)
	if errInt != nil {
		err = fmt.Errorf("error doing DT UPLOAD %v", err)
	}
	buff := bytes.NewBuffer(data)

	if address.datatypes == nil {
		address.datatypes = map[string]ADSSymbolUploadDataType{}
	}

	for buff.Len() > 0 {
		header, _ := decodeSymbolUploadDataType(buff, "")
		address.datatypes[header.Name] = header
	}
	return
	//   log.Warn(hex.Dump(header));
}

func decodeSymbolUploadDataType(data *bytes.Buffer, parent string) (header ADSSymbolUploadDataType, err error) {

	result := AdsDatatypeEntry{}
	header = ADSSymbolUploadDataType{}

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
			if header.Childs == nil {
				header.Childs = map[string]ADSSymbolUploadDataType{}
			}

			child, _ := decodeSymbolUploadDataType(buff, header.Name)
			header.Childs[child.Name] = child
		}
	}

	return
}

func makeArrayChilds(levels []AdsDatatypeArrayInfo, dt string, size uint32) (childs map[string]ADSSymbolUploadDataType) {
	childs = map[string]ADSSymbolUploadDataType{}

	if len(levels) < 1 {
		return
	}

	level := levels[:1][0]
	subChilds := makeArrayChilds(levels[1:], dt, size)

	var offset uint32

	for i := level.LBound; i < level.LBound+level.Elements; i++ {
		name := fmt.Sprint("[", i, "]")

		child := ADSSymbolUploadDataType{}
		child.Name = name
		child.DataType = dt
		child.DatatypeEntry.Offs = offset
		child.DatatypeEntry.Size = size / level.Elements
		child.Childs = subChilds

		//child.Walk("")

		childs[name] = child
		offset += size / level.Elements
	}

	return
}
func (node *ADSSymbol) getStringValue() (value string, err error) {
	if node.Handle == nil {
		node.getHandle()
	}
	data, err := getValueByHandle(
		*node.Handle,
		node.Length)
	node.parse(data, 0)

	return node.Value, err
}

func getValueByHandle(handle uint32, size uint32) (data []byte, err error) {
	data, err = adsSyncReadReq(
		ADSIGRP_SYM_VALBYHND,
		uint32(handle),
		uint32(size))

	return data, err
}

func (node *ADSSymbol) getHandle() (err error) {
	var handle uint32
	if node.Handle != nil {
		handle = *node.Handle

	} else {
		handleData, _ := adsSyncReadWriteReq(
			ADSIGRP_SYM_HNDBYNAME,
			0x0,
			uint32(unsafe.Sizeof(handle)),
			[]byte(node.FullName))

		handle = binary.LittleEndian.Uint32(handleData)
		address.handles[handle] = node
		node.Handle = &handle
	}
	return err
}

func releaseHandle(handle uint32) (err error) {
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(handle))
	err = adsSyncWriteReq(
		ADSIGRP_SYM_RELEASEHND,
		0x0,
		a)
	if err != nil {
		delete(address.handles, handle)
		fmt.Println("handle deleted ", handle)
	}
	return

}
