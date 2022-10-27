package oracle

import "sync"

type InstanceList struct {
	OraInstances []*OracleInstance
	mutex        sync.Mutex
}

func NewIntanceList() *InstanceList {
	return &InstanceList{}
}

func (il *InstanceList) GetList() []*OracleInstance {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	return il.OraInstances
}

func (il *InstanceList) SetList(list []*OracleInstance) {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	il.OraInstances = list
}

var OraList *InstanceList
