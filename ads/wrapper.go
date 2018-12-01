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

void  notificationFun(AmsAddr*, AdsNotificationHeader*, unsigned long);
void  routerNotificationFun(long);
*/
import "C"

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
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
		return false, fmt.Errorf("Error checking port %d", errInt)
	}
	return bool(portOpen), nil
}

//export notificationFun
func notificationFun(addr *C.AmsAddr, notification *C.AdsNotificationHeader, user C.ulong) {
	goAmsAddr := (*AmsAddr)(unsafe.Pointer(addr))
	connection := getConnectionFromAddress(*goAmsAddr)
	variable, ok := connection.notificationHandles[uint32(notification.hNotification)]
	if !ok {
		fmt.Println(notification.hNotification)
		return
	}
	cBytes := C.GoBytes(unsafe.Pointer(&notification.data), C.int(notification.cbSampleSize))
	variable.parse(cBytes, 0)
	changed := false
	if len(variable.Childs) == 0 {
		changed = true
	} else {
		changed = variable.isNodeChanged()
	}
	if changed {
		for _, callback := range variable.ChangedHandlers {
			variable.Lock.Lock()
			callback(*variable)
			variable.Lock.Unlock()
		}
	}
	variable.clearNodeChangedFlag()
}

func (node *ADSSymbol) clearNodeChangedFlag() {
	lock.Lock()
	node.Lock.Lock()
	node.Changed = false
	node.Lock.Unlock()
	lock.Unlock()
	for _, child := range node.Childs {
		child.clearNodeChangedFlag()
	}
}

func (node *ADSSymbol) isNodeChanged() (changed bool) {
	if node.Changed {
		return true
	}
	for _, child := range node.Childs {
		if len(child.Childs) > 0 {
			changed = child.isNodeChanged()
			if changed {
				return true
			}
		}
		return true
	}
	return
}

func AdsGetDllVersion() (version AdsVersion) {
	adsLock.Lock()
	cAdsVersion := C.AdsGetDllVersion()
	adsLock.Unlock()
	version = *(*AdsVersion)(unsafe.Pointer(&cAdsVersion))
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
	return
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
		err = fmt.Errorf("error %v", errInt)
	}
	return
}

func (conn *Connection) adsGetLocalAddressEx() (err error) {
	adsLock.Lock()
	errInt := C.AdsGetLocalAddressEx(C.long(conn.port), (*C.AmsAddr)(unsafe.Pointer(conn.addr)))
	adsLock.Unlock()
	if errInt != 0 {
		err = fmt.Errorf("error %v", errInt)
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
		err = fmt.Errorf("Error writing adsSyncWriteReq %s", data)
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
		err = fmt.Errorf("Error writing adsSyncWriteReq %s", data)
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
		err = fmt.Errorf("Error adsSyncReadReq")
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
		err = fmt.Errorf("Error adsSyncReadReqEx", errInt)
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
	fmt.Println("amount of data returned", amountOfDataReturned)
	data = C.GoBytes(unsafe.Pointer(cData), C.int(amountOfDataReturned))
	fmt.Println(errInt)
	if errInt != 0 {
		err = fmt.Errorf("Error adsSyncReadReqEx", errInt)
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
		err = fmt.Errorf("Error adsSyncReadWriteReq %v", errInt)
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
		err = fmt.Errorf("Error adsSyncReadWriteReq %v", errInt)
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
	fmt.Println("handle for notification", handle)
	fmt.Println("error for notification", nErrInt)

	node.NotificationHandle = handle
	lock.Lock()
	node.Connection.notificationHandles[node.NotificationHandle] = node
	lock.Unlock()

	fmt.Println(nErrInt)
	fmt.Println(node.FullName)
	fmt.Println(node.NotificationHandle)
	fmt.Println("done")
}

func (node *ADSSymbol) adsSyncAddDeviceNotificationReqEx(transMode uint32, maxDelay uint32, cycleTime uint32) {
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
	fmt.Println("handle for notification", handle)
	fmt.Println("error for notification", nErrInt)

	node.NotificationHandle = handle
	node.Connection.notificationHandles[node.NotificationHandle] = node
	fmt.Println(nErrInt)
	fmt.Println(node.FullName)
	fmt.Println(node.NotificationHandle)
	fmt.Println("done")
}

func (conn *Connection) adsSyncDelDeviceNotificationReq(handle uint32) (err error) {
	adsLock.Lock()
	nErrInt := int(C.AdsSyncDelDeviceNotificationReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		C.ulong(handle)))
	adsLock.Unlock()
	if nErrInt != 0 {
		err = fmt.Errorf("Del Notification Error %d", nErrInt)
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
		err = fmt.Errorf("Del Notification Error %d", nErrInt)
	}
	return
}

func (node *ADSSymbol) addCallback(function func(ADSSymbol)) {
	if node.ChangedHandlers == nil {
		node.ChangedHandlers = make([]func(ADSSymbol), 1)
		node.ChangedHandlers[0] = function
		return
	}
	lock.Lock()
	node.ChangedHandlers = append(node.ChangedHandlers, function)
	lock.Unlock()
}

func (node *ADSSymbol) getHandle() (err error) {
	var handle uint32
	if node.Handle != 0 {
		return
	} else {
		handleData, err := node.Connection.adsSyncReadWriteReq(
			ADSIGRP_SYM_HNDBYNAME,
			0x0,
			uint32(unsafe.Sizeof(handle)),
			[]byte(node.FullName))
		if err != nil {
			return err
		}
		handle = binary.LittleEndian.Uint32(handleData)
		node.Lock.Lock()
		node.Handle = handle
		node.Connection.handles[handle] = node
		node.Lock.Unlock()
	}
	return err
}

func (conn *Connection) getValueByHandle(handle uint32, size uint32) (data []byte, err error) {
	data, err = conn.adsSyncReadReq(
		ADSIGRP_SYM_VALBYHND,
		uint32(handle),
		uint32(size))
	return data, err
}

func (conn *Connection) releaseHandle(handle uint32) (err error) {
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(handle))
	err = conn.adsSyncWriteReq(
		ADSIGRP_SYM_RELEASEHND,
		0x0,
		a)
	if err != nil {
		// conn.handles[handle].Handle = 0
		fmt.Println("handle deleted ", handle)
	}
	return
}

func (conn *Connection) releasNotificationeHandle(handle uint32) (err error) {
	conn.adsSyncDelDeviceNotificationReq(handle)
	if err != nil {
		// conn.notificationHandles[handle].NotificationHandle = 0
		fmt.Println("notification handle deleted ", handle)
	}
	return
}

func (conn *Connection) AdsSyncReadStateReq() (adsState int, deviceState int, err error) {
	cAdsState := C.ushort(0)
	cDeviceState := C.ushort(0)
	adsLock.Lock()
	nErr := C.AdsSyncReadStateReq(
		(*C.AmsAddr)(unsafe.Pointer(conn.addr)),
		&cAdsState,
		&cDeviceState)
	adsLock.Unlock()
	if nErr != 0 {
		return 0, 0, fmt.Errorf("Error ", nErr)
	}
	return int(cAdsState), int(cDeviceState), nil
}

func RegisterRouterNotification(callback func(int)) (err error) {
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
	RouterNotification(int(response))
}
