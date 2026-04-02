package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var ErrNotFound = errors.New("not found")

type Inbound struct {
	ID        int64     `json:"id"`
	Tag       string    `json:"tag"`
	Protocol  string    `json:"protocol"`
	Port      uint16    `json:"port"`
	Settings  string    `json:"settings"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (s *Store) CreateInbound(ib *Inbound) error {
	result, err := s.db.Exec(
		`INSERT INTO inbounds (tag, protocol, port, settings) VALUES (?, ?, ?, ?)`,
		ib.Tag, ib.Protocol, ib.Port, ib.Settings,
	)
	if err != nil {
		return fmt.Errorf("failed to create inbound: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	ib.ID = id
	return nil
}

func (s *Store) GetInbound(id int64) (*Inbound, error) {
	ib := &Inbound{}
	err := s.db.QueryRow(
		`SELECT id, tag, protocol, port, settings, created_at, updated_at FROM inbounds WHERE id = ?`,
		id,
	).Scan(&ib.ID, &ib.Tag, &ib.Protocol, &ib.Port, &ib.Settings, &ib.CreatedAt, &ib.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get inbound: %w", err)
	}
	return ib, nil
}

func (s *Store) ListInbounds() ([]*Inbound, error) {
	rows, err := s.db.Query(
		`SELECT id, tag, protocol, port, settings, created_at, updated_at FROM inbounds ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list inbounds: %w", err)
	}
	defer rows.Close()

	var inbounds []*Inbound
	for rows.Next() {
		ib := &Inbound{}
		if err := rows.Scan(&ib.ID, &ib.Tag, &ib.Protocol, &ib.Port, &ib.Settings, &ib.CreatedAt, &ib.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan inbound: %w", err)
		}
		inbounds = append(inbounds, ib)
	}
	return inbounds, rows.Err()
}

func (s *Store) UpdateInbound(ib *Inbound) error {
	result, err := s.db.Exec(
		`UPDATE inbounds SET tag = ?, protocol = ?, port = ?, settings = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		ib.Tag, ib.Protocol, ib.Port, ib.Settings, ib.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update inbound: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteInbound(id int64) error {
	result, err := s.db.Exec(`DELETE FROM inbounds WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete inbound: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
