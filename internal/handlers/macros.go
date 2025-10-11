package handlers

func isServerOwner(userID int64, serverID int64) (bool, error) {
	var ownsServer bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM servers WHERE id = $1 AND owner_id = $2)", serverID, userID).Scan(&ownsServer)
	if err != nil {
		return false, err
	}

	if !ownsServer {
		return false, nil
	}
	return true, nil
}

func addServerMember(serverID int64, userID int64) error {
	_, err := db.Exec("INSERT INTO server_members (server_id, user_id) VALUES ($1, $2)", serverID, userID)
	if err != nil {
		return err
	}
	return nil
}
