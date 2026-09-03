package squirrel

import (
	"context"
	"fmt"
	"time"
)

// Noticed is one line about one thing.
type Noticed struct {
	ID    int64
	Kind  string
	RefID int64
	Words string
	At    time.Time
}

// Notice keeps a line about one thing, replacing whatever was there.
//
// Replacing rather than stacking: two lines under one strip is a conversation,
// and a strip is not somewhere a conversation happens. A refusal is cleared
// with the words it was about, because the new line is not the one that was
// refused.
func (s *Store) Notice(ctx context.Context, personID int64, kind string, refID int64, words string, at time.Time) error {
	if _, err := s.pool.Exec(ctx, `
		insert into noticed (person_id, kind, ref_id, words, made_at)
		values ($1, $2, $3, $4, $5)
		on conflict (person_id, kind, ref_id)
		do update set words = excluded.words, made_at = excluded.made_at, refused_at = null`,
		personID, kind, refID, words, at); err != nil {
		return fmt.Errorf("keeping what was noticed: %w", err)
	}
	return nil
}

// WhatWasNoticed is every line this person has not refused.
func (s *Store) WhatWasNoticed(ctx context.Context, personID int64) ([]Noticed, error) {
	rows, err := s.pool.Query(ctx, `
		select id, kind, ref_id, words, made_at from noticed
		 where person_id = $1 and refused_at is null
		 order by made_at desc`, personID)
	if err != nil {
		return nil, fmt.Errorf("reading what was noticed: %w", err)
	}
	defer rows.Close()

	out := []Noticed{}
	for rows.Next() {
		var one Noticed
		if err := rows.Scan(&one.ID, &one.Kind, &one.RefID, &one.Words, &one.At); err != nil {
			return nil, fmt.Errorf("reading what was noticed: %w", err)
		}
		out = append(out, one)
	}
	return out, rows.Err()
}

// NoticedAt is when it last noticed anything, which is what paces it.
func (s *Store) NoticedAt(ctx context.Context, personID int64) (time.Time, error) {
	var at *time.Time
	if err := s.pool.QueryRow(ctx,
		`select max(made_at) from noticed where person_id = $1`, personID).Scan(&at); err != nil {
		return time.Time{}, fmt.Errorf("reading when it last noticed: %w", err)
	}
	if at == nil {
		return time.Time{}, nil
	}
	return *at, nil
}

// NotUseful marks one line as not worth having said.
//
// The row stays and the words stay: what was refused is what the next pass is
// told not to write again, and a deleted row teaches nothing.
func (s *Store) NotUseful(ctx context.Context, personID, id int64, at time.Time) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`update noticed set refused_at = $3
		  where id = $2 and person_id = $1 and refused_at is null`, personID, id, at)
	if err != nil {
		return false, fmt.Errorf("marking a line as not useful: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// WhatWasRefused is the lines this person said were not useful, newest first.
// It is what the next pass is shown so that it does not write them again.
func (s *Store) WhatWasRefused(ctx context.Context, personID int64, limit int) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		select words from noticed
		 where person_id = $1 and refused_at is not null
		 order by refused_at desc limit $2`, personID, limit)
	if err != nil {
		return nil, fmt.Errorf("reading what was refused: %w", err)
	}
	defer rows.Close()

	out := []string{}
	for rows.Next() {
		var words string
		if err := rows.Scan(&words); err != nil {
			return nil, fmt.Errorf("reading what was refused: %w", err)
		}
		out = append(out, words)
	}
	return out, rows.Err()
}
