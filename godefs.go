// Created by cgo -godefs - DO NOT EDIT
// cgo.exe -godefs cdefs.go

package main

type AmsNetId struct {
	B [6]uint8
}
type AmsAddr struct {
	NetId AmsNetId
	Port  uint16
}
type AdsVersion struct {
	Version  uint8
	Revision uint8
	Build    uint16
}
type AdsNotificationAttrib struct {
	CbLength   uint32
	NTransMode uint32
	NMaxDelay  uint32
	NCycleTime uint32
}
type AdsNotificationHeader struct {
	HNotification uint32
	Pad_cgo_0     [8]byte
	CbSampleSize  uint32
	Data          [1]uint8
}
type AdsSymbolEntry struct {
	EntryLength   uint32
	IGroup        uint32
	IOffs         uint32
	Size          uint32
	DataType      uint32
	Flags         uint32
	NameLength    uint16
	TypeLength    uint16
	CommentLength uint16
}
type AdsDatatypeArrayInfo struct {
	LBound   uint32
	Elements uint32
}
type AdsDatatypeEntry struct {
	EntryLength   uint32
	Version       uint32
	HashValue     uint32
	TypeHashValue uint32
	Size          uint32
	Offs          uint32
	DataType      uint32
	Flags         uint32
	NameLength    uint16
	TypeLength    uint16
	CommentLength uint16
	ArrayDim      uint16
	SubItems      uint16
}
type AdsSymbolUploadInfo struct {
	NSymbols uint32
	NSymSize uint32
}
type AdsSymbolUploadInfo2 struct {
	NSymbols        uint32
	NSymSize        uint32
	NDatatypes      uint32
	NDatatypeSize   uint32
	NMaxDynSymbols  uint32
	NUsedDynSymbols uint32
}
type AdsSymbolInfoByName struct {
	IndexGroup  uint32
	IndexOffset uint32
	CbLength    uint32
}

const ANYSIZE_ARRAY = 1
const ADS_FIXEDNAMESIZE = 16

////////////////////////////////////////////////////////////////////////////////
// AMS Ports
const AMSPORT_LOGGER = 100
const AMSPORT_R0_RTIME = 200
const AMSPORT_R0_TRACE = (AMSPORT_R0_RTIME + 90)
const AMSPORT_R0_IO = 300
const AMSPORT_R0_SPS = 400
const AMSPORT_R0_NC = 500
const AMSPORT_R0_ISG = 550
const AMSPORT_R0_PCS = 600
const AMSPORT_R0_PLC = 801
const AMSPORT_R0_PLC_RTS1 = 801
const AMSPORT_R0_PLC_RTS2 = 811
const AMSPORT_R0_PLC_RTS3 = 821
const AMSPORT_R0_PLC_RTS4 = 831
const AMSPORT_R0_PLC_TC3 = 851

////////////////////////////////////////////////////////////////////////////////
// ADS Cmd Ids
const ADSSRVID_INVALID = 0x00
const ADSSRVID_READDEVICEINFO = 0x01
const ADSSRVID_READ = 0x02
const ADSSRVID_WRITE = 0x03
const ADSSRVID_READSTATE = 0x04
const ADSSRVID_WRITECTRL = 0x05
const ADSSRVID_ADDDEVICENOTE = 0x06
const ADSSRVID_DELDEVICENOTE = 0x07
const ADSSRVID_DEVICENOTE = 0x08
const ADSSRVID_READWRITE = 0x09

////////////////////////////////////////////////////////////////////////////////
// ADS reserved index groups
const ADSIGRP_SYMTAB = 0xF000
const ADSIGRP_SYMNAME = 0xF001
const ADSIGRP_SYMVAL = 0xF002

const ADSIGRP_SYM_HNDBYNAME = 0xF003
const ADSIGRP_SYM_VALBYNAME = 0xF004
const ADSIGRP_SYM_VALBYHND = 0xF005
const ADSIGRP_SYM_RELEASEHND = 0xF006
const ADSIGRP_SYM_INFOBYNAME = 0xF007
const ADSIGRP_SYM_VERSION = 0xF008
const ADSIGRP_SYM_INFOBYNAMEEX = 0xF009

const ADSIGRP_SYM_DOWNLOAD = 0xF00A
const ADSIGRP_SYM_UPLOAD = 0xF00B
const ADSIGRP_SYM_UPLOADINFO = 0xF00C
const ADSIGRP_SYM_DOWNLOAD2 = 0xF00D
const ADSIGRP_SYM_DT_UPLOAD = 0xF00E
const ADSIGRP_SYM_UPLOADINFO2 = 0xF00F

const ADSIGRP_SYMNOTE = 0xF010 // notification of named handle

