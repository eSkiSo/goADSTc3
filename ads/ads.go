package ads

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

type updateStruct struct {
	variable  string
	timestamp time.Time
	value     []byte
}

// NotificationStruct is the structure to notify clients of notifications
type NotificationStruct struct {
	Variable  string
	Value     string
	TimeStamp time.Time
}

var connection *adsConnection

type adsConnection struct {
	addr       *amsAddr
	symbolLock sync.Mutex

	SymbolsLoaded bool
	update        chan updateStruct

	symbols             map[string]*adsSymbol
	datatypes           map[string]adsSymbolUploadDataType
	handles             map[uint32]string
	notificationHandles map[uint32]string
	pollVariables       []string
}

type adsSymbol struct {
	FullName           string
	LastUpdateTime     time.Time
	MinUpdateInterval  time.Duration
	Name               string
	DataType           string
	Comment            string
	Handle             uint32
	NotificationHandle uint32
	Group              uint32
	Offset             uint32
	Length             uint32
	Changed            bool

	Value string
	Valid bool

	Parent *adsSymbol
	Childs map[string]*adsSymbol
}

// Client is the consumer facing struct to manage connections
type Client struct {
	port         int
	adsLock      sync.Mutex
	Notification chan NotificationStruct
}

var client *Client

func init() {
	client = &Client{}
	port := portOpenEx()
	client.port = port
	client.Notification = make(chan NotificationStruct, 100)
	connection = &adsConnection{}
	connection.update = make(chan updateStruct, 100)
	connection.symbols = map[string]*adsSymbol{}
	connection.datatypes = map[string]adsSymbolUploadDataType{}
	connection.handles = map[uint32]string{}
	connection.notificationHandles = map[uint32]string{}
}

// AddLocalConnection adds a connection to localhost
func AddLocalConnection(ctx context.Context) error {
	getLocalAddressEx()
	fmt.Printf("local connection at %d %d %d \n", client.port, connection.addr.Port, connection.addr.NetId[0])
	connection.addr.Port = 851

	err := initializeConnVariables()
	if err != nil {
		return err
	}
	go readWritePump(ctx)
	return nil
}

// AddRemoteConnection adds a connection to outside computer
func AddRemoteConnection(ctx context.Context, netID string, port uint16) (*Client, error) {
	log.Println("local package")
	adsVersion := GetDllVersion()
	log.Printf("Ads Version: Version: %v, Build %v, Revision %v", adsVersion.Version, adsVersion.Build, adsVersion.Revision)

	connection.addr = &amsAddr{}
	connection.addr.NetId = stringToNetID(netID)
	fmt.Printf("Remote connection at Port: %d Address: %d %d %d %d %d %d\n",
		connection.addr.Port,
		connection.addr.NetId[0],
		connection.addr.NetId[1],
		connection.addr.NetId[2],
		connection.addr.NetId[3],
		connection.addr.NetId[4],
		connection.addr.NetId[5])
	connection.addr.Port = port

	err := initializeConnVariables()
	if err != nil {
		return nil, err
	}
	go readWritePump(ctx)
	return client, nil
}

func initializeConnVariables() error {
	uploadInfo, err := getSymbolUploadInfo()
	if err != nil {
		return err
	}
	fmt.Println("uploadinfo  loaded", uploadInfo.NDatatypeSize, uploadInfo.NSymSize)
	err = uploadSymbolInfoDataTypes(uploadInfo.NDatatypeSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoDataTypes  loaded")
	err = uploadSymbolInfoSymbols(uploadInfo.NSymSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoSymbols  loaded")
	connection.SymbolsLoaded = true
	return nil
}

// CloseConnection closes current connection
func CloseConnection() {
	for k := range connection.notificationHandles {
		err := releaseNotificationeHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted notification handle %d\n", k)
		}
	}
	for k := range connection.handles {
		err := releaseHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted handle %d\n", k)
		}
	}
	return
}

func showComments(info *adsSymbolUploadDataType) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showComments(value)
	}
}

func showInfoComments(info *adsSymbol) {
	fmt.Println(info.Name)
	for _, v := range info.Childs {
		showInfoComments(v)
	}
}

