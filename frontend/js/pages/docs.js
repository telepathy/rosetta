async function pageDocs() {
  document.title = 'Rosetta - 文档汇编';
  var html = sidebarHtml('/docs');
  html += '<div class="page-header"><h2>📋 数据库文档汇编</h2><div class="page-desc">所有逻辑模型的完整参考文档，含表备注、字段备注</div></div>';
  html += '<div class="page-toolbar">';
  html += '<button class="btn btn-primary" onclick="fetchAllDocs()">🔄 刷新</button>';
  html += '<div class="flex-spacer"></div>';
  html += '<button class="btn btn-outline" onclick="printDocs()">🖨️ 打印</button>';
  html += '</div>';
  html += '<div id="docs-content"><div style="text-align:center;padding:60px;color:var(--text-secondary)">加载中...</div></div>';
  html += '</div></div>';
  setTimeout(fetchAllDocs, 0);
  return html;
}

async function fetchAllDocs() {
  var container = document.getElementById('docs-content');
  try {
    var data = await api.get('/models?page=1&page_size=200');
    var models = data.data.items;

    if (models.length === 0) {
      container.innerHTML = '<div class="empty-state"><div class="empty-icon">📋</div><p>暂无模型数据</p></div>';
      return;
    }

    var allDetails = [];
    for (var i = 0; i < models.length; i++) {
      try {
        var d = (await api.get('/models/' + models[i].id)).data;
        allDetails.push(d);
      } catch(e) {}
    }

    renderDocs(container, allDetails);
  } catch(e) {
    container.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function renderDocs(container, models) {
  var html = '';
  html += '<div style="margin-bottom:16px;color:var(--text-secondary)">共 <b>' + models.length + '</b> 个模型</div>';

  models.forEach(function(m) {
    html += '<div class="card" style="margin-bottom:20px;page-break-inside:avoid">';

    html += '<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px;padding-bottom:8px;border-bottom:2px solid var(--border)">';
    html += '<div><h3 style="margin:0;font-size:16px">📐 ' + m.table_name + '</h3>';
    html += '<div style="font-size:12px;color:var(--text-secondary);margin-top:4px">' + (m.table_comment || '无表备注') + '</div></div>';
    html += '<div><span class="badge badge-info">' + (m.table_status || 'DRAFT') + '</span> ' +
      '<span class="badge badge-gray">' + (m.source || 'MANUAL') + '</span></div>';
    html += '</div>';

    var cols = m.columns || [];
    if (cols.length > 0) {
      html += '<table><thead><tr><th style="width:40px">#</th><th>字段名</th><th>逻辑类型</th><th style="width:60px">非空</th><th style="width:60px">主键</th><th>默认值</th><th>备注</th></tr></thead><tbody>';
      cols.forEach(function(col) {
        var typeStr = col.logical_type;
        if (col.type_length) { typeStr += '(' + col.type_length; if (col.type_scale) typeStr += ',' + col.type_scale; typeStr += ')'; }
        html += '<tr>';
        html += '<td>' + col.ordinal + '</td>';
        html += '<td><strong>' + col.column_name + '</strong></td>';
        html += '<td><code>' + typeStr + '</code></td>';
        html += '<td>' + (col.nullable ? '' : '<span class="badge badge-warning">NOT NULL</span>') + '</td>';
        html += '<td>' + (col.is_primary_key ? '🔑' : '') + '</td>';
        html += '<td>' + (col.default_value || '-') + '</td>';
        html += '<td style="color:var(--text-secondary)">' + (col.comment || '-') + '</td>';
        html += '</tr>';
      });
      html += '</tbody></table>';
    } else {
      html += '<p style="color:var(--text-secondary);text-align:center;padding:20px">暂无字段定义</p>';
    }

    var indexes = m.indexes || [];
    if (indexes.length > 0) {
      html += '<div style="margin-top:12px;font-weight:600;font-size:13px">索引</div>';
      html += '<table><thead><tr><th>名称</th><th>类型</th><th>列</th></tr></thead><tbody>';
      indexes.forEach(function(idx) {
        var icols = []; try { icols = JSON.parse(idx.columns); } catch(e) {}
        var colStr = icols.map(function(c) { return c.name + (c.order === 'DESC' ? ' DESC' : ''); }).join(', ');
        html += '<tr><td>' + idx.index_name + '</td><td>' + idx.index_type + '</td><td>' + colStr + '</td></tr>';
      });
      html += '</tbody></table>';
    }

    var fks = m.foreign_keys || [];
    if (fks.length > 0) {
      html += '<div style="margin-top:12px;font-weight:600;font-size:13px">外键</div>';
      html += '<table><thead><tr><th>外键名</th><th>列</th><th>引用表</th><th>引用列</th></tr></thead><tbody>';
      fks.forEach(function(fk) {
        html += '<tr><td>' + fk.fk_name + '</td><td>' + fk.column_name + '</td><td><a href="#/models/' + fk.ref_model_id + '" style="color:var(--primary)">' + fk.ref_table_name + '</a></td><td>' + fk.ref_column_name + '</td></tr>';
      });
      html += '</tbody></table>';
    }

    html += '<details style="margin-top:12px"><summary style="cursor:pointer;color:var(--text-secondary);font-size:12px">DDL (MySQL)</summary>';
    html += '<div class="json-view" id="ddldoc-' + m.id + '" style="margin-top:8px;max-height:300px">加载中...</div>';
    html += '</details>';

    html += '</div>';
  });

  container.innerHTML = html;

  models.forEach(function(m) {
    (function(id) {
      api.get('/models/' + id + '/ddl?dialect=MYSQL').then(function(r) {
        var el = document.getElementById('ddldoc-' + id);
        if (el) el.textContent = r.data.ddl;
      });
    })(m.id);
  });
}

function printDocs() {
  window.print();
}
window.fetchAllDocs = fetchAllDocs;
window.printDocs = printDocs;
