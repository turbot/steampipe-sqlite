package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"math"
	"strconv"
	"sync/atomic"

	"github.com/turbot/go-kit/helpers"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/sperr"
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
	Rows    int64 `json:"-"`   // the number of rows to return - populated during xFilter
	ArgvIdx int   `json:"idx"` // the index in the values that Cursor.Filter receives
}

type Qual struct {
	ArgvIndex        int                     `json:"argv_index"`
	FieldName        string                  `json:"field_name"`
	Operator         string                  `json:"operator"`
	ColumnDefinition *proto.ColumnDefinition `json:"column_definition"`
}
type QualOperator struct {
	Op   string  `json:"op"`
	Cost float64 `json:"cost"`
}

type PluginTable struct {
	name        string
	tableSchema *proto.TableSchema
	planNumber  int64
}

func (p *PluginTable) getLimit(info *sqlite.IndexInfoInput) (limit *QueryLimit) {
	log.Println("[DEBUG] table.getLimit")
	defer log.Println("[DEBUG] end table.getLimit")

	for idx, ic := range info.Constraints {
		if ic.Op == sqlite.ConstraintOp(SQLITE_INDEX_CONSTRAINT_LIMIT) {
			// sqlite passes LIMIT as a constraint (sort of makes sense)
			// use it
			limit = &QueryLimit{
				ArgvIdx: idx,
			}
			break
		}
	}
	return limit
}

// if there are unusable constraints on any of start, stop, or step then
// this plan is unusable and the xBestIndex method should return a SQLITE_CONSTRAINT error.
func (p *PluginTable) BestIndex(info *sqlite.IndexInfoInput) (output *sqlite.IndexInfoOutput, err error) {
	log.Println("[DEBUG] table.BestIndex start", p.name)
	defer log.Println("[DEBUG] table.BestIndex end", p.name)

	qc := &QueryContext{
		Columns: p.getColumnsFromIndexInfo(info),
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println("[ERROR] table.BestIndex recover: ", r)
			err = sperr.ToError(r)
		}
		log.Println("[TRACE] table.BestIndex idxnum: ", output.IndexNumber)
		log.Println("[TRACE] table.BestIndex idxStr: ", output.IndexString)
		log.Println("[TRACE] table.BestIndex output EstimatedCost: ", output.EstimatedCost)
		for _, cu := range output.ConstraintUsage {
			log.Println("[TRACE] table.BestIndex output.ConstraintUsage: ", cu.ArgvIndex, cu.Omit)
		}
	}()

	newPlanNumber := atomic.AddInt64(&p.planNumber, 1)

	output = &sqlite.IndexInfoOutput{
		EstimatedCost:   math.MaxFloat64,
		IndexNumber:     int(newPlanNumber),                   // although this should not be required, lets put a unique value to this field
		IndexString:     strconv.FormatInt(newPlanNumber, 10), // just set this to a unique number
		ConstraintUsage: make([]*sqlite.ConstraintUsage, len(info.Constraints)),
	}

	var currentArgvIndex = atomic.Int64{}

	for idx, ic := range info.Constraints {
		log.Println("[TRACE] table.BestIndex idx >>>: ", idx)
		log.Println("[TRACE] table.BestIndex constraint >>>: ", ic.ColumnIndex, ic.Op, ic.Usable)

		// if this constraint is not usable, then skip it - it will be omitted
		// Note: ROWID (-1 in ColumnIndex) cannot be used, since plugin tables do not have a similar concept
		if !ic.Usable || ic.ColumnIndex == -1 {
			log.Println("[TRACE] table.BestIndex constraint not usable or ROWID")
			output.ConstraintUsage[idx] = &sqlite.ConstraintUsage{
				// return an argvIndex of -1 so that this does not get passed in to xFilter
				ArgvIndex: -1,
				Omit:      true,
			}
			continue
		}

		log.Println("[TRACE] table.BestIndex column >>>: ", p.tableSchema.Columns[ic.ColumnIndex])

		// default to using this constraint
		nextArgvIndex := int(currentArgvIndex.Add(1))
		output.ConstraintUsage[idx] = &sqlite.ConstraintUsage{
			ArgvIndex: nextArgvIndex,
			Omit:      false,
		}

		// if this is a limit constraint, then we need to handle it differently
		// since it is not a constraint on a column
		if ic.Op == sqlite.ConstraintOp(SQLITE_INDEX_CONSTRAINT_LIMIT) {
			qc.Limit = &QueryLimit{
				ArgvIdx: nextArgvIndex,
			}
			continue
		}

		cost := p.getConstraintCost(ic)
		if cost < output.EstimatedCost {
			output.EstimatedCost = cost
		}
		qualOperator := getPluginOperator(ic.Op)
		qc.Quals = append(qc.Quals, &Qual{
			ArgvIndex:        nextArgvIndex,
			FieldName:        p.tableSchema.Columns[ic.ColumnIndex].GetName(),
			Operator:         qualOperator.Op,
			ColumnDefinition: p.tableSchema.Columns[ic.ColumnIndex],
		})
	}

	// we cannot do this and short circuit at the top of the function
	// since we need to set the output.ConstraintUsage for all constraints
	// even if they are not usable
	if !p.allRequiredKeyColsInConstraints(info) {
		// all key columns are not provided
		// this is going to be a very high cost plan
		// we do this instead of a SQLITE_CONSTRAINT error
		// so that the query reaches the plugin which can provide
		// a much richer error message around required
		// key columns not being provided
		output.EstimatedCost = math.MaxFloat64
	}

	// serialize the QueryContext to JSON
	// we need to use the json encoder here since the Operator field
	// may contain '>' or '<' which will be escaped by the default encoder
	buffer := bytes.NewBuffer([]byte{})
	jsonEncoder := json.NewEncoder(buffer)
	jsonEncoder.SetEscapeHTML(false)
	err = jsonEncoder.Encode(qc)
	if err != nil {
		log.Println("[WARN] table.BestIndex jsonEncoder.Encode failed: ", err)
		return nil, err
	}
	output.IndexString = string(buffer.Bytes())

	return output, nil
}

