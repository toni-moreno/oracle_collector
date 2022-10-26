package agent

import (
	"fmt"
	"strconv"
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

func (dt *DataTable) Length() int {
	return len(dt.Row)
}

func (dt *DataTable) GetMetrics() ([]*telegraf.Metric, error) {
	result := []*telegraf.Metric{}

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

	now := time.Now()
	for _, row := range dt.Row {
		tags := make(map[string]string)
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
				value, cerr = strconv.ParseInt(strings.TrimSpace(raw_value.(string)), 10, 64)
			case "float", "FLOAT":
				value, cerr = strconv.ParseFloat(strings.TrimSpace(raw_value.(string)), 64)
			case "bool", "BOOL", "BOOLEAN":
				value, cerr = strconv.ParseBool(strings.TrimSpace(raw_value.(string)))
			}
			if cerr != nil {
				return result, cerr
			}

			fields[fieldname] = value
		}

		m := metric.New(dt.mcfg.Context, tags, fields, now)
		result = append(result, &m)
	}
	return result, nil
}
