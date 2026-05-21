var diagramState = { cy: null, connectMode: false, connectSource: null };

async function pageDiagram() {
  document.title = 'Rosetta - ER 图';
  var html = sidebarHtml('/diagram');
  html += '<div class="page-header"><h2>📊 数据库模型图</h2><div class="page-desc">自动布局 ｜ 滚轮缩放 ｜ 拖拽平移 ｜ 点击表节点查看详情</div></div>';
  html += '<div class="page-toolbar">';
  html += '<select id="diagram-database" onchange="onDiagramDBChange()"><option value="">选择逻辑库...</option></select>';
  html += '<select id="diagram-schema" onchange="loadDiagram()"><option value="">先选择逻辑库</option></select>';
  html += '<div class="flex-spacer"></div>';
  html += '<button id="vfk-mode-btn" class="btn btn-sm btn-outline" onclick="toggleConnectMode()">🔗 虚拟外键</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="fitDiagram()">⊞ 适应</button>';
  html += '<button class="btn btn-sm btn-outline" onclick="if(diagramState.cy){diagramState.cy.zoom(1);diagramState.cy.center();}">1:1</button>';
  html += '</div>';
  html += '<div id="vfk-status" style="display:none;padding:4px 12px;background:#eef2ff;border-radius:var(--radius);font-size:13px;color:var(--primary);margin-bottom:8px"></div>';
  html += '<div style="display:flex;gap:16px;height:calc(100vh - 240px)">';
  html += '<div id="diagram-viewport" style="flex:1;border:1px solid var(--border);border-radius:var(--radius);background:#fafbfc;position:relative;min-width:0">';
  html += '<div class="empty-state"><div class="empty-icon">📊</div><p>请先选择逻辑库和 Schema 查看 ER 图</p></div>';
  html += '</div>';
  html += '<div id="diagram-detail" class="card" style="width:340px;flex-shrink:0;overflow-y:auto;display:none">';
  html += '<div class="card-title" id="detail-title">表详情</div>';
  html += '<div id="detail-content"></div>';
  html += '</div>';
  html += '</div></div>';
  setTimeout(function() { loadDiagramDBOptions(); }, 0);
  return html;
}

async function loadDiagramDBOptions() {
  try {
    var dbs = (await api.get('/databases')).data || [];
    var sel = document.getElementById('diagram-database');
    sel.innerHTML = '<option value="">选择逻辑库...</option>';
    dbs.forEach(function(d) { sel.innerHTML += '<option value="' + d.id + '">' + escapeHtml(d.name) + '</option>'; });
    if (dbs.length === 1) { sel.value = dbs[0].id; onDiagramDBChange(); }
  } catch(e) {}
}

async function onDiagramDBChange() {
  var dbId = document.getElementById('diagram-database').value;
  var schemaSel = document.getElementById('diagram-schema');
  schemaSel.innerHTML = '<option value="">先选择逻辑库</option>';
  if (!dbId) { schemaSel.innerHTML = '<option value="">选择 Schema...</option>'; return; }
  try {
    var schemas = (await api.get('/databases/' + dbId + '/schemas')).data || [];
    schemaSel.innerHTML = '<option value="">选择 Schema...</option>';
    schemas.forEach(function(s) { schemaSel.innerHTML += '<option value="' + s.id + '">' + escapeHtml(s.name) + '</option>'; });
  } catch(e) {}
}

async function loadDiagram() {
  var schemaId = document.getElementById('diagram-schema').value;
  if (!schemaId) return;
  if (diagramState.cy) { diagramState.cy.destroy(); diagramState.cy = null; }
  document.getElementById('diagram-detail').style.display = 'none';
  diagramState.connectMode = false;
  diagramState.connectSource = null;
  updateConnectModeUI();

  var vp = document.getElementById('diagram-viewport');
  vp.innerHTML = '<div style="text-align:center;padding:60px">加载中...</div>';

  try {
    var er = (await api.get('/logical-schemas/' + schemaId + '/er-diagram')).data;
    if (!er || !er.tables || er.tables.length === 0) {
      vp.innerHTML = '<div class="empty-state"><p>该 Schema 下没有部署的模型</p></div>'; return;
    }
    var tables = er.tables, edges = er.edges || [];
    var modelDetails = {};
    for (var i = 0; i < tables.length; i += 8) {
      var batch = tables.slice(i, i + 8);
      var results = await Promise.all(batch.map(function(t) { return api.get('/models/' + t.id).catch(function() { return null; }); }));
      results.forEach(function(r, j) { if (r && r.data) modelDetails[batch[j].id] = r.data; });
    }
    vp.innerHTML = '';
    renderGraph(vp, tables, edges, modelDetails);
  } catch(e) {
    vp.innerHTML = '<div class="empty-state"><p>加载失败: ' + e.message + '</p></div>';
  }
}

