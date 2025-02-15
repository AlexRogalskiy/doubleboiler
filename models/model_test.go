package models

import (
	"context"
	"database/sql"
	"doubleboiler/config"
	"fmt"
	"testing"

	bandname "github.com/davidbanham/bandname_go"
	"github.com/stretchr/testify/assert"
)

func randString() string {
	return bandname.Bandname()
}

func getCtx(t *testing.T) context.Context {
	ctx := context.Background()
	tx, err := config.Db.BeginTx(ctx, nil)
	assert.Nil(t, err)
	return context.WithValue(ctx, "tx", tx)
}

func closeTx(t *testing.T, ctx context.Context) {
	db := ctx.Value("tx")
	switch v := db.(type) {
	case *sql.Tx:
		t.Log("Rolling back")
		v.Rollback()
	case Querier:
		t.Log("No tx")
		return
	default:
		t.Log(fmt.Errorf("Unknown db type"))
		t.FailNow()
	}
}

var modelsUnderTest [][]model

type modelCollectionFixture struct {
	deps       []model
	collection models
}

var modelCollectionsUnderTest []modelCollectionFixture

func TestSave(t *testing.T) {
	t.Parallel()

	for _, c := range modelsUnderTest {
		ctx := getCtx(t)
		for _, m := range c {
			t.Run(m.tablename(), func(t *testing.T) {
				assert.Nil(t, m.Save(ctx))
				switch m.(type) {
				case auditableModel:
					db := ctx.Value("tx").(Querier)
					row := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM audit_log WHERE entity_id = $1", m.id())
					count := 0
					assert.Nil(t, row.Scan(&count))
					assert.Greater(t, count, 0)
				}
			})
		}
		closeTx(t, ctx)
	}
}

func TestFindByID(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	models := modelsUnderTest

	for _, c := range models {
		for _, m := range c {
			t.Run(m.tablename(), func(t *testing.T) {
				assert.Nil(t, m.Save(ctx))
			})
		}
	}

	for _, c := range models {
		m := c[len(c)-1]
		found := m.blank()
		err := found.FindByID(ctx, m.id())
		assert.Nil(t, err)
		m.nullDynamicValues()
		found.nullDynamicValues()
		assert.Equal(t, m, found)
	}

	closeTx(t, ctx)
}

func TestFindByColumn(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	models := modelsUnderTest

	for _, c := range models {
		for _, m := range c {
			t.Run(m.tablename(), func(t *testing.T) {
				assert.Nil(t, m.Save(ctx))
			})
		}
	}

	for _, c := range models {
		m := c[len(c)-1]
		t.Run(m.tablename(), func(t *testing.T) {
			found := m.blank()
			err := found.FindByColumn(ctx, "id", m.id())
			assert.Nil(t, err)
			found.nullDynamicValues()
			m.nullDynamicValues()
			assert.Equal(t, m, found)
		})
	}

	closeTx(t, ctx)
}

func TestAuditLog(t *testing.T) {
	t.Parallel()
	ctx := getCtx(t)

	models := modelsUnderTest

	for _, c := range models {
		for _, m := range c {
			t.Run(m.tablename(), func(t *testing.T) {
				assert.Nil(t, m.Save(ctx))
			})
		}
	}

	for _, c := range models {
		m := c[len(c)-1]
		t.Run(m.tablename(), func(t *testing.T) {
			found := m.blank()
			err := found.FindByColumn(ctx, "id", m.id())
			assert.Nil(t, err)
			found.nullDynamicValues()
			m.nullDynamicValues()
			assert.Equal(t, m, found)
		})
	}

	closeTx(t, ctx)
}

func TestFindAll(t *testing.T) {
	t.Parallel()
	for _, c := range modelCollectionsUnderTest {
		ctx := getCtx(t)
		m := c.collection
		t.Run(m.tablename(), func(t *testing.T) {
			for _, i := range c.deps {
				err := i.Save(ctx)
				assert.Nil(t, err)
			}
			for i := range m.Iter() {
				err := i.Save(ctx)
				assert.Nil(t, err)
			}

			found := m.blank()
			err := found.FindAll(ctx, All{})
			assert.Nil(t, err)

			matched := 0
			for i := range m.Iter() {
				for j := range found.Iter() {
					if i.id() == j.id() {
						matched++
					}
				}
			}
			assert.Equal(t, 2, matched)
		})
		closeTx(t, ctx)
	}
}

type model interface {
	Save(context.Context) error
	FindByID(context.Context, string) error
	FindByColumn(context.Context, string, string) error
	nullDynamicValues()
	blank() model
	id() string
	tablename() string
}

type auditableModel interface {
	auditQuery(context.Context, string) string
}

type models interface {
	FindAll(context.Context, Query) error
	Iter() <-chan model
	tablename() string
	blank() models
}
