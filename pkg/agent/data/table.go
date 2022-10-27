package data

import (
	"fmt"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/toni-moreno/oracle_collector/pkg/config"
)

type Index struct {
	Num  int
	Type string
}

type Row []interface{}

type DataTable struct {
	columns int
	Header  []string
	Row     []Row
	last    int
	mcfg    *config.OracleMetricConfig
}

func NewDatatableWithConfig(cfg *config.OracleMetricConfig) *DataTable {
	dt := DataTable{
		mcfg: cfg,
	}
	return &dt
}

func NewDatatable(header []string) *DataTable {
	dt := DataTable{
		columns: len(header),
		Header:  header,
	}
	return &dt
}

func (dt *DataTable) SetHeader(header []string) {
	dt.Header = nil
	for _, h := range header {
		dt.Header = append(dt.Header, strings.ToLower(h))
	}
	dt.columns = len(header)
}

func (dt *DataTable) AppendEmptyRow() []interface{} {
	// Set Header Should be called first
	row := make([]interface{}, dt.columns)
	dt.Row = append(dt.Row, row)
	dt.last++
	rowPointers := make([]interface{}, dt.columns)
	for i := range row {
		rowPointers[i] = &row[i]
	}
	return rowPointers
}

func (dt *DataTable) AppendRow(row []interface{}) {
	// Set Header Should be called first
	dt.Row = append(dt.Row, row)
	dt.last++
}

func (dt *DataTable) Length() int {
	return len(dt.Row)
}

func (dt *DataTable) Transpose(columnName string) (*DataTable, error) {
	tfield := dt.mcfg.FieldToAppend
	if len(tfield) == 0 {
		return nil, fmt.Errorf("Can not transpose without 'fieldtoappend' info")
	}
	// Get New Headers
	// Get Index for header column

	idx := -1
	for i, v := range dt.Header {
		if v == tfield {
			idx = i
			break
		}
	}
	// check if Idx not found
	if idx < 0 {
		err := fmt.Errorf("Error on Transpose, Transpose field  [%s] not found on table with headers [%+v]", tfield, dt.Header)
		return nil, err
	}

	var newheader []string
	newmetrictype := make(map[string]string)
	for _, r := range dt.Row {
		// fields should be equal in config and in headers , and alwasy lowercase
		h := strings.ReplaceAll(strings.ToLower(convert2String(r[idx])), " ", "_")
		newheader = append(newheader, h)
		newmetrictype[h] = dt.mcfg.MetricsType["value"]
	}

	// Get Index for data column

	idx = -1
	for i, v := range dt.Header {
		if v == "value" {
			idx = i
			break
		}
	}
	// check if Idx not found
	if idx < 0 {
		err := fmt.Errorf("Error on Transpose, value field  found on table with headers [%+v]", tfield, dt.Header)
		return nil, err
	}

	// new data

	var newdata []interface{}

	for _, r := range dt.Row {
		newdata = append(newdata, r[idx])
	}

	// new config
	nconf := &config.OracleMetricConfig{
		Context:     dt.mcfg.Context,
		Labels:      dt.mcfg.Labels,
		MetricsDesc: dt.mcfg.MetricsDesc, // sure?
		MetricsType: newmetrictype,
		// FieldToAppend: , not needed once transformation done
		// Request: , not needed once transformation done
		IgnoreZeroResult: dt.mcfg.IgnoreZeroResult,
	}

	newtab := NewDatatableWithConfig(nconf)
	newtab.SetHeader(newheader)
	// data for value
	newtab.AppendRow(newdata)

	return newtab, nil
}

func (dt *DataTable) getMetrics(extraLabels map[string]string) ([]telegraf.Metric, error) {
	result := []telegraf.Metric{}
	// some checks

	// Get Colunms index for tags
	tagIndexes := make(map[string]int)

	for _, val := range dt.mcfg.Labels {
		for j, h := range dt.Header {
			if val == h {
				tagIndexes[val] = j
			}
		}
	}
	if len(tagIndexes) != len(dt.mcfg.Labels) {
		err := fmt.Errorf("Error on query or config, not all labels  [%+v] found on query  headers results [%+v]", dt.mcfg.Labels, dt.Header)
		return nil, err
	}
	// Get Field Indexes
	fieldIndexes := make(map[string]*Index)

	for fk, ft := range dt.mcfg.MetricsType {
		for j, h := range dt.Header {
			if fk == h {
				fieldIndexes[fk] = &Index{
					Num:  j,
					Type: ft,
				}
			}
		}
	}
	// check if field is empty

	if len(fieldIndexes) == 0 {
		return nil, fmt.Errorf("Fields not found with type config [%+v] and Query  Headers [%+v]", dt.mcfg.MetricsType, dt.Header)
	}

	now := time.Now()
	for _, row := range dt.Row {
		tags := make(map[string]string)
		// first added extra tags
		for k, v := range extraLabels {
			tags[k] = v
		}
		// then added table tags
		for tag, index := range tagIndexes {
			tags[tag] = row[index].(string)
		}
		fields := make(map[string]interface{})
		for fieldname, index := range fieldIndexes {
			var value interface{}
			var cerr error
			raw_value := row[index.Num]
			switch index.Type {
			case "INTEGER", "COUNTER":
				fallthrough
			case "integer", "counter":
				value = convert2Int64(raw_value)
			case "float", "FLOAT":
				value = convert2Float(raw_value)
			case "bool", "BOOL", "BOOLEAN":
				value = convert2Bool(raw_value)
			case "string", "STRING":
				value = convert2String(raw_value)
			}
			if cerr != nil {
				return result, cerr
			}

			fields[fieldname] = value
		}

		m := metric.New(dt.mcfg.Context, tags, fields, now)
		result = append(result, m)
	}
	return result, nil
}

func (dt *DataTable) GetMetrics(extraLabels map[string]string) ([]telegraf.Metric, error) {
	if len(dt.mcfg.FieldToAppend) > 0 {
		new, err := dt.Transpose(dt.mcfg.FieldToAppend)
		if err != nil {
			return nil, err
		}
		return new.getMetrics(extraLabels)
	}
	return dt.getMetrics(extraLabels)
}
