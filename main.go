package ads

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

var portOpen bool

type Connection struct {
	addr                *AmsAddr
	port                int
	symbolsLoaded       bool
	Symbols             map[string]ADSSymbol
	datatypes           map[string]ADSSymbolUploadDataType
	handles             map[uint32]*ADSSymbol
	notificationHandles map[uint32]*ADSSymbol
}

type ADSSymbol struct {
	Connection         *Connection
	Self               *ADSSymbol
	FullName           string
	LastUpdateTime     int64
	MinUpdateInterval  int64
	Name               string
	DataType           string
	Comment            string
	Handle             uint32
	NotificationHandle uint32
	ChangedHandlers    []func(ADSSymbol)

	Group  uint32
	Offset uint32
	Length uint32

	Value   string
	Valid   bool
	Changed bool

	Parent *ADSSymbol
	Childs map[string]*ADSSymbol
}

var lock *sync.Mutex

func init() {
	lock = &sync.Mutex{}
}

func AddLocalConnection() (conn *Connection, err error) {

	localConnection := Connection{}
	open, err := adsAmsPortEnabled()
	if err != nil {
		return nil, err
	}

	if !open {
		adsPortOpen()
	}

	localConnection.addr = &AmsAddr{}
	localConnection.adsGetLocalAddress()
	fmt.Printf("local connection at %d %d %d \n", localConnection.port, localConnection.addr.Port, localConnection.addr.NetId.B[0])
	localConnection.addr.Port = 851

	localConnection.initializeConnection()
	err = localConnection.initializeConnVariables()
	if err != nil {
		return
	}

	connections = append(connections, &localConnection)
	conn = &localConnection
	return
}

func AddRemoteConnection(netID string) (conn *Connection, err error) {

	localConnection := Connection{}
	open, err := adsAmsPortEnabled()
	if !open {
		adsPortOpen()
	}
	if err != nil {
		return nil, err
	}
	localConnection.addr = &AmsAddr{}
	localConnection.setRemoteAddress(netID)
	fmt.Printf("remote connection at %d %d %d \n", localConnection.port, localConnection.addr.Port, localConnection.addr.NetId.B[0])
	localConnection.addr.Port = 851

	localConnection.initializeConnection()
	err = localConnection.initializeConnVariables()
	if err != nil {
		return
	}

	connections = append(connections, &localConnection)
	conn = &localConnection
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
	return err
}

func (localConnection *Connection) initializeConnection() {
	localConnection.Symbols = map[string]ADSSymbol{}
	localConnection.datatypes = map[string]ADSSymbolUploadDataType{}
	localConnection.handles = map[uint32]*ADSSymbol{}
	localConnection.notificationHandles = map[uint32]*ADSSymbol{}
}

// CloseAllConnections closes open connections
func CloseAllConnections() {
	for _, conn := range connections {
		conn.CloseConnection()
	}
	err := adsPortClose()
	if err != nil {
		log.Println(err)
	}
}

// CloseConnection closes current connection
func (localConnection *Connection) CloseConnection() {
	for k := range localConnection.handles {
		err := localConnection.releaseHandle(k)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted handle %d", k)
		}
	}
	for k := range localConnection.notificationHandles {
		err := localConnection.releasNotificationeHandle(k)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Printf("deleted notification handle %d", k)
		}
	}
}

func showComments(info ADSSymbolUploadDataType) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showComments(value)
	}
}

func showInfoComments(info ADSSymbol) {
	fmt.Println(info.Name)
	for _, value := range info.Childs {
		showInfoComments(*value)
	}

}

// AddNotification adds event notification to handle
func (node *ADSSymbol) AddNotification(mode uint32, cycleTime uint32, maxTime uint32, callback func(ADSSymbol)) {
	node.adsSyncAddDeviceNotificationReq(mode, maxTime, cycleTime)
	node.addCallback(callback)
}

// GetStringValue returns value from PLC in string format
func (node *ADSSymbol) GetStringValue() (value string, err error) {
	if node.Handle == 0 {
		err = node.getHandle()
	}
	if err != nil {
		return "", err
	}
	data, err := node.Connection.getValueByHandle(
		node.Handle,
		node.Length)
	if err != nil {
		return "", err
	}

	node.parse(data, 0)
	return node.Value, err
}

func (node *ADSSymbol) Write(value string) {
	if node.Handle == 0 {
		node.getHandle()
	}
	node.writeToNode(value, 0)
}

// GetJSON (onlyChanged bool) string
func (node *ADSSymbol) GetJSON(onlyChanged bool) (string, error) {
	if !onlyChanged {
		_, err := node.GetStringValue()
		if err != nil {
			return "", err
		}
	}
	data := make(map[string]interface{})
	data[node.FullName] = node.parseNode(onlyChanged)
	if jsonData, err := json.Marshal(data); err == nil {
		return string(jsonData), nil
	}
	return "", nil
}

// ParseNode returns JSON interface for symbol
func (node *ADSSymbol) parseNode(onlyChanged bool) (rData interface{}) {
	if len(node.Childs) == 0 {
		rData = node.Value
		// node.Changed = false
	} else {
		// if strings.HasPrefix(node.DataType, "ARRAY") {
		// 	re := regexp.MustCompile(`\[.*\.\.(\d+)\]`)
		// 	arraySize, _ := strconv.Atoi(re.FindAllStringSubmatch(node.DataType, 1)[0][1])
		// 	arraySize++
		// 	localArray := make([]interface{}, arraySize)
		// 	for _, child := range node.Childs {
		// 		re := regexp.MustCompile(`\[(\d+)\]`)
		// 		arrayIndex, _ := strconv.Atoi(re.FindAllStringSubmatch(child.Name, 1)[0][1])
		// 		localArray[arrayIndex] = child.ParseNode()
		// 	}
		// 	rData = localArray
		// } else {
		localMap := make(map[string]interface{})
		for _, child := range node.Childs {
			if onlyChanged {
				if child.Changed {
					localMap[child.Name] = child.parseNode(true)
					// child.Changed = false
				}
			} else {
				localMap[child.Name] = child.parseNode(false)
			}
		}
		rData = localMap
		return
	}
	// if node.Parent == nil {
	// 	tempData := make(map[string]interface{})
	// 	tempData[node.Name] = rData
	// 	rData = tempData
	// }
	return
}

// 	return
// }
