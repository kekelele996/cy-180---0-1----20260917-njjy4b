-- 受访者授权表：授权闭环（登记 → 核验 → 撤销）。
-- 同一项目可存在多条记录（撤销后允许重新登记），最新一条决定当前授权状态。

CREATE TABLE IF NOT EXISTS consents (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  project_id BIGINT UNSIGNED NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  note VARCHAR(512) DEFAULT '',
  revoke_reason VARCHAR(512) DEFAULT '',
  registered_by BIGINT UNSIGNED NOT NULL,
  verified_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  revoked_by BIGINT UNSIGNED NOT NULL DEFAULT 0,
  verified_at DATETIME(3) NULL,
  revoked_at DATETIME(3) NULL,
  created_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  INDEX idx_consents_project (project_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
