-- 创建网约车演示司机维表，供明细查询联表与司机过滤场景复用。
CREATE TABLE IF NOT EXISTS drivers (
  driver_id VARCHAR(64) NOT NULL COMMENT '司机编号',
  driver_name VARCHAR(64) NOT NULL COMMENT '司机姓名',
  PRIMARY KEY (driver_id)
) COMMENT='网约车司机维表';

-- 创建网约车演示订单事实表，供聚合、排行、趋势和明细查询统一复用。
CREATE TABLE IF NOT EXISTS trip_orders (
  order_id VARCHAR(64) NOT NULL COMMENT '订单编号',
  city_code VARCHAR(32) NOT NULL COMMENT '城市编码',
  service_type VARCHAR(32) NOT NULL COMMENT '服务类型',
  order_status VARCHAR(32) NOT NULL COMMENT '订单状态',
  called_at DATETIME NOT NULL COMMENT '呼叫时间',
  finished_at DATETIME NULL COMMENT '完单时间',
  is_cancelled TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否取消',
  driver_id VARCHAR(64) NULL COMMENT '司机编号',
  PRIMARY KEY (order_id),
  KEY idx_trip_orders_called_at (called_at),
  KEY idx_trip_orders_city_called_at (city_code, called_at),
  KEY idx_trip_orders_status_called_at (order_status, called_at),
  KEY idx_trip_orders_driver_id (driver_id)
) COMMENT='网约车订单事实表';

