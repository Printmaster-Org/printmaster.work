(() => {
  const origin = 'https://scans.printmaster.work';
  const frame = document.getElementById('scans-frame');
  const status = document.getElementById('scans-status');
  let ready = false;
  const initialize = () => frame.contentWindow.postMessage({
    type: 'printmaster:scans:init', hash: location.hash
  }, origin);
  frame.addEventListener('load', initialize);
  window.addEventListener('message', event => {
    if (event.origin !== origin || event.source !== frame.contentWindow) return;
    const data = event.data;
    if (!data || typeof data !== 'object') return;
    if (data.type === 'printmaster:scans:ready') {
      initialize();
    } else if (data.type === 'printmaster:scans:resize' &&
        Number.isFinite(data.height) && data.height >= 200 && data.height <= 100000) {
      frame.style.height = `${Math.ceil(data.height)}px`;
      ready = true;
      status.textContent = '';
    } else if (ready && data.type === 'printmaster:scans:anchor' &&
        typeof data.hash === 'string' && /^#[a-z][a-z0-9-]*$/.test(data.hash) &&
        Number.isFinite(data.top) && data.top >= 0 && data.top <= parseFloat(frame.style.height)) {
      history.replaceState(null, '', data.hash);
      const headerHeight = document.querySelector('.site-header').getBoundingClientRect().height;
      window.scrollTo({ top: frame.getBoundingClientRect().top + window.scrollY + data.top - headerHeight - 16 });
    }
  });
  window.addEventListener('hashchange', initialize);
  window.setTimeout(() => {
    if (!ready) status.textContent = 'Scans could not connect to this page. Open the standalone site below if it is unavailable or does not fit correctly.';
  }, 15000);
})();
