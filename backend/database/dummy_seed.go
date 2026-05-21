package database

import (
	"encoding/json"
	"fmt"
	"rosetta/models"

	"gorm.io/gorm"
)

type colDef struct {
	Name     string
	Type     string
	Length   *int
	Scale    *int
	Nullable bool
	PK       bool
	Default  string
	Comment  string
}
type idxDef struct {
	Name    string
	Type    string
	Columns []indexCol
}
type indexCol struct {
	Name  string `json:"name"`
	Order string `json:"order"`
}
type fkDef struct {
	Col, RefTable, RefCol string
}
type tableDef struct {
	Domain  string
	Comment string
	Cols    []colDef
	Indexes []idxDef
	FKs     []fkDef
}

var domainDBConfig = []struct {
	DBName      string
	Description string
	Domain      string
	SchemaNames []string
}{
	{DBName: "电商数据库", Description: "电商业务域", Domain: "电商", SchemaNames: []string{"ods", "dwd"}},
	{DBName: "CRM数据库", Description: "客户关系管理域", Domain: "CRM", SchemaNames: []string{"ods", "dwd"}},
	{DBName: "财务数据库", Description: "财务核算域", Domain: "财务", SchemaNames: []string{"ods", "dwd"}},
	{DBName: "HR数据库", Description: "人力资源域", Domain: "HR", SchemaNames: []string{"ods", "dwd"}},
	{DBName: "物流数据库", Description: "物流供应链域", Domain: "物流", SchemaNames: []string{"ods", "dwd"}},
	{DBName: "分析数据库", Description: "数据分析域", Domain: "分析", SchemaNames: []string{"ods", "dwd"}},
}

func SeedDummyTables(db *gorm.DB) error {
	defs := dummyTableDefs()

	physicalSchemaID := ensureDummySchema(db)
	if physicalSchemaID == 0 {
		return fmt.Errorf("no schema available for deployment")
	}

	var adminID uint64 = 1
	tx := db.Begin()

	logicalDBs := make(map[string]*models.LogicalDatabase)
	logicalSchemas := make(map[string]*models.LogicalSchema)
	allLogicalSchemas := make([]*models.LogicalSchema, 0)

	for _, dbc := range domainDBConfig {
		ldb := models.LogicalDatabase{Name: dbc.DBName, Description: dbc.Description}
		if tx.Where("name = ?", dbc.DBName).First(&ldb).Error != nil {
			if err := tx.Create(&ldb).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("create logical db %s: %w", dbc.DBName, err)
			}
		}
		logicalDBs[dbc.Domain] = &ldb

		for _, sn := range dbc.SchemaNames {
			key := dbc.Domain + "/" + sn
			ls := models.LogicalSchema{DatabaseID: ldb.ID, Name: sn, Description: dbc.Domain + " " + sn}
			if tx.Where("database_id = ? AND name = ?", ldb.ID, sn).First(&ls).Error != nil {
				if err := tx.Create(&ls).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("create logical schema %s: %w", key, err)
				}
			}
			logicalSchemas[key] = &ls
			allLogicalSchemas = append(allLogicalSchemas, &ls)
		}
	}

	modelIDs := make(map[string]uint64)
	tableOrder := make([]string, 0, len(defs))

	domainTableCounts := make(map[string]int)
	for _, def := range defs {
		domainTableCounts[def.Domain]++
	}

	schemaIndex := make(map[string]int)
	for _, dbc := range domainDBConfig {
		schemaIndex[dbc.Domain] = 0
	}

	schemaTableLimit := make(map[string]int)
	for _, dbc := range domainDBConfig {
		total := domainTableCounts[dbc.Domain]
		n := len(dbc.SchemaNames)
		for i, sn := range dbc.SchemaNames {
			perSchema := total / n
			if i < total%n {
				perSchema++
			}
			key := dbc.Domain + "/" + sn
			schemaTableLimit[key] = perSchema
		}
	}

	schemaTableCounts := make(map[string]int)

	for name, def := range defs {
		schemas := getSchemasForDomain(def.Domain)
		si := schemaIndex[def.Domain] % len(schemas)
		sk := def.Domain + "/" + schemas[si]
		if schemaTableCounts[sk] >= schemaTableLimit[sk] {
			si = (si + 1) % len(schemas)
			sk = def.Domain + "/" + schemas[si]
		}
		schemaIndex[def.Domain] = si + 1
		schemaTableCounts[sk]++

		s := logicalSchemas[sk]
		if s == nil {
			tx.Rollback()
			return fmt.Errorf("schema not found: %s", sk)
		}
		ldb := logicalDBs[def.Domain]
		if ldb == nil {
			tx.Rollback()
			return fmt.Errorf("logical db not found for domain: %s", def.Domain)
		}

		var exist models.LogicalModel
		if tx.Where("schema_id = ? AND table_name = ?", s.ID, name).First(&exist).Error == nil {
			modelIDs[name] = exist.ID
			tableOrder = append(tableOrder, name)
			continue
		}

		m := models.LogicalModel{
			DatabaseID:   ldb.ID,
			SchemaID:     s.ID,
			TabName:      name,
			TableComment: fmt.Sprintf("[%s] %s", def.Domain, def.Comment),
			TableStatus:  "PUBLISHED",
			Source:       "MANUAL",
			CreatedBy:    &adminID,
		}
		if err := tx.Create(&m).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("create model %s: %w", name, err)
		}
		modelIDs[name] = m.ID
		tableOrder = append(tableOrder, name)

		for i, c := range def.Cols {
			col := models.ModelColumn{
				ModelID:      m.ID,
				Ordinal:      i + 1,
				ColumnName:   c.Name,
				LogicalType:  c.Type,
				TypeLength:   c.Length,
				TypeScale:    c.Scale,
				Nullable:     c.Nullable,
				DefaultValue: c.Default,
				Comment:      c.Comment,
				IsPrimaryKey: c.PK,
			}
			if err := tx.Create(&col).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("create column %s.%s: %w", name, c.Name, err)
			}
		}

		for _, idx := range def.Indexes {
			colsJSON, _ := json.Marshal(idx.Columns)
			mi := models.ModelIndex{
				ModelID:   m.ID,
				IndexName: idx.Name,
				IndexType: idx.Type,
				Columns:   string(colsJSON),
			}
			if err := tx.Create(&mi).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("create index %s.%s: %w", name, idx.Name, err)
			}
		}

		dep := models.ModelDeployment{
			ModelID:  m.ID,
			SchemaID: physicalSchemaID,
			Dialect:  "MYSQL",
		}
		tx.Where("model_id = ? AND schema_id = ?", m.ID, physicalSchemaID).Delete(&models.ModelDeployment{})
		if err := tx.Create(&dep).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("deploy %s: %w", name, err)
		}
	}

	for name, def := range defs {
		for _, fk := range def.FKs {
			refID, ok := modelIDs[fk.RefTable]
			if !ok {
				continue
			}
			fkName := fmt.Sprintf("fk_%s_%s", name, fk.Col)

			var exist models.ModelForeignKey
			if tx.Where("model_id = ? AND column_name = ?", modelIDs[name], fk.Col).First(&exist).Error == nil {
				continue
			}

			mfk := models.ModelForeignKey{
				ModelID:       modelIDs[name],
				FkName:        fkName,
				ColumnName:    fk.Col,
				RefModelID:    refID,
				RefColumnName: fk.RefCol,
			}
			if err := tx.Create(&mfk).Error; err != nil {
				continue
			}
		}
	}

	instanceMappingCount := int64(0)
	tx.Model(&models.DatabaseInstanceMapping{}).Count(&instanceMappingCount)
	if instanceMappingCount == 0 {
		var dummyInst models.DatasourceInstance
		tx.Where("type = ?", "MYSQL").First(&dummyInst)
		if dummyInst.ID > 0 {
			for _, dbc := range domainDBConfig {
				ldb := logicalDBs[dbc.Domain]
				mapping := models.DatabaseInstanceMapping{
					DatabaseID: ldb.ID,
					InstanceID: dummyInst.ID,
				}
				tx.Where("database_id = ? AND instance_id = ?", ldb.ID, dummyInst.ID).Delete(&models.DatabaseInstanceMapping{})
				tx.Create(&mapping)
			}
		}
	}

	tx.Commit()
	fmt.Printf("Seeded %d tables across %d logical databases\n", len(modelIDs), len(domainDBConfig))
	return nil
}

