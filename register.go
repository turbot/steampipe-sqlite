package main

import (
	"fmt"

	"go.riyazali.net/sqlite"
)

type CreateVirtualTablesSqliteFunction struct{}

func register() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		if err := setConnectionConfig(""); err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		schema, err := getSchema()
		if err != nil {
			return sqlite.SQLITE_ERROR, err
		}
		if err := setupSchemaTables(schema, api); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		configureFn := NewConfigureFn(api)
		if err := api.CreateFunction(fmt.Sprintf("configure_%s", pluginAlias), configureFn); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		return sqlite.SQLITE_OK, nil
	})
}