func readWritePump(ctx context.Context) {
ForLoop:
	for {
		select {
		case s := <-connection.update:
			connection.symbolLock.Lock()
			symbol, err := getSymbol(s.variable)
			if err != nil {
				continue
			}
			symbol.parse(s.value, 0)
			client.Notification <- NotificationStruct{Variable: s.variable, Value: symbol.getJSON(true), TimeStamp: s.timestamp}
			symbol.clearChanged()
			connection.symbolLock.Unlock()
		case <-time.After(50 * time.Millisecond):
			for _, pollVariable := range connection.pollVariables {
				connection.symbolLock.Lock()
				symbol, err := getSymbol(pollVariable)
				err = updateVariable(pollVariable)
				if err != nil {
					fmt.Printf("error here %v/n", err)
				} else {
					value := symbol.getJSON(true)
					client.Notification <- NotificationStruct{Variable: pollVariable, Value: value, TimeStamp: time.Now()}
				}
				connection.symbolLock.Unlock()

			}
		case <-ctx.Done():
			break ForLoop
		}
	}
}

func getSymbol(variable string) (*adsSymbol, error) {
	symbol, ok := connection.symbols[variable]
	if !ok {
		return nil, fmt.Errorf("symbol not found")
	}
	if symbol.Handle != 0 {
		return symbol, nil
	}
	handle, err := getHandleByString(variable)
	if err != nil {
		return nil, fmt.Errorf("unable to get handle for symbol %w", err)
	}
	symbol.Handle = handle
	return symbol, nil
}

// updateVariable returns value from PLC in string format
func updateVariable(variable string) error {
	symbol, err := getSymbol(variable)
	if err != nil {
		return fmt.Errorf("symbol not found")
	}
	if time.Since(symbol.LastUpdateTime) > symbol.MinUpdateInterval {
		data, err := getValueByHandle(
			symbol.Handle,
			symbol.Length)
		if err != nil {
			symbol.Handle = 0
			return err
		}
		symbol.parse(data, 0)
	}
	return nil
}

// Write writes value to variable
func Write(variable string, value string) error {
	connection.symbolLock.Lock()
	defer connection.symbolLock.Unlock()
	symbol, err := getSymbol(variable)
	if err != nil {
		return fmt.Errorf("symbol not found")
	}
	symbol.writeToNode(value, 0)
	return nil

}

// Read writes value to variable
func Read(variable string) (string, error) {
	connection.symbolLock.Lock()
	defer connection.symbolLock.Unlock()
	symbol, err := getSymbol(variable)
	if err != nil {
		return "", fmt.Errorf("symbol not found")
	}
	value := symbol.getJSON(true)
	return value, nil
}

// AddNotification adds
func AddNotification(variable string, mode AdsTransMode, cycleTime time.Duration, maxTime time.Duration) {
	connection.symbolLock.Lock()
	defer connection.symbolLock.Unlock()
	symbol, err := getSymbol(variable)
	if err != nil {
		return
	}
	handle, err := syncAddDeviceNotificationReqEx(symbol.Handle, symbol.Length, mode, uint32(maxTime), uint32(cycleTime))
	connection.notificationHandles[handle] = symbol.FullName
	// addNotificationChannel(variable, ADSTRANS_SERVERONCHA, 5*time.Millisecond, 5*time.Millisecond)
}

// GetJSON (onlyChanged bool) string
func (node *adsSymbol) getJSON(onlyChanged bool) string {
	data := node.parseNode(onlyChanged)
	if jsonData, err := json.Marshal(data); err == nil {
		return string(jsonData)
	}
	return ""
}

var openBracketRegex = regexp.MustCompile(`\[`)
var closeBracketRegex = regexp.MustCompile(`\]`)

// ParseNode returns JSON interface for symbol
func (node *adsSymbol) parseNode(onlyChanged bool) (rData interface{}) {
	if len(node.Childs) == 0 {
		rData = node.Value
		// node.Changed = false
	} else {
		localMap := make(map[string]interface{})
		for _, child := range node.Childs {
			if onlyChanged {
				if child.Changed {
					s := openBracketRegex.ReplaceAllString(child.Name, `"[`)
					s = closeBracketRegex.ReplaceAllString(s, `]"`)
					localMap[s] = child.parseNode(true)
					// child.Changed = false
				}
			} else {
				s := openBracketRegex.ReplaceAllString(child.Name, `"[`)
				s = closeBracketRegex.ReplaceAllString(s, `]"`)
				localMap[s] = child.parseNode(false)
			}
		}
		rData = localMap
	}
	return
}

func (node *adsSymbol) clearChanged() {
	for _, localNode := range node.Childs {
		localNode.clearChanged()
	}
	node.Changed = false
}
