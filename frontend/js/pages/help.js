async function pageHelp() {
  document.title = 'Rosetta - 帮助';
  var html = sidebarHtml('/help');
  html += '<div class="page-header"><h2>📖 使用帮助</h2><div class="page-desc">Rosetta 企业数据治理平台操作指南</div></div>';

  html += '<div class="card" id="help-login">';
  html += '<h3 style="margin-bottom:12px">1. 登录</h3>';
  html += '<p style="color:var(--text-secondary)">使用系统管理员分配的账号登录。登录后进入仪表盘，显示各模块数据概览。点击统计卡片可跳转到对应管理页。</p>';
  html += '</div>';

  html += '<div class="card" id="help-users">';
  html += '<h3 style="margin-bottom:12px">2. 用户管理</h3>';
  html += '<p style="margin-bottom:8px">路径: <code style="background:var(--bg);padding:2px 6px;border-radius:4px">#/users</code></p>';
  html += '<table><thead><tr><th>操作</th><th>说明</th></tr></thead><tbody>';
  html += '<tr><td>新建用户</td><td>点击「+ 新建用户」，填写用户名、密码、显示名，勾选角色</td></tr>';
  html += '<tr><td>编辑用户</td><td>点击「编辑」，修改显示名、邮箱、角色</td></tr>';
  html += '<tr><td>分配角色</td><td>点击「角色」，勾选要分配的角色</td></tr>';
  html += '<tr><td>重置密码</td><td>编辑弹窗中点击「重置密码」，输入新密码</td></tr>';
  html += '<tr><td>启/禁用</td><td>点击「禁用」或「启用」切换状态</td></tr>';
  html += '</tbody></table>';
  html += '<p style="margin-top:12px;font-weight:600">系统预置 4 个角色：</p>';
  html += '<table><thead><tr><th>角色</th><th>权限</th></tr></thead><tbody>';
  html += '<tr><td><span class="badge badge-info">SUPER_ADMIN</span> 超级管理员</td><td>所有权限</td></tr>';
  html += '<tr><td><span class="badge badge-info">GOVERNANCE_ADMIN</span> 数据治理管理员</td><td>全局可读</td></tr>';
  html += '<tr><td><span class="badge badge-info">DATA_DEVELOPER</span> 数据开发</td><td>被授权模型可读写</td></tr>';
  html += '<tr><td><span class="badge badge-info">DATA_ANALYST</span> 数据分析</td><td>被授权模型只读</td></tr>';
  html += '</tbody></table>';
  html += '</div>';

  html += '<div class="card" id="help-instances">';
  html += '<h3 style="margin-bottom:12px">3. 实例管理</h3>';
  html += '<p style="margin-bottom:8px">路径: <code style="background:var(--bg);padding:2px 6px;border-radius:4px">#/instances</code></p>';
  html += '<p style="margin-bottom:8px">管理数据库连接信息，支持 MySQL 和 GaussDB M 两种类型。</p>';
  html += '<table><thead><tr><th>操作</th><th>说明</th></tr></thead><tbody>';
  html += '<tr><td>注册实例</td><td>填写名称、类型、主机、端口、用户名、密码、数据库</td></tr>';
  html += '<tr><td>编辑实例</td><td>修改连接信息</td></tr>';
  html += '<tr><td>删除实例</td><td>确认后删除，关联的 Schema 一并删除</td></tr>';
  html += '<tr><td>管理 Schema</td><td>点击「Schema」，查看已有 Schema，可新建</td></tr>';
  html += '</tbody></table>';
  html += '<p style="margin-top:8px"><strong>Schema 层级</strong>：</p>';
  html += '<p style="color:var(--text-secondary)">';
  html += '<span class="badge badge-info">ODS</span> 贴源层 &nbsp; ';
  html += '<span class="badge badge-info">DWD</span> 明细层 &nbsp; ';
  html += '<span class="badge badge-info">DWS</span> 汇总层 &nbsp; ';
  html += '<span class="badge badge-info">ADS</span> 应用层';
  html += '</p>';
  html += '</div>';

  html += '<div class="card" id="help-dicts">';
  html += '<h3 style="margin-bottom:12px">4. 字典维护</h3>';
  html += '<p style="margin-bottom:8px">路径: <code style="background:var(--bg);padding:2px 6px;border-radius:4px">#/dicts</code></p>';
  html += '<p style="margin-bottom:8px">统一管理业务术语和数据标准。</p>';
  html += '<table><thead><tr><th>操作</th><th>说明</th></tr></thead><tbody>';
  html += '<tr><td>新建字典</td><td>填写名称、编码，选择类型</td></tr>';
  html += '<tr><td>编辑/删除</td><td>在右侧详情区顶部操作</td></tr>';
  html += '<tr><td>添加条目</td><td>选中字典后点击「+ 添加条目」</td></tr>';
  html += '<tr><td>编辑条目</td><td>点击条目行的「编辑」</td></tr>';
  html += '<tr><td>删除条目</td><td>点击条目行的「删除」</td></tr>';
  html += '</tbody></table>';
  html += '<p style="margin-top:12px;font-weight:600">三种字典类型：</p>';
  html += '<table><thead><tr><th>类型</th><th>用途</th><th>示例</th></tr></thead><tbody>';
  html += '<tr><td>STANDARD 标准字典</td><td>业务术语</td><td>用户状态：启用/禁用/注销</td></tr>';
  html += '<tr><td>TYPE_MAPPING 类型映射</td><td>逻辑类型到物理类型映射</td><td>BIGINT → MySQL:BIGINT</td></tr>';
  html += '<tr><td>REFERENCE 参考数据</td><td>系统级枚举</td><td>性别、是否删除</td></tr>';
  html += '</tbody></table>';
  html += '</div>';

  html += '<div class="card" id="help-models">';
  html += '<h3 style="margin-bottom:12px">5. 模型管理</h3>';
  html += '<p style="margin-bottom:8px">路径: <code style="background:var(--bg);padding:2px 6px;border-radius:4px">#/models</code></p>';
  html += '<p style="margin-bottom:12px">这是平台核心功能，定义数据表结构。</p>';

  html += '<h4 style="margin:16px 0 8px 0">5.1 模型列表</h4>';
  html += '<ul style="padding-left:20px;color:var(--text-secondary);line-height:1.8">';
  html += '<li>展示所有逻辑模型，支持按表名搜索</li>';
  html += '<li>点击模型名进入详情编辑</li>';
  html += '<li>点击「DDL」弹窗预览双方言 DDL</li>';
  html += '</ul>';

  html += '<h4 style="margin:16px 0 8px 0">5.2 模型详情 — 4 个 Tab</h4>';
  html += '<table><thead><tr><th>Tab</th><th>功能</th></tr></thead><tbody>';
  html += '<tr><td><strong>字段</strong></td><td>添加/编辑/删除字段。配置：字段名、逻辑类型、长度、精度、非空、主键、默认值、注释</td></tr>';
  html += '<tr><td><strong>索引</strong></td><td>添加/删除索引，支持普通(NORMAL)和唯一(UNIQUE)，配置索引列及排序方向</td></tr>';
  html += '<tr><td><strong>外键</strong></td><td>添加/删除外键约束，指定引用模型ID和引用列</td></tr>';
  html += '<tr><td><strong>DDL 预览</strong></td><td>实时切换 MySQL / GaussDB M 查看生成的建表语句，支持复制</td></tr>';
  html += '</tbody></table>';

  html += '<h4 style="margin:16px 0 8px 0">5.3 DDL 方言差异</h4>';
  html += '<table><thead><tr><th>特性</th><th style="width:40%">MySQL</th><th style="width:40%">GaussDB M</th></tr></thead><tbody>';
  html += '<tr><td>自增主键</td><td><code>AUTO_INCREMENT</code></td><td><code>GENERATED BY DEFAULT AS IDENTITY</code></td></tr>';
  html += '<tr><td>日期时间</td><td><code>DATETIME</code></td><td><code>TIMESTAMP</code></td></tr>';
  html += '<tr><td>大文本</td><td><code>TEXT</code></td><td><code>CLOB</code></td></tr>';
  html += '<tr><td>布尔</td><td><code>TINYINT(1)</code></td><td><code>BOOLEAN</code></td></tr>';
  html += '<tr><td>列注释</td><td>内联 <code>COMMENT \'xxx\'</code></td><td>独立 <code>COMMENT ON COLUMN</code></td></tr>';
  html += '<tr><td>表注释</td><td>建表尾缀</td><td>独立 <code>COMMENT ON TABLE</code></td></tr>';
  html += '<tr><td>引擎</td><td><code>ENGINE=InnoDB</code></td><td>不支持，自动跳过</td></tr>';
  html += '<tr><td>引用符</td><td><code>`backtick`</code></td><td>无</td></tr>';
  html += '</tbody></table>';
  html += '</div>';

  html += '<div class="card" id="help-workflow">';
  html += '<h3 style="margin-bottom:12px">6. 典型操作流程</h3>';

  html += '<h4 style="margin:12px 0 8px 0">新建表并生成 DDL</h4>';
  html += '<ol style="padding-left:20px;color:var(--text-secondary);line-height:1.8">';
  html += '<li><strong>创建模型</strong> — 模型管理 → + 新建模型，输入表名</li>';
  html += '<li><strong>添加字段</strong> — 点击模型名进入详情 → Tab「字段」→ + 添加字段</li>';
  html += '<li><strong>添加索引</strong> — Tab「索引」→ + 添加索引</li>';
  html += '<li><strong>添加外键</strong> — Tab「外键」→ + 添加外键（引用另一个模型）</li>';
  html += '<li><strong>预览 DDL</strong> — Tab「DDL 预览」，切换 MySQL/GaussDB 查看</li>';
  html += '</ol>';

  html += '<h4 style="margin:16px 0 8px 0">反向工程已有表</h4>';
  html += '<ol style="padding-left:20px;color:var(--text-secondary);line-height:1.8">';
  html += '<li><strong>注册实例</strong> — 实例管理 → + 注册实例，填写数据库连接信息</li>';
  html += '<li><strong>管理 Schema</strong> — 为实例创建 Schema</li>';
  html += '<li><strong>导入表</strong> — 通过 API 调用反向工程接口导入</li>';
  html += '</ol>';
  html += '</div>';

  html += '<div class="card" id="help-faq">';
  html += '<h3 style="margin-bottom:12px">7. 常见问题</h3>';
  html += '<dl style="line-height:1.8">';
  html += '<dt style="font-weight:600;margin-top:12px">Q: 登录失败？</dt>';
  html += '<dd style="color:var(--text-secondary);margin-bottom:8px">确保 MySQL 容器正在运行：<code>docker compose ps</code>。确认密码正确。</dd>';
  html += '<dt style="font-weight:600;margin-top:12px">Q: 页面空白？</dt>';
  html += '<dd style="color:var(--text-secondary);margin-bottom:8px">检查浏览器控制台是否有 JS 错误。刷新页面重试。</dd>';
  html += '<dt style="font-weight:600;margin-top:12px">Q: 字段的"可为空"不生效？</dt>';
  html += '<dd style="color:var(--text-secondary);margin-bottom:8px">编辑字段后点击保存，模型详情页会重新加载。</dd>';
  html += '<dt style="font-weight:600;margin-top:12px">Q: 如何切换 MySQL / GaussDB DDL 预览？</dt>';
  html += '<dd style="color:var(--text-secondary);margin-bottom:8px">在模型详情的「DDL 预览」Tab 或列表页的「DDL」弹窗中，点击 Tab 按钮切换。</dd>';
  html += '<dt style="font-weight:600;margin-top:12px">Q: 如何添加 GaussDB 实例？</dt>';
  html += '<dd style="color:var(--text-secondary);margin-bottom:8px">实例类型选择「GaussDB M」，填写 GaussDB 的连接信息。</dd>';
  html += '</dl>';
  html += '</div>';

  html += '</div></div>';
  return html;
}
