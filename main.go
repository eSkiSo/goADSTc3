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
	
	// router := mux.NewRouter().StrictSlash(true)
    // router.HandleFunc("/{key}", ShortHandler)
	
	// sigs := make(chan os.Signal, 1)
	// signal.Notify(sigs, os.Interrupt,syscall.SIGTERM, syscall.SIGTERM)
	// done := make(chan bool, 1)
	// go func() {
	// 	sig := <- sigs 
	// 	closeServer()
	// 	log.Info(sig)
	// 	done <- true
			
	// }()

	connectToServer()
	node := createAdsNode("Main.blargh")
	node2 := createAdsNode("Main.test")
	writeLongToHandle(node.handle, 14.25)
	writeStringToHandle(node2.handle,"making sure it's good")
	deleteHandles()
	closeServer()
	// log.Fatal(http.ListenAndServe(":8080", router))
	// <-done
	// 	log.Info("Exiting")
}

type AdsNode struct {
	handle uint64
	typeof string
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

func getTypeOfVariable(node AdsNode) string {
	return ""
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

func (node AdsNode) WriteToNode(value string) {
	
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
		C.ADSIGRP_SYM_RELEASEHND,
		0,
		C.sizeof_ulong,
		unsafe.Pointer(&cHandle))
	log.Infof("removeHandle error: %d\n", nErr)
}

func closeServer() {
	C.AdsPortClose()
	log.Infof("Closed Port")
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
