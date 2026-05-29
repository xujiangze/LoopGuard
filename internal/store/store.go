package store

import (
	"LoopGuard/internal/model"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) AutoMigrate() error {
	return s.db.AutoMigrate(
		&model.User{}, &model.APIKey{}, &model.Program{},
		&model.Ticket{}, &model.Execution{},
	)
}

// Users
func (s *Store) CreateUser(u *model.User) error { return s.db.Create(u).Error }
func (s *Store) GetUserByUsername(name string) (*model.User, error) {
	var u model.User
	err := s.db.Where("username = ?", name).First(&u).Error
	return &u, err
}
func (s *Store) GetUser(id uint64) (*model.User, error) {
	var u model.User
	err := s.db.First(&u, id).Error
	return &u, err
}
func (s *Store) UpdateUserPassword(id uint64, hash string) error {
	return s.db.Model(&model.User{}).Where("id = ?", id).Update("password_hash", hash).Error
}

// API Keys
func (s *Store) CreateAPIKey(k *model.APIKey) error { return s.db.Create(k).Error }
func (s *Store) ListAPIKeys() ([]model.APIKey, error) {
	var ks []model.APIKey
	err := s.db.Order("id desc").Find(&ks).Error
	return ks, err
}
func (s *Store) GetAPIKeyByHash(hash string) (*model.APIKey, error) {
	var k model.APIKey
	err := s.db.Where("key_hash = ? AND enabled = ?", hash, true).First(&k).Error
	return &k, err
}
func (s *Store) UpdateAPIKey(k *model.APIKey) error { return s.db.Save(k).Error }
func (s *Store) DeleteAPIKey(id uint64) error       { return s.db.Delete(&model.APIKey{}, id).Error }

// Programs
func (s *Store) CreateProgram(p *model.Program) error { return s.db.Create(p).Error }
func (s *Store) UpdateProgram(p *model.Program) error { return s.db.Save(p).Error }
func (s *Store) GetProgram(id uint64) (*model.Program, error) {
	var p model.Program
	err := s.db.First(&p, id).Error
	return &p, err
}
func (s *Store) GetProgramByProjectName(project, name string) (*model.Program, error) {
	var p model.Program
	err := s.db.Where("project = ? AND name = ?", project, name).First(&p).Error
	return &p, err
}
func (s *Store) ListPrograms() ([]model.Program, error) {
	var ps []model.Program
	err := s.db.Order("id desc").Find(&ps).Error
	return ps, err
}

// Tickets
func (s *Store) CreateTicket(t *model.Ticket) error { return s.db.Create(t).Error }
func (s *Store) UpdateTicket(t *model.Ticket) error { return s.db.Save(t).Error }
func (s *Store) GetTicket(id uint64) (*model.Ticket, error) {
	var t model.Ticket
	err := s.db.First(&t, id).Error
	return &t, err
}
func (s *Store) ListTicketsByApprover(approverID uint64, status model.TicketStatus) ([]model.Ticket, error) {
	q := s.db.Where("approver_id = ?", approverID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var ts []model.Ticket
	err := q.Order("id desc").Find(&ts).Error
	return ts, err
}

// Executions
func (s *Store) CreateExecution(e *model.Execution) error { return s.db.Create(e).Error }
func (s *Store) ListExecutionsByTicket(ticketID uint64) ([]model.Execution, error) {
	var es []model.Execution
	err := s.db.Where("ticket_id = ?", ticketID).Order("id asc").Find(&es).Error
	return es, err
}
