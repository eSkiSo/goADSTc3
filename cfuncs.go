package main

/*
#cgo CFLAGS: -I .
#cgo LDFLAGS: -LC:/TwinCAT/AdsApi/TcAdsDll/x64/lib -lTcAdsDll
#include <stdbool.h>
#include <stdlib.h>
#include <inttypes.h>
#define BOOL bool
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsDef.h"
#include "C:/TwinCAT/AdsApi/TcAdsDll/Include/TcAdsAPI.h"

void  notificationFun(AmsAddr*, AdsNotificationHeader*,unsigned long);

void  Callback(AmsAddr* pAddr, AdsNotificationHeader* pNotification, unsigned long hUser) {
	notificationFun(pAddr, pNotification, hUser);
}

*/
import "C"