function renderDetail(modelId, modelDetails) {
  var detail = modelDetails[modelId];
  if (!detail) return;
  var title = document.getElementById('detail-title');
  title.textContent = detail.table_name || '表详情';
  var cols = detail.columns || [];
  var html = '<table style="font-size:12px"><thead><tr><th>字段</th><th>类型</th><th>键</th><th>可空</th></tr></thead><tbody>';
  cols.forEach(function(c) {
    var ts = c.logical_type;
    if (c.type_length) ts += '(' + c.type_length + (c.type_scale ? ',' + c.type_scale : '') + ')';
    var key = c.is_primary_key ? '🔑' : (c.is_foreign_key ? '🔗' : '');
    html += '<tr><td style="font-family:monospace;white-space:nowrap">' + c.column_name + '</td><td style="font-family:monospace;font-size:11px;color:var(--text-secondary)">' + ts + '</td><td>' + key + '</td><td>' + (c.nullable ? '✓' : '✗') + '</td></tr>';
  });
  html += '</tbody></table>';
  if (detail.table_comment) html += '<div style="margin-top:8px;font-size:12px;color:var(--text-secondary)">备注: ' + detail.table_comment + '</div>';

  // Virtual FK section
  var vfks = detail.virtual_foreign_keys || [];
  if (vfks.length > 0) {
    html += '<div style="margin-top:12px;font-weight:600;font-size:13px;color:#6366f1">🔗 虚拟外键</div>';
    html += '<table style="font-size:12px;margin-top:4px"><thead><tr><th>字段</th><th>引用表</th><th>引用列</th><th></th></tr></thead><tbody>';
    vfks.forEach(function(vfk) {
      html += '<tr><td>' + escapeHtml(vfk.column_name) + '</td><td>' + escapeHtml(vfk.ref_table_name || '') + '</td><td>' + escapeHtml(vfk.ref_column_name) + '</td>';
      html += '<td><button class="btn btn-sm" style="color:#ef4444;font-size:11px;padding:2px 6px" onclick="deleteVFKFromDetail(' + detail.id + ',' + vfk.id + ')">✕</button></td>';
      html += '</tr>';
    });
    html += '</tbody></table>';
  }

  document.getElementById('detail-content').innerHTML = html;
  document.getElementById('diagram-detail').style.display = 'block';
}

async function deleteVFKFromDetail(modelId, vfkId) {
  if (!confirm('确定删除该虚拟外键吗？')) return;
  try {
    await api.del('/models/' + modelId + '/virtual-foreign-keys/' + vfkId);
    toast('虚拟外键已删除', 'success');
    loadDiagram();
  } catch(e) { toast(e.message, 'error'); }
}

