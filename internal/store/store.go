package store

import (
	"LoopGuard/internal/model"
	"os"
	"path/filepath"
	"strconv"

	"gorm.io/gorm"
)

type Store struct {
	db *gorm.DB
}

func New(db *gorm.DB) *Store { return &Store{db: db} }

func (s *Store) DB() *gorm.DB { return s.db }

func (s *Store) AutoMigrate() error {
	if err := s.db.AutoMigrate(
		&model.User{}, &model.APIKey{}, &model.Program{},
		&model.ProgramVersion{}, &model.Ticket{}, &model.Execution{},
		&model.WebhookConfig{}, &model.WebhookDelivery{},
	); err != nil {
		return err
	}
	// 清理重构残留：旧 binary_path 列已从模型移除
	if s.db.Migrator().HasColumn(&model.Program{}, "binary_path") {
		if err := s.db.Migrator().DropColumn(&model.Program{}, "binary_path"); err != nil {
			return err
		}
	}
	return nil
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
func (s *Store) GetAPIKey(id uint64) (*model.APIKey, error) {
	var k model.APIKey
	err := s.db.First(&k, id).Error
	return &k, err
}

func (s *Store) GetAPIKeyByHash(hash string) (*model.APIKey, error) {
	var k model.APIKey
	err := s.db.Where("key_hash = ? AND enabled = ?", hash, true).First(&k).Error
	return &k, err
}
func (s *Store) UpdateAPIKey(k *model.APIKey) error { return s.db.Save(k).Error }
func (s *Store) DeleteAPIKey(id uint64) error { return s.db.Delete(&model.APIKey{}, id).Error }
func (s *Store) GetAPIKeysByIDs(ids []uint64) (map[uint64]model.APIKey, error) {
	m := make(map[uint64]model.APIKey)
	if len(ids) == 0 {
		return m, nil
	}
	var ks []model.APIKey
	if err := s.db.Where("id IN ?", ids).Find(&ks).Error; err != nil {
		return nil, err
	}
	for _, k := range ks {
		m[k.ID] = k
	}
	return m, nil
}

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
func (s *Store) DeleteProgram(id uint64) error { return s.db.Delete(&model.Program{}, id).Error }
func (s *Store) GetProgramsByIDs(ids []uint64) (map[uint64]model.Program, error) {
	m := make(map[uint64]model.Program)
	if len(ids) == 0 {
		return m, nil
	}
	var ps []model.Program
	if err := s.db.Where("id IN ?", ids).Find(&ps).Error; err != nil {
		return nil, err
	}
	for _, p := range ps {
		m[p.ID] = p
	}
	return m, nil
}

// ProgramVersions
func (s *Store) CreateProgramVersion(pv *model.ProgramVersion) error {
	return s.db.Create(pv).Error
}
func (s *Store) GetProgramVersion(programID uint64, version int) (*model.ProgramVersion, error) {
	var pv model.ProgramVersion
	err := s.db.Where("program_id = ? AND version = ?", programID, version).First(&pv).Error
	return &pv, err
}
func (s *Store) ListProgramVersions(programID uint64) ([]model.ProgramVersion, error) {
	var pvs []model.ProgramVersion
	err := s.db.Where("program_id = ?", programID).Order("version desc").Find(&pvs).Error
	return pvs, err
}
func (s *Store) DeleteProgramVersionsByProgramID(programID uint64) error {
	return s.db.Where("program_id = ?", programID).Delete(&model.ProgramVersion{}).Error
}

func (s *Store) DeleteProgramWithCascade(id uint64, workspaceDir string) error {
	p, err := s.GetProgram(id)
	if err != nil {
		return err
	}
	if err := s.DeleteProgramVersionsByProgramID(id); err != nil {
		return err
	}
	if err := s.DeleteProgram(id); err != nil {
		return err
	}
	// 清理当前文件目录
	os.RemoveAll(filepath.Join(workspaceDir, p.Project, p.Name))
	// 清理版本快照目录
	os.RemoveAll(filepath.Join(workspaceDir, ".versions", strconv.FormatUint(id, 10)))
	return nil
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

// Webhooks
func (s *Store) CreateWebhook(w *model.WebhookConfig) error { return s.db.Create(w).Error }
func (s *Store) GetWebhook(id uint64) (*model.WebhookConfig, error) {
	var w model.WebhookConfig
	err := s.db.First(&w, id).Error
	return &w, err
}
func (s *Store) GetWebhooksByProgram(programID uint64) ([]model.WebhookConfig, error) {
	var ws []model.WebhookConfig
	err := s.db.Where("program_id = ?", programID).Order("id desc").Find(&ws).Error
	return ws, err
}
func (s *Store) GetWebhooksByEventType(eventType string) ([]model.WebhookConfig, error) {
	var ws []model.WebhookConfig
	// 使用 LIKE 查询，因为 EventTypes 是逗号分隔的字符串
	err := s.db.Where("event_types LIKE ?", "%"+eventType+"%").Find(&ws).Error
	return ws, err
}
func (s *Store) DeleteWebhook(id uint64) error { return s.db.Delete(&model.WebhookConfig{}, id).Error }
func (s *Store) UpdateWebhook(w *model.WebhookConfig) error { return s.db.Save(w).Error }

// Webhook Deliveries
func (s *Store) CreateWebhookDelivery(d *model.WebhookDelivery) error { return s.db.Create(d).Error }
func (s *Store) GetWebhookDeliveries(webhookID uint64) ([]model.WebhookDelivery, error) {
	var ds []model.WebhookDelivery
	err := s.db.Where("webhook_id = ?", webhookID).Order("id desc").Find(&ds).Error
	return ds, err
}
