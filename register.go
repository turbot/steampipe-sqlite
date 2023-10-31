package main

import (
	"fmt"

	"go.riyazali.net/sqlite"
)

type CreateVirtualTablesSqliteFunction struct{}

func register() {
	sqlite.Register(func(api *sqlite.ExtensionApi) (sqlite.ErrorCode, error) {
		createFn := NewCreateVTab(api)
		if err := api.CreateFunction(fmt.Sprintf("setup_%s", pluginAlias), createFn); err != nil {
			return sqlite.SQLITE_ERROR, err
		}

		return sqlite.SQLITE_OK, nil
	})
}
