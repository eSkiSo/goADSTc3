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

extern void notificationFun(AmsAddr*, AdsNotificationHeader*, unsigned long);
extern void  routerNotificationFun(long);
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

func adsAmsPortEnabledEx(port int) (bool, error) {
	var portOpen C.bool
	var cPort = C.long(port)
	client.adsLock.Lock()
	errInt := C.AdsAmsPortEnabledEx(cPort, &portOpen)
	client.adsLock.Unlock()
	if errInt != 0 && errInt != 1864 {
		return false, fmt.Errorf("error checking port %w", errInt)
	}
	return bool(portOpen), nil
}

//export notificationFun
func notificationFun(addr *C.AmsAddr, notification *C.AdsNotificationHeader, user C.ulong) {
	// goamsAddr := (*AmsAddr)(unsafe.Pointer(addr))
	cdata := C.GoBytes(unsafe.Pointer(notification), C.sizeof_AdsNotificationHeader)
	buf := bytes.NewBuffer(cdata)
	notificationHeader := &AdsNotificationHeader{}
	binary.Read(buf, binary.LittleEndian, &notificationHeader.HNotification)
	binary.Read(buf, binary.LittleEndian, &notificationHeader.Timestamp)
	binary.Read(buf, binary.LittleEndian, &notificationHeader.CbSampleSize)
	cBytes := C.GoBytes(unsafe.Pointer(&notification.data), C.int(notification.cbSampleSize))
	connection.symbolLock.Lock()
	variable, ok := connection.notificationHandles[uint32(notification.hNotification)]
	connection.symbolLock.Unlock()
	if !ok {
		fmt.Printf("notification error: %w", uint32(notification.hNotification))
		return
	}
	unixTime := time.Unix(int64(notificationHeader.Timestamp/10000000)-11644473600, 0)
	var update = updateStruct{}
	update.variable = variable
	update.value = cBytes
	update.timestamp = unixTime
	connection.update <- update
	return
}

// GetDllVersion gets ads dll version number
func GetDllVersion() AdsVersion {
	version := &AdsVersion{}
	client.adsLock.Lock()
	cAdsVersion := C.AdsGetDllVersion()
	client.adsLock.Unlock()
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(cAdsVersion))
	buf := bytes.NewBuffer(b)
	binary.Read(buf, binary.LittleEndian, version.Build)
	binary.Read(buf, binary.LittleEndian, version.Revision)
	binary.Read(buf, binary.LittleEndian, version.Version)
	return *version
}

// PortOpenEx opens AdsPort
func portOpenEx() int {
	client.adsLock.Lock()
	port := int(C.AdsPortOpenEx())
	client.adsLock.Unlock()
	return port
}

// PortCloseEx opens AdsPort
func portCloseEx(port int) error {
	client.adsLock.Lock()
	errInt := C.AdsPortCloseEx(C.long(port))
	client.adsLock.Unlock()
	if errInt != 0 {
		return fmt.Errorf(string(errInt))
	}
	return nil
}

// GetLocalAddressEx gets local NetId
func getLocalAddressEx() error {
	client.adsLock.Lock()
	errInt := C.AdsGetLocalAddressEx(C.long(client.port), (*C.AmsAddr)(unsafe.Pointer(connection.addr)))
	client.adsLock.Unlock()
	if errInt != 0 {
		return fmt.Errorf("adsGetLocalAddressEx error: %w", errInt)
	}
	return nil
}

func stringToNetID(amsID string) (id amsNetId) {
	stringBytes := strings.Split(amsID, ".")
	byte0, _ := strconv.Atoi(stringBytes[0])
	byte1, _ := strconv.Atoi(stringBytes[1])
	byte2, _ := strconv.Atoi(stringBytes[2])
	byte3, _ := strconv.Atoi(stringBytes[3])
	byte4, _ := strconv.Atoi(stringBytes[4])
	byte5, _ := strconv.Atoi(stringBytes[5])

	id[0] = uint8(byte0)
	id[1] = uint8(byte1)
	id[2] = uint8(byte2)
	id[3] = uint8(byte3)
	id[4] = uint8(byte4)
	id[5] = uint8(byte5)
	return id
}

func syncWriteReqEx(group uint32, offset uint32, data []byte) error {
	cDataToWrite := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToWrite))
	client.adsLock.Lock()
	errInt := int(C.AdsSyncWriteReqEx(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(len(data)),
		unsafe.Pointer(cDataToWrite)))
	client.adsLock.Unlock()
	if errInt != 0 {
		return fmt.Errorf("error writing adsSyncWriteReq %w", errInt)
	}
	return nil
}

func syncReadReqEx2(group uint32, offset uint32, length uint32) (data []byte, err error) {
	amountOfDataReturned := C.ulong(length)
	cData := C.CString(string(make([]byte, length)))
	defer C.free(unsafe.Pointer(cData))
	client.adsLock.Lock()
	errInt := int(C.AdsSyncReadReqEx2(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(length),
		unsafe.Pointer(cData),
		&amountOfDataReturned))
	client.adsLock.Unlock()
	// fmt.Println("amount of data returned", amountOfDataReturned)
	data = C.GoBytes(unsafe.Pointer(cData), C.int(amountOfDataReturned))
	//fmt.Println(errInt)
	if errInt != 0 {
		return nil, fmt.Errorf("error adsSyncReadReqEx: %w", errInt)
	}
	return data, err
}

