package ads

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sync"
	"time"
)

var portOpen bool
var RouterNotification func(response int)

type AdsSyncMap struct {
	sync.Map
}

func (testMap *AdsSyncMap) Empty() bool {
	var count int
	testMap.Range(func(k, v interface{}) bool {
		count++
		return false
	})
	if count > 0 {
		return false
	}
	return true
}

type Connection struct {
	addr          *AmsAddr
	port          int
	SymbolsLoaded bool

	Symbols             map[string]*ADSSymbol
	datatypes           map[string]ADSSymbolUploadDataType
	handles             map[uint32]*ADSSymbol
	notificationHandles map[uint32]*ADSSymbol
	// notificationHandles sync.map
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
	ChangedHandlers    []func(ADSSymbol) // Fix: doesn't allow change values
	ChangedChannel     []chan ChangedRepsonse
	Group              uint32
	Offset             uint32
	Length             uint32

	Value   string
	Valid   bool
	Changed bool

	Parent *ADSSymbol
	Childs map[string]*ADSSymbol
}

var lock *sync.RWMutex
var adsLock *sync.Mutex

func init() {
	lock = &sync.RWMutex{}
	adsLock = &sync.Mutex{}
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

func AddRemoteConnection(netID string, port uint16) (conn *Connection, err error) {
	fmt.Println("local package")
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
	fmt.Printf("remote connection at %d %d %d %d %d %d\n",
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
	go conn.notificationPump()
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
	localConnection.Symbols = map[string]*ADSSymbol{}
	localConnection.datatypes = map[string]ADSSymbolUploadDataType{}
	localConnection.handles = map[uint32]*ADSSymbol{}
	localConnection.notificationHandles = map[uint32]*ADSSymbol{}
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
func (localConnection *Connection) CloseConnection() {
	for k := range localConnection.notificationHandles {
		err := localConnection.releasNotificationeHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("deleted notification handle %d", k)
		}
	}
	for k := range localConnection.handles {
		err := localConnection.releaseHandle(uint32(k))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("deleted handle %d", k)
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

// AddNotification adds event notification to handle
func (node *ADSSymbol) AddNotification(mode uint32, cycleTime time.Duration, maxTime time.Duration, callback func(ADSSymbol)) {
	node.adsSyncAddDeviceNotificationReqEx(mode, uint32(maxTime), uint32(cycleTime))
	node.addCallback(callback)
}

func (node *ADSSymbol) AddNotificationChannel(mode uint32, cycleTime time.Duration, maxTime time.Duration, callback chan ChangedRepsonse) {
	node.adsSyncAddDeviceNotificationReqEx(mode, uint32(maxTime), uint32(cycleTime))
	node.addCallbackChannel(callback)
}

var index uint32

// AddResponseChannel adds event notification to handle
func (node *ADSSymbol) AddResponseChannel(mode uint32, cycleTime time.Duration, maxTime time.Duration, callback chan ChangedRepsonse) {
	node.MinUpdateInterval = time.Millisecond * 100
	node.Connection.notificationHandles[index] = node
	index++
	node.addRepsoneChannel(callback)
}

func (localConnection *Connection) notificationPump() {
	for {
		for _, variable := range localConnection.notificationHandles {
			if time.Since(variable.LastUpdateTime) > variable.MinUpdateInterval {
				value, _ := variable.GetJSON(false)
				for _, callback := range variable.ChangedChannel {
					callback <- ChangedRepsonse{Variable: variable.FullName, Value: value}
				}
			}
		}
	}
}

// GetStringValue returns value from PLC in string format
func (node *ADSSymbol) GetStringValue() (value string, err error) {

	if node.Handle == 0 {
		err = node.getHandle()
	}
	if err != nil {
		return "", err
	}
	lock.Lock()
	data, err := node.Connection.getValueByHandle(
		node.Handle,
		node.Length)
	if err != nil {
		node.Handle = 0
		return "", err
	}
	node.parse(data, 0)
	lock.Unlock()
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
	// data := make(map[string]interface{})
	// data[node.FullName] = node.parseNode(onlyChanged)
	lock.RLock()
	defer lock.RUnlock()
	data := node.parseNode(onlyChanged)
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
					re := regexp.MustCompile(`\[`)
					s := re.ReplaceAllString(child.Name, `"[`)
					re = regexp.MustCompile(`\]`)
					s = re.ReplaceAllString(s, `]"`)
					localMap[s] = child.parseNode(true)
					// child.Changed = false
				}
			} else {
				re := regexp.MustCompile(`\[`)
				s := re.ReplaceAllString(child.Name, `"[`)
				re = regexp.MustCompile(`\]`)
				s = re.ReplaceAllString(s, `]"`)
				localMap[s] = child.parseNode(false)
			}
		}
		rData = localMap
		return
	}
	return
}