function renderGraph(vp, tables, edges, modelDetails) {
  if (!window.cytoscape) { vp.innerHTML = '<div class="empty-state"><p>可视化库未加载，请刷新页面</p></div>'; return; }

  var elements = [];
  tables.forEach(function(t) {
    elements.push({ data: { id: 't' + t.id, tableName: t.table_name, modelId: t.id } });
  });
  var edgeIdx = 0;
  edges.forEach(function(e) {
    if (e.source !== e.target) {
      elements.push({
        data: {
          id: 'e_' + e.source + '_' + e.target + '_' + edgeIdx,
          source: 't' + e.source,
          target: 't' + e.target,
          label: e.source_col + ' → ' + e.target_col,
          virtual: e.virtual,
          sourceCol: e.source_col,
          targetCol: e.target_col,
          fkName: e.fk_name
        }
      });
      edgeIdx++;
    }
  });

  var cy = window.cytoscape({
    container: vp, elements: elements,
    style: [
      { selector: 'node', style: {
        'label': 'data(tableName)',
        'shape': 'round-rectangle', 'width': 140, 'height': 40,
        'background-color': '#ffffff', 'border-color': '#4f46e5', 'border-width': 2,
        'text-valign': 'center', 'text-halign': 'center',
        'font-size': '12px', 'font-weight': 'bold', 'color': '#1e293b',
      }},
      { selector: 'edge', style: {
        'width': 1.5, 'line-color': '#94a3b8', 'target-arrow-color': '#94a3b8',
        'target-arrow-shape': 'triangle', 'curve-style': 'bezier',
        'label': 'data(label)',
        'font-size': '10px', 'color': '#64748b',
        'text-background-color': '#ffffff', 'text-background-opacity': 0.85,
        'text-background-padding': '2px',
        'text-rotation': 'autorotate',
      }},
      { selector: 'edge[?virtual]', style: {
        'line-style': 'dashed', 'line-color': '#f59e0b', 'target-arrow-color': '#f59e0b',
        'target-arrow-shape': 'triangle-backcurve', 'width': 2,
        'line-dash-pattern': [6, 4],
        'label': 'data(label)',
        'font-size': '10px', 'color': '#d97706', 'font-weight': 'bold',
        'text-background-color': '#fffbeb', 'text-background-opacity': 0.9,
        'text-background-padding': '2px',
        'text-rotation': 'autorotate',
      }},
      { selector: 'node.connect-source', style: {
        'border-color': '#22c55e', 'border-width': 4,
      }},
    ],
    layout: { name: 'cose', idealEdgeLength: 120, nodeRepulsion: 150000, gravity: 0.8, gravityRange: 500, nodeOverlap: 200, nodeDimensionsIncludeLabels: true, padding: 60, numIter: 2000 },
    minZoom: 0.1, maxZoom: 3,
  });

  // Spread overlapping nodes after layout
  cy.on('layoutstop', function() {
    function getBB(n) { return n.boundingBox({ includeLabels: true }); }
    function rectsOverlap(a, b) {
      return !(a.x2 <= b.x1 || a.x1 >= b.x2 || a.y2 <= b.y1 || a.y1 >= b.y2);
    }
    var done = false;
    while (!done) {
      done = true;
      var nodes = cy.nodes().toArray();
      for (var i = 0; i < nodes.length; i++) {
        for (var j = i + 1; j < nodes.length; j++) {
          var a = getBB(nodes[i]), b = getBB(nodes[j]);
          if (rectsOverlap(a, b)) {
            done = false;
            var dx = nodes[j].position('x') - nodes[i].position('x');
            var dy = nodes[j].position('y') - nodes[i].position('y');
            var overlapX = Math.min(a.x2 - b.x1, b.x2 - a.x1) / 2 + 5;
            var overlapY = Math.min(a.y2 - b.y1, b.y2 - a.y1) / 2 + 5;
            if (Math.abs(overlapX) > Math.abs(overlapY)) {
              if (dx > 0) { nodes[i].position('x', nodes[i].position('x') - overlapX); nodes[j].position('x', nodes[j].position('x') + overlapX); }
              else { nodes[i].position('x', nodes[i].position('x') + overlapX); nodes[j].position('x', nodes[j].position('x') - overlapX); }
            } else {
              if (dy > 0) { nodes[i].position('y', nodes[i].position('y') - overlapY); nodes[j].position('y', nodes[j].position('y') + overlapY); }
              else { nodes[i].position('y', nodes[i].position('y') + overlapY); nodes[j].position('y', nodes[j].position('y') - overlapY); }
            }
          }
        }
      }
    }
    cy.fit(undefined, 50);
  });

  cy.on('tap', 'edge', function(ev) {
    var edge = ev.target;
    if (edge.data('virtual')) {
      var srcName = edge.source().data('tableName');
      var tgtName = edge.target().data('tableName');
      var info = '虚拟外键: ' + srcName + '.' + edge.data('sourceCol') + ' → ' + tgtName + '.' + edge.data('targetCol');
      if (edge.data('fkName')) info += '\n名称: ' + edge.data('fkName');
      if (confirm(info + '\n\n删除该虚拟外键？')) {
        var srcModelId = edge.source().data('modelId');
        api.get('/logical-schemas/' + document.getElementById('diagram-schema').value + '/virtual-foreign-keys').then(function(res) {
          var vfks = res.data || [];
          var vfk = vfks.find(function(v) { return v.model_id === srcModelId && v.column_name === edge.data('sourceCol') && v.ref_column_name === edge.data('targetCol'); });
          if (vfk) {
            api.del('/models/' + srcModelId + '/virtual-foreign-keys/' + vfk.id).then(function() {
              toast('虚拟外键已删除', 'success');
              loadDiagram();
            }).catch(function(e) { toast(e.message, 'error'); });
          }
        });
      }
    }
  });

  cy.on('tap', 'node', function(ev) {
    var n = ev.target;
    if (diagramState.connectMode) {
      ev.originalEvent && ev.originalEvent.stopPropagation();
      if (!diagramState.connectSource) {
        diagramState.connectSource = n;
        n.addClass('connect-source');
        document.getElementById('vfk-status').textContent = '已选择源表: ' + n.data('tableName') + '，请点击目标表';
      } else {
        var src = diagramState.connectSource;
        var tgt = n;
        n.removeClass('connect-source');
        src.removeClass('connect-source');
        diagramState.connectSource = null;
        showVFKModal(src.data('modelId'), src.data('tableName'), tgt.data('modelId'), tgt.data('tableName'), modelDetails);
      }
      return;
    }
    renderDetail(n.data('modelId'), modelDetails);
    n.style('border-width', 3);
    cy.nodes().not(n).style('border-width', 2);
  });

  cy.on('tap', function(ev) {
    if (ev.target === cy) {
      if (diagramState.connectMode && diagramState.connectSource) {
        diagramState.connectSource.removeClass('connect-source');
        diagramState.connectSource = null;
        document.getElementById('vfk-status').textContent = '连线模式：点击源表，再点击目标表';
      }
      document.getElementById('diagram-detail').style.display = 'none';
      cy.nodes().style('border-width', 2);
    }
  });

  diagramState.cy = cy;
  diagramState.modelDetails = modelDetails;
  cy.on('layoutstop', function() { cy.fit(undefined, 50); });
}