const ADSIGRP_SUMUP_READ = 0xF080 // AdsRW  IOffs list size or 0 (=0 -> list size == WLength/3*sizeof(ULONG))
// W: {list of IGrp, IOffs, Length}
// if IOffs != 0 then R: {list of results} and {list of data}
// if IOffs == 0 then R: only data (sum result)
const ADSIGRP_SUMUP_WRITE = 0xF081 // AdsRW  IOffs list size
// W: {list of IGrp, IOffs, Length} followed by {list of data}
// R: list of results
const ADSIGRP_SUMUP_READWRITE = 0xF082 // AdsRW  IOffs list size
// W: {list of IGrp, IOffs, RLength, WLength} followed by {list of data}
// R: {list of results, RLength} followed by {list of data}
const ADSIGRP_SUMUP_READEX = 0xF083 // AdsRW  IOffs list size
// W: {list of IGrp, IOffs, Length}
const ADSIGRP_SUMUP_READEX2 = 0xF084 // AdsRW  IOffs list size
// W: {list of IGrp, IOffs, Length}
// R: {list of results, Length} followed by {list of data (returned lengths)}
const ADSIGRP_SUMUP_ADDDEVNOTE = 0xF085 // AdsRW  IOffs list size
// W: {list of IGrp, IOffs, Attrib}
// R: {list of results, handles}
const ADSIGRP_SUMUP_DELDEVNOTE = 0xF086 // AdsRW  IOffs list size
// W: {list of handles}
// R: {list of results, Length} followed by {list of data}

const ADSIGRP_IOIMAGE_RWIB = 0xF020   // read/write input byte(s)
const ADSIGRP_IOIMAGE_RWIX = 0xF021   // read/write input bit
const ADSIGRP_IOIMAGE_RISIZE = 0xF025 // read input size (in byte)
const ADSIGRP_IOIMAGE_RWOB = 0xF030   // read/write output byte(s)
const ADSIGRP_IOIMAGE_RWOX = 0xF031   // read/write output bit
const ADSIGRP_IOIMAGE_CLEARI = 0xF040 // write inputs to null
const ADSIGRP_IOIMAGE_CLEARO = 0xF050 // write outputs to null
const ADSIGRP_IOIMAGE_RWIOB = 0xF060  // read input and write output byte(s)

const ADSIGRP_DEVICE_DATA = 0xF100       // state, name, etc...
const ADSIOFFS_DEVDATA_ADSSTATE = 0x0000 // ads state of device
const ADSIOFFS_DEVDATA_DEVSTATE = 0x0002 // device state

////////////////////////////////////////////////////////////////////////////////
// ADS Return codes
const ADSERR_NOERR = 0x00
const ERR_ADSERRS = 0x0700

