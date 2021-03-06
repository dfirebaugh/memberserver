package database

import (
	"context"
	"fmt"
	"memberserver/api/models"
	"strings"

	"github.com/jackc/pgx/v4"
	log "github.com/sirupsen/logrus"
)

func NewDBMemberStore(db *Database) *DBMemberStore {
	return &DBMemberStore{
		db: db,
	}
}

type DBMemberStore struct {
	db *Database
}

func (m *DBMemberStore) GetMembers() []models.Member {
	var members []models.Member
	rows, err := m.db.getConn().Query(m.db.ctx, memberDbMethod.getMember())
	if err != nil {
		log.Errorf("conn.Query failed: %v", err)
	}

	defer rows.Close()

	resourceMemo := make(map[string]models.MemberResource)

	for rows.Next() {
		var rIDs []string
		var member models.Member
		err = rows.Scan(&member.ID, &member.Name, &member.Email, &member.RFID, &member.Level, &rIDs)
		if err != nil {
			log.Errorf("error scanning row: %s", err)
		}

		// having issues with unmarshalling a jsonb object array from pgx
		// using a less efficient approach for now
		// TODO: fix this on the query level
		for _, rID := range rIDs {
			if _, exist := resourceMemo[rID]; exist {
				member.Resources = append(member.Resources, models.MemberResource{ResourceID: rID, Name: resourceMemo[rID].Name})
				continue
			}

			resource, err := m.db.GetResourceByID(rID)
			if err != nil {
				log.Debugf("error getting resource by id in memberResource lookup: %s %s_\n", err.Error(), rID)
				continue
			}

			resourceMemo[rID] = models.MemberResource{
				ResourceID: resource.ID,
				Name:       resource.Name,
			}

			member.Resources = append(member.Resources, models.MemberResource{ResourceID: rID, Name: resource.Name})
		}

		members = append(members, member)
	}

	return members

	return members
}

func (db *Database) GetMemberByEmail(memberEmail string) (models.Member, error) {
	m := NewDBMemberStore(db)
	return m.GetMemberByEmail(memberEmail)
}

// GetMemberByEmail - lookup a member by their email address
func (m *DBMemberStore) GetMemberByEmail(memberEmail string) (models.Member, error) {
	var member models.Member
	var rIDs []string

	err := m.db.getConn().QueryRow(context.Background(), memberDbMethod.getMemberByEmail(), memberEmail).Scan(&member.ID, &member.Name, &member.Email, &member.RFID, &member.Level, &rIDs)
	if err == pgx.ErrNoRows {
		return member, err
	}
	if err != nil {
		log.Errorf("error getting member by email: %v", memberEmail)
		return member, fmt.Errorf("conn.Query failed: %w", err)
	}

	resourceMemo := make(map[string]models.MemberResource)

	// having issues with unmarshalling a jsonb object array from pgx
	// using a less efficient approach for now
	// TODO: fix this on the query level
	for _, rID := range rIDs {
		if _, exist := resourceMemo[rID]; exist {
			member.Resources = append(member.Resources, models.MemberResource{ResourceID: rID, Name: resourceMemo[rID].Name})
			continue
		}
		resource, err := m.db.GetResourceByID(rID)
		if err != nil {
			log.Debugf("error getting resource by id in memberResource lookup: %s %s\n", err.Error(), rID)
		}

		resourceMemo[rID] = models.MemberResource{
			ResourceID: resource.ID,
			Name:       resource.Name,
		}
		member.Resources = append(member.Resources, models.MemberResource{ResourceID: rID, Name: resource.Name})
	}

	return member, nil
}

func (m *DBMemberStore) AssignRFID(email string, rfid string) (models.Member, error) {
	member, err := m.GetMemberByEmail(email)
	if err != nil {
		log.Errorf("error retrieving a member with that email address %s", err.Error())
		return member, err
	}

	err = m.db.getConn().QueryRow(context.Background(), memberDbMethod.setMemberRFIDTag(), email, encodeRFID(rfid)).Scan(&member.RFID)
	if err != nil {
		return member, fmt.Errorf("conn.Query failed: %v", err)
	}

	return member, err
}

func (m *DBMemberStore) AddNewMember(newMember models.Member) (models.Member, error) {
	err := m.db.AddMembers([]models.Member{newMember})
	if err != nil {
		return models.Member{}, err
	}
	return newMember, nil
}

// GetMemberTiers - gets the member tiers from DB
func (m *DBMemberStore) GetTiers() []models.Tier {
	rows, err := m.db.getConn().Query(context.Background(), tierDbMethod.getMemberTiers())
	if err != nil {
		log.Errorf("conn.Query failed: %v", err)
	}

	defer rows.Close()

	var tiers []models.Tier

	for rows.Next() {
		var t models.Tier
		err = rows.Scan(&t.ID, &t.Name)
		if err == nil {
			tiers = append(tiers, t)
		}
	}

	return tiers
}

var memberDbMethod MemberDatabaseMethod

// GetMembersWithCredit - gets members that have been credited a membership
//  if a member exists in the member_credits table
//  they are credited a membership
func (db *Database) GetMembersWithCredit() []models.Member {
	rows, err := db.getConn().Query(db.ctx, memberDbMethod.getMembersWithCredit())
	if err != nil {
		log.Errorf("error getting credited members: %v", err)
	}

	defer rows.Close()

	var members []models.Member

	for rows.Next() {
		var m models.Member
		err = rows.Scan(&m.ID, &m.Name, &m.Email, &m.RFID, &m.Level)
		if err != nil {
			log.Errorf("error scanning row: %s", err)
		}

		members = append(members, m)
	}

	return members
}

// AddMembers adds multiple members to the database
func (db *Database) AddMembers(members []models.Member) error {
	sqlStr := `INSERT INTO membership.members(
name, email, member_tier_id)
VALUES `

	var valStr []string
	for _, m := range members {
		// postgres doesn't like apostrophes
		memberName := strings.Replace(m.Name, "'", "''", -1)

		// if member level isn't set them to inactive,
		//   otherwise, use the level they already have.
		if m.Level == 0 {
			m.Level = uint8(Inactive)
		}

		valStr = append(valStr, fmt.Sprintf("('%s', '%s', %d)", memberName, m.Email, m.Level))
	}

	str := strings.Join(valStr, ",")

	_, err := db.getConn().Exec(context.Background(), sqlStr+str+" ON CONFLICT DO NOTHING;")
	if err != nil {
		return fmt.Errorf("add members query failed: %v", err)
	}
	for _, m := range members {
		log.Println("Adding default resource")
		db.AddUserToDefaultResources(m.Email)
	}

	return err
}
