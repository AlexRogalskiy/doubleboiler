package models

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type SearchResult struct {
	Path       string
	EntityType string
	ID         string
	Label      string
	Rank       float64
}

type SearchResults struct {
	Data  []SearchResult
	Query Query
}

func (results *SearchResults) FindAll(ctx context.Context, q Query) error {
	results.Query = q

	db := ctx.Value("tx").(Querier)

	var rows *sql.Rows
	var err error

	switch v := q.(type) {
	default:
		return fmt.Errorf("Unknown query")
	case ByPhrase:
		parts := []string{}
		for _, entity := range Searchables {
			include := len(v.EntityFilter) == 0 || v.EntityFilter[entity.Label]

			if include {
				parts = append(parts, entity.searchFunc(v))
			}
		}
		filteredParts := []string{}
		for _, part := range parts {
			if part == "" {
				continue
			}
			filteredParts = append(filteredParts, part)
		}

		if len(filteredParts) == 0 {
			return nil
		}

		query := strings.Join(filteredParts, " UNION ALL ")

		query += " ORDER BY rank DESC " + q.Pagination()

		rows, err = db.QueryContext(ctx, query, v.OrgID, v.Phrase)
	}
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		result := SearchResult{}
		err = rows.Scan(
			&result.EntityType, &result.Path, &result.ID, &result.Label, &result.Rank,
		)
		if err != nil {
			return err
		}
		(*results).Data = append((*results).Data, result)
	}

	return err
}
