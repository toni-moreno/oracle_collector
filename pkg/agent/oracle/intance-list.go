package oracle

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/toni-moreno/oracle_collector/pkg/utils"
)

type InstanceList struct {
	OraInstances []*OracleInstance
	mutex        sync.Mutex
}

// NewInstanceList Creates a Oracle List with empty instances
func NewInstanceList() *InstanceList {
	return &InstanceList{
		OraInstances: []*OracleInstance{},
	}
}

func GetSidNames(ia []*OracleInstance) []string {
	ret := []string{}
	for _, i := range ia {
		ret = append(ret, i.DiscoveredSid)
	}
	return ret
}

func (il *InstanceList) GetList() []*OracleInstance {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	return il.OraInstances
}

func (il *InstanceList) GetFilteredListBySid(sreg string) []*OracleInstance {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	new_array := []*OracleInstance{}
	for _, i := range il.OraInstances {
		match, _ := regexp.MatchString(sreg, i.InstInfo.InstName)
		if match {
			new_array = append(new_array, i)
		}
	}
	return new_array
}

func (il *InstanceList) SetList(list []*OracleInstance) {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	il.OraInstances = list
}

func (il *InstanceList) GetNewAndOldInstances(updated []*OracleInstance) ([]*OracleInstance, []*OracleInstance, []*OracleInstance) {
	cur_sids := GetSidNames(il.OraInstances)
	new_sids := GetSidNames(updated)

	new := utils.SliceDiff(new_sids, cur_sids)
	old := utils.SliceDiff(cur_sids, new_sids)
	same := utils.SliceIntersect(cur_sids, new_sids)
	// log.Debugf("[DISCOVERY] NEW: %+v", new)
	// log.Debugf("[DISCOVERY] OLD: %+v", old)
	new_oi := []*OracleInstance{}
	old_oi := []*OracleInstance{}
	same_oi := []*OracleInstance{}

	// creating new OracleInstance array
	for _, sid := range new {
		for _, upd_inst := range updated {
			if upd_inst.DiscoveredSid == sid {
				new_oi = append(new_oi, upd_inst)
				break
			}
		}
	}
	// creating old OracleInstance Array
	for _, sid := range old {
		for _, upd_inst := range il.OraInstances {
			if upd_inst.DiscoveredSid == sid {
				old_oi = append(old_oi, upd_inst)
				break
			}
		}
	}

	// creating smae OracleInstance Array
	for _, sid := range same {
		for _, upd_inst := range il.OraInstances {
			if upd_inst.DiscoveredSid == sid {
				same_oi = append(same_oi, upd_inst)
				break
			}
		}
	}

	return new_oi, old_oi, same_oi
}

func (il *InstanceList) Add(i *OracleInstance) {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	il.OraInstances = append(il.OraInstances, i)
}

func remove(slice []*OracleInstance, i int) []*OracleInstance {
	return append(slice[:i], slice[i+1:]...)
}

func (il *InstanceList) Delete(inst *OracleInstance) error {
	il.mutex.Lock()
	defer il.mutex.Unlock()
	// search for index.
	idx := -1
	for i, e := range il.OraInstances {
		if e.DiscoveredSid == inst.DiscoveredSid {
			idx = i
			break
		}
	}
	if idx == -1 {
		// not found
		return fmt.Errorf("Error Deleting Instance [%] ,from Orcle List : NOT FOUND", inst.DiscoveredSid)
	}

	il.OraInstances = remove(il.OraInstances, idx)
	return nil
}

var OraList *InstanceList