func syncReadWriteReqEx2(group uint32, offset uint32, readLength uint32, dataToWrite []byte) (data []byte, err error) {
	data = make([]byte, readLength)
	cDataToRead := C.CString(string(data))
	defer C.free(unsafe.Pointer(cDataToRead))

	cDataToWrite := C.CString(string(dataToWrite))
	defer C.free(unsafe.Pointer(cDataToWrite))

	cLengthOfReturnedBytes := C.ulong(0)
	client.adsLock.Lock()
	errInt := int(C.AdsSyncReadWriteReqEx2(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		C.ulong(group),
		C.ulong(offset),
		C.ulong(readLength),
		unsafe.Pointer(cDataToRead),
		C.ulong(len(dataToWrite)),
		unsafe.Pointer(cDataToWrite),
		&cLengthOfReturnedBytes))
	client.adsLock.Unlock()
	data = C.GoBytes(unsafe.Pointer(cDataToRead), C.int(readLength))
	if errInt != 0 {
		return nil, fmt.Errorf("error adsSyncReadWriteReq %w", errInt)
	}
	return data, err
}

func syncAddDeviceNotificationReqEx(handle uint32, size uint32, transMode AdsTransMode, maxDelay uint32, cycleTime uint32) (uint32, error) {
	notAttrib := AdsNotificationAttrib{}
	notAttrib.NMaxDelay = uint32(maxDelay / 100.0)
	notAttrib.NCycleTime = uint32(cycleTime / 100.0)
	notAttrib.CbLength = size
	notAttrib.NTransMode = uint32(transMode)

	hNotification := C.ulong(0)
	client.adsLock.Lock()
	nErrInt := int(C.AdsSyncAddDeviceNotificationReqEx(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		ADSIGRP_SYM_VALBYHND,
		C.ulong(handle),
		(*C.AdsNotificationAttrib)(unsafe.Pointer(&notAttrib)),
		(C.PAdsNotificationFuncEx)(C.notificationFun),
		C.ulong(0),
		&hNotification))
	client.adsLock.Unlock()
	if nErrInt != 0 {
		return 0, fmt.Errorf("could not create notification %w", nErrInt)
	}
	notHandle := uint32(hNotification)

	fmt.Printf("Notification Added - Handle: %d\n", handle)
	return notHandle, nil
}

func syncDelDeviceNotificationReqEx(handle uint32) (err error) {
	client.adsLock.Lock()
	nErrInt := int(C.AdsSyncDelDeviceNotificationReqEx(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		C.ulong(handle)))
	client.adsLock.Unlock()
	if nErrInt != 0 {
		return fmt.Errorf("del notification error %w", nErrInt)
	}
	return
}

func writeBuffArrayEx(handle uint32, data []byte) {
	syncWriteReqEx(
		ADSIGRP_SYM_VALBYHND,
		uint32(handle),
		data)
}

func getHandleByString(variableName string) (handle uint32, err error) {
	handleData, err := syncReadWriteReqEx2(
		ADSIGRP_SYM_HNDBYNAME,
		0x0,
		uint32(unsafe.Sizeof(handle)),
		[]byte(variableName))
	if err != nil {
		return 0, err
	}
	handle = binary.LittleEndian.Uint32(handleData)

	return handle, nil
}

func getValueByHandle(handle uint32, size uint32) (data []byte, err error) {
	data, err = syncReadReqEx2(
		ADSIGRP_SYM_VALBYHND,
		uint32(handle),
		uint32(size))
	return data, err
}

func releaseHandle(handle uint32) error {
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(handle))
	err := syncWriteReqEx(
		ADSIGRP_SYM_RELEASEHND,
		0x0,
		a)
	if err != nil {
		// conn.handles[handle].Handle = 0
		return fmt.Errorf("handle not deleted %w", err)
	}
	return nil
}

func releaseNotificationeHandle(handle uint32) (err error) {
	syncDelDeviceNotificationReqEx(handle)
	if err != nil {
		// conn.notificationHandles[handle].NotificationHandle = 0
		return fmt.Errorf("notification handle not deleted %d err: %w", handle, err)
	}
	return nil
}

func syncReadStateReqEx() (adsState int, deviceState int, err error) {
	cAdsState := C.ushort(0)
	cDeviceState := C.ushort(0)
	client.adsLock.Lock()
	nErr := C.AdsSyncReadStateReqEx(
		C.long(client.port),
		(*C.AmsAddr)(unsafe.Pointer(connection.addr)),
		&cAdsState,
		&cDeviceState)
	client.adsLock.Unlock()
	if nErr != 0 {
		return 0, 0, fmt.Errorf("error %d", nErr)
	}
	return int(cAdsState), int(cDeviceState), nil
}

func registerRouterNotification() error {
	log.Println("adding router notification")
	client.adsLock.Lock()
	C.AdsAmsRegisterRouterNotification(
		(C.PAmsRouterNotificationFuncEx)(C.routerNotificationFun),
	)
	client.adsLock.Unlock()
	return nil
}

func unregisterRouterNotification() error {
	client.adsLock.Lock()
	C.AdsAmsUnRegisterRouterNotification()
	client.adsLock.Unlock()
	return nil
}

//export routerNotificationFun
func routerNotificationFun(response C.long) {
	log.Printf("notification received %d\n", response)
	// for _, client := range routerNotificationClients {
	// 	client <- int(response)
	// }
}