func getSchemasForDomain(domain string) []string {
	for _, d := range domainDBConfig {
		if d.Domain == domain {
			return d.SchemaNames
		}
	}
	return []string{"default"}
}

func ensureDummySchema(db *gorm.DB) uint64 {
	var inst models.DatasourceInstance
	db.Where("type = ?", "MYSQL").First(&inst)
	if inst.ID == 0 {
		inst = models.DatasourceInstance{
			Name:       "dummy-mysql",
			Type:       "MYSQL",
			Host:       "127.0.0.1",
			Port:       3306,
			Credential: `{"user":"root","password":"rosetta123"}`,
			Status:     "ACTIVE",
		}
		db.Create(&inst)
	}
	var schema models.DatasourceSchema
	db.Where("instance_id = ? AND schema_name = ?", inst.ID, "dummy_erp").First(&schema)
	if schema.ID == 0 {
		schema = models.DatasourceSchema{InstanceID: inst.ID, SchemaName: "dummy_erp", Layer: "DWD"}
		db.Create(&schema)
	}
	return schema.ID
}

func intPtr(i int) *int { return &i }
func dummyTableDefs() map[string]tableDef {
	return map[string]tableDef{
		// ===== E-COMMERCE (25 tables) =====
		"product": {
			Domain: "电商", Comment: "商品主表",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "商品ID"},
				{Name: "product_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "商品编码"},
				{Name: "product_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "商品名称"},
				{Name: "category_id", Type: "BIGINT", Nullable: false, Comment: "分类ID"},
				{Name: "brand", Type: "VARCHAR", Length: intPtr(128), Comment: "品牌"},
				{Name: "unit", Type: "VARCHAR", Length: intPtr(32), Comment: "单位"},
				{Name: "price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "售价"},
				{Name: "cost_price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "成本价"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:ON_SALE/OFF_SALE"},
				{Name: "created_at", Type: "DATETIME", Nullable: false, Comment: "创建时间"},
				{Name: "updated_at", Type: "DATETIME", Comment: "更新时间"},
			},
			Indexes: []idxDef{
				{Name: "idx_product_code", Type: "UNIQUE", Columns: []indexCol{{Name: "product_code", Order: "ASC"}}},
				{Name: "idx_category_id", Type: "NORMAL", Columns: []indexCol{{Name: "category_id", Order: "ASC"}}},
				{Name: "idx_status", Type: "NORMAL", Columns: []indexCol{{Name: "status", Order: "ASC"}}},
			},
			FKs: []fkDef{{Col: "category_id", RefTable: "product_category", RefCol: "id"}},
		},
		"product_category": {
			Domain: "电商", Comment: "商品分类",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "分类ID"},
				{Name: "parent_id", Type: "BIGINT", Comment: "父分类ID"},
				{Name: "category_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "分类名称"},
				{Name: "category_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "分类编码"},
				{Name: "level", Type: "INT", Nullable: false, Comment: "层级"},
				{Name: "sort_order", Type: "INT", Comment: "排序"},
			},
			Indexes: []idxDef{{Name: "idx_parent_id", Type: "NORMAL", Columns: []indexCol{{Name: "parent_id", Order: "ASC"}}}},
			FKs:     []fkDef{{Col: "parent_id", RefTable: "product_category", RefCol: "id"}},
		},
		"product_sku": {
			Domain: "电商", Comment: "商品SKU",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "SKU ID"},
				{Name: "product_id", Type: "BIGINT", Nullable: false, Comment: "商品ID"},
				{Name: "sku_code", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "SKU编码"},
				{Name: "spec_attr", Type: "VARCHAR", Length: intPtr(512), Comment: "规格属性JSON"},
				{Name: "price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "SKU价格"},
				{Name: "stock_quantity", Type: "INT", Nullable: false, Comment: "库存数量"},
			},
			Indexes: []idxDef{{Name: "uk_sku_code", Type: "UNIQUE", Columns: []indexCol{{Name: "sku_code", Order: "ASC"}}}},
			FKs:     []fkDef{{Col: "product_id", RefTable: "product", RefCol: "id"}},
		},
		"inventory": {
			Domain: "电商", Comment: "库存",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "库存ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "warehouse_id", Type: "BIGINT", Comment: "仓库ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "库存数量"},
				{Name: "locked_quantity", Type: "INT", Comment: "锁定数量"},
				{Name: "safety_stock", Type: "INT", Comment: "安全库存"},
				{Name: "last_count_at", Type: "DATETIME", Comment: "最近盘点时间"},
			},
			FKs: []fkDef{{Col: "sku_id", RefTable: "product_sku", RefCol: "id"}, {Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},
		"inventory_log": {
			Domain: "电商", Comment: "库存变动日志",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "日志ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "change_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "变动类型:IN/OUT/ADJUST"},
				{Name: "quantity_change", Type: "INT", Nullable: false, Comment: "变动数量"},
				{Name: "before_quantity", Type: "INT", Comment: "变动前数量"},
				{Name: "after_quantity", Type: "INT", Comment: "变动后数量"},
				{Name: "order_id", Type: "BIGINT", Comment: "关联订单ID"},
				{Name: "created_at", Type: "DATETIME", Nullable: false, Comment: "创建时间"},
			},
			FKs: []fkDef{{Col: "sku_id", RefTable: "product_sku", RefCol: "id"}, {Col: "order_id", RefTable: "order", RefCol: "id"}},
		},

		"order": {
			Domain: "电商", Comment: "订单主表",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "订单ID"},
				{Name: "order_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "订单编号"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "order_status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "订单状态"},
				{Name: "total_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "订单总额"},
				{Name: "discount_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "优惠金额"},
				{Name: "payment_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "实付金额"},
				{Name: "payment_status", Type: "VARCHAR", Length: intPtr(32), Comment: "支付状态"},
				{Name: "shipping_address", Type: "VARCHAR", Length: intPtr(512), Comment: "收货地址"},
				{Name: "order_time", Type: "DATETIME", Nullable: false, Comment: "下单时间"},
			},
			Indexes: []idxDef{
				{Name: "uk_order_no", Type: "UNIQUE", Columns: []indexCol{{Name: "order_no", Order: "ASC"}}},
				{Name: "idx_customer_id", Type: "NORMAL", Columns: []indexCol{{Name: "customer_id", Order: "ASC"}}},
				{Name: "idx_order_time", Type: "NORMAL", Columns: []indexCol{{Name: "order_time", Order: "DESC"}}},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}},
		},
		"order_item": {
			Domain: "电商", Comment: "订单明细",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "明细ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "product_name", Type: "VARCHAR", Length: intPtr(256), Comment: "商品名称快照"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "数量"},
				{Name: "unit_price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "单价"},
				{Name: "total_price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "小计"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}, {Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"order_payment": {
			Domain: "电商", Comment: "支付记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "支付ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "pay_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "支付流水号"},
				{Name: "pay_method", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "支付方式"},
				{Name: "pay_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "支付金额"},
				{Name: "pay_time", Type: "DATETIME", Comment: "支付时间"},
				{Name: "pay_status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "支付状态"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}},
		},
		"order_shipment": {
			Domain: "电商", Comment: "发货记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "发货ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "shipment_no", Type: "VARCHAR", Length: intPtr(128), Comment: "物流单号"},
				{Name: "carrier", Type: "VARCHAR", Length: intPtr(64), Comment: "承运商"},
				{Name: "ship_status", Type: "VARCHAR", Length: intPtr(32), Comment: "发货状态"},
				{Name: "ship_time", Type: "DATETIME", Comment: "发货时间"},
				{Name: "deliver_time", Type: "DATETIME", Comment: "签收时间"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}},
		},
		"shopping_cart": {
			Domain: "电商", Comment: "购物车",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "购物车ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "数量"},
				{Name: "added_at", Type: "DATETIME", Nullable: false, Comment: "添加时间"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"product_review": {
			Domain: "电商", Comment: "商品评价",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "评价ID"},
				{Name: "product_id", Type: "BIGINT", Nullable: false, Comment: "商品ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "rating", Type: "INT", Nullable: false, Comment: "评分1-5"},
				{Name: "content", Type: "TEXT", Comment: "评价内容"},
				{Name: "created_at", Type: "DATETIME", Nullable: false, Comment: "创建时间"},
			},
			FKs: []fkDef{{Col: "product_id", RefTable: "product", RefCol: "id"}, {Col: "customer_id", RefTable: "customer", RefCol: "id"}},
		},
		"coupon": {
			Domain: "电商", Comment: "优惠券",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "优惠券ID"},
				{Name: "coupon_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "券码"},
				{Name: "coupon_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "类型:DISCOUNT/CASH"},
				{Name: "discount_value", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "优惠金额/折扣率"},
				{Name: "min_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "最低消费"},
				{Name: "total_count", Type: "INT", Nullable: false, Comment: "发放数量"},
				{Name: "used_count", Type: "INT", Comment: "已使用数量"},
				{Name: "start_time", Type: "DATETIME", Comment: "开始时间"},
				{Name: "end_time", Type: "DATETIME", Comment: "结束时间"},
			},
		},
		"coupon_usage": {
			Domain: "电商", Comment: "优惠券使用记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "记录ID"},
				{Name: "coupon_id", Type: "BIGINT", Nullable: false, Comment: "优惠券ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "used_time", Type: "DATETIME", Nullable: false, Comment: "使用时间"},
			},
			FKs: []fkDef{{Col: "coupon_id", RefTable: "coupon", RefCol: "id"}, {Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "order_id", RefTable: "order", RefCol: "id"}},
		},
		"supplier": {
			Domain: "电商", Comment: "供应商",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "供应商ID"},
				{Name: "supplier_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "供应商编码"},
				{Name: "supplier_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "供应商名称"},
				{Name: "contact_person", Type: "VARCHAR", Length: intPtr(64), Comment: "联系人"},
				{Name: "phone", Type: "VARCHAR", Length: intPtr(32), Comment: "电话"},
				{Name: "address", Type: "VARCHAR", Length: intPtr(512), Comment: "地址"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
		},

		// ===== CRM (20 tables) =====
		"customer": {
			Domain: "CRM", Comment: "客户主表",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "客户ID"},
				{Name: "customer_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "客户编号"},
				{Name: "customer_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "客户名称"},
				{Name: "customer_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:PERSONAL/ENTERPRISE"},
				{Name: "level_id", Type: "BIGINT", Comment: "等级ID"},
				{Name: "mobile", Type: "VARCHAR", Length: intPtr(32), Comment: "手机号"},
				{Name: "email", Type: "VARCHAR", Length: intPtr(256), Comment: "邮箱"},
				{Name: "register_time", Type: "DATETIME", Nullable: false, Comment: "注册时间"},
			},
			FKs: []fkDef{{Col: "level_id", RefTable: "customer_level", RefCol: "id"}},
		},
		"customer_address": {
			Domain: "CRM", Comment: "客户地址",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "地址ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "province", Type: "VARCHAR", Length: intPtr(64), Comment: "省"},
				{Name: "city", Type: "VARCHAR", Length: intPtr(64), Comment: "市"},
				{Name: "district", Type: "VARCHAR", Length: intPtr(64), Comment: "区"},
				{Name: "detail", Type: "VARCHAR", Length: intPtr(512), Comment: "详细地址"},
				{Name: "is_default", Type: "BOOLEAN", Comment: "是否默认"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}},
		},
		"customer_level": {
			Domain: "CRM", Comment: "客户等级",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "等级ID"},
				{Name: "level_name", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "等级名称"},
				{Name: "min_consumption", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "最低消费门槛"},
				{Name: "discount_rate", Type: "DECIMAL", Length: intPtr(4), Scale: intPtr(2), Comment: "折扣率"},
			},
		},
		"account": {
			Domain: "CRM", Comment: "企业账户",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "账户ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "account_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "账户名称"},
				{Name: "industry", Type: "VARCHAR", Length: intPtr(128), Comment: "行业"},
				{Name: "annual_revenue", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "年收入"},
				{Name: "employee_count", Type: "INT", Comment: "员工人数"},
				{Name: "owner_id", Type: "BIGINT", Comment: "负责人"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "owner_id", RefTable: "employee", RefCol: "id"}},
		},
		"contact": {
			Domain: "CRM", Comment: "联系人",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "联系人ID"},
				{Name: "account_id", Type: "BIGINT", Nullable: false, Comment: "账户ID"},
				{Name: "contact_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "联系人姓名"},
				{Name: "title", Type: "VARCHAR", Length: intPtr(128), Comment: "职位"},
				{Name: "phone", Type: "VARCHAR", Length: intPtr(32), Comment: "电话"},
				{Name: "email", Type: "VARCHAR", Length: intPtr(256), Comment: "邮箱"},
				{Name: "is_primary", Type: "BOOLEAN", Comment: "是否主要联系人"},
			},
			FKs: []fkDef{{Col: "account_id", RefTable: "account", RefCol: "id"}},
		},
		"lead": {
			Domain: "CRM", Comment: "销售线索",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "线索ID"},
				{Name: "lead_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "线索名称"},
				{Name: "company", Type: "VARCHAR", Length: intPtr(256), Comment: "公司名称"},
				{Name: "source", Type: "VARCHAR", Length: intPtr(64), Comment: "来源"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:NEW/QUALIFIED/CONVERTED"},
				{Name: "owner_id", Type: "BIGINT", Comment: "负责人ID"},
				{Name: "created_at", Type: "DATETIME", Nullable: false, Comment: "创建时间"},
			},
			FKs: []fkDef{{Col: "owner_id", RefTable: "employee", RefCol: "id"}},
		},
		"opportunity": {
			Domain: "CRM", Comment: "商机",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "商机ID"},
				{Name: "opportunity_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "商机名称"},
				{Name: "account_id", Type: "BIGINT", Nullable: false, Comment: "账户ID"},
				{Name: "amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "金额"},
				{Name: "stage_id", Type: "BIGINT", Nullable: false, Comment: "阶段ID"},
				{Name: "probability", Type: "INT", Comment: "赢单概率%"},
				{Name: "close_date", Type: "DATE", Comment: "预计关闭日期"},
				{Name: "owner_id", Type: "BIGINT", Comment: "负责人ID"},
			},
			FKs: []fkDef{{Col: "account_id", RefTable: "account", RefCol: "id"}, {Col: "stage_id", RefTable: "opportunity_stage", RefCol: "id"}, {Col: "owner_id", RefTable: "employee", RefCol: "id"}},
		},
		"opportunity_stage": {
			Domain: "CRM", Comment: "商机阶段",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "阶段ID"},
				{Name: "stage_name", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "阶段名称"},
				{Name: "sort_order", Type: "INT", Comment: "排序"},
				{Name: "is_won", Type: "BOOLEAN", Comment: "是否赢单阶段"},
			},
		},
		"contract": {
			Domain: "CRM", Comment: "合同",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "合同ID"},
				{Name: "contract_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "合同编号"},
				{Name: "opportunity_id", Type: "BIGINT", Comment: "商机ID"},
				{Name: "account_id", Type: "BIGINT", Nullable: false, Comment: "账户ID"},
				{Name: "contract_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "合同金额"},
				{Name: "start_date", Type: "DATE", Comment: "开始日期"},
				{Name: "end_date", Type: "DATE", Comment: "结束日期"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
			},
			FKs: []fkDef{{Col: "opportunity_id", RefTable: "opportunity", RefCol: "id"}, {Col: "account_id", RefTable: "account", RefCol: "id"}},
		},
		"service_ticket": {
			Domain: "CRM", Comment: "服务工单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "工单ID"},
				{Name: "ticket_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "工单号"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "subject", Type: "VARCHAR", Length: intPtr(512), Nullable: false, Comment: "主题"},
				{Name: "priority", Type: "VARCHAR", Length: intPtr(32), Comment: "优先级"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
				{Name: "assignee_id", Type: "BIGINT", Comment: "处理人"},
				{Name: "created_at", Type: "DATETIME", Nullable: false, Comment: "创建时间"},
				{Name: "resolved_at", Type: "DATETIME", Comment: "解决时间"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "assignee_id", RefTable: "employee", RefCol: "id"}},
		},
		"activity": {
			Domain: "CRM", Comment: "活动记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "活动ID"},
				{Name: "related_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "关联类型:LEAD/OPPORTUNITY/ACCOUNT"},
				{Name: "related_id", Type: "BIGINT", Nullable: false, Comment: "关联ID"},
				{Name: "activity_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "类型:CALL/EMAIL/MEETING"},
				{Name: "subject", Type: "VARCHAR", Length: intPtr(512), Comment: "主题"},
				{Name: "owner_id", Type: "BIGINT", Comment: "执行人"},
				{Name: "activity_time", Type: "DATETIME", Comment: "活动时间"},
			},
		},

		// ===== FINANCE (15 tables) =====
		"gl_account": {
			Domain: "财务", Comment: "总账科目",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "科目ID"},
				{Name: "account_code", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "科目编码"},
				{Name: "account_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "科目名称"},
				{Name: "parent_id", Type: "BIGINT", Comment: "上级科目ID"},
				{Name: "account_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "类型:ASSET/LIABILITY/EQUITY/REVENUE/EXPENSE"},
				{Name: "dc_flag", Type: "VARCHAR", Length: intPtr(4), Nullable: false, Comment: "借贷方向:D/C"},
				{Name: "is_leaf", Type: "BOOLEAN", Comment: "是否叶子科目"},
			},
			FKs: []fkDef{{Col: "parent_id", RefTable: "gl_account", RefCol: "id"}},
		},
		"gl_journal": {
			Domain: "财务", Comment: "总账凭证",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "凭证ID"},
				{Name: "journal_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "凭证号"},
				{Name: "fiscal_period_id", Type: "BIGINT", Nullable: false, Comment: "会计期间ID"},
				{Name: "journal_date", Type: "DATE", Nullable: false, Comment: "凭证日期"},
				{Name: "description", Type: "VARCHAR", Length: intPtr(512), Comment: "摘要"},
				{Name: "total_debit", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "借方合计"},
				{Name: "total_credit", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "贷方合计"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:DRAFT/POSTED"},
			},
			FKs: []fkDef{{Col: "fiscal_period_id", RefTable: "fiscal_period", RefCol: "id"}},
		},
		"gl_journal_entry": {
			Domain: "财务", Comment: "凭证分录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "分录ID"},
				{Name: "journal_id", Type: "BIGINT", Nullable: false, Comment: "凭证ID"},
				{Name: "account_id", Type: "BIGINT", Nullable: false, Comment: "科目ID"},
				{Name: "debit_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "借方金额"},
				{Name: "credit_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "贷方金额"},
				{Name: "description", Type: "VARCHAR", Length: intPtr(512), Comment: "分录摘要"},
			},
			FKs: []fkDef{{Col: "journal_id", RefTable: "gl_journal", RefCol: "id"}, {Col: "account_id", RefTable: "gl_account", RefCol: "id"}},
		},
		"fiscal_period": {
			Domain: "财务", Comment: "会计期间",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "期间ID"},
				{Name: "period_name", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "期间名称"},
				{Name: "fiscal_year", Type: "INT", Nullable: false, Comment: "会计年度"},
				{Name: "period_month", Type: "INT", Nullable: false, Comment: "月份"},
				{Name: "start_date", Type: "DATE", Nullable: false, Comment: "开始日期"},
				{Name: "end_date", Type: "DATE", Nullable: false, Comment: "结束日期"},
				{Name: "is_closed", Type: "BOOLEAN", Comment: "是否已关闭"},
			},
		},
		"ap_invoice": {
			Domain: "财务", Comment: "应付发票",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "发票ID"},
				{Name: "invoice_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "发票号"},
				{Name: "supplier_id", Type: "BIGINT", Nullable: false, Comment: "供应商ID"},
				{Name: "invoice_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "发票金额"},
				{Name: "tax_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "税额"},
				{Name: "invoice_date", Type: "DATE", Nullable: false, Comment: "发票日期"},
				{Name: "due_date", Type: "DATE", Comment: "到期日"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
			},
			FKs: []fkDef{{Col: "supplier_id", RefTable: "supplier", RefCol: "id"}},
		},
		"ap_payment": {
			Domain: "财务", Comment: "付款记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "付款ID"},
				{Name: "invoice_id", Type: "BIGINT", Nullable: false, Comment: "发票ID"},
				{Name: "payment_no", Type: "VARCHAR", Length: intPtr(128), Comment: "付款单号"},
				{Name: "payment_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "付款金额"},
				{Name: "payment_date", Type: "DATE", Comment: "付款日期"},
				{Name: "payment_method", Type: "VARCHAR", Length: intPtr(32), Comment: "付款方式"},
			},
			FKs: []fkDef{{Col: "invoice_id", RefTable: "ap_invoice", RefCol: "id"}},
		},
		"ar_invoice": {
			Domain: "财务", Comment: "应收发票",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "发票ID"},
				{Name: "invoice_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "发票号"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "contract_id", Type: "BIGINT", Comment: "合同ID"},
				{Name: "invoice_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "发票金额"},
				{Name: "invoice_date", Type: "DATE", Nullable: false, Comment: "发票日期"},
				{Name: "due_date", Type: "DATE", Comment: "到期日"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "contract_id", RefTable: "contract", RefCol: "id"}},
		},
		"budget": {
			Domain: "财务", Comment: "预算",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "预算ID"},
				{Name: "budget_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "预算名称"},
				{Name: "fiscal_year", Type: "INT", Nullable: false, Comment: "年度"},
				{Name: "department_id", Type: "BIGINT", Comment: "部门ID"},
				{Name: "total_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "预算总额"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "department_id", RefTable: "department", RefCol: "id"}},
		},
		"budget_line": {
			Domain: "财务", Comment: "预算明细",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "明细ID"},
				{Name: "budget_id", Type: "BIGINT", Nullable: false, Comment: "预算ID"},
				{Name: "account_id", Type: "BIGINT", Nullable: false, Comment: "科目ID"},
				{Name: "budget_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "预算金额"},
				{Name: "used_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "已使用金额"},
			},
			FKs: []fkDef{{Col: "budget_id", RefTable: "budget", RefCol: "id"}, {Col: "account_id", RefTable: "gl_account", RefCol: "id"}},
		},
		"expense_report": {
			Domain: "财务", Comment: "费用报销单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "报销单ID"},
				{Name: "report_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "单据号"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "total_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "报销总额"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
				{Name: "submit_date", Type: "DATE", Comment: "提交日期"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},
		"expense_item": {
			Domain: "财务", Comment: "报销明细",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "明细ID"},
				{Name: "report_id", Type: "BIGINT", Nullable: false, Comment: "报销单ID"},
				{Name: "expense_type", Type: "VARCHAR", Length: intPtr(64), Comment: "费用类型"},
				{Name: "amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Nullable: false, Comment: "金额"},
				{Name: "description", Type: "VARCHAR", Length: intPtr(512), Comment: "说明"},
			},
			FKs: []fkDef{{Col: "report_id", RefTable: "expense_report", RefCol: "id"}},
		},
		"fixed_asset": {
			Domain: "财务", Comment: "固定资产",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "资产ID"},
				{Name: "asset_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "资产编码"},
				{Name: "asset_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "资产名称"},
				{Name: "category_id", Type: "BIGINT", Comment: "资产类别ID"},
				{Name: "purchase_date", Type: "DATE", Comment: "购置日期"},
				{Name: "original_value", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "原值"},
				{Name: "residual_value", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "残值"},
				{Name: "useful_life", Type: "INT", Comment: "使用年限"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态:IN_USE/DISPOSED"},
			},
		},
		"asset_depreciation": {
			Domain: "财务", Comment: "折旧记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "折旧记录ID"},
				{Name: "asset_id", Type: "BIGINT", Nullable: false, Comment: "资产ID"},
				{Name: "fiscal_period_id", Type: "BIGINT", Nullable: false, Comment: "会计期间ID"},
				{Name: "depreciation_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Nullable: false, Comment: "折旧额"},
				{Name: "accumulated_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "累计折旧"},
			},
			FKs: []fkDef{{Col: "asset_id", RefTable: "fixed_asset", RefCol: "id"}, {Col: "fiscal_period_id", RefTable: "fiscal_period", RefCol: "id"}},
		},

		// ===== HR (15 tables) =====
		"employee": {
			Domain: "HR", Comment: "员工",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "员工ID"},
				{Name: "employee_no", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "工号"},
				{Name: "employee_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "姓名"},
				{Name: "gender", Type: "VARCHAR", Length: intPtr(8), Comment: "性别"},
				{Name: "department_id", Type: "BIGINT", Nullable: false, Comment: "部门ID"},
				{Name: "email", Type: "VARCHAR", Length: intPtr(256), Comment: "邮箱"},
				{Name: "mobile", Type: "VARCHAR", Length: intPtr(32), Comment: "手机号"},
				{Name: "hire_date", Type: "DATE", Comment: "入职日期"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:ACTIVE/RESIGNED"},
			},
			FKs: []fkDef{{Col: "department_id", RefTable: "department", RefCol: "id"}},
		},
		"department": {
			Domain: "HR", Comment: "部门",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "部门ID"},
				{Name: "parent_id", Type: "BIGINT", Comment: "上级部门ID"},
				{Name: "dept_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "部门名称"},
				{Name: "dept_code", Type: "VARCHAR", Length: intPtr(32), Comment: "部门编码"},
				{Name: "manager_id", Type: "BIGINT", Comment: "部门负责人"},
			},
			FKs: []fkDef{{Col: "parent_id", RefTable: "department", RefCol: "id"}, {Col: "manager_id", RefTable: "employee", RefCol: "id"}},
		},
		"position": {
			Domain: "HR", Comment: "岗位",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "岗位ID"},
				{Name: "position_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "岗位名称"},
				{Name: "department_id", Type: "BIGINT", Nullable: false, Comment: "所属部门"},
				{Name: "position_level", Type: "VARCHAR", Length: intPtr(32), Comment: "职级"},
			},
			FKs: []fkDef{{Col: "department_id", RefTable: "department", RefCol: "id"}},
		},
		"employee_position": {
			Domain: "HR", Comment: "员工任职",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "任职ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "position_id", Type: "BIGINT", Nullable: false, Comment: "岗位ID"},
				{Name: "start_date", Type: "DATE", Nullable: false, Comment: "开始日期"},
				{Name: "end_date", Type: "DATE", Comment: "结束日期"},
				{Name: "is_current", Type: "BOOLEAN", Comment: "是否当前任职"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}, {Col: "position_id", RefTable: "position", RefCol: "id"}},
		},
		"salary": {
			Domain: "HR", Comment: "薪资",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "薪资ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "salary_month", Type: "VARCHAR", Length: intPtr(8), Nullable: false, Comment: "薪资月份"},
				{Name: "basic_salary", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "基本工资"},
				{Name: "bonus", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "奖金"},
				{Name: "deduction", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "扣款"},
				{Name: "net_salary", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "实发工资"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},
		"attendance": {
			Domain: "HR", Comment: "考勤",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "考勤ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "attendance_date", Type: "DATE", Nullable: false, Comment: "日期"},
				{Name: "check_in_time", Type: "DATETIME", Comment: "签到时间"},
				{Name: "check_out_time", Type: "DATETIME", Comment: "签退时间"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态:NORMAL/LATE/EARLY/ABSENT"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},
		"leave_request": {
			Domain: "HR", Comment: "请假申请",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "请假ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "leave_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "类型:ANNUAL/SICK/PERSONAL"},
				{Name: "start_date", Type: "DATE", Nullable: false, Comment: "开始日期"},
				{Name: "end_date", Type: "DATE", Nullable: false, Comment: "结束日期"},
				{Name: "reason", Type: "VARCHAR", Length: intPtr(512), Comment: "请假原因"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:PENDING/APPROVED/REJECTED"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},
		"job_posting": {
			Domain: "HR", Comment: "招聘职位",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "职位ID"},
				{Name: "job_title", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "职位名称"},
				{Name: "department_id", Type: "BIGINT", Comment: "部门ID"},
				{Name: "headcount", Type: "INT", Comment: "招聘人数"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:OPEN/CLOSED"},
			},
		},
		"candidate": {
			Domain: "HR", Comment: "候选人",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "候选人ID"},
				{Name: "posting_id", Type: "BIGINT", Comment: "应聘职位ID"},
				{Name: "candidate_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "姓名"},
				{Name: "phone", Type: "VARCHAR", Length: intPtr(32), Comment: "电话"},
				{Name: "email", Type: "VARCHAR", Length: intPtr(256), Comment: "邮箱"},
				{Name: "resume_url", Type: "VARCHAR", Length: intPtr(512), Comment: "简历链接"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态:SCREENING/INTERVIEW/OFFER/HIRED/REJECTED"},
			},
			FKs: []fkDef{{Col: "posting_id", RefTable: "job_posting", RefCol: "id"}},
		},
		"interview": {
			Domain: "HR", Comment: "面试记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "面试ID"},
				{Name: "candidate_id", Type: "BIGINT", Nullable: false, Comment: "候选人ID"},
				{Name: "interviewer_id", Type: "BIGINT", Nullable: false, Comment: "面试官ID"},
				{Name: "interview_time", Type: "DATETIME", Comment: "面试时间"},
				{Name: "result", Type: "VARCHAR", Length: intPtr(32), Comment: "结果:PASS/FAIL/PENDING"},
				{Name: "feedback", Type: "TEXT", Comment: "面试评价"},
			},
			FKs: []fkDef{{Col: "candidate_id", RefTable: "candidate", RefCol: "id"}, {Col: "interviewer_id", RefTable: "employee", RefCol: "id"}},
		},
		"training_course": {
			Domain: "HR", Comment: "培训课程",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "课程ID"},
				{Name: "course_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "课程名称"},
				{Name: "trainer", Type: "VARCHAR", Length: intPtr(128), Comment: "培训师"},
				{Name: "duration_hours", Type: "INT", Comment: "时长(小时)"},
				{Name: "start_date", Type: "DATE", Comment: "开始日期"},
			},
		},
		"training_enrollment": {
			Domain: "HR", Comment: "培训报名",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "报名ID"},
				{Name: "course_id", Type: "BIGINT", Nullable: false, Comment: "课程ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "enrollment_date", Type: "DATE", Comment: "报名日期"},
				{Name: "completion_status", Type: "VARCHAR", Length: intPtr(32), Comment: "完成状态"},
			},
			FKs: []fkDef{{Col: "course_id", RefTable: "training_course", RefCol: "id"}, {Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},

		// ===== LOGISTICS (15 tables) =====
		"warehouse": {
			Domain: "物流", Comment: "仓库",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "仓库ID"},
				{Name: "warehouse_code", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "仓库编码"},
				{Name: "warehouse_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "仓库名称"},
				{Name: "address", Type: "VARCHAR", Length: intPtr(512), Comment: "地址"},
				{Name: "manager_id", Type: "BIGINT", Comment: "负责人"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
		},
		"warehouse_zone": {
			Domain: "物流", Comment: "库区",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "库区ID"},
				{Name: "warehouse_id", Type: "BIGINT", Nullable: false, Comment: "仓库ID"},
				{Name: "zone_code", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "库区编码"},
				{Name: "zone_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:RECEIVING/STORAGE/PICKING/SHIPPING"},
			},
			FKs: []fkDef{{Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},
		"stock": {
			Domain: "物流", Comment: "库存",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "库存ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "warehouse_id", Type: "BIGINT", Nullable: false, Comment: "仓库ID"},
				{Name: "zone_id", Type: "BIGINT", Comment: "库区ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "数量"},
				{Name: "available_quantity", Type: "INT", Comment: "可用数量"},
			},
			FKs: []fkDef{{Col: "sku_id", RefTable: "product_sku", RefCol: "id"}, {Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}, {Col: "zone_id", RefTable: "warehouse_zone", RefCol: "id"}},
		},
		"stock_movement": {
			Domain: "物流", Comment: "库存移动",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "移动ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "from_zone_id", Type: "BIGINT", Comment: "来源库区"},
				{Name: "to_zone_id", Type: "BIGINT", Comment: "目标库区"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "移动数量"},
				{Name: "movement_time", Type: "DATETIME", Nullable: false, Comment: "移动时间"},
			},
			FKs: []fkDef{{Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"shipment": {
			Domain: "物流", Comment: "发货单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "发货单ID"},
				{Name: "shipment_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "发货单号"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "warehouse_id", Type: "BIGINT", Comment: "发货仓库"},
				{Name: "carrier_id", Type: "BIGINT", Comment: "承运商ID"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
				{Name: "ship_time", Type: "DATETIME", Comment: "发货时间"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}, {Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},
		"shipment_item": {
			Domain: "物流", Comment: "发货明细",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "明细ID"},
				{Name: "shipment_id", Type: "BIGINT", Nullable: false, Comment: "发货单ID"},
				{Name: "order_item_id", Type: "BIGINT", Comment: "订单明细ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "发货数量"},
			},
			FKs: []fkDef{{Col: "shipment_id", RefTable: "shipment", RefCol: "id"}, {Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"carrier": {
			Domain: "物流", Comment: "承运商",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "承运商ID"},
				{Name: "carrier_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "承运商名称"},
				{Name: "carrier_code", Type: "VARCHAR", Length: intPtr(64), Comment: "编码"},
				{Name: "contact", Type: "VARCHAR", Length: intPtr(128), Comment: "联系方式"},
			},
		},
		"purchase_order": {
			Domain: "物流", Comment: "采购订单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "采购单ID"},
				{Name: "po_no", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "采购单号"},
				{Name: "supplier_id", Type: "BIGINT", Nullable: false, Comment: "供应商ID"},
				{Name: "warehouse_id", Type: "BIGINT", Comment: "收货仓库"},
				{Name: "total_amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "总金额"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "状态"},
				{Name: "order_date", Type: "DATE", Nullable: false, Comment: "下单日期"},
			},
			FKs: []fkDef{{Col: "supplier_id", RefTable: "supplier", RefCol: "id"}, {Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},
		"purchase_order_item": {
			Domain: "物流", Comment: "采购明细",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "明细ID"},
				{Name: "po_id", Type: "BIGINT", Nullable: false, Comment: "采购单ID"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "数量"},
				{Name: "unit_price", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "单价"},
			},
			FKs: []fkDef{{Col: "po_id", RefTable: "purchase_order", RefCol: "id"}, {Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"vendor": {
			Domain: "物流", Comment: "供应商评价",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "评价ID"},
				{Name: "supplier_id", Type: "BIGINT", Nullable: false, Comment: "供应商ID"},
				{Name: "quality_score", Type: "INT", Comment: "质量评分"},
				{Name: "delivery_score", Type: "INT", Comment: "交付评分"},
				{Name: "evaluate_date", Type: "DATE", Comment: "评价日期"},
			},
			FKs: []fkDef{{Col: "supplier_id", RefTable: "supplier", RefCol: "id"}},
		},
		"receiving": {
			Domain: "物流", Comment: "收货单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "收货ID"},
				{Name: "po_id", Type: "BIGINT", Nullable: false, Comment: "采购单ID"},
				{Name: "warehouse_id", Type: "BIGINT", Comment: "仓库ID"},
				{Name: "receive_date", Type: "DATE", Comment: "收货日期"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "po_id", RefTable: "purchase_order", RefCol: "id"}, {Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},
		"quality_inspection": {
			Domain: "物流", Comment: "质检记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "质检ID"},
				{Name: "receiving_id", Type: "BIGINT", Comment: "收货ID"},
				{Name: "inspector_id", Type: "BIGINT", Comment: "质检员ID"},
				{Name: "result", Type: "VARCHAR", Length: intPtr(32), Comment: "结果:PASS/FAIL"},
				{Name: "defect_count", Type: "INT", Comment: "不良品数"},
			},
			FKs: []fkDef{{Col: "receiving_id", RefTable: "receiving", RefCol: "id"}, {Col: "inspector_id", RefTable: "employee", RefCol: "id"}},
		},
		"return_order": {
			Domain: "物流", Comment: "退货单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "退货ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "原订单ID"},
				{Name: "return_reason", Type: "VARCHAR", Length: intPtr(256), Comment: "退货原因"},
				{Name: "return_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "退款金额"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}},
		},
		"inventory_transfer": {
			Domain: "物流", Comment: "库存调拨",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "调拨ID"},
				{Name: "from_warehouse_id", Type: "BIGINT", Nullable: false, Comment: "来源仓库"},
				{Name: "to_warehouse_id", Type: "BIGINT", Nullable: false, Comment: "目标仓库"},
				{Name: "sku_id", Type: "BIGINT", Nullable: false, Comment: "SKU ID"},
				{Name: "quantity", Type: "INT", Nullable: false, Comment: "数量"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "from_warehouse_id", RefTable: "warehouse", RefCol: "id"}, {Col: "to_warehouse_id", RefTable: "warehouse", RefCol: "id"}, {Col: "sku_id", RefTable: "product_sku", RefCol: "id"}},
		},
		"cycle_count": {
			Domain: "物流", Comment: "盘点记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "盘点ID"},
				{Name: "warehouse_id", Type: "BIGINT", Comment: "仓库ID"},
				{Name: "count_date", Type: "DATE", Comment: "盘点日期"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "warehouse_id", RefTable: "warehouse", RefCol: "id"}},
		},

		// ===== E-COMMERCE EXT (追加) =====
		"product_image": {
			Domain: "电商", Comment: "商品图片",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "图片ID"},
				{Name: "product_id", Type: "BIGINT", Nullable: false, Comment: "商品ID"},
				{Name: "image_url", Type: "VARCHAR", Length: intPtr(512), Comment: "图片URL"},
				{Name: "is_primary", Type: "BOOLEAN", Comment: "是否主图"},
				{Name: "sort_order", Type: "INT", Comment: "排序"},
			},
			FKs: []fkDef{{Col: "product_id", RefTable: "product", RefCol: "id"}},
		},
		"product_tag": {
			Domain: "电商", Comment: "商品标签",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "标签ID"},
				{Name: "tag_name", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "标签名"},
				{Name: "tag_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型"},
			},
		},
		"wishlist": {
			Domain: "电商", Comment: "收藏夹",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "收藏ID"},
				{Name: "customer_id", Type: "BIGINT", Nullable: false, Comment: "客户ID"},
				{Name: "product_id", Type: "BIGINT", Nullable: false, Comment: "商品ID"},
				{Name: "added_at", Type: "DATETIME", Comment: "添加时间"},
			},
			FKs: []fkDef{{Col: "customer_id", RefTable: "customer", RefCol: "id"}, {Col: "product_id", RefTable: "product", RefCol: "id"}},
		},
		"promotion": {
			Domain: "电商", Comment: "促销活动",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "活动ID"},
				{Name: "promo_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "活动名称"},
				{Name: "promo_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:FULL_REDUCE/DISCOUNT"},
				{Name: "start_time", Type: "DATETIME", Comment: "开始时间"},
				{Name: "end_time", Type: "DATETIME", Comment: "结束时间"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
		},
		"refund": {
			Domain: "电商", Comment: "退款记录",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "退款ID"},
				{Name: "order_id", Type: "BIGINT", Nullable: false, Comment: "订单ID"},
				{Name: "refund_amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "退款金额"},
				{Name: "refund_reason", Type: "VARCHAR", Length: intPtr(256), Comment: "退款原因"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "order_id", RefTable: "order", RefCol: "id"}},
		},
		"shopping_session": {
			Domain: "电商", Comment: "浏览会话",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "会话ID"},
				{Name: "customer_id", Type: "BIGINT", Comment: "客户ID"},
				{Name: "session_token", Type: "VARCHAR", Length: intPtr(256), Comment: "会话Token"},
				{Name: "start_time", Type: "DATETIME", Comment: "开始时间"},
				{Name: "end_time", Type: "DATETIME", Comment: "结束时间"},
			},
		},

		// ===== CRM EXT =====
		"campaign": {
			Domain: "CRM", Comment: "营销活动",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "活动ID"},
				{Name: "campaign_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "活动名称"},
				{Name: "campaign_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:EMAIL/SMS/SOCIAL"},
				{Name: "budget", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "预算"},
				{Name: "start_date", Type: "DATE", Comment: "开始日期"},
				{Name: "end_date", Type: "DATE", Comment: "结束日期"},
			},
		},
		"email_template": {
			Domain: "CRM", Comment: "邮件模板",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "模板ID"},
				{Name: "template_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "模板名称"},
				{Name: "subject", Type: "VARCHAR", Length: intPtr(512), Comment: "邮件主题"},
				{Name: "body_html", Type: "TEXT", Comment: "HTML内容"},
			},
		},
		"competitor": {
			Domain: "CRM", Comment: "竞争对手",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "对手ID"},
				{Name: "competitor_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "名称"},
				{Name: "strength", Type: "TEXT", Comment: "优势"},
				{Name: "weakness", Type: "TEXT", Comment: "劣势"},
			},
		},

		// ===== FINANCE EXT =====
		"tax_rate": {
			Domain: "财务", Comment: "税率表",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "税率ID"},
				{Name: "tax_code", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "税码"},
				{Name: "tax_name", Type: "VARCHAR", Length: intPtr(128), Comment: "税种名称"},
				{Name: "rate", Type: "DECIMAL", Length: intPtr(5), Scale: intPtr(2), Comment: "税率%"},
				{Name: "is_active", Type: "BOOLEAN", Comment: "是否有效"},
			},
		},
		"bank_account": {
			Domain: "财务", Comment: "银行账户",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "账户ID"},
				{Name: "bank_name", Type: "VARCHAR", Length: intPtr(256), Comment: "银行名称"},
				{Name: "account_no", Type: "VARCHAR", Length: intPtr(64), Comment: "账号"},
				{Name: "currency", Type: "VARCHAR", Length: intPtr(8), Comment: "币种"},
				{Name: "balance", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "余额"},
			},
		},
		"cash_flow": {
			Domain: "财务", Comment: "现金流",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "记录ID"},
				{Name: "flow_type", Type: "VARCHAR", Length: intPtr(32), Nullable: false, Comment: "类型:INFLOW/OUTFLOW"},
				{Name: "amount", Type: "DECIMAL", Length: intPtr(16), Scale: intPtr(2), Comment: "金额"},
				{Name: "flow_date", Type: "DATE", Comment: "日期"},
				{Name: "description", Type: "VARCHAR", Length: intPtr(512), Comment: "说明"},
			},
		},
		"payment_method": {
			Domain: "财务", Comment: "支付方式",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "方式ID"},
				{Name: "method_name", Type: "VARCHAR", Length: intPtr(64), Nullable: false, Comment: "方式名称"},
				{Name: "method_code", Type: "VARCHAR", Length: intPtr(32), Comment: "编码"},
				{Name: "is_online", Type: "BOOLEAN", Comment: "是否线上支付"},
				{Name: "fee_rate", Type: "DECIMAL", Length: intPtr(5), Scale: intPtr(2), Comment: "手续费率"},
			},
		},

		// ===== HR EXT =====
		"performance_review": {
			Domain: "HR", Comment: "绩效评估",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "评估ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "review_period", Type: "VARCHAR", Length: intPtr(16), Comment: "评估周期"},
				{Name: "score", Type: "DECIMAL", Length: intPtr(4), Scale: intPtr(1), Comment: "评分"},
				{Name: "reviewer_id", Type: "BIGINT", Comment: "评估人"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}, {Col: "reviewer_id", RefTable: "employee", RefCol: "id"}},
		},
		"skill": {
			Domain: "HR", Comment: "技能库",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "技能ID"},
				{Name: "skill_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "技能名称"},
				{Name: "skill_category", Type: "VARCHAR", Length: intPtr(64), Comment: "技能分类"},
			},
		},
		"employee_skill": {
			Domain: "HR", Comment: "员工技能",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "记录ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "skill_id", Type: "BIGINT", Nullable: false, Comment: "技能ID"},
				{Name: "proficiency", Type: "VARCHAR", Length: intPtr(32), Comment: "熟练度"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}, {Col: "skill_id", RefTable: "skill", RefCol: "id"}},
		},
		"payroll": {
			Domain: "HR", Comment: "工资单",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "工资单ID"},
				{Name: "employee_id", Type: "BIGINT", Nullable: false, Comment: "员工ID"},
				{Name: "pay_period", Type: "VARCHAR", Length: intPtr(16), Comment: "发放周期"},
				{Name: "gross_pay", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "应发工资"},
				{Name: "net_pay", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "实发工资"},
				{Name: "pay_date", Type: "DATE", Comment: "发放日期"},
			},
			FKs: []fkDef{{Col: "employee_id", RefTable: "employee", RefCol: "id"}},
		},
		"benefit": {
			Domain: "HR", Comment: "福利",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "福利ID"},
				{Name: "benefit_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "福利名称"},
				{Name: "benefit_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:INSURANCE/ALLOWANCE"},
				{Name: "amount", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "金额"},
			},
		},

		// ===== ANALYTICS (15 tables) =====
		"report_config": {
			Domain: "分析", Comment: "报表配置",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "报表ID"},
				{Name: "report_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "报表名称"},
				{Name: "report_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:TABLE/CHART/DASHBOARD"},
				{Name: "query_sql", Type: "TEXT", Comment: "查询SQL"},
				{Name: "created_by", Type: "BIGINT", Comment: "创建人"},
			},
		},
		"report_schedule": {
			Domain: "分析", Comment: "报表调度",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "调度ID"},
				{Name: "report_id", Type: "BIGINT", Nullable: false, Comment: "报表ID"},
				{Name: "cron_expr", Type: "VARCHAR", Length: intPtr(64), Comment: "Cron表达式"},
				{Name: "recipients", Type: "VARCHAR", Length: intPtr(512), Comment: "接收人"},
				{Name: "is_active", Type: "BOOLEAN", Comment: "是否启用"},
			},
			FKs: []fkDef{{Col: "report_id", RefTable: "report_config", RefCol: "id"}},
		},
		"dashboard": {
			Domain: "分析", Comment: "仪表盘",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "仪表盘ID"},
				{Name: "dashboard_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "名称"},
				{Name: "layout_json", Type: "TEXT", Comment: "布局JSON"},
				{Name: "owner_id", Type: "BIGINT", Comment: "拥有者"},
			},
		},
		"dashboard_widget": {
			Domain: "分析", Comment: "仪表盘组件",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "组件ID"},
				{Name: "dashboard_id", Type: "BIGINT", Nullable: false, Comment: "仪表盘ID"},
				{Name: "widget_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型"},
				{Name: "report_id", Type: "BIGINT", Comment: "关联报表"},
				{Name: "position", Type: "INT", Comment: "位置"},
			},
			FKs: []fkDef{{Col: "dashboard_id", RefTable: "dashboard", RefCol: "id"}, {Col: "report_id", RefTable: "report_config", RefCol: "id"}},
		},
		"kpi_definition": {
			Domain: "分析", Comment: "KPI定义",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "KPI ID"},
				{Name: "kpi_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "指标名称"},
				{Name: "kpi_code", Type: "VARCHAR", Length: intPtr(64), Comment: "指标编码"},
				{Name: "unit", Type: "VARCHAR", Length: intPtr(32), Comment: "单位"},
				{Name: "target_value", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "目标值"},
				{Name: "department_id", Type: "BIGINT", Comment: "所属部门"},
			},
		},
		"kpi_value": {
			Domain: "分析", Comment: "KPI数据",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "数据ID"},
				{Name: "kpi_id", Type: "BIGINT", Nullable: false, Comment: "KPI ID"},
				{Name: "actual_value", Type: "DECIMAL", Length: intPtr(12), Scale: intPtr(2), Comment: "实际值"},
				{Name: "record_date", Type: "DATE", Comment: "记录日期"},
			},
			FKs: []fkDef{{Col: "kpi_id", RefTable: "kpi_definition", RefCol: "id"}},
		},
		"data_source": {
			Domain: "分析", Comment: "数据源配置",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "配置ID"},
				{Name: "source_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "数据源名称"},
				{Name: "source_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:MYSQL/HIVE/API"},
				{Name: "connection_json", Type: "TEXT", Comment: "连接配置JSON"},
			},
		},
		"etl_job": {
			Domain: "分析", Comment: "ETL任务",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "任务ID"},
				{Name: "job_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "任务名称"},
				{Name: "source_id", Type: "BIGINT", Comment: "源数据源"},
				{Name: "target_table", Type: "VARCHAR", Length: intPtr(256), Comment: "目标表"},
				{Name: "schedule_cron", Type: "VARCHAR", Length: intPtr(64), Comment: "调度表达式"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "source_id", RefTable: "data_source", RefCol: "id"}},
		},
		"etl_log": {
			Domain: "分析", Comment: "ETL日志",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "日志ID"},
				{Name: "job_id", Type: "BIGINT", Nullable: false, Comment: "任务ID"},
				{Name: "start_time", Type: "DATETIME", Comment: "开始时间"},
				{Name: "end_time", Type: "DATETIME", Comment: "结束时间"},
				{Name: "rows_processed", Type: "INT", Comment: "处理行数"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态:SUCCESS/FAILED"},
				{Name: "error_msg", Type: "TEXT", Comment: "错误信息"},
			},
			FKs: []fkDef{{Col: "job_id", RefTable: "etl_job", RefCol: "id"}},
		},
		"data_lineage": {
			Domain: "分析", Comment: "数据血缘",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "血缘ID"},
				{Name: "source_table", Type: "VARCHAR", Length: intPtr(256), Comment: "源表"},
				{Name: "source_column", Type: "VARCHAR", Length: intPtr(256), Comment: "源列"},
				{Name: "target_table", Type: "VARCHAR", Length: intPtr(256), Comment: "目标表"},
				{Name: "target_column", Type: "VARCHAR", Length: intPtr(256), Comment: "目标列"},
				{Name: "etl_job_id", Type: "BIGINT", Comment: "ETL任务ID"},
			},
			FKs: []fkDef{{Col: "etl_job_id", RefTable: "etl_job", RefCol: "id"}},
		},
		"data_quality_rule": {
			Domain: "分析", Comment: "数据质量规则",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "规则ID"},
				{Name: "rule_name", Type: "VARCHAR", Length: intPtr(256), Nullable: false, Comment: "规则名称"},
				{Name: "rule_type", Type: "VARCHAR", Length: intPtr(32), Comment: "类型:NOT_NULL/UNIQUE/RANGE"},
				{Name: "target_table", Type: "VARCHAR", Length: intPtr(256), Comment: "目标表"},
				{Name: "target_column", Type: "VARCHAR", Length: intPtr(256), Comment: "目标列"},
				{Name: "rule_config", Type: "TEXT", Comment: "规则配置JSON"},
			},
		},
		"data_quality_result": {
			Domain: "分析", Comment: "质量检查结果",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "结果ID"},
				{Name: "rule_id", Type: "BIGINT", Nullable: false, Comment: "规则ID"},
				{Name: "check_time", Type: "DATETIME", Comment: "检查时间"},
				{Name: "pass_count", Type: "INT", Comment: "通过数"},
				{Name: "fail_count", Type: "INT", Comment: "失败数"},
				{Name: "status", Type: "VARCHAR", Length: intPtr(32), Comment: "状态"},
			},
			FKs: []fkDef{{Col: "rule_id", RefTable: "data_quality_rule", RefCol: "id"}},
		},
		"metric_dimension": {
			Domain: "分析", Comment: "指标维度",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "维度ID"},
				{Name: "dimension_name", Type: "VARCHAR", Length: intPtr(128), Nullable: false, Comment: "维度名称"},
				{Name: "dimension_code", Type: "VARCHAR", Length: intPtr(64), Comment: "维度编码"},
			},
		},
		"audit_snapshot": {
			Domain: "分析", Comment: "审计快照",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "快照ID"},
				{Name: "snapshot_time", Type: "DATETIME", Comment: "快照时间"},
				{Name: "table_name", Type: "VARCHAR", Length: intPtr(256), Comment: "表名"},
				{Name: "row_count", Type: "INT", Comment: "行数"},
				{Name: "checksum", Type: "VARCHAR", Length: intPtr(128), Comment: "校验和"},
			},
		},
		"data_dictionary": {
			Domain: "分析", Comment: "数据字典快照",
			Cols: []colDef{
				{Name: "id", Type: "BIGINT", PK: true, Nullable: false, Comment: "字典ID"},
				{Name: "table_name", Type: "VARCHAR", Length: intPtr(256), Comment: "表名"},
				{Name: "column_name", Type: "VARCHAR", Length: intPtr(256), Comment: "列名"},
				{Name: "business_term", Type: "VARCHAR", Length: intPtr(256), Comment: "业务术语"},
				{Name: "data_type", Type: "VARCHAR", Length: intPtr(64), Comment: "数据类型"},
				{Name: "description", Type: "TEXT", Comment: "描述"},
			},
		},
	}
}