function toggleConnectMode() {
  diagramState.connectMode = !diagramState.connectMode;
  if (!diagramState.connectMode) {
    diagramState.connectSource = null;
  }
  updateConnectModeUI();
}

function updateConnectModeUI() {
  var btn = document.getElementById('vfk-mode-btn');
  var status = document.getElementById('vfk-status');
  if (!btn || !status) return;
  if (diagramState.connectMode) {
    btn.className = 'btn btn-sm btn-primary';
    status.style.display = 'block';
    status.textContent = '连线模式：点击源表，再点击目标表';
  } else {
    btn.className = 'btn btn-sm btn-outline';
    status.style.display = 'none';
    if (diagramState.cy) {
      diagramState.cy.nodes().removeClass('connect-source');
    }
  }
}

function showVFKModal(srcModelId, srcTableName, tgtModelId, tgtTableName, modelDetails) {
  var srcCols = (modelDetails[srcModelId] || {}).columns || [];
  var tgtCols = (modelDetails[tgtModelId] || {}).columns || [];
  var html = '<div style="padding:8px 0">';
  html += '<div style="display:flex;gap:24px">';
  html += '<div style="flex:1"><label style="font-weight:600;font-size:13px">源表: ' + escapeHtml(srcTableName) + '</label>';
  html += '<select id="vfk-src-col" style="width:100%;margin-top:6px;padding:6px;border:1px solid var(--border);border-radius:var(--radius)">';
  srcCols.forEach(function(c) { html += '<option value="' + escapeHtml(c.column_name) + '">' + escapeHtml(c.column_name) + '</option>'; });
  html += '</select></div>';
  html += '<div style="display:flex;align-items:center;padding-top:24px">→</div>';
  html += '<div style="flex:1"><label style="font-weight:600;font-size:13px">目标表: ' + escapeHtml(tgtTableName) + '</label>';
  html += '<select id="vfk-tgt-col" style="width:100%;margin-top:6px;padding:6px;border:1px solid var(--border);border-radius:var(--radius)">';
  tgtCols.forEach(function(c) { html += '<option value="' + escapeHtml(c.column_name) + '">' + escapeHtml(c.column_name) + '</option>'; });
  html += '</select></div>';
  html += '</div>';
  html += '<div style="margin-top:12px"><label style="font-weight:600;font-size:13px">外键名称（可选）</label>';
  html += '<input id="vfk-name" style="width:100%;margin-top:6px;padding:6px;border:1px solid var(--border);border-radius:var(--radius)" placeholder="如 fk_order_customer"></div>';
  html += '</div>';

  openModal('创建虚拟外键', html,
    '<button class="btn btn-primary" onclick="doCreateVFK(' + srcModelId + ',' + tgtModelId + ')">确认创建</button>' +
    '<button class="btn btn-outline" onclick="closeModal()">取消</button>'
  );
}

async function doCreateVFK(srcModelId, tgtModelId) {
  var srcCol = document.getElementById('vfk-src-col').value;
  var tgtCol = document.getElementById('vfk-tgt-col').value;
  var fkName = document.getElementById('vfk-name').value.trim();
  try {
    await api.post('/models/' + srcModelId + '/virtual-foreign-keys', {
      column_name: srcCol,
      ref_model_id: tgtModelId,
      ref_column_name: tgtCol,
      fk_name: fkName || undefined
    });
    toast('虚拟外键创建成功', 'success');
    closeModal();
    // Exit connect mode
    diagramState.connectMode = false;
    updateConnectModeUI();
    loadDiagram();
  } catch(e) { toast(e.message, 'error'); }
}

function fitDiagram() { if (diagramState.cy) diagramState.cy.fit(undefined, 50); }

window.loadDiagram = loadDiagram;
window.fitDiagram = fitDiagram;
window.onDiagramDBChange = onDiagramDBChange;
window.toggleConnectMode = toggleConnectMode;
window.doCreateVFK = doCreateVFK;
window.deleteVFKFromDetail = deleteVFKFromDetail;