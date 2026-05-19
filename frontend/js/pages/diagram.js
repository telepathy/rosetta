var diagramState = { mode: 'view', schemaId: null, dragInfo: null, fkSource: null, panStart: null, panScroll: null };

async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">所见即所得编辑 ｜ 拖拽移动表 ｜ 点击列编辑 ｜ 连线建立外键</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">选择 Schema...</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button id="btn-mode-view" class="btn btn-sm btn-primary" onclick="setMode(\'view\')">👁 浏览</button>';
  html += '<button id="btn-mode-edit" class="btn btn-sm btn-outline" onclick="setMode(\'edit\')">✏️ 编辑</button>';
  html += '<button id="btn-mode-fk" class="btn btn-sm btn-outline" onclick="setMode(\'fk\')">🔗 建外键</button>';
  html += '<button id="btn-add-table" class="btn btn-sm btn-primary hidden" onclick="addTableOnDiagram()">+ 新表</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="zoomDiagram(0.2)">🔍+</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="zoomDiagram(-0.2)">🔍-</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="resetZoom()">重置</button>';
  html += '</div>';
  html += '<div id="diagram-container" style="overflow:auto;border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;min-height:500px;position:relative">';
  html += '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择一个 Schema 查看 ER 图</p></div>';
  html += '</div>';
  html += '</div></div>';
  setTimeout(loadSchemaOptions, 0);
  return html;
}

function setMode(mode) {
  diagramState.mode = mode;
  diagramState.fkSource = null;
  ['view','edit','fk'].forEach(function(m) {
    var btn = document.getElementById('btn-mode-' + m);
    if (btn) btn.className = 'btn btn-sm ' + (m === mode ? 'btn-primary' : 'btn-outline');
  });
  var addBtn = document.getElementById('btn-add-table');
  if (addBtn) addBtn.classList.toggle('hidden', mode === 'view');
  if (diagramState.schemaId) loadDiagram();
}

