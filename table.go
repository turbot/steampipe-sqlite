package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"go.riyazali.net/sqlite"
	"golang.org/x/exp/maps"
)

/*
QueryContext contains important query properties:
  - The columns requested.
  - The constraints specified.
  - The query qualifiers (where clauses).
  - The limit (number of rows to return).
*/
type QueryContext struct {
	Columns []string    `json:"columns"`
	Quals   []*Qual     `json:"quals"`
	Limit   *QueryLimit `json:"limit"`
}

type QueryLimit struct {
	Rows int64 // the number of rows to return
	Idx  int   `json:"idx"` // the index in the values that Cursor.Filter receives
}

type Qual struct {
	FieldName        string                  `json:"field_name"`
	Operator         string                  `json:"operator"`
	ColumnDefinition *proto.ColumnDefinition `json:"-"`
}
type QualOperator struct {
	Op   string  `json:"op"`
	Cost float64 `json:"cost"`
}

type PluginTable struct {
	name        string
	tableSchema *proto.TableSchema
}

func (p *PluginTable) getLimit(info *sqlite.IndexInfoInput) (limit *QueryLimit) {
	// fmt.Println("table.getLimit")
	// defer fmt.Println("end table.getLimit")

	for idx, ic := range info.Constraints {
		if ic.Op == sqlite.ConstraintOp(SQLITE_INDEX_CONSTRAINT_LIMIT) {
			// sqlite passes LIMIT as a constraint (sort of makes sense)
			// use it
			limit = &QueryLimit{
				Idx: idx,
			}
			break
		}
	}
	return limit
}

// if there are unusable constraints on any of start, stop, or step then
// this plan is unusable and the xBestIndex method should return a SQLITE_CONSTRAINT error.
func (p *PluginTable) BestIndex(info *sqlite.IndexInfoInput) (*sqlite.IndexInfoOutput, error) {
	// fmt.Println("table.BestIndex")
	// defer fmt.Println("end table.BestIndex")

	qc := &QueryContext{
		Columns: p.getColumnsFromIndexInfo(info),
	}

	output := &sqlite.IndexInfoOutput{
		EstimatedCost:   math.MaxFloat64,
		IndexNumber:     0,
		IndexString:     "",
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(info.Constraints)),
	}

	for idx, ic := range info.Constraints {
		fmt.Println(">>>: ", ic.ColumnIndex, ic.Op, ic.Usable)

		output.ConstraintUsage[idx] = &sqlite.ConstraintUsage{
			Omit: true,
		}

		if ic.Op == sqlite.ConstraintOp(SQLITE_INDEX_CONSTRAINT_LIMIT) {
			// sqlite passes LIMIT as a constraint (sort of makes sense)
			// use it
			limit := &QueryLimit{
				Idx: idx,
			}
			output.ConstraintUsage[idx] = &sqlite.ConstraintUsage{
				Omit:      false,
				ArgvIndex: idx + 1, // according to https://www.sqlite.org/vtab.html, this should be 1-indexed
			}
			qc.Limit = limit
			continue
		}

		cost := p.getConstraintCost(ic)
		if cost < output.EstimatedCost {
			output.EstimatedCost = cost
		}
		output.ConstraintUsage[idx] = &sqlite.ConstraintUsage{
			Omit:      false,
			ArgvIndex: idx + 1, // according to https://www.sqlite.org/vtab.html, this should be 1-indexed
		}
		qualOperator := getPluginOperator(ic.Op)
		qc.Quals = append(qc.Quals, &Qual{
			FieldName:        p.tableSchema.Columns[ic.ColumnIndex].GetName(),
			Operator:         qualOperator.Op,
			ColumnDefinition: p.tableSchema.Columns[ic.ColumnIndex],
		})

	}

	qcBytes, err := json.Marshal(qc)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	output.IndexString = string(qcBytes)

	return output, nil
}

func (p *PluginTable) getConstraintCost(ic *sqlite.IndexConstraint) (cost float64) {
	// fmt.Println("table.constraintCost")
	// defer fmt.Println("end table.constraintCost")

	if !ic.Usable {
		return math.MaxFloat64
	}
	schemaColumn := p.tableSchema.Columns[ic.ColumnIndex]
	sqliteOp := ic.Op

	// is this a usable key column?
	for _, keyColumn := range p.tableSchema.GetAllKeyColumns() {
		if keyColumn.GetName() != schemaColumn.GetName() {
			// not me
			continue
		}

		// does this key column support this operator?
		for _, operator := range keyColumn.Operators {
			if qualOp := getPluginOperator(sqliteOp); qualOp.Op == operator {
				return cost
			}
		}
	}

	// this can be used, but with a high cost
	return math.MaxFloat64
}

func (p *PluginTable) Open() (sqlite.VirtualCursor, error) {
	// fmt.Println("table.Open")
	// defer fmt.Println("end table.Open")

	cursor := NewPluginCursor(context.Background(), p)
	return cursor, nil
}

func (p *PluginTable) Disconnect() error {
	// fmt.Println("table.Disconnect")
	// defer fmt.Println("end table.Disconnect")
	return nil
}

func (p *PluginTable) Destroy() error {
	// fmt.Println("table.Destroy")
	// defer fmt.Println("end table.Destroy")
	return nil
}

func (p *PluginTable) getColumnsFromIndexInfo(info *sqlite.IndexInfoInput) []string {
	// get the columns from the index info
	if info.ColUsed == nil {
		// no cols used, so return all columns - not sure if this can ever happen
		return maps.Keys(p.tableSchema.GetColumnMap())
	}
	// get the columns from the index info
	// the ColUsed field is a bitmask of the columns used in the query
	// if the 0th bit is set, then the 0th column is used
	// if the 1st bit is set, then the 1st column is used and so on
	// so we need to iterate over the columns by index and check that
	// the bit for that index is set
	// if the 64th bit is set, then any column over 63 is used (need to handle this)
	columns := []string{}
	for i, col := range p.tableSchema.GetColumns() {
		// check if the  bit is set in info.ColUsed
		if checkKthBitSet(*info.ColUsed, i) {
			columns = append(columns, col.GetName())
		}
	}
	return columns
}

// checkKthBitSet checks if the kth (0-indexed) bit is set in n
func checkKthBitSet(n int64, k int) bool {
	return n&(1<<(k)) == 0
}
