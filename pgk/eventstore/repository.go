package eventstore

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"github.com/goccy/go-json"

	"github.com/saveblush/reraw-relay/core/config"
	"github.com/saveblush/reraw-relay/core/generic"
	"github.com/saveblush/reraw-relay/core/utils"
	"github.com/saveblush/reraw-relay/models"
)

// repository interface
type Repository interface {
	Find(db *gorm.DB, req *Request) (*models.Event, error)
	FindAll(db *gorm.DB, req *Request) ([]*models.Event, error)
	FindByID(db *gorm.DB, ID string) (*models.Event, error)
	Count(db *gorm.DB, req *Request) (*int64, error)
	Insert(db *gorm.DB, req *models.Event) error
	SoftDelete(db *gorm.DB, req *models.Event) error
	Delete(db *gorm.DB, req *models.Event) error
	InsertBlacklist(db *gorm.DB, req *models.Blacklist) error
	FindBlacklists(db *gorm.DB, req *models.Blacklist) ([]*models.Blacklist, error)
	FindEventsExpiration(db *gorm.DB) ([]*models.Event, error)
}

type repository struct {
	ctx context.Context
}

func NewRepository() Repository {
	return &repository{}
}

func makePlaceParams(n int) string {
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

func (r *repository) query(req *Request) (string, []any, error) {
	var conditions []string
	var params []any

	//conditions = append(conditions, `(deleted_at IS NULL AND (CASE WHEN `+strconv.Itoa(int(utils.Now().Unix()))+` > expiration THEN 1 ELSE 0 END) = ?)`)
	//params = append(params, 0)
	conditions = append(conditions, `(deleted_at IS NULL)`)

	if len(req.NostrFilter.IDs) > 0 {
		for _, v := range req.NostrFilter.IDs {
			params = append(params, v)
		}
		conditions = append(conditions, `id IN (`+makePlaceParams(len(req.NostrFilter.IDs))+`)`)
	}

	if len(req.NostrFilter.Kinds) > 0 {
		for _, v := range req.NostrFilter.Kinds {
			params = append(params, v)
		}
		conditions = append(conditions, `kind IN (`+makePlaceParams(len(req.NostrFilter.Kinds))+`)`)
	}

	if len(req.NostrFilter.Authors) > 0 {
		for _, v := range req.NostrFilter.Authors {
			params = append(params, v)
		}
		conditions = append(conditions, `pubkey IN (`+makePlaceParams(len(req.NostrFilter.Authors))+`)`)
	}

	if req.NostrFilter.Search != "" {
		conditions = append(conditions, `content LIKE ?`)
		params = append(params, `%`+strings.ReplaceAll(req.NostrFilter.Search, `%`, `\%`)+`%`)
	}

	if !generic.IsEmpty(req.NostrFilter.Since) {
		conditions = append(conditions, `created_at >= ?`)
		params = append(params, req.NostrFilter.Since)
	}

	if !generic.IsEmpty(req.NostrFilter.Until) {
		conditions = append(conditions, `created_at <= ?`)
		params = append(params, req.NostrFilter.Until)
	}

	tagQuery := make([]string, 0, 1)
	if !generic.IsEmpty(req.NostrFilter.Tags) {
		for _, v := range req.NostrFilter.Tags {
			tagQuery = append(tagQuery, v...)
		}
	}
	if !generic.IsEmpty(tagQuery) {
		for _, tagValue := range tagQuery {
			params = append(params, tagValue)
		}
		conditions = append(conditions, `tagvalues && ARRAY[`+makePlaceParams(len(tagQuery))+`]`)
	}

	if len(conditions) == 0 {
		conditions = append(conditions, `1`)
	}

	var limit int
	var sqlLimit string
	if req.NostrFilter.Limit > 0 {
		limit = req.NostrFilter.Limit
	} else if config.CF.Info.Limitation.MaxLimit > 0 {
		if !req.DoCount {
			limit = config.CF.Info.Limitation.MaxLimit
		}
	} else {
		// กรณีไม่กำหนดช่วงในการหาเหตุการณ์
		if generic.IsEmpty(req.NostrFilter.Since) {
			limit = 50
		} else {
			req.NoLimit = true
		}
	}

	// กรณี noLimit = true จะมาฟังชั่นอื่น เพราะไม่ต้องการ limit เช่น การลบเหตุการณ์(NIP-09)
	if !req.NoLimit {
		sqlLimit = "LIMIT ?"
		params = append(params, limit)
	}

	var sqlField string
	var sqlOrderBy string
	if req.DoCount {
		sqlField = "COUNT(1)"
		sqlOrderBy = ""
	} else {
		sqlField = "id, created_at, pubkey, kind, content, tags, sig"
		sqlOrderBy = "ORDER BY created_at DESC, id"
	}

	sql := `SELECT ` + sqlField + `
			FROM ` + models.Event{}.TableName() + ` 
			WHERE ` + strings.Join(conditions, " AND ") + `
			` + sqlOrderBy + ` ` + sqlLimit

	return sql, params, nil
}

func (r *repository) Find(db *gorm.DB, req *Request) (*models.Event, error) {
	sql, params, err := r.query(req)
	if err != nil {
		return nil, err
	}

	entities := &models.Event{}
	err = db.WithContext(r.ctx).Limit(1).Raw(sql, params...).Scan(entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) FindAll(db *gorm.DB, req *Request) ([]*models.Event, error) {
	sql, params, err := r.query(req)
	if err != nil {
		return nil, err
	}

	entities := []*models.Event{}
	err = db.WithContext(r.ctx).Raw(sql, params...).Scan(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) FindByID(db *gorm.DB, ID string) (*models.Event, error) {
	entities := &models.Event{}
	err := db.WithContext(r.ctx).Limit(1).Where("id = ?", ID).Find(entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) Count(db *gorm.DB, req *Request) (*int64, error) {
	sql, params, err := r.query(req)
	if err != nil {
		return nil, err
	}

	var entities *int64
	err = db.WithContext(r.ctx).Raw(sql, params...).Scan(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) Insert(db *gorm.DB, req *models.Event) error {
	tags, errTags := json.Marshal(&req.Tags)
	if errTags != nil {
		return errTags
	}

	data := map[string]interface{}{
		"id":         req.ID,
		"created_at": req.CreatedAt,
		"updated_at": req.UpdatedAt,
		"deleted_at": req.DeletedAt,
		"pubkey":     req.Pubkey,
		"Kind":       req.Kind,
		"content":    req.Content,
		"tags":       tags,
		"sig":        req.Sig,
		"expiration": req.Expiration,
	}
	err := db.Model(&models.Event{}).Create(&data).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) SoftDelete(db *gorm.DB, req *models.Event) error {
	err := db.Model(&req).Select("DeletedAt").Updates(&models.Event{DeletedAt: utils.Pointer(models.Timestamp(utils.Now().Unix()))}).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) Delete(db *gorm.DB, req *models.Event) error {
	err := db.Delete(&req).Error
	if err != nil {
		return err
	}

	return nil
}

func (r *repository) InsertBlacklist(db *gorm.DB, req *models.Blacklist) error {
	query := db.Model(&models.Blacklist{})
	query.Where("pubkey = ?", req.Pubkey)
	query.Updates(&req)
	row := query.RowsAffected
	if row == 0 {
		err := db.Model(&models.Blacklist{}).Create(&req).Error
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) queryFindBots(db *gorm.DB, req *models.Blacklist) *gorm.DB {
	if !generic.IsEmpty(req.Pubkey) {
		db = db.Where("pubkey = ?", req.Pubkey)
	}

	return db
}

func (r *repository) FindBlacklists(db *gorm.DB, req *models.Blacklist) ([]*models.Blacklist, error) {
	entities := []*models.Blacklist{}
	query := r.queryFindBots(db, req)
	err := query.WithContext(r.ctx).Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) FindEventsExpiration(db *gorm.DB) ([]*models.Event, error) {
	entities := []*models.Event{}
	query := db.Where("expiration < ?", utils.Now().Unix())
	query.Where("deleted_at IS NULL")
	err := query.WithContext(r.ctx).Find(&entities).Error
	if err != nil {
		return nil, err
	}

	return entities, nil
}
