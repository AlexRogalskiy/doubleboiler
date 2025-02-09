package routes

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"doubleboiler/logger"
	"fmt"
	"net/http"
)

func txMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method == "GET" {
			ctx = context.WithValue(ctx, "tx", config.Db)
			h.ServeHTTP(w, r.WithContext(ctx))
		} else {
			var tx *sql.Tx
			var err error

			tx, err = config.Db.BeginTx(ctx, nil)
			if err != nil {
				errRes(w, r, 500, "Cannot begin transaction", err)
				return
			}

			if c, err := r.Cookie("doubleboiler-user"); err == nil {
				cookieValue := make(map[string]string)
				if err := secureCookie.Decode("doubleboiler-user", c.Value, &cookieValue); err == nil {
					tx.ExecContext(ctx, "SET application_name = '"+cookieValue["Id"]+"'")
				}
			}

			ctx = context.WithValue(ctx, "tx", tx)

			h.ServeHTTP(w, r.WithContext(ctx))

			if err := tx.Commit(); err != nil {
				if err != sql.ErrTxDone {
					logger.Log(r.Context(), logger.Error, fmt.Sprintf("committing transaction: %+v", err))
				}
			}
		}
	})
}
