// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.17.2
// source: entry_tag.sql

package db

import (
	"context"
)

const getEntryTags = `-- name: GetEntryTags :many
SELECT id, entry_id, tag_id, organisation_id, created_at, deleted_at
FROM entry_tags
WHERE organisation_id = $1 AND entry_id = $2
`

type GetEntryTagsParams struct {
	OrganisationID string `json:"organisation_id"`
	EntryID        string `json:"entry_id"`
}

func (q *Queries) GetEntryTags(ctx context.Context, arg GetEntryTagsParams) ([]EntryTag, error) {
	rows, err := q.db.QueryContext(ctx, getEntryTags, arg.OrganisationID, arg.EntryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []EntryTag
	for rows.Next() {
		var i EntryTag
		if err := rows.Scan(
			&i.ID,
			&i.EntryID,
			&i.TagID,
			&i.OrganisationID,
			&i.CreatedAt,
			&i.DeletedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}
