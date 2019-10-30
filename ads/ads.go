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
	notificationIndex int
	timestamp         time.Time
	value             []byte
}

// NotificationStruct is the structure to notify clients of notifications
type NotificationStruct struct {
	Variable  string
	Value     string
	TimeStamp time.Time
}

type Connection struct {
	addr       *amsAddr
	symbolLock sync.Mutex

	SymbolsLoaded bool

	symbols       map[string]*Symbol
	datatypes     map[string]symbolUploadDataType
	handles       map[uint32]string
	pollVariables []string
	Update        chan NotificationStruct
}

type Symbol struct {
	connection         *Connection
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

	Parent *Symbol
	Childs map[string]*Symbol
}

type clientNotification struct {
	symbol *Symbol
	handle uint32
}

// Client is the consumer facing struct to manage connections
type Client struct {
	ctx           context.Context
	cancel        context.CancelFunc
	port          int
	adsLock       sync.Mutex
	notifications []*clientNotification
	connections   []*Connection
	update        chan updateStruct
}

var client *Client

func init() {
	client = &Client{}
	client.update = make(chan updateStruct)
	port := portOpenEx()
	client.port = port
	client.ctx, client.cancel = context.WithCancel(context.Background())
	go readWritePump(client.ctx)
}

func initConnection() *Connection {
	connection := &Connection{}
	connection.Update = make(chan NotificationStruct)
	connection.symbols = make(map[string]*Symbol)
	connection.datatypes = make(map[string]symbolUploadDataType)
	connection.handles = make(map[uint32]string)
	return connection
}

// AddLocalConnection adds a connection to localhost
func AddLocalConnection(ctx context.Context) (*Connection, error) {
	connection := initConnection()

	connection.getLocalAddressEx()
	fmt.Printf("local connection at %d %d %d \n", client.port, connection.addr.Port, connection.addr.NetId[0])
	connection.addr.Port = 851

	err := connection.initializeConnVariables()
	if err != nil {
		return nil, err
	}
	client.connections = append(client.connections, connection)
	return connection, nil
}

// AddRemoteConnection adds a connection to outside computer
func AddRemoteConnection(ctx context.Context, netID string, port uint16) (*Connection, error) {
	connection := initConnection()

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

	err := connection.initializeConnVariables()
	if err != nil {
		return nil, err
	}

	client.connections = append(client.connections, connection)
	return connection, nil
}

func (connection *Connection) initializeConnVariables() error {
	uploadInfo, err := connection.getSymbolUploadInfo()
	if err != nil {
		return err
	}
	fmt.Println("uploadinfo  loaded", uploadInfo.NDatatypeSize, uploadInfo.NSymSize)
	err = connection.uploadSymbolInfoDataTypes(uploadInfo.NDatatypeSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoDataTypes  loaded")
	err = connection.uploadSymbolInfoSymbols(uploadInfo.NSymSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoSymbols  loaded")
	connection.SymbolsLoaded = true
	return nil
}

func Shutdown() {
	client.cancel()
	for _, connection := range client.connections {
		connection.closeConnection()
	}
	closeClient()
}

// CloseConnection closes current connection
func (connection *Connection) closeConnection() {
	for k := range connection.handles {
		err := connection.releaseHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted handle %d\n", k)
		}
	}
	return
}

func closeClient() {
	for _, k := range client.notifications {
		err := k.symbol.connection.releaseNotificationeHandle(k.handle)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted notification handle %d\n", k)
		}
	}
}

func (info *symbolUploadDataType) showComments() {
	fmt.Println(info.Name)
	for _, v := range info.Childs {
		v.showComments()
	}
}

func (symbol *Symbol) showInfoComments() {
	fmt.Println(symbol.Name)
	for _, v := range symbol.Childs {
		v.showInfoComments()
	}
}

func readWritePump(ctx context.Context) {
ForLoop:
	for {
		select {
		case s := <-client.update:
			notification := client.notifications[s.notificationIndex]
			symbol := notification.symbol
			symbol.connection.symbolLock.Lock()
			symbol.parse(s.value, 0)
			hmiNotification := NotificationStruct{Variable: symbol.FullName, Value: symbol.GetJSON(true), TimeStamp: s.timestamp}
			symbol.connection.Update <- hmiNotification
			symbol.clearChanged()
			symbol.connection.symbolLock.Unlock()
			// connection.symbolLock.Unlock()
		// case <-time.After(50 * time.Millisecond):
		// 	for _, pollVariable := range connection.pollVariables {
		// 		connection.symbolLock.Lock()
		// 		symbol, err := connection.GetSymbol(pollVariable)
		// 		err = symbol.updateVariable()
		// 		if err != nil {
		// 			fmt.Printf("error here %v/n", err)
		// 		} else {
		// 			value := symbol.getJSON(true)
		// 			client.Notification <- NotificationStruct{Variable: pollVariable, Value: value, TimeStamp: time.Now()}
		// 		}
		// 		connection.symbolLock.Unlock()
		// 	}
		case <-ctx.Done():
			break ForLoop
		}
	}
}

