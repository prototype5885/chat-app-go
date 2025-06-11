package handlers

func isServerOwner(userID uint64, serverID uint64) (bool, error) {
	var ownsServer bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM servers WHERE id = ? AND owner_id = ?)", serverID, userID).Scan(&ownsServer)
	if err != nil {
		return false, err
	}

	if !ownsServer {
		return false, nil
	}
	return true, nil
}
