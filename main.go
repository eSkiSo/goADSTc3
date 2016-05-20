package main

/*
#cgo LDFLAGS: -LC:/TwinCAT/AdsApi/TcAdsDll/x64/lib -lTcAdsDll
#include <stdbool.h>
#include <stdlib.h>
#include <inttypes.h>
#define BOOL bool
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsDef.h"
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsAPI.h"
*/
import "C"

import (
	_ "encoding/binary"
	"net/http"
	"os"
	_ "os/signal"
	_ "syscall"
	"unsafe"
	"github.com/op/go-logging"
	_ "github.com/gorilla/mux"
	_ "encoding/json"
	"strings"
	_ "strconv"
	"encoding/binary"
	"bytes"
)

var handles map[string]AdsNode

var (
	addr = C.AmsAddr{}
)

var log = logging.MustGetLogger("test")
var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.4s} %{id:03x}%{color:reset} %{message}`,
)

func main() {
	handles = make(map[string]AdsNode)
	backend1 := logging.NewLogBackend(os.Stderr, "", 0)
	backend1Leveled := logging.AddModuleLevel(backend1)
	logging.SetBackend(backend1Leveled)
	

	// connectToServer()

	// handle := getHandle("Main.i")
	// writeShortToHandle2(handle, 10)
	// deleteHandles()
	// closeServer()
	log.Info(adsGetDllVersion())
}



func deleteHandles() {
	for _, element := range handles {
		removeHandle(element.handle)
		log.Infof("removed handle", element.handle)
	}
}

func writeToVariable(variableName string, value string) {
	// node := createAdsNode(variableName)
	
}

func getTypeOfVariable(variableName string) string {
	buffer := make([]byte, 0xFFFF)
	cBuffer := C.CString(string(buffer))
	defer C.free(unsafe.Pointer(cBuffer))
	sizeOfBuffer := C.ulong(unsafe.Sizeof(buffer))
	cVariableNmae := C.CString(variableName)
	defer C.free(unsafe.Pointer(cVariableNmae))
	nErr := C.AdsSyncReadWriteReq(&addr,
		C.ADSIGRP_SYM_INFOBYNAMEEX, 
		0, 
		sizeOfBuffer, 
		unsafe.Pointer(&cBuffer), 
		C.ulong(len(variableName)+1), 
		unsafe.Pointer(cVariableNmae))

	log.Infof("getTypeOfVariable error: %d", nErr)	
	
	byteArray :=  C.GoBytes(unsafe.Pointer(&cBuffer), 0xFFFF)
	getAdsSymbol(byteArray)
	return "typeOf"
}

func createAdsNode(variableName string) AdsNode{
	val, ok := handles[variableName]
	if ok {
		return val
	}
	var newNode AdsNode
	newNode.handle = getHandle(variableName)
	handles[variableName] = newNode
	return newNode
}





func  getAdsSymbol(data []byte) (symbol AdsSymHandle){
	dataBuffer := bytes.NewBuffer(data)
 	headerRead := AdsHeaderType{}
	
	binary.Read(dataBuffer, binary.LittleEndian, &headerRead)
	// adsType := AdsSymHandle{}
	name := make([]byte, headerRead.LenName)
	dt := make([]byte, headerRead.LenDataType)
	comment := make([]byte, headerRead.LenComment)
	binary.Read(dataBuffer, binary.LittleEndian, name)
	dataBuffer.Next(1)
	binary.Read(dataBuffer, binary.LittleEndian, dt)
	dataBuffer.Next(1)
	binary.Read(dataBuffer, binary.LittleEndian, comment)
	dataBuffer.Next(1)
	symbol.Name = string(name)
	symbol.DataType = string(dt)
	symbol.Comment = string(comment)	
	
	return symbol
}

func (node AdsNode) WriteToNode(value string) {
	// valueInC := C.float(value)
	// cHandle := C.ulong(handle)
	// nErr := C.AdsSyncWriteReq(&addr,
	// 	C.ADSIGRP_SYM_VALBYHND,
	// 	cHandle,
	// 	C.sizeof_float,
	// 	unsafe.Pointer(&valueInC))
	// log.Infof("writeLongToHandle error: %d", nErr)		
}

func ShortHandler(w http.ResponseWriter, r *http.Request) {
    // vars := mux.Vars(r)
	// key := vars["key"]
	// handle := getHandle(key)
	// value := readShortFromHandle(handle)
	// removeHandle(handle)
	// if err := json.NewEncoder(w).Encode(value); err != nil {
    //     panic(err)
    // }
}

func writeShortToHandle(handle uint64, value uint16) {
	valueInCShort := C.short(value)
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncWriteReq(&addr,
		C.ADSIGRP_SYM_VALBYHND,
		cHandle,
		C.sizeof_short,
		unsafe.Pointer(&valueInCShort))
	log.Infof("writeShortToHandle error: %d", nErr)
}

func writeLongToHandle(handle uint64, value float32) {
	valueInC := C.float(value)
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncWriteReq(&addr,
		C.ADSIGRP_SYM_VALBYHND,
		cHandle,
		C.sizeof_float,
		unsafe.Pointer(&valueInC))
	log.Infof("writeLongToHandle error: %d", nErr)	
}

func writeShortToHandle2(handle uint64, value uint16) {
	buf := bytes.NewBuffer([]byte{})
	binary.Write(buf,binary.LittleEndian, value )
	writeToHandle(handle, buf.Bytes())
}

func writeToHandle(handle uint64, data []byte) {
	valueInC := C.CString(string(data))
	defer C.free(unsafe.Pointer(valueInC))
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncWriteReq(&addr,
		C.ADSIGRP_SYM_VALBYHND,
		cHandle,
		C.ulong(len(data)),
		unsafe.Pointer(valueInC))
	log.Infof("writeStringToHandle error: %d", nErr)		
}



func writeStringToHandle(handle uint64, value string) {
	valueInC := C.CString(value)
	defer C.free(unsafe.Pointer(valueInC))
	//lengthOfString := len(value)
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncWriteReq(&addr,
		C.ADSIGRP_SYM_VALBYHND,
		cHandle,
		C.ulong(len(value)+1),
		unsafe.Pointer(valueInC))
	log.Infof("writeStringToHandle error: %d", nErr)	
}

func readShortFromHandle(handle uint64) int16 {
	nData := C.short(1)
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncReadReq(&addr,
		C.ADSIGRP_SYM_VALBYHND,
		cHandle,
		C.sizeof_short,
		unsafe.Pointer(&nData))
	log.Infof("readShortFromHandle nData: %d\n", nData)
	log.Infof("readShortFromHandle error: %d", nErr)
	return int16(nData)
	
}

func connectToServer() {
	nPort := C.AdsPortOpen()
	nErr := C.AdsGetLocalAddress(&addr)

	log.Infof("connectToServer error: %d\n", nErr)
	log.Infof("connectToServer nport: %d\n", nPort)
	addr.port = 851
}

func removeHandle(handle uint64) {
	cHandle := C.ulong(handle)
	nErr := C.AdsSyncWriteReq(&addr,
		ADSIGRP_SYM_RELEASEHND,
		0,
		C.sizeof_ulong,
		unsafe.Pointer(&cHandle))
	log.Infof("removeHandle error: %d\n", nErr)
}


func getHandle(variableName string) (returnedHandle uint64) {
	handleFromC := C.ulong(0)
	trimmedHandle := strings.TrimSpace(variableName)
	handleName := C.CString(trimmedHandle)
	sizeOfName := len(trimmedHandle)
	defer C.free(unsafe.Pointer(handleName))
	nErr := C.AdsSyncReadWriteReq(&addr,
		C.ADSIGRP_SYM_HNDBYNAME,
		0x0,
		C.sizeof_ulong,
		unsafe.Pointer(&handleFromC),
		C.ulong(sizeOfName),
		unsafe.Pointer(handleName))
	log.Infof("getHandle error: %d\n", nErr)
	returnedHandle = uint64(handleFromC)
	return
}

func adsGetDllVersion() int {
	return int(C.AdsGetDllVersion())
}

/// opens port on local server
/// returns port number
func adsPortOpen() int {
	return int(C.AdsPortOpen())
}

func adsPortClose() int {
	return int(C.AdsPortClose())
}

func adsGetLocalAddress() (err int, address AmsAddr) {
	address := AmsAddr{}
	C.AdsGetLocalAddress  
}