// GetSymbol retrieves symbol based on FullName
func (connection *Connection) GetSymbol(variable string) (*Symbol, error) {
	symbol, ok := connection.symbols[variable]
	if !ok {
		return nil, fmt.Errorf("symbol not found")
	}
	if symbol.Handle != 0 {
		return symbol, nil
	}
	handle, err := connection.getHandleByString(variable)
	if err != nil {
		return nil, fmt.Errorf("unable to get handle for symbol %w", err)
	}
	connection.handles[handle] = symbol.FullName
	symbol.Handle = handle
	return symbol, nil
}

// Write writes value to variable
func (connection *Connection) Write(variable string, value string) error {
	connection.symbolLock.Lock()
	symbol, err := connection.GetSymbol(variable)
	if err != nil {
		connection.symbolLock.Unlock()
		return fmt.Errorf("symbol not found")
	}
	connection.symbolLock.Unlock()
	symbol.Write(value)
	return nil

}

func (symbol *Symbol) Write(value string) error {
	symbol.connection.symbolLock.Lock()
	err := symbol.writeToNode(value, 0)
	symbol.connection.symbolLock.Unlock()
	if err != nil {
		fmt.Println(err)
	}
	return err
}

// Read writes value to variable
func (connection *Connection) Read(variable string) (string, error) {
	connection.symbolLock.Lock()
	symbol, err := connection.GetSymbol(variable)
	if err != nil {
		connection.symbolLock.Unlock()
		return "", fmt.Errorf("error: %w", err)
	}
	connection.symbolLock.Unlock()
	value, _ := symbol.Read()
	return value, nil
}

func (symbol *Symbol) Read() (string, error) {
	symbol.connection.symbolLock.Lock()
	err := symbol.updateVariable()
	if err != nil {
		symbol.connection.symbolLock.Unlock()
		return "", err
	}
	value := symbol.GetJSON(false)
	symbol.connection.symbolLock.Unlock()
	return value, nil
}

// AddNotification adds
func (connection *Connection) AddNotification(variable string, mode AdsTransMode, cycleTime time.Duration, maxTime time.Duration) {
	connection.symbolLock.Lock()
	defer connection.symbolLock.Unlock()
	symbol, err := connection.GetSymbol(variable)
	if err != nil {
		return
	}
	symbol.AddNotification(mode, maxTime, cycleTime)
}

// AddNotification adds
func (symbol *Symbol) AddNotification(mode AdsTransMode, cycleTime time.Duration, maxTime time.Duration) {
	notification := &clientNotification{
		symbol: symbol,
	}
	client.notifications = append(client.notifications, notification)
	index := len(client.notifications) - 1
	handle, _ := symbol.connection.syncAddDeviceNotificationReqEx(symbol.Handle, symbol.Length, mode, uint32(maxTime), uint32(cycleTime), uint32(index))
	notification.handle = handle
}

// updateVariable returns value from PLC in string format
func (symbol *Symbol) updateVariable() error {
	if time.Since(symbol.LastUpdateTime) > symbol.MinUpdateInterval {
		data, err := symbol.connection.getValueByHandle(
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

// GetJSON (onlyChanged bool) string
func (symbol *Symbol) GetJSON(onlyChanged bool) string {
	data := symbol.parseSymbol(onlyChanged)
	if jsonData, err := json.Marshal(data); err == nil {
		return string(jsonData)
	}
	return ""
}

var openBracketRegex = regexp.MustCompile(`\[`)
var closeBracketRegex = regexp.MustCompile(`\]`)

// parseSymbol returns JSON interface for symbol
func (symbol *Symbol) parseSymbol(onlyChanged bool) (rData interface{}) {
	if len(symbol.Childs) == 0 {
		rData = symbol.Value
		// symbol.Changed = false
	} else {
		localMap := make(map[string]interface{})
		for _, child := range symbol.Childs {
			s := openBracketRegex.ReplaceAllString(child.Name, `"[`)
			s = closeBracketRegex.ReplaceAllString(s, `]"`)
			if onlyChanged {
				if child.Changed {
					localMap[s] = child.parseSymbol(true)
					// child.Changed = false
				}
			} else {
				localMap[s] = child.parseSymbol(false)
			}
		}
		rData = localMap
	}
	return
}

func (symbol *Symbol) clearChanged() {
	for _, localsymbol := range symbol.Childs {
		localsymbol.clearChanged()
	}
	symbol.Changed = false
}
