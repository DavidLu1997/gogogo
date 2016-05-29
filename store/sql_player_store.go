// Copyright (c) 2016 David Lu
// See License.txt

package store

import (
	"database/sql"
	"fmt"
	"github.com/davidlu1997/gogogo/model"
	"strings"
)

type SqlPlayerStore struct {
	*SqlStore
}

func NewSqlPlayerStore(sqlStore *SqlStore) PlayerStore {
	ps := &SqlPlayerStore{sqlStore}

	for _, db := range sqlStore.GetAllConns() {
		table := db.AddTableWithName(model.Player{}, "Players").SetKeys(false, "Id")
		table.ColMap("Id").SetMaxSize(24)
		table.ColMap("Playername").SetMaxSize(64).SetUnique(true)
		table.ColMap("Password").SetMaxSize(128)
		table.ColMap("Email").SetMaxSize(128).SetUnique(true)
		table.ColMap("AllowStats").SetMaxSize(1)
		table.ColMap("Locale").SetMaxSize(5)
	}

	return ps
}

func (ps SqlPlayerStore) CreateIndexesIfNotExists() {
	ps.CreateIndexesIfNotExists("idx_player_email", "Players", "Email")
}

func (ps SqlPlayerStore) Save(player *model.Player) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		player.PreSave()
		if result.Err = player.IsValid(); result.Err != nil {
			storeChannel <- result
			close(storeChannel)
			return
		}

		if err := ps.GetMaster().Insert(player); err != nil {
			if IsUniqueConstraintError(err.Error(), []string{"Email", "players_email_key", "idx_players_email_unique"}) {
				result.Err = model.NewLocError("SqlPlayerStore.Save", "Email already exists", nil, "player_id="+player.Id+", "+err.Error())
			} else if IsUniqueConstraintError(err.Error(), []string{"Playername", "players_username_key", "idx_players_username_unique"}) {
				result.Err = model.NewLocError("SqlPlayerStore.Save", "Username already exists", nil, "player_id="+player.Id+", "+err.Error())
			} else {
				result.Err = model.NewLocError("SqlPlayerStore.Save", "Player saving error", nil, "player_id="+player.Id+", "+err.Error())
			}
		} else {
			result.Data = player
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) Update(player *model.Player) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		player.PreUpdate()

		if result.Err = player.IsValid(); result.Err != nil {
			storeChannel <- result
			close(storeChannel)
			return
		}

		if oldPlayerResult, err := ps.GetMaster().Get(model.Player{}, Player.Id); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.Update", "", nil, "player_id="+player.Id+", "+err.Error())
		} else if oldPlayerResult == nil {
			result.Err = model.NewLocError("SqlPlayerStore.Update", "Cannot find player to update", nil, "player_id="+player.Id)
		} else {
			oldPlayer := oldPlayerResult.(*model.Player)
			player.CreateAt = oldPlayer.CreateAt
			player.Password = oldPlayer.Password

			if !trustedUpdateData {
				player.DeleteAt = oldPlayer.DeleteAt
			}

			if count, err := ps.GetMaster().Update(player); err != nil {
				if IsUniqueConstraintError(err.Error(), []string{"Email", "players_email_key", "idx_players_email_unique"}) {
					result.Err = model.NewLocError("SqlPlayerStore.Update", "Email already exists", nil, "player_id="+player.Id+", "+err.Error())
				} else if IsUniqueConstraintError(err.Error(), []string{"Username", "players_username_key", "idx_players_username_unique"}) {
					result.Err = model.NewLocError("SqlPlayerStore.Update", "Username already exists", nil, "player_id="+player.Id+", "+err.Error())
				} else {
					result.Err = model.NewLocError("SqlPlayerStore.Update", "Player updating error", nil, "player_id="+player.Id+", "+err.Error())
				}
			} else if count != 1 {
				result.Err = model.NewLocError("SqlPlayerStore.Update", "player update error", nil, fmt.Sprintf("player_id=%v, count=%v", player.Id, count))
			} else {
				result.Data = [2]*model.Player{player, oldPlayer}
			}
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) UpdateUpdateAt(playerId string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		curTime := model.GetMillis()

		if _, err := ps.GetMaster().Exec("UPDATE Players SET UpdateAt = :Time WHERE Id = :PlayerId", map[string]interface{}{"Time": curTime, "PlayerId": playerId}); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.UpdateUpdateAt", "Player updated at error", nil, "player_id="+playerId)
		} else {
			result.Data = playerId
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) UpdatePassword(playerId string, newPassword string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if _, err := ps.GetMaster().Exec("UPDATE Players SET Password = :Password WHERE Id = :PlayerId", map[string]interface{}{"Password": newPassword, "PlayerId": playerId}); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.UpdatePassword", "Player update password error", nil, "player_id="+playerId)
		} else {
			result.Data = playerId
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) Get(id string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if obj, err := ps.GetMaster().Get(model.Player{}, id); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.Get", "Get player by id error", nil, "player_id="+id+", "+err.Error())
		} else if obj == nil {
			result.Err = model.NewLocError("SqlPlayerStore.Get", "Missing player error", nil, "player_id="+id)
		} else {
			result.Data = obj.(*model.Player)
		}

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}

func (ps SqlPlayerStore) GetAll() StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		var data []*model.Player
		if _, err := ps.GetMaster().Select(&data, "SELECT * FROM Players"); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.GetAll", "Get all players error", nil, err.Error())
		}

		result.Data = data

		storeChannel <- result
		close(storeChannel)

	}()

	return storeChannel
}

func (ps SqlPlayerStore) GetByEmail(email string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		email = strings.ToLower(email)

		player := model.Player{}

		if err := ps.GetMaster().SelectOne(&player, "SELECT * FROM Players WHERE Email = :Email", map[string]interface{}{"Email": email}); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.GetByEmail", "Missing player error", nil, "email="+email+", "+err.Error())
		}

		result.Data = &player

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) GetByUsername(username string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		username = strings.ToLower(username)

		player := model.Player{}

		if err := ps.GetMaster().SelectOne(&player, "SELECT * FROM Players WHERE Username = :Username", map[string]interface{}{"Username": username}); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.GetByUsername", "Missing player error", nil, "username="+username+", "+err.Error())
		}

		result.Data = &player

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) GetTotalPlayersCount() StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if count, err := ps.GetMaster().SelectInt("SELECT COUNT(Id) FROM Players"); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.GetTotalPlayersCount", "Get total players count error", nil, err.Error())
		} else {
			result.Data = count
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}

func (ps SqlPlayerStore) PermanentDelete(playerId string) StoreChannel {
	storeChannel := make(StoreChannel)

	go func() {
		result := StoreResult{}

		if _, err := ps.GetMaster().Exec("DELETE FROM Players WHERE Id = :PlayerId", map[string]interface{}{"PlayerId": playerId}); err != nil {
			result.Err = model.NewLocError("SqlPlayerStore.PermanentDelete", "Permanent delete player error", nil, "playerId="+playerId+", "+err.Error())
		}

		storeChannel <- result
		close(storeChannel)
	}()

	return storeChannel
}
