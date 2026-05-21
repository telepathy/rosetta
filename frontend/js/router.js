const router = {
  routes: {},
  _guard: null,

  on(path, handler) {
    this.routes[path] = handler;
  },

  guard(fn) {
    this._guard = fn;
  },

  navigate(path) {
    window.location.hash = path;
  },

  _match(hash) {
    const path = hash.replace(/^#/, '') || '/';
    for (const [pattern, handler] of Object.entries(this.routes)) {
      const regex = new RegExp('^' + pattern.replace(/:\w+/g, '(\\w+)') + '$');
      const match = path.match(regex);
      if (match) {
        const params = {};
        const paramNames = [...pattern.matchAll(/:(\w+)/g)].map(m => m[1]);
        paramNames.forEach((name, i) => params[name] = match[i + 1]);
        return { handler, params };
      }
    }
    return { handler: this.routes['/404'] || (() => '<h2>404 Not Found</h2>') };
  },

  async _resolve() {
    try {
      const { handler, params } = this._match(window.location.hash);
      if (this._guard && !window.location.hash.startsWith('#/login')) {
        const ok = await this._guard();
        if (!ok) {
          this.navigate('/login');
          return;
        }
      }
      const app = document.getElementById('app');
      if (!app) return;
      app.innerHTML = await handler(params);
    } catch (e) {
      var app = document.getElementById('app');
      if (app) app.innerHTML = '<div style="padding:40px;text-align:center"><h2>页面错误</h2><p style="color:#dc2626;margin:12px 0">' + e.message + '</p><pre style="text-align:left;background:#1e293b;color:#e2e8f0;padding:16px;border-radius:8px;overflow:auto;max-height:300px;font-size:12px">' + (e.stack || '').replace(/</g,'&lt;') + '</pre><br><a href="#/login" style="color:#4f46e5">返回登录</a></div>';
    }
  },

  start() {
    window.addEventListener('hashchange', () => this._resolve());
    this._resolve();
  }
};
