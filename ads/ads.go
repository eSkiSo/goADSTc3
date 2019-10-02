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

type WriteStruct struct {
	Variable string
	Value    string
}

type updateStruct struct {
	variable  string
	timestamp time.Time
	value     []byte
}

type NotificationStruct struct {
	Variable  string
	Value     string
	TimeStamp time.Time
}

type Connection struct {
	addr                   *AmsAddr
	port                   int
	SymbolsLoaded          bool
	Write                  chan WriteStruct
	WriteRead              chan WriteStruct
	Update                 chan updateStruct
	UpdateResponse         chan string
	Read                   chan string
	ReadResponse           chan string
	AddNotification        chan string
	AddPollingNotification chan string
	Notification           chan NotificationStruct

	symbols             map[string]*ADSSymbol
	datatypes           map[string]ADSSymbolUploadDataType
	handles             map[uint32]string
	notificationHandles map[uint32]string
	pollVariables       []string
}

type ChangedRepsonse struct {
	Variable string
	Value    string
}

type ADSSymbol struct {
	Connection         *Connection
	Self               *ADSSymbol
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

	Parent *ADSSymbol
	Childs map[string]*ADSSymbol
}

var registeredRouterNotification bool
var routerNotificationClients []chan int

var adsLock *sync.Mutex

func init() {
	adsLock = &sync.Mutex{}
}

func AddRouterNotification(response chan int) {
	if !registeredRouterNotification {
		RegisterRouterNotification()
		registeredRouterNotification = true
	}
	routerNotificationClients = append(routerNotificationClients, response)
}

func AddLocalConnection(ctx context.Context) (conn *Connection, err error) {
	localConnection := Connection{}
	open, err := adsAmsPortEnabled()
	if err != nil {
		return nil, err
	}

	if !open {
		localConnection.port = adsPortOpenEx()
		fmt.Println(localConnection.port)
	}
	if err != nil {
		return nil, err
	}

	localConnection.addr = &AmsAddr{}
	localConnection.adsGetLocalAddressEx()
	fmt.Printf("local connection at %d %d %d \n", localConnection.port, localConnection.addr.Port, localConnection.addr.NetId.B[0])
	localConnection.addr.Port = 851

	localConnection.initializeConnection()
	err = localConnection.initializeConnVariables()
	if err != nil {
		return
	}

	connections = append(connections, &localConnection)
	conn = &localConnection
	go localConnection.readWritePump(ctx)
	return
}

func AddRemoteConnection(ctx context.Context, netID string, port uint16) (conn *Connection, err error) {
	log.Println("local package")
	adsVersion := AdsGetDllVersion()
	log.Printf("Ads Version: Version: %v, Build %v, Revision %v", adsVersion.Version, adsVersion.Build, adsVersion.Revision)
	localConnection := Connection{}

	open, err := adsAmsPortEnabled()
	if !open {
		localConnection.port = adsPortOpenEx()
		fmt.Println(localConnection.port)
	}
	if err != nil {
		return nil, err
	}
	localConnection.addr = &AmsAddr{}
	localConnection.setRemoteAddress(netID)
	fmt.Printf("Remote connection at Port: %d Port: %d Address: %d %d %d %d %d %d\n",
		localConnection.port,
		localConnection.addr.Port,
		localConnection.addr.NetId.B[0],
		localConnection.addr.NetId.B[1],
		localConnection.addr.NetId.B[2],
		localConnection.addr.NetId.B[3],
		localConnection.addr.NetId.B[4],
		localConnection.addr.NetId.B[5])
	localConnection.addr.Port = port

	localConnection.initializeConnection()
	err = localConnection.initializeConnVariables()
	if err != nil {
		return
	}

	connections = append(connections, &localConnection)
	conn = &localConnection
	go localConnection.readWritePump(ctx)
	return conn, err
}

func (localConnection *Connection) initializeConnVariables() error {
	uploadInfo, err := localConnection.getSymbolUploadInfo()
	if err != nil {
		return err
	}
	fmt.Println("uploadinfo  loaded", uploadInfo.NDatatypeSize, uploadInfo.NSymSize)
	err = localConnection.uploadSymbolInfoDataTypes(uploadInfo.NDatatypeSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoDataTypes  loaded")
	err = localConnection.uploadSymbolInfoSymbols(uploadInfo.NSymSize)
	if err != nil {
		return err
	}
	fmt.Println("uploadSymbolInfoSymbols  loaded")
	localConnection.SymbolsLoaded = true
	return err
}

func (localConnection *Connection) initializeConnection() {
	localConnection.Read = make(chan string)
	localConnection.ReadResponse = make(chan string)
	localConnection.Notification = make(chan NotificationStruct, 100)
	localConnection.AddNotification = make(chan string)
	localConnection.AddPollingNotification = make(chan string)
	localConnection.UpdateResponse = make(chan string)
	localConnection.Write = make(chan WriteStruct)
	localConnection.WriteRead = make(chan WriteStruct)
	localConnection.Update = make(chan updateStruct, 100)
	localConnection.symbols = map[string]*ADSSymbol{}
	localConnection.datatypes = map[string]ADSSymbolUploadDataType{}
	localConnection.handles = map[uint32]string{}
	localConnection.notificationHandles = map[uint32]string{}
}

// CloseAllConnections closes open connections
func CloseAllConnections() {
	for _, conn := range connections {
		conn.CloseConnection()
		err := adsPortCloseEx(conn.port)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Closed")
	}
}

// CloseConnection closes current connection
func (localConnection Connection) CloseConnection() {
	for k := range localConnection.notificationHandles {
		err := localConnection.releasNotificationeHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted notification handle %d\n", k)
		}
	}
	for k := range localConnection.handles {
		err := localConnection.releaseHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted handle %d\n", k)
		}
	}
	return
}

