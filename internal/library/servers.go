package library

import "fmt"

// Server represents a streaming server configuration.
type Server struct {
	ID       string
	Name     string
	Type     string
	URL      string
	Username string
	Password string
}

// AddServer inserts a new server.
func (db *DB) AddServer(s Server) error {
	_, err := db.Exec(
		"INSERT INTO servers (id, name, type, url, username, password) VALUES (?, ?, ?, ?, ?, ?)",
		s.ID, s.Name, s.Type, s.URL, s.Username, s.Password,
	)
	if err != nil {
		return fmt.Errorf("add server: %w", err)
	}
	return nil
}

// GetServer returns a server by ID.
func (db *DB) GetServer(id string) (Server, error) {
	var s Server
	err := db.QueryRow(
		"SELECT id, name, type, url, username, password FROM servers WHERE id = ?", id,
	).Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.Username, &s.Password)
	if err != nil {
		return Server{}, fmt.Errorf("get server: %w", err)
	}
	return s, nil
}

// GetServers returns all servers ordered by name.
func (db *DB) GetServers() ([]Server, error) {
	rows, err := db.Query("SELECT id, name, type, url, username, password FROM servers ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("get servers: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var servers []Server
	for rows.Next() {
		var s Server
		if err := rows.Scan(&s.ID, &s.Name, &s.Type, &s.URL, &s.Username, &s.Password); err != nil {
			return nil, fmt.Errorf("scan server: %w", err)
		}
		servers = append(servers, s)
	}
	return servers, rows.Err()
}

// UpdateServer updates an existing server.
func (db *DB) UpdateServer(s Server) error {
	_, err := db.Exec(
		"UPDATE servers SET name = ?, type = ?, url = ?, username = ?, password = ? WHERE id = ?",
		s.Name, s.Type, s.URL, s.Username, s.Password, s.ID,
	)
	if err != nil {
		return fmt.Errorf("update server: %w", err)
	}
	return nil
}

// DeleteServer deletes a server by ID.
func (db *DB) DeleteServer(id string) error {
	_, err := db.Exec("DELETE FROM servers WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete server: %w", err)
	}
	return nil
}
