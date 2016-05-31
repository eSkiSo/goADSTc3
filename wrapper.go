package ads

/*
#cgo CFLAGS: -I .
#cgo LDFLAGS: -LC:/TwinCAT/AdsApi/TcAdsDll/x64/lib -lTcAdsDll
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <inttypes.h>
#define BOOL bool
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsDef.h"
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsAPI.h"

void  Callback(AmsAddr*, AdsNotificationHeader*, unsigned long);

*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

type NotificationReturn struct {
	handle    uint32
	timestamp uint64
	data      []byte
}

//export notificationFun
func notificationFun(addr *C.AmsAddr, notification *C.AdsNotificationHeader, user C.ulong) {
	//fmt.Println("printed")
	variable := address.notificationHandles[uint32(notification.hNotification)]
	fmt.Println(variable.FullName)
	//fmt.Println(notification.data)
	//fmt.Println(notification.cbSampleSize)
	cBytes := C.GoBytes(unsafe.Pointer(&notification.data), C.int(notification.cbSampleSize))
	//fmt.Println(cBytes)
	variable.parse(cBytes, 0)

	//fmt.Println(&notification.data)
}

type AmsAddr C.AmsAddr

type Connection struct {
	addr                C.AmsAddr
	symbols             map[string]ADSSymbol
	datatypes           map[string]ADSSymbolUploadDataType
	handles             map[uint32]*ADSSymbol
	notificationHandles map[uint32]*ADSSymbol
}

var address = Connection{}

func adsGetDllVersion() (version AdsVersion) {
	cAdsVersion := C.AdsGetDllVersion()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(cAdsVersion))
	buff := bytes.NewBuffer(b)
	binary.Read(buff, binary.LittleEndian, &version)
	return
}

/// opens port on local server
/// returns port number
func adsPortOpen() (port int) {
	port = int(C.AdsPortOpen())
	return
}

func adsPortClose() (err error) {
	errInt := C.AdsPortClose()
	if errInt != 0 {
		err = fmt.Errorf(string(errInt))
	}
	return
}

func adsGetLocalAddress() (err error) {
	errInt := C.AdsGetLocalAddress(&address.addr)
	if errInt != 0 {
		err = fmt.Errorf("error %v", errInt)
	}
	return
}

func setRemoteAddress(amsId string) {
	stringBytes := strings.Split(amsId, ".")
	byte0, _ := strconv.Atoi(stringBytes[0])
	byte1, _ := strconv.Atoi(stringBytes[1])
	byte2, _ := strconv.Atoi(stringBytes[2])
	byte3, _ := strconv.Atoi(stringBytes[3])
	byte4, _ := strconv.Atoi(stringBytes[4])
	byte5, _ := strconv.Atoi(stringBytes[5])

	address.addr.netId.b[0] = C.uchar(byte0)
	address.addr.netId.b[1] = C.uchar(byte1)
	address.addr.netId.b[2] = C.uchar(byte2)
	address.addr.netId.b[3] = C.uchar(byte3)
	address.addr.netId.b[4] = C.uchar(byte4)
	address.addr.netId.b[5] = C.uchar(byte5)
}

func adsSyncWriteReq(group uint32, offset uint32, data []byte) (err error) {
	cDataToWrite := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToWrite))
	errInt := int(C.AdsSyncWriteReq(
		&address.addr,
		C.ulong(group),
		C.ulong(offset),
		C.ulong(len(data)),
		unsafe.Pointer(cDataToWrite)))
	if errInt != 0 {
		err = fmt.Errorf("Error writing adsSyncWriteReq")
	}
	return err
}

func adsSyncReadReq(group uint32, offset uint32, length uint32) (data []byte, err error) {
	data = make([]byte, length)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))

	errInt := int(C.AdsSyncReadReq(
		&address.addr,
		C.ulong(group),
		C.ulong(offset),
		C.ulong(length),
		unsafe.Pointer(cDataToRead)))
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(length))
	if errInt != 0 {
		err = fmt.Errorf("Error adsSyncReadReq")
	}
	return data, err
}

func adsSyncReadReqEx(group uint32, offset uint32, length uint32) (data []byte, err error) {
	amountOfDataReturned := C.ulong(0)
	cData := C.CString(string(make([]byte, length)))
	defer C.free(unsafe.Pointer(cData))
	errInt := int(C.AdsSyncReadReqEx(
		&address.addr,
		C.ulong(group),
		C.ulong(offset),
		C.ulong(len(data)),
		unsafe.Pointer(cData),
		&amountOfDataReturned))
	data = C.GoBytes(unsafe.Pointer(cData), C.int(length))
	if errInt != 0 {
		err = fmt.Errorf("Error adsSyncReadReqEx")
	}
	return data, err
}

func adsSyncReadWriteReq(group uint32, offset uint32, readLength uint32, dataToWrite []byte) (data []byte, err error) {
	data = make([]byte, readLength)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))

	cDataToWrite := C.CString(string(dataToWrite))
	defer C.free(unsafe.Pointer(cDataToWrite))

	errInt := int(C.AdsSyncReadWriteReq(
		&address.addr,
		C.ulong(group),
		C.ulong(offset),
		C.ulong(readLength),
		unsafe.Pointer(cDataToRead),
		C.ulong(len(dataToWrite)),
		unsafe.Pointer(cDataToWrite)))
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(readLength))
	if errInt != 0 {
		err = fmt.Errorf("Error vadsSyncReadWriteReq %v", errInt)
	}
	return data, err
}

const (
	ADSTRANS_NOTRANS     = 0
	ADSTRANS_CLIENTCYCLE = 1
	ADSTRANS_CLIENTONCHA = 2
	ADSTRANS_SERVERCYCLE = 3
	ADSTRANS_SERVERONCHA = 4
)

func (node *ADSSymbol) AdsSyncAddDeviceNotificationReq(transMode int, maxDelay uint32, cycleTime uint32) {

	notAttrib := AdsNotificationAttrib{}
	notAttrib.NMaxDelay = 0
	notAttrib.NCycleTime = uint32(time.Second / 100.0)
	notAttrib.CbLength = node.Length
	notAttrib.NTransMode = ADSTRANS_SERVERONCHA

	if node.Handle == nil {
		node.getHandle()
	}

	if address.notificationHandles == nil {
		address.notificationHandles = make(map[uint32]*ADSSymbol)
	}

	var handle uint32

	hNotification := C.ulong(0)
	//f := C.Callback
	nErrInt := int(C.AdsSyncAddDeviceNotificationReq(
		&address.addr,
		ADSIGRP_SYM_VALBYHND,
		C.ulong(*node.Handle),
		(*C.AdsNotificationAttrib)(unsafe.Pointer(&notAttrib)),
		(C.PAdsNotificationFuncEx)(C.Callback),
		C.ulong(*node.Handle),
		&hNotification))

	handle = uint32(hNotification)
	fmt.Println("handle for notification", handle)
	fmt.Println("error for notification", nErrInt)

	address.notificationHandles[handle] = node
	node.NotificationHandle = &handle
	fmt.Println(nErrInt)
	fmt.Println("done")
}
