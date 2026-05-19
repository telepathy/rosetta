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
    const { handler, params } = this._match(window.location.hash);
    if (this._guard && !window.location.hash.startsWith('#/login')) {
      const ok = await this._guard();
      if (!ok) {
        this.navigate('/login');
        return;
      }
    }
    const app = document.getElementById('app');
    const result = await handler(params);
    if (typeof result === 'string') {
      app.innerHTML = result;
    } else if (result && result.el) {
      app.innerHTML = '';
      app.appendChild(result.el);
    }
    if (result && result.afterRender) result.afterRender();
  },

  start() {
    window.addEventListener('hashchange', () => this._resolve());
    this._resolve();
  }
};
