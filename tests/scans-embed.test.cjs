const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const source = fs.readFileSync(path.join(__dirname, '..', 'web', 'assets', 'scans.js'), 'utf8');

function render() {
  const listeners = {};
  const sent = [];
  const scrolls = [];
  const hashes = [];
  const status = { textContent: 'Loading' };
  const frame = {
    contentWindow: { postMessage: (data, origin) => sent.push({ data, origin }) },
    style: {}, addEventListener: (type, fn) => { listeners[type] = fn; },
    getBoundingClientRect: () => ({ top: 100 })
  };
  let timeout;
  vm.runInNewContext(source, {
    document: {
      getElementById: id => id === 'scans-frame' ? frame : status,
      querySelector: () => ({ getBoundingClientRect: () => ({ height: 76 }) })
    },
    location: { hash: '#releases' },
    history: { replaceState: (_state, _title, hash) => hashes.push(hash) },
    window: {
      scrollY: 20, addEventListener: (type, fn) => { listeners[type] = fn; },
      scrollTo: value => scrolls.push(value.top),
      setTimeout: fn => { timeout = fn; }
    }
  });
  const message = data => listeners.message({
    origin: 'https://scans.printmaster.work', source: frame.contentWindow, data
  });
  return { listeners, frame, status, sent, scrolls, hashes, message, timeout: () => timeout() };
}

test('only accepts finite bounded dimensions from the exact Scans frame', () => {
  const page = render();
  for (const event of [
    { origin: 'https://evil.example', source: page.frame.contentWindow },
    { origin: 'https://scans.printmaster.work', source: {} }
  ]) page.listeners.message({ ...event, data: { type: 'printmaster:scans:resize', height: 2000 } });
  for (const height of [NaN, Infinity, -1, 0, 199, 100001, '2000']) {
    page.message({ type: 'printmaster:scans:resize', height });
  }
  assert.equal(page.frame.style.height, undefined);
  page.message({ type: 'printmaster:scans:resize', height: 2000.2 });
  assert.equal(page.frame.style.height, '2001px');
  assert.equal(page.status.textContent, '');
  page.timeout();
  assert.equal(page.status.textContent, '');
});

test('handshakes on load/ready/hashchange and scrolls only to validated anchors', () => {
  const page = render();
  page.listeners.load();
  page.message({ type: 'printmaster:scans:ready' });
  page.listeners.hashchange();
  assert.equal(page.sent.length, 3);
  assert.ok(page.sent.every(message => message.origin === 'https://scans.printmaster.work' &&
    message.data.hash === '#releases'));
  page.message({ type: 'printmaster:scans:resize', height: 2000 });
  for (const [hash, top] of [['https://evil.example', 10], ['#releases', Infinity], ['#releases', 2001]]) {
    page.message({ type: 'printmaster:scans:anchor', hash, top });
  }
  assert.equal(page.scrolls.length, 0);
  page.message({ type: 'printmaster:scans:anchor', hash: '#releases', top: 1200 });
  assert.equal(page.hashes[0], '#releases');
  assert.equal(page.scrolls[0], 1228);
});

test('reports bridge failure while the static standalone link remains available', () => {
  const page = render();
  page.timeout();
  assert.match(page.status.textContent, /could not connect/);
});
