const API_BASE = '/api';

const api = {
  token: null,
  user: null,

  setToken(t) { this.token = t; localStorage.setItem('rosetta_token', t); },
  getToken() { return this.token || localStorage.getItem('rosetta_token'); },
  clearToken() { this.token = null; this.user = null; localStorage.removeItem('rosetta_token'); },

  async request(method, path, body) {
    const headers = { 'Content-Type': 'application/json' };
    const token = this.getToken();
    if (token) headers['Authorization'] = 'Bearer ' + token;

    const opts = { method, headers };
    if (body && method !== 'GET') opts.body = JSON.stringify(body);

    const res = await fetch(API_BASE + path, opts);
    const data = await res.json();

    if (data.code === 401) {
      this.clearToken();
      router.navigate('/login');
      throw new Error(data.message);
    }
    if (data.code !== 0) throw new Error(data.message);
    return data;
  },

  get(path) { return this.request('GET', path); },
  post(path, body) { return this.request('POST', path, body); },
  put(path, body) { return this.request('PUT', path, body); },
  del(path) { return this.request('DELETE', path); },

  async login(username, password) {
    const data = await this.post('/auth/login', { username, password });
    this.setToken(data.data.token);
    this.user = data.data.user;
    return data;
  },

  async fetchMe() {
    const data = await this.get('/auth/me');
    this.user = data.data;
    return data;
  }
};