func showComments(info *ADSSymbolUploadDataType) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showComments(value)
	}
}

func showInfoComments(info *ADSSymbol) {
	fmt.Println(info.Name)
	for _, v := range info.Childs {
		showInfoComments(v)
	}
}

func (node *ADSSymbol) addNotificationChannel(mode AdsTransMode, cycleTime time.Duration, maxTime time.Duration) {
	node.adsSyncAddDeviceNotificationReqEx(mode, uint32(maxTime), uint32(cycleTime))
}

func (localConnection *Connection) readWritePump(ctx context.Context) {
ForLoop:
	for {
		select {
		case s := <-localConnection.Read:
			symbol, ok := localConnection.symbols[s]
			if ok {
				err := symbol.updateVariable()
				if err != nil {
					fmt.Printf("error here %v/n", err)
					localConnection.ReadResponse <- "{}"
				} else {
					value := symbol.getJSON(false)
					localConnection.ReadResponse <- value
				}
			} else {
				localConnection.ReadResponse <- "{}"
				fmt.Println(s)
			}
		case s := <-localConnection.Write:
			value, ok := localConnection.symbols[s.Variable]
			if ok {
				value.Write(s.Value)
			} else {
				fmt.Printf("bad variable call: %s", s.Variable)
			}
		case s := <-localConnection.WriteRead:
			variable, ok := localConnection.symbols[s.Variable]
			value := "{}"
			if ok {
				variable.Write(s.Value)
				variable.updateVariable()
				value = variable.getJSON(false)
			}
			localConnection.ReadResponse <- value
		case s := <-localConnection.Update:
			variable, ok := localConnection.symbols[s.variable]
			if ok {
				variable.parse(s.value, 0)
				localConnection.Notification <- NotificationStruct{Variable: s.variable, Value: variable.getJSON(true), TimeStamp: s.timestamp}
				// localConnection.UpdateResponse <- value.getJSON(true)
				variable.clearChanged()
			}
			// } else {
			// 	localConnection.UpdateResponse <- ""
			// }
		case <-time.After(50 * time.Millisecond):
			for _, pollVariable := range localConnection.pollVariables {
				symbol, ok := localConnection.symbols[pollVariable]
				if ok {
					err := symbol.updateVariable()
					if err != nil {
						fmt.Printf("error here %v/n", err)
					} else {
						value := symbol.getJSON(true)
						localConnection.Notification <- NotificationStruct{Variable: pollVariable, Value: value, TimeStamp: time.Now()}
					}

				}
			}
		case s := <-localConnection.AddNotification:
			value, ok := localConnection.symbols[s]
			if ok {
				value.addNotificationChannel(ADSTRANS_SERVERONCHA, time.Millisecond*50, time.Millisecond*50)
			}
		case s := <-localConnection.AddPollingNotification:
			localConnection.pollVariables = append(localConnection.pollVariables, s)
		case <-ctx.Done():
			break ForLoop
		}
	}
}

// GetStringValue returns value from PLC in string format
func (node *ADSSymbol) updateVariable() error {
	if time.Since(node.LastUpdateTime) > node.MinUpdateInterval {
		if node.Handle == 0 {
			err := node.getHandle()
			if err != nil {
				return err
			}
		}
		data, err := node.Connection.getValueByHandle(
			node.Handle,
			node.Length)
		if err != nil {
			node.Handle = 0
			return err
		}
		node.parse(data, 0)
	}
	return nil
}

func (node *ADSSymbol) Write(value string) {
	if node.Handle == 0 {
		node.getHandle()
	}
	node.writeToNode(value, 0)
}

// GetJSON (onlyChanged bool) string
func (node *ADSSymbol) getJSON(onlyChanged bool) string {
	data := node.parseNode(onlyChanged)
	if jsonData, err := json.Marshal(data); err == nil {
		return string(jsonData)
	}
	return ""
}

var openBracketRegex = regexp.MustCompile(`\[`)
var closeBracketRegex = regexp.MustCompile(`\]`)

// ParseNode returns JSON interface for symbol
func (node *ADSSymbol) parseNode(onlyChanged bool) (rData interface{}) {
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

func (node *ADSSymbol) clearChanged() {
	for _, localNode := range node.Childs {
		localNode.clearChanged()
	}
	node.Changed = false
}