async function loadSchemaOptions() {
  try {
    var schemas = [];
    var instances = (await api.get('/instances?page=1&page_size=50')).data.items || [];
    for (var i = 0; i < instances.length; i++) {
      try {
        var s = (await api.get('/instances/' + instances[i].id + '/schemas')).data;
        if (Array.isArray(s)) {
          s.forEach(function(sc) { schemas.push({ id: sc.id, name: instances[i].name + ' / ' + sc.schema_name, instanceId: instances[i].id }); });
        }
      } catch(e) {}
    }
    var sel = document.getElementById('diagram-schema');
    sel.innerHTML = '<option value="">选择 Schema...</option>';
    schemas.forEach(function(s) { sel.innerHTML += '<option value="' + s.id + '">' + s.name + '</option>'; });

    if (schemas.length === 0) {
      document.getElementById('diagram-container').innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>没有可用的 Schema</p><p style="font-size:12px;color:var(--text-secondary);margin-top:8px">请先在「实例管理」中创建实例和 Schema，然后在「模型管理」中将模型部署到 Schema</p></div>';
    } else if (schemas.length === 1) {
      sel.value = schemas[0].id;
      loadDiagram();
    }
  } catch(e) {
    document.getElementById('diagram-container').innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;
  diagramState.schemaId = schemaId;

  var container = document.getElementById('diagram-container');
  container.innerHTML = '<div style="text-align:center;padding:60px;color:var(--text-secondary)">加载中...</div>';

  try {
    var er = (await api.get('/schemas/' + schemaId + '/er-diagram')).data;
    if (!er.data || !er.data.tables || er.data.tables.length === 0) {
      container.innerHTML = '<div class="empty-state"><div class="empty-icon">📊</div><p>该 Schema 下没有部署的模型，请先部署模型到此 Schema</p></div>';
      return;
    }
    var tables = er.data.tables;
    var edges = er.data.edges || [];
    var modelDetails = {};
    for (var i = 0; i < tables.length; i++) {
      try {
        var d = (await api.get('/models/' + tables[i].id)).data;
        modelDetails[tables[i].id] = d.data;
      } catch(e) {}
    }
    renderDiagram(container, tables, edges, modelDetails);
  } catch(e) {
    container.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function renderDiagram(container, tables, edges, modelDetails) {
  var cardW = 300, colH = 26, padX = 40, padY = 60;
  var cols = Math.max(1, Math.floor(Math.min(3, Math.ceil(Math.sqrt(tables.length)))));
  var positions = {};
  tables.forEach(function(t, i) {
    var col = i % cols, row = Math.floor(i / cols);
    positions[t.id] = { x: padX + col * (cardW + padX), y: padY + row * 200 };
  });
  var headerH = diagramState.mode === 'view' ? 32 : 42;

  var totalH = 0, totalW = 0;
  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var colCount = detail ? (detail.columns ? detail.columns.length : 0) : 0;
    var h = headerH + colCount * colH + (diagramState.mode !== 'view' ? 30 : 12);
    var p = positions[t.id]; totalH = Math.max(totalH, p.y + h + padY);
  });
  totalW = cols * (cardW + padX) + padX;

  var svgNS = 'http://www.w3.org/2000/svg';
  var svg = document.createElementNS(svgNS, 'svg');
  svg.setAttribute('width', Math.max(totalW, 1200)); svg.setAttribute('height', Math.max(totalH, 600));
  svg.setAttribute('id', 'diagram-svg'); svg.style.minWidth = Math.max(totalW, 1200) + 'px';

  var defs = document.createElementNS(svgNS, 'defs');
  var marker = document.createElementNS(svgNS, 'marker');
  marker.setAttribute('id', 'arrowhead'); marker.setAttribute('markerWidth', '8'); marker.setAttribute('markerHeight', '6');
  marker.setAttribute('refX', '8'); marker.setAttribute('refY', '3'); marker.setAttribute('orient', 'auto');
  var ap = document.createElementNS(svgNS, 'path'); ap.setAttribute('d', 'M0,0 L8,3 L0,6 Z'); ap.setAttribute('fill', '#94a3b8');
  marker.appendChild(ap); defs.appendChild(marker); svg.appendChild(defs);

  var edgePaths = buildEdgePaths(edges, positions, modelDetails, headerH, colH);
  edgePaths.forEach(function(ep) { svg.appendChild(ep); });

  tables.forEach(function(t) {
    var detail = modelDetails[t.id];
    var columns = detail ? (detail.columns || []) : [];
    var height = headerH + columns.length * colH + (diagramState.mode !== 'view' ? 30 : 12);
    tableCard(svg, t, detail, columns, positions[t.id], cardW, height, headerH, colH, edges);
  });

  container.innerHTML = '';
  container.appendChild(svg);

  if (diagramState.mode !== 'view') setupDrag(container);
  window._diagramZoom = 1;
  window._diagramContainer = container;
}

function buildEdgePaths(edges, positions, modelDetails, headerH, colH) {
  var svgNS = 'http://www.w3.org/2000/svg';
  var result = [];
  edges.forEach(function(e) {
    if (!positions[e.source] || !positions[e.target]) return;
    var sp = positions[e.source], tp = positions[e.target];
    var srcDetail = modelDetails[e.source], tgtDetail = modelDetails[e.target];
    var srcCols = srcDetail ? (srcDetail.columns || []) : [];
    var tgtCols = tgtDetail ? (tgtDetail.columns || []) : [];
    var si = srcCols.findIndex(function(c) { return c.column_name === e.source_col; });
    var ti = tgtCols.findIndex(function(c) { return c.column_name === e.target_col; });
    var srcY = sp.y + headerH + 4 + (si >= 0 ? si : 0) * colH + colH / 2 + 2;
    var tgtY = tp.y + headerH + 4 + (ti >= 0 ? ti : 0) * colH + colH / 2 + 2;
    var sx = sp.x + 300, tx = tp.x;
    var mx = (sx + tx) / 2;
    var path = document.createElementNS(svgNS, 'path');
    path.setAttribute('d', 'M' + sx + ',' + srcY + ' C' + mx + ',' + srcY + ' ' + mx + ',' + tgtY + ' ' + tx + ',' + tgtY);
    path.setAttribute('fill', 'none'); path.setAttribute('stroke', '#94a3b8'); path.setAttribute('stroke-width', '1.5');
    path.setAttribute('marker-end', 'url(#arrowhead)');
    result.push(path);
    var lb = document.createElementNS(svgNS, 'rect');
    var lx = mx - 32, ly = (srcY + tgtY) / 2 - 8;
    lb.setAttribute('x', lx); lb.setAttribute('y', ly); lb.setAttribute('width', '64'); lb.setAttribute('height', '16');
    lb.setAttribute('rx', '4'); lb.setAttribute('fill', '#f1f5f9'); lb.setAttribute('stroke', '#e2e8f0');
    result.push(lb);
    var lt = document.createElementNS(svgNS, 'text');
    lt.setAttribute('x', mx); lt.setAttribute('y', (srcY + tgtY) / 2 + 3);
    lt.setAttribute('fill', '#64748b'); lt.setAttribute('font-size', '9'); lt.setAttribute('text-anchor', 'middle');
    lt.textContent = (e.source_col.length > 8 ? e.source_col.slice(0,8) : e.source_col) + '→' + (e.target_col.length > 8 ? e.target_col.slice(0,8) : e.target_col);
    result.push(lt);
  });
  return result;
}

function tableCard(svg, t, detail, columns, pos, cardW, height, headerH, colH, edges) {
  var svgNS = 'http://www.w3.org/2000/svg';
  var g = document.createElementNS(svgNS, 'g');
  g.setAttribute('transform', 'translate(' + pos.x + ',' + pos.y + ')');
  g.setAttribute('data-table-id', t.id);
  g.classList.add('er-table-group');

  var bg = svgRect(svgNS, 0, 0, cardW, height, 6, '#fff', '#cbd5e1', 1.5);
  g.appendChild(bg);

  var hbg = svgRect(svgNS, 1, 1, cardW - 2, headerH - 2, 5, '#4f46e5');
  g.appendChild(hbg);
  var hbg2 = svgRect(svgNS, 1, headerH - 4, cardW - 2, 4, 0, '#4f46e5');
  g.appendChild(hbg2);

  var title = svgText(svgNS, 12, 22, '#fff', 13, '600', t.table_name);
  g.appendChild(title);

  if (diagramState.mode !== 'view') {
    var editBtn = svgText(svgNS, cardW - 60, 22, 'rgba(255,255,255,0.8)', 11, 'normal', '✏️编辑');
    editBtn.style.cursor = 'pointer';
    editBtn.addEventListener('click', function(ev) { ev.stopPropagation(); editTableName(t.id); });
    g.appendChild(editBtn);
  }

  columns.forEach(function(col, ci) {
    var cy = headerH + 4 + ci * colH + colH / 2 + 2;

    if (col.is_primary_key) {
      g.appendChild(svgText(svgNS, 8, cy, '#f59e0b', 10, 'normal', '🔑'));
    }

    var isFK = false;
    edges.forEach(function(e) { if (e.source === t.id && e.source_col === col.column_name) isFK = true; });
    var colX = 26;
    if (isFK) {
      g.appendChild(svgText(svgNS, 24, cy, '#4f46e5', 9, '600', 'FK'));
      colX = 46;
    }

    g.appendChild(svgText(svgNS, colX, cy, '#334155', 11, '500', col.column_name));

    var typeStr = col.logical_type + (col.type_length ? '(' + col.type_length + (col.type_scale ? ',' + col.type_scale : '') + ')' : '');
    g.appendChild(svgText(svgNS, cardW - 10, cy, '#94a3b8', 10, 'normal', typeStr, 'end'));

    if (diagramState.mode === 'edit') {
      var editIcon = svgText(svgNS, cardW - 72, cy, '#94a3b8', 10, 'normal', '✎');
      editIcon.style.cursor = 'pointer';
      (function(mid, colId) {
        editIcon.addEventListener('click', function(ev) { ev.stopPropagation(); editColumnOnDiagram(mid, colId); });
      })(t.id, col.id);
      g.appendChild(editIcon);
    }

    if (diagramState.mode === 'fk') {
      var fkDot = document.createElementNS(svgNS, 'circle');
      fkDot.setAttribute('cx', cardW - 80); fkDot.setAttribute('cy', cy);
      fkDot.setAttribute('r', '6'); fkDot.setAttribute('fill', '#e2e8f0'); fkDot.setAttribute('stroke', '#94a3b8');
      fkDot.style.cursor = 'pointer';
      (function(mid, colName) {
        fkDot.addEventListener('click', function(ev) { ev.stopPropagation(); startFK(mid, colName); });
        fkDot.addEventListener('mouseenter', function() { fkDot.setAttribute('fill', '#4f46e5'); });
        fkDot.addEventListener('mouseleave', function() { fkDot.setAttribute('fill', diagramState.fkSource && diagramState.fkSource.modelId === mid && diagramState.fkSource.colName === colName ? '#f59e0b' : '#e2e8f0'); });
      })(t.id, col.column_name);
      g.appendChild(fkDot);
    }
  });

  if (diagramState.mode === 'edit') {
    var addY = headerH + columns.length * colH + 16;
    var addBg = svgRect(svgNS, 4, addY - 4, cardW - 8, 24, 4, '#f8fafc', 'transparent', 0);
    g.appendChild(addBg);
    var addTxt = svgText(svgNS, cardW / 2, addY + 8, '#94a3b8', 11, 'normal', '+ 添加字段', 'middle');
    addTxt.style.cursor = 'pointer';
    (function(mid) {
      addTxt.addEventListener('click', function(ev) { ev.stopPropagation(); addColumnOnDiagram(mid); });
      addBg.addEventListener('click', function(ev) { ev.stopPropagation(); addColumnOnDiagram(mid); });
    })(t.id);
    g.appendChild(addTxt);
  }

  g.addEventListener('click', function() {
    if (diagramState.mode === 'view') router.navigate('/models/' + t.id);
  });

  svg.appendChild(g);
}

var colTypes = ['BIGINT','INT','VARCHAR','DECIMAL','FLOAT','DOUBLE','DATE','DATETIME','TIMESTAMP','TEXT','BOOLEAN','JSON'];

function editColumnOnDiagram(modelId, colId) {
  var promptText = '编辑字段 (格式: 字段名,类型,备注)\n类型选: ' + colTypes.join(',');
  var input = prompt(promptText);
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 1 || !parts[0]) return;

  api.get('/models/' + modelId).then(function(r) {
    var cols = r.data.columns || [];
    var col = cols.find(function(c) { return c.id === colId; });
    if (!col) { toast('字段未找到', 'error'); return; }
    col.column_name = parts[0];
    if (parts[1]) col.logical_type = parts[1].toUpperCase();
    if (parts[2]) col.comment = parts[2];
    api.put('/models/' + modelId + '/columns', { columns: cols }).then(function() {
      toast('字段已更新', 'success'); loadDiagram();
    }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function addColumnOnDiagram(modelId) {
  var input = prompt('添加字段\n格式: 字段名,类型,备注\n类型选: ' + colTypes.join(','));
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (parts.length < 2 || !parts[0] || !parts[1]) { toast('至少需要字段名和类型', 'error'); return; }

  api.get('/models/' + modelId).then(function(r) {
    var cols = r.data.columns || [];
    cols.push({
      ordinal: cols.length + 1, column_name: parts[0],
      logical_type: parts[1].toUpperCase(),
      nullable: true, is_primary_key: false, default_value: '',
      comment: parts[2] || '', type_length: null, type_scale: null
    });
    api.put('/models/' + modelId + '/columns', { columns: cols }).then(function() {
      toast('字段已添加', 'success'); loadDiagram();
    }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function editTableName(modelId) {
  var input = prompt('编辑表名和备注\n格式: 表名,备注');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  api.get('/models/' + modelId).then(function(r) {
    api.put('/models/' + modelId, {
      table_name: parts[0] || r.data.table_name,
      table_comment: parts[1] || '',
      table_status: r.data.table_status
    }).then(function() {
      toast('已更新', 'success'); loadDiagram();
    }).catch(function(e) { toast(e.message, 'error'); });
  });
}

function startFK(modelId, colName) {
  if (!diagramState.fkSource) {
    diagramState.fkSource = { modelId: modelId, colName: colName };
    toast('已选择 ' + colName + ' 作为外键源，请点击目标列', 'info');
  } else {
    completeFK(modelId, colName);
  }
}

function completeFK(modelId, colName) {
  var src = diagramState.fkSource;
  diagramState.fkSource = null;
  if (src.modelId === modelId && src.colName === colName) { toast('不能引用自身', 'error'); return; }

  var fkName = 'fk_' + src.colName;
  api.post('/models/' + src.modelId + '/foreign-keys', {
    fk_name: fkName, column_name: src.colName,
    ref_model_id: modelId, ref_column_name: colName
  }).then(function() {
    toast('外键 ' + fkName + ' 已创建', 'success'); loadDiagram();
  }).catch(function(e) { toast(e.message, 'error'); });
}

function addTableOnDiagram() {
  var input = prompt('新建模型\n格式: 表名,备注');
  if (!input) return;
  var parts = input.split(',').map(function(s) { return s.trim(); });
  if (!parts[0]) { toast('请输入表名', 'error'); return; }
  api.post('/models', { table_name: parts[0], table_comment: parts[1] || '' }).then(function(r) {
    var modelId = r.data.id;
    api.post('/models/' + modelId + '/deploy', { schema_id: parseInt(diagramState.schemaId), dialect: 'MYSQL' }).then(function() {
      toast('模型已创建并部署', 'success'); loadDiagram();
    }).catch(function() {
      toast('模型已创建，请手动部署到 Schema', 'success'); loadDiagram();
    });
  }).catch(function(e) { toast(e.message, 'error'); });
}

function setupDrag(container) {
  var svg = document.getElementById('diagram-svg');
  if (!svg) return;

  container.addEventListener('mousedown', function(e) {
    if (diagramState.mode === 'view') return;
    var target = e.target.closest('.er-table-group');
    if (target) {
      var tableId = target.getAttribute('data-table-id');
      var m = target.getAttribute('transform').match(/translate\(([\d.]+),\s*([\d.]+)\)/);
      if (!m) return;
      diagramState.dragInfo = { elem: target, tableId: tableId, sx: e.clientX, sy: e.clientY, ox: parseFloat(m[1]), oy: parseFloat(m[2]) };
      e.preventDefault();
    }
  });

  container.addEventListener('mousemove', function(e) {
    if (!diagramState.dragInfo) return;
    var dx = e.clientX - diagramState.dragInfo.sx, dy = e.clientY - diagramState.dragInfo.sy;
    diagramState.dragInfo.elem.setAttribute('transform', 'translate(' + (diagramState.dragInfo.ox + dx) + ',' + (diagramState.dragInfo.oy + dy) + ')');
  });

  container.addEventListener('mouseup', function() { diagramState.dragInfo = null; });
  container.addEventListener('mouseleave', function() { diagramState.dragInfo = null; });
}

function svgRect(ns, x, y, w, h, r, fill, stroke, sw) {
  var rect = document.createElementNS(ns, 'rect');
  rect.setAttribute('x', x); rect.setAttribute('y', y);
  rect.setAttribute('width', w); rect.setAttribute('height', h);
  if (r > 0) { rect.setAttribute('rx', r); rect.setAttribute('ry', r); }
  rect.setAttribute('fill', fill || '#fff');
  if (stroke) rect.setAttribute('stroke', stroke);
  if (sw) rect.setAttribute('stroke-width', sw);
  return rect;
}

function svgText(ns, x, y, fill, size, weight, content, anchor) {
  var t = document.createElementNS(ns, 'text');
  t.setAttribute('x', x); t.setAttribute('y', y);
  t.setAttribute('fill', fill); t.setAttribute('font-size', size);
  t.setAttribute('font-weight', weight || 'normal');
  if (anchor) t.setAttribute('text-anchor', anchor);
  t.textContent = content;
  return t;
}

function zoomDiagram(delta) {
  window._diagramZoom = Math.max(0.3, Math.min(2, (window._diagramZoom || 1) + delta));
  var svg = document.getElementById('diagram-svg');
  if (svg) svg.style.transform = 'scale(' + window._diagramZoom + ')';
}

function resetZoom() {
  window._diagramZoom = 1;
  var svg = document.getElementById('diagram-svg');
  if (svg) svg.style.transform = 'scale(1)';
}

window.loadDiagram = loadDiagram;
window.setMode = setMode;
window.zoomDiagram = zoomDiagram;
window.resetZoom = resetZoom;
window.editColumnOnDiagram = editColumnOnDiagram;
window.addColumnOnDiagram = addColumnOnDiagram;
window.editTableName = editTableName;
window.startFK = startFK;
window.completeFK = completeFK;
window.addTableOnDiagram = addTableOnDiagram;