func (p *PluginTable) allRequiredKeyColsInConstraints(info *sqlite.IndexInfoInput) bool {
	log.Println("[DEBUG] table.verifyAllKeyColumnsInConstraints start")
	defer log.Println("[DEBUG] table.verifyAllKeyColumnsInConstraints end")

	// make a slice of all constraints (by name)
	constraintColumns := make([]string, 0, len(info.Constraints))
	for _, ic := range info.Constraints {
		if ic.ColumnIndex == -1 {
			// ROWID (-1 in ColumnIndex) cannot be used, since plugin tables do not have a parallel
			continue
		}
		column := p.tableSchema.Columns[ic.ColumnIndex]
		constraintColumns = append(constraintColumns, column.GetName())
	}

	// get a slice of all key columns
	keyColumns := make([]string, 0, len(p.tableSchema.GetAllKeyColumns()))
	for _, keyColumn := range p.tableSchema.GetAllKeyColumns() {
		if keyColumn.GetRequire() == plugin.Required {
			// not concerned about optional columns
			keyColumns = append(keyColumns, keyColumn.GetName())
		}
	}

	// check if all key columns are in the constraints
	for _, keyColumn := range keyColumns {
		if !helpers.StringSliceContains(constraintColumns, keyColumn) {
			return false
		}
	}

	return true
}

func (p *PluginTable) getConstraintCost(ic *sqlite.IndexConstraint) (cost float64) {
	log.Println("[DEBUG] table.getConstraintCost start")
	defer log.Println("[DEBUG] table.getConstraintCost end")

	schemaColumn := p.tableSchema.Columns[ic.ColumnIndex]
	sqliteOp := ic.Op

	log.Println("[DEBUG] >>> column: ", schemaColumn.GetName())
	log.Println("[DEBUG] >>> sqliteOp: ", sqliteOp)

	// is this a usable key column?
	for _, keyColumn := range p.tableSchema.GetAllKeyColumns() {
		if keyColumn.GetName() != schemaColumn.GetName() {
			// not me
			continue
		}

		// does this key column support this operator?
		for _, operator := range keyColumn.Operators {
			log.Println("[DEBUG] >>> operator: ", operator, sqliteOp)
			if qualOp := getPluginOperator(sqliteOp); qualOp.Op == operator {
				return qualOp.Cost
			}
		}
	}

	// this can be used, but with a high cost
	return math.MaxFloat64
}

func (p *PluginTable) Open() (sqlite.VirtualCursor, error) {
	log.Println("[DEBUG] table.Open")
	defer log.Println("[DEBUG] end table.Open")

	cursor := NewPluginCursor(context.Background(), p)
	return cursor, nil
}

func (p *PluginTable) Disconnect() error {
	log.Println("[DEBUG] table.Disconnect")
	defer log.Println("[DEBUG] end table.Disconnect")
	return nil
}

func (p *PluginTable) Destroy() error {
	log.Println("[DEBUG] table.Destroy")
	defer log.Println("[DEBUG] end table.Destroy")
	return nil
}

func (p *PluginTable) getColumnsFromIndexInfo(info *sqlite.IndexInfoInput) (columns []string) {
	log.Println("[DEBUG] table.getColumnsFromIndexInfo")
	defer log.Println("[DEBUG] end table.getColumnsFromIndexInfo")

	log.Println("[TRACE] table.getColumnsFromIndexInfo info.ColUsed: ", *info.ColUsed)

	defer func() {
		log.Println("[TRACE] table.getColumnsFromIndexInfo columns: ", columns)
	}()

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
	for i, col := range p.tableSchema.GetColumns() {
		// check if the bit is set in info.ColUsed
		// if it is, then this column is used
		// if it is not, then this column is not used
		// if the bit is set for the 63rd column, then any column over 63 is used
		// we just include all of them and rely on the SQLite core to do the rest of the selection
		isColUsed := checkKthBitSet(*info.ColUsed, i) || (i > 63 && checkKthBitSet(*info.ColUsed, 63))
		log.Println("[TRACE] table.getColumnsFromIndexInfo col used: ", col.GetName(), i, isColUsed)
		if isColUsed {
			columns = append(columns, col.GetName())
		}
	}
	return columns
}

// checkKthBitSet checks if the kth (0-indexed) bit is set in n
// if k is more than 63, then it returns true
func checkKthBitSet(n int64, bitIdxK int) bool {
	log.Println("[TRACE] table.checkKthBitSet", n, bitIdxK)
	defer log.Println("[TRACE] end table.checkKthBitSet", n, bitIdxK)
	return n&(1<<bitIdxK) != 0
}
