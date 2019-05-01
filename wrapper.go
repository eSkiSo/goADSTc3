package ads

/*
#cgo CFLAGS: -I .
#cgo LDFLAGS: -LC:/TwinCAT/AdsApi/TcAdsDll/x64/lib -lTcAdsDll
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#define BOOL bool
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsDef.h"
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsAPI.h"

void  notificationFun(AmsAddr*, AdsNotificationHeader*, unsigned long);
void  routerNotificationFun(long);
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

var connections []*Connection

type NotificationReturn struct {
	handle    uint32
	timestamp uint64
	data      []byte
}

func getConnectionFromAddress(addr AmsAddr) (conn *Connection) {
	for _, value := range connections {
		if *value.addr == addr {
			conn = value
			return
		}
	}
	return
}

func adsAmsPortEnabled() (bool, error) {
	var portOpen C.bool
	adsLock.Lock()
	errInt := C.AdsAmsPortEnabled(&portOpen)
	adsLock.Unlock()
	if errInt != 0 && errInt != 1864 {
		return false, fmt.Errorf("Error checking port %d\n", errInt)
	}
	return bool(portOpen), nil
}

//export notificationFun
func notificationFun(addr *C.AmsAddr, notification *C.AdsNotificationHeader, user C.ulong) {
		goAmsAddr := (*AmsAddr)(unsafe.Pointer(addr))
		connection := getConnectionFromAddress(*goAmsAddr)
	cdata := C.GoBytes(unsafe.Pointer(notification), C.sizeof_AdsNotificationHeader)
	buf := bytes.NewBuffer(cdata)
	notificationHeader := &AdsNotificationHeader{}
	binary.Read(buf, binary.LittleEndian, &notificationHeader.HNotification)
	binary.Read(buf, binary.LittleEndian, &notificationHeader.Timestamp)
	binary.Read(buf, binary.LittleEndian, &notificationHeader.CbSampleSize)
	cBytes := C.GoBytes(unsafe.Pointer(&notification.data), C.int(notification.cbSampleSize))
		variable, ok := connection.notificationHandles[uint32(notification.hNotification)]
		if !ok {
			fmt.Printf("note error: %v", uint32(notification.hNotification))
			return
		}
	unixTime := time.Unix(int64(notificationHeader.Timestamp/10000000)-11644473600, 0)
			var update = updateStruct{}
			update.variable = variable
			update.value = cBytes
			update.timestamp = unixTime
			connection.Update <- update
		}()
}

func AdsGetDllVersion() (version AdsVersion) {
	adsLock.Lock()
	cAdsVersion := C.AdsGetDllVersion()
	adsLock.Unlock()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(cAdsVersion))
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, &version.Build)
	binary.Read(buf, binary.LittleEndian, &version.Revision)
	binary.Read(buf, binary.LittleEndian, &version.Version)
	return
}

/// opens port on local server
/// returns port number
func adsPortOpen() (port int) {
	adsLock.Lock()
	port = int(C.AdsPortOpen())
	adsLock.Unlock()
	return
}

func adsPortOpenEx() (port int) {
	adsLock.Lock()
	port = int(C.AdsPortOpenEx())
	adsLock.Unlock()
	return port
}

func adsPortClose() (err error) {
	adsLock.Lock()
	errInt := C.AdsPortClose()
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf(string(errInt))
	}
	return
}

func adsPortCloseEx(port int) (err error) {
	adsLock.Lock()
	errInt := C.AdsPortCloseEx(C.long(port))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf(string(errInt))
	}
	return
}

func (conn *Connection) adsGetLocalAddress() (err error) {
	adsLock.Lock()
	errInt := C.AdsGetLocalAddress((*C.AmsAddr)(unsafe.Pointer(conn.addr)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("get local address error: %d\n", errInt)
	}
	return
}

func (conn *Connection) adsGetLocalAddressEx() (err error) {
	adsLock.Lock()
	errInt := C.AdsGetLocalAddressEx(C.long(conn.port), (*C.AmsAddr)(unsafe.Pointer(conn.addr)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("adsGetLocalAddressEx error: %d\n", errInt)
	}
	return
}

func (conn *Connection) setRemoteAddress(amsId string) {
	stringBytes := strings.Split(amsId, ".")
	byte0, _ := strconv.Atoi(stringBytes[0])
	byte1, _ := strconv.Atoi(stringBytes[1])
	byte2, _ := strconv.Atoi(stringBytes[2])
	byte3, _ := strconv.Atoi(stringBytes[3])
	byte4, _ := strconv.Atoi(stringBytes[4])
	byte5, _ := strconv.Atoi(stringBytes[5])

	conn.addr.NetId.B[0] = uint8(byte0)
	conn.addr.NetId.B[1] = uint8(byte1)
	conn.addr.NetId.B[2] = uint8(byte2)
	conn.addr.NetId.B[3] = uint8(byte3)
	conn.addr.NetId.B[4] = uint8(byte4)
	conn.addr.NetId.B[5] = uint8(byte5)
}

func (conn *Connection) adsSyncWriteReq(group uint32, offset uint32, data []byte) (err error) {
	cDataToWrite := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToWrite))
	adsLock.Lock()
	errInt := int(C.AdsSyncWriteReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(len(data)),
		unsafe.Pointer(cDataToWrite)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("error writing adsSyncWriteReq %d", errInt)
	}
	return err
}

func (conn *Connection) adsSyncWriteReqEx(group uint32, offset uint32, data []byte) (err error) {
	cDataToWrite := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToWrite))
	adsLock.Lock()
	errInt := int(C.AdsSyncWriteReqEx(
		C.long(conn.port),
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(len(data)),
		unsafe.Pointer(cDataToWrite)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("error writing adsSyncWriteReq %d", errInt)
	}
	return err
}

func (conn *Connection) adsSyncReadReq(group uint32, offset uint32, length uint32) (data []byte, err error) {
	data = make([]byte, length)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))
	adsLock.Lock()
	errInt := int(C.AdsSyncReadReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(length),
		unsafe.Pointer(cDataToRead)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("error adsSyncReadReq: %d\n", errInt)
		return data, err
	}
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(length))
	return data, err
}

func (conn *Connection) adsSyncReadReqEx(group uint32, offset uint32, length uint32) (data []byte, err error) {
	amountOfDataReturned := C.ulong(0)
	cData := C.CString(string(make([]byte, length)))
	defer C.free(unsafe.Pointer(cData))
	adsLock.Lock()
	errInt := int(C.AdsSyncReadReqEx(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(length),
		unsafe.Pointer(cData),
		&amountOfDataReturned))
	adsLock.Unlock()
	data = C.GoBytes(unsafe.Pointer(cData), C.int(amountOfDataReturned))
	if errInt != 0 {
		err = fmt.Errorf("error adsSyncReadReqEx: %d", errInt)
	}
	return data, err
}

func (conn *Connection) adsSyncReadReqEx2(group uint32, offset uint32, length uint32) (data []byte, err error) {
	amountOfDataReturned := C.ulong(length)
	cData := C.CString(string(make([]byte, length)))
	defer C.free(unsafe.Pointer(cData))
	adsLock.Lock()
	errInt := int(C.AdsSyncReadReqEx2(
		C.long(conn.port),
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(length),
		unsafe.Pointer(cData),
		&amountOfDataReturned))
	adsLock.Unlock()
	// fmt.Println("amount of data returned", amountOfDataReturned)
	data = C.GoBytes(unsafe.Pointer(cData), C.int(amountOfDataReturned))
	//fmt.Println(errInt)
	if errInt != 0 {
		err = fmt.Errorf("error adsSyncReadReqEx: %d", errInt)
	}
	return data, err
}

func (node *ADSSymbol) writeBuffArray(data []byte) {
	if node.Handle == 0 {
		node.getHandle()
	}
	node.Connection.adsSyncWriteReq(
		ADSIGRP_SYM_VALBYHND,
		uint32(node.Handle),
		data)
}

func (node *ADSSymbol) writeBuffArrayEx(data []byte) {
	if node.Handle == 0 {
		node.getHandle()
	}
	node.Connection.adsSyncWriteReqEx(
		ADSIGRP_SYM_VALBYHND,
		uint32(node.Handle),
		data)
}

func (conn *Connection) adsSyncReadWriteReq(group uint32, offset uint32, readLength uint32, dataToWrite []byte) (data []byte, err error) {
	data = make([]byte, readLength)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))

	cDataToWrite := C.CString(string(dataToWrite))
	defer C.free(unsafe.Pointer(cDataToWrite))
	adsLock.Lock()
	errInt := int(C.AdsSyncReadWriteReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(readLength),
		unsafe.Pointer(cDataToRead),
		C.ulong(len(dataToWrite)),
		unsafe.Pointer(cDataToWrite)))
	adsLock.Unlock()
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(readLength))
	if errInt != 0 {
		err = fmt.Errorf("error adsSyncReadWriteReq %v", errInt)
	}
	return data, err
}

func (conn *Connection) adsSyncReadWriteReqEx2(group uint32, offset uint32, readLength uint32, dataToWrite []byte) (data []byte, err error) {
	data = make([]byte, readLength)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))

	cDataToWrite := C.CString(string(dataToWrite))
	defer C.free(unsafe.Pointer(cDataToWrite))

	cLengthOfReturnedBytes := C.ulong(0)
	adsLock.Lock()
	errInt := int(C.AdsSyncReadWriteReqEx2(
		C.long(conn.port),
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(readLength),
		unsafe.Pointer(cDataToRead),
		C.ulong(len(dataToWrite)),
		unsafe.Pointer(cDataToWrite),
		&cLengthOfReturnedBytes))
	adsLock.Unlock()
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(readLength))
	if errInt != 0 {
		err = fmt.Errorf("error adsSyncReadWriteReq %d", errInt)
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

func (node *ADSSymbol) adsSyncAddDeviceNotificationReq(transMode uint32, maxDelay uint32, cycleTime uint32) {

	notAttrib := AdsNotificationAttrib{}
	notAttrib.NMaxDelay = uint32(maxDelay / 100.0)
	notAttrib.NCycleTime = uint32(cycleTime / 100.0)
	notAttrib.CbLength = node.Length
	notAttrib.NTransMode = uint32(transMode)

	if node.Handle == 0 {
		node.getHandle()
		fmt.Println("node handle", &node.Handle)
	}
	var handle uint32

	hNotification := C.ulong(0)
	//f := C.Callback
	adsLock.Lock()
	nErrInt := int(C.AdsSyncAddDeviceNotificationReq(
		(*C.AmsAddr)(unsafe.Pointer(node.Connection.addr)),
		ADSIGRP_SYM_VALBYHND,
		C.ulong(node.Handle),
		(*C.AdsNotificationAttrib)(unsafe.Pointer(&notAttrib)),
		(C.PAdsNotificationFuncEx)(C.notificationFun),
		C.ulong(node.Handle),
		&hNotification))
	adsLock.Unlock()
	handle = uint32(hNotification)

	node.NotificationHandle = handle
	node.Connection.notificationHandles[handle] = node.FullName

	log.Printf("Notification Added - Variable: %s Handle: %d, Error: %d\n", node.FullName, node.NotificationHandle, nErrInt)

}

func (node *ADSSymbol) adsSyncAddDeviceNotificationReqEx(transMode uint32, maxDelay uint32, cycleTime uint32) {
	notAttrib := AdsNotificationAttrib{}
	notAttrib.NMaxDelay = uint32(maxDelay / 100.0)
	notAttrib.NCycleTime = uint32(cycleTime / 100.0)
	notAttrib.CbLength = node.Length
	notAttrib.NTransMode = uint32(transMode)

	if node.Handle == 0 {
		node.getHandle()
	}
	var handle uint32

	hNotification := C.ulong(0)
	adsLock.Lock()
	nErrInt := int(C.AdsSyncAddDeviceNotificationReqEx(
		C.long(node.Connection.port),
		(*C.AmsAddr)(unsafe.Pointer(node.Connection.addr)),
		ADSIGRP_SYM_VALBYHND,
		C.ulong(node.Handle),
		(*C.AdsNotificationAttrib)(unsafe.Pointer(&notAttrib)),
		(C.PAdsNotificationFuncEx)(C.notificationFun),
		C.ulong(node.Handle),
		&hNotification))
	adsLock.Unlock()
	handle = uint32(hNotification)
	node.NotificationHandle = handle
	node.Connection.notificationHandles[handle] = node.FullName
	fmt.Printf("Notification Added - Variable: %s Handle: %d, Error: %d\n", node.FullName, node.NotificationHandle, nErrInt)
}

func (conn *Connection) adsSyncDelDeviceNotificationReq(handle uint32) (err error) {
	adsLock.Lock()
	nErrInt := int(C.AdsSyncDelDeviceNotificationReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(handle)))
	adsLock.Unlock()
	if nErrInt != 0 {
		err = fmt.Errorf("del notification error %d", nErrInt)
	}
	return
}

func (conn *Connection) adsSyncDelDeviceNotificationReqEx(handle uint32) (err error) {
	adsLock.Lock()
	nErrInt := int(C.AdsSyncDelDeviceNotificationReqEx(
		C.long(conn.port),
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(handle)))
	adsLock.Unlock()
	if nErrInt != 0 {
		err = fmt.Errorf("del notification error %d", nErrInt)
	}
	return
}

func (node *ADSSymbol) getHandle() (err error) {
	var handle uint32
	if node.Handle != 0 {
		return
	}

	handleData, err := node.Connection.adsSyncReadWriteReqEx2(
		ADSIGRP_SYM_HNDBYNAME,
		0x0,
		uint32(unsafe.Sizeof(handle)),
		[]byte(node.FullName))
	if err != nil {
		return err
	}
	handle = binary.LittleEndian.Uint32(handleData)
	lock.Lock()
	node.Handle = handle
	node.Connection.handles[handle] = node.FullName
	lock.Unlock()

	return err
}

func (conn *Connection) getValueByHandle(handle uint32, size uint32) (data []byte, err error) {
	data, err = conn.adsSyncReadReqEx2(
		ADSIGRP_SYM_VALBYHND,
		uint32(handle),
		uint32(size))
	return data, err
}

func (conn *Connection) releaseHandle(handle uint32) (err error) {
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(handle))
	err = conn.adsSyncWriteReqEx(
		ADSIGRP_SYM_RELEASEHND,
		0x0,
		a)
	if err != nil {
		// conn.handles[handle].Handle = 0
		fmt.Printf("handle deleted %d\n", handle)
	}
	return
}

func (conn *Connection) releasNotificationeHandle(handle uint32) (err error) {
	conn.adsSyncDelDeviceNotificationReqEx(handle)
	if err != nil {
		// conn.notificationHandles[handle].NotificationHandle = 0
		fmt.Printf("notification handle deleted %d\n", handle)
	}
	return
}

func (conn *Connection) adsSyncReadStateReq() (adsState int, deviceState int, err error) {
	cAdsState := C.ushort(0)
	cDeviceState := C.ushort(0)
	adsLock.Lock()
	nErr := C.AdsSyncReadStateReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		&cAdsState,
		&cDeviceState)
	adsLock.Unlock()
	if nErr != 0 {
		return 0, 0, fmt.Errorf("error %d", nErr)
	}
	return int(cAdsState), int(cDeviceState), nil
}

	log.Println("adding router notification")
	adsLock.Lock()
	C.AdsAmsRegisterRouterNotification(
		(C.PAmsRouterNotificationFuncEx)(C.routerNotificationFun),
	)
	adsLock.Unlock()
	return
}

func UnregisterRouterNotification() (err error) {
	adsLock.Lock()
	C.AdsAmsUnRegisterRouterNotification()
	adsLock.Unlock()
	return
}

//export routerNotificationFun
func routerNotificationFun(response C.long) {
	log.Printf("notification received %d\n", response)
}