const ADSERR_DEVICE_ERROR = (0x00 + ERR_ADSERRS)                // Error class < device error >
const ADSERR_DEVICE_SRVNOTSUPP = (0x01 + ERR_ADSERRS)           // Service is not supported by server
const ADSERR_DEVICE_INVALIDGRP = (0x02 + ERR_ADSERRS)           // invalid indexGroup
const ADSERR_DEVICE_INVALIDOFFSET = (0x03 + ERR_ADSERRS)        // invalid indexOffset
const ADSERR_DEVICE_INVALIDACCESS = (0x04 + ERR_ADSERRS)        // reading/writing not permitted
const ADSERR_DEVICE_INVALIDSIZE = (0x05 + ERR_ADSERRS)          // parameter size not correct
const ADSERR_DEVICE_INVALIDDATA = (0x06 + ERR_ADSERRS)          // invalid parameter value(s)
const ADSERR_DEVICE_NOTREADY = (0x07 + ERR_ADSERRS)             // device is not in a ready state
const ADSERR_DEVICE_BUSY = (0x08 + ERR_ADSERRS)                 // device is busy
const ADSERR_DEVICE_INVALIDCONTEXT = (0x09 + ERR_ADSERRS)       // invalid context (must be InWindows)
const ADSERR_DEVICE_NOMEMORY = (0x0A + ERR_ADSERRS)             // out of memory
const ADSERR_DEVICE_INVALIDPARM = (0x0B + ERR_ADSERRS)          // invalid parameter value(s)
const ADSERR_DEVICE_NOTFOUND = (0x0C + ERR_ADSERRS)             // not found (files, ...)
const ADSERR_DEVICE_SYNTAX = (0x0D + ERR_ADSERRS)               // syntax error in comand or file
const ADSERR_DEVICE_INCOMPATIBLE = (0x0E + ERR_ADSERRS)         // objects do not match
const ADSERR_DEVICE_EXISTS = (0x0F + ERR_ADSERRS)               // object already exists
const ADSERR_DEVICE_SYMBOLNOTFOUND = (0x10 + ERR_ADSERRS)       // symbol not found
const ADSERR_DEVICE_SYMBOLVERSIONINVALID = (0x11 + ERR_ADSERRS) // symbol version invalid
const ADSERR_DEVICE_INVALIDSTATE = (0x12 + ERR_ADSERRS)         // server is in invalid state
const ADSERR_DEVICE_TRANSMODENOTSUPP = (0x13 + ERR_ADSERRS)     // AdsTransMode not supported
const ADSERR_DEVICE_NOTIFYHNDINVALID = (0x14 + ERR_ADSERRS)     // Notification handle is invalid
const ADSERR_DEVICE_CLIENTUNKNOWN = (0x15 + ERR_ADSERRS)        // Notification client not registered
const ADSERR_DEVICE_NOMOREHDLS = (0x16 + ERR_ADSERRS)           // no more notification handles
const ADSERR_DEVICE_INVALIDWATCHSIZE = (0x17 + ERR_ADSERRS)     // size for watch to big
const ADSERR_DEVICE_NOTINIT = (0x18 + ERR_ADSERRS)              // device not initialized
const ADSERR_DEVICE_TIMEOUT = (0x19 + ERR_ADSERRS)              // device has a timeout
const ADSERR_DEVICE_NOINTERFACE = (0x1A + ERR_ADSERRS)          // query interface failed
const ADSERR_DEVICE_INVALIDINTERFACE = (0x1B + ERR_ADSERRS)     // wrong interface required
const ADSERR_DEVICE_INVALIDCLSID = (0x1C + ERR_ADSERRS)         // class ID is invalid
const ADSERR_DEVICE_INVALIDOBJID = (0x1D + ERR_ADSERRS)         // object ID is invalid
const ADSERR_DEVICE_PENDING = (0x1E + ERR_ADSERRS)              // request is pending
const ADSERR_DEVICE_ABORTED = (0x1F + ERR_ADSERRS)              // request is aborted
const ADSERR_DEVICE_WARNING = (0x20 + ERR_ADSERRS)              // signal warning
const ADSERR_DEVICE_INVALIDARRAYIDX = (0x21 + ERR_ADSERRS)      // invalid array index
const ADSERR_DEVICE_SYMBOLNOTACTIVE = (0x22 + ERR_ADSERRS)      // symbol not active -> release handle and try again
const ADSERR_DEVICE_ACCESSDENIED = (0x23 + ERR_ADSERRS)         // access denied
const ADSERR_DEVICE_LICENSENOTFOUND = (0x24 + ERR_ADSERRS)      // no license found
const ADSERR_DEVICE_LICENSEEXPIRED = (0x25 + ERR_ADSERRS)       // license expired
const ADSERR_DEVICE_LICENSEEXCEEDED = (0x26 + ERR_ADSERRS)      // license exceeded
const ADSERR_DEVICE_LICENSEINVALID = (0x27 + ERR_ADSERRS)       // license invalid
const ADSERR_DEVICE_LICENSESYSTEMID = (0x28 + ERR_ADSERRS)      // license invalid system id
const ADSERR_DEVICE_LICENSENOTIMELIMIT = (0x29 + ERR_ADSERRS)   // license not time limited
const ADSERR_DEVICE_LICENSEFUTUREISSUE = (0x2A + ERR_ADSERRS)   // license issue time in the future
const ADSERR_DEVICE_LICENSETIMETOLONG = (0x2B + ERR_ADSERRS)    // license time period to long
const ADSERR_DEVICE_EXCEPTION = (0x2C + ERR_ADSERRS)            // exception in device specific code
const ADSERR_DEVICE_LICENSEDUPLICATED = (0x2D + ERR_ADSERRS)    // license file read twice
const ADSERR_DEVICE_SIGNATUREINVALID = (0x2E + ERR_ADSERRS)     // invalid signature
const ADSERR_DEVICE_CERTIFICATEINVALID = (0x2F + ERR_ADSERRS)   // public key certificate
//
const ADSERR_CLIENT_ERROR = (0x40 + ERR_ADSERRS)          // Error class < client error >
const ADSERR_CLIENT_INVALIDPARM = (0x41 + ERR_ADSERRS)    // invalid parameter at service call
const ADSERR_CLIENT_LISTEMPTY = (0x42 + ERR_ADSERRS)      // polling list	is empty
const ADSERR_CLIENT_VARUSED = (0x43 + ERR_ADSERRS)        // var connection already in use
const ADSERR_CLIENT_DUPLINVOKEID = (0x44 + ERR_ADSERRS)   // invoke id in use
const ADSERR_CLIENT_SYNCTIMEOUT = (0x45 + ERR_ADSERRS)    // timeout elapsed
const ADSERR_CLIENT_W32ERROR = (0x46 + ERR_ADSERRS)       // error in win32 subsystem
const ADSERR_CLIENT_TIMEOUTINVALID = (0x47 + ERR_ADSERRS) // ?
const ADSERR_CLIENT_PORTNOTOPEN = (0x48 + ERR_ADSERRS)    // ads dll
const ADSERR_CLIENT_NOAMSADDR = (0x49 + ERR_ADSERRS)      // ads dll
const ADSERR_CLIENT_SYNCINTERNAL = (0x50 + ERR_ADSERRS)   // internal error in ads sync
const ADSERR_CLIENT_ADDHASH = (0x51 + ERR_ADSERRS)        // hash table overflow
const ADSERR_CLIENT_REMOVEHASH = (0x52 + ERR_ADSERRS)     // key not found in hash table
const ADSERR_CLIENT_NOMORESYM = (0x53 + ERR_ADSERRS)      // no more symbols in cache
const ADSERR_CLIENT_SYNCRESINVALID = (0x54 + ERR_ADSERRS) // invalid response received
const ADSERR_CLIENT_SYNCPORTLOCKED = (0x55 + ERR_ADSERRS) // sync port is locked
