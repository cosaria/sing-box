package store

import (
	"fmt"
	"time"
)

type TrafficLog struct {
	ID         int64     `json:"id"`
	InboundTag string    `json:"inbound_tag"`
	Upload     int64     `json:"upload"`
	Download   int64     `json:"download"`
	Timestamp  time.Time `json:"timestamp"`
}

type TrafficSummary struct {
	Tag      string `json:"tag"`
	Upload   int64  `json:"upload"`
	Download int64  `json:"download"`
}

func (s *Store) InsertTrafficLog(inboundTag string, upload, download int64) error {
	_, err := s.db.Exec(
		`INSERT INTO traffic_logs (inbound_tag, upload, download) VALUES (?, ?, ?)`,
		inboundTag, upload, download,
	)
	if err != nil {
		return fmt.Errorf("failed to insert traffic log: %w", err)
	}
	return nil
}

func (s *Store) GetTrafficSummary() ([]TrafficSummary, error) {
	rows, err := s.db.Query(
		`SELECT inbound_tag, SUM(upload), SUM(download) FROM traffic_logs GROUP BY inbound_tag ORDER BY inbound_tag`,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query traffic summary: %w", err)
	}
	defer rows.Close()
	var summaries []TrafficSummary
	for rows.Next() {
		var ts TrafficSummary
		if err := rows.Scan(&ts.Tag, &ts.Upload, &ts.Download); err != nil {
			return nil, fmt.Errorf("failed to scan traffic summary: %w", err)
		}
		summaries = append(summaries, ts)
	}
	return summaries, rows.Err()
}

func (s *Store) GetTrafficByTag(tag string) ([]TrafficLog, error) {
	rows, err := s.db.Query(
		`SELECT id, inbound_tag, upload, download, timestamp FROM traffic_logs WHERE inbound_tag = ? ORDER BY timestamp DESC`,
		tag,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query traffic by tag: %w", err)
	}
	defer rows.Close()
	var logs []TrafficLog
	for rows.Next() {
		var tl TrafficLog
		if err := rows.Scan(&tl.ID, &tl.InboundTag, &tl.Upload, &tl.Download, &tl.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan traffic log: %w", err)
		}
		logs = append(logs, tl)
	}
	return logs, rows.Err()
}
