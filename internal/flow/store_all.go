package flow

import "time"

// All returns every recorded command, oldest first.
func (s *Store) All() ([]Record, error) {
	rows, err := s.db.Query(`SELECT ts, command, dir, project, exit_code, duration_ms
FROM commands ORDER BY ts ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var ts, ms int64
		var r Record
		if err := rows.Scan(&ts, &r.Command, &r.Dir, &r.Project, &r.ExitCode, &ms); err != nil {
			return nil, err
		}
		r.Time = time.Unix(ts, 0)
		r.Duration = time.Duration(ms) * time.Millisecond
		out = append(out, r)
	}
	return out, rows.Err()
}
