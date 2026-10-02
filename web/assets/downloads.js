(() => {
  const API_URL = '/api/releases';
  const repoReleases = 'https://github.com/printmaster-org/printmaster/releases';
  const status = document.querySelector('#release-status');
  const latestSection = document.querySelector('#latest-release');
  const previousSection = document.querySelector('#previous-releases');
  const betaSection = document.querySelector('#beta-releases');
  const panel = document.querySelector('#release-content');
  const tabs = [...document.querySelectorAll('.component-tab')];
  let catalog = null;
  let selectedComponent = location.hash === '#server' ? 'server' : 'agent';

  document.querySelectorAll('.menu-toggle').forEach((button) => {
    const nav = document.querySelector(`#${button.getAttribute('aria-controls')}`);
    button.addEventListener('click', () => {
      const open = button.getAttribute('aria-expanded') === 'true';
      button.setAttribute('aria-expanded', String(!open));
      nav?.classList.toggle('open', !open);
    });
    nav?.querySelectorAll('a').forEach((link) => link.addEventListener('click', () => {
      button.setAttribute('aria-expanded', 'false');
      nav.classList.remove('open');
    }));
  });

  const year = document.querySelector('#year');
  if (year) year.textContent = String(new Date().getFullYear());

  tabs.forEach((tab) => tab.addEventListener('click', () => {
    setComponent(tab.dataset.component);
  }));

  window.addEventListener('hashchange', () => {
    const requested = location.hash === '#server' ? 'server' : 'agent';
    if (requested !== selectedComponent) setComponent(requested, false);
  });

  function setComponent(component, updateHash = true) {
    selectedComponent = component === 'server' ? 'server' : 'agent';
    tabs.forEach((tab) => {
      const active = tab.dataset.component === selectedComponent;
      tab.classList.toggle('active', active);
      tab.setAttribute('aria-selected', String(active));
    });
    panel.setAttribute('aria-labelledby', `tab-${selectedComponent}`);
    if (updateHash && location.hash !== `#${selectedComponent}`) history.replaceState(null, '', `#${selectedComponent}`);
    if (catalog) renderCatalog();
  }

  function element(tag, className, text) {
    const node = document.createElement(tag);
    if (className) node.className = className;
    if (text !== undefined) node.textContent = text;
    return node;
  }

  function safeLink(rawURL) {
    try {
      const parsed = new URL(rawURL);
      return parsed.protocol === 'https:' && parsed.hostname.toLowerCase() === 'github.com' ? parsed.href : '';
    } catch {
      return '';
    }
  }

  function externalLink(label, rawURL, className = '') {
    const href = safeLink(rawURL);
    if (!href) return null;
    const link = element('a', className, label);
    link.href = href;
    link.target = '_blank';
    link.rel = 'noopener noreferrer';
    return link;
  }

  function formatDate(rawDate) {
    const date = new Date(rawDate);
    if (Number.isNaN(date.getTime())) return 'Date unavailable';
    return new Intl.DateTimeFormat(undefined, { year: 'numeric', month: 'short', day: 'numeric' }).format(date);
  }

  function formatSize(bytes) {
    if (!Number.isFinite(bytes) || bytes <= 0) return '';
    const units = ['B', 'KB', 'MB', 'GB'];
    let value = bytes;
    let unit = 0;
    while (value >= 1024 && unit < units.length - 1) {
      value /= 1024;
      unit += 1;
    }
    return `${value >= 10 || unit === 0 ? value.toFixed(0) : value.toFixed(1)} ${units[unit]}`;
  }

  function assetLabel(name) {
    const lower = name.toLowerCase();
    if (lower.endsWith('.msi')) return 'Windows installer · MSI';
    if (lower.endsWith('.deb')) return `Debian / Ubuntu · ${lower.includes('_arm64') ? 'ARM64' : 'x64'}`;
    if (lower.endsWith('.rpm')) return `Fedora / RHEL · ${lower.includes('aarch64') ? 'ARM64' : 'x64'}`;
    if (lower.includes('windows') && lower.endsWith('.exe')) return 'Windows executable · x64';
    if (lower.includes('darwin')) return `macOS · ${lower.includes('arm64') ? 'Apple silicon' : 'Intel'}`;
    if (lower.includes('linux')) return `Linux binary · ${lower.includes('arm64') ? 'ARM64' : 'x64'}`;
    return name;
  }

  function primaryAsset(release, component) {
    const preference = component === 'agent'
      ? ['.msi', '.deb', '.rpm', 'windows-amd64.exe', 'linux-amd64', 'darwin-arm64']
      : ['linux-amd64', 'windows-amd64.exe', 'darwin-arm64'];
    for (const suffix of preference) {
      const asset = release.assets.find((candidate) => candidate.name.toLowerCase().endsWith(suffix));
      if (asset) return asset;
    }
    return release.assets[0];
  }

  function assetList(release, component, featured = false) {
    const list = element('div', featured ? 'asset-list featured-assets' : 'asset-list');
    const primary = primaryAsset(release, component);
    const sorted = [...release.assets].sort((a, b) => {
      if (a === primary) return -1;
      if (b === primary) return 1;
      return assetLabel(a.name).localeCompare(assetLabel(b.name));
    });

    sorted.forEach((asset) => {
      const link = externalLink('', asset.url, `asset-link${asset === primary ? ' asset-link-primary' : ''}`);
      if (!link) return;
      const label = element('span', 'asset-label', assetLabel(asset.name));
      const name = element('span', 'asset-filename', asset.name);
      const size = element('span', 'asset-size', formatSize(asset.size));
      link.append(label, name, size);
      list.append(link);
    });
    return list;
  }

  function releaseHeading(release, badgeText) {
    const heading = element('div', 'release-heading');
    const copy = element('div', 'release-heading-copy');
    const eyebrow = element('div', 'release-version-line');
    eyebrow.append(element('span', 'release-version', `v${release.version}`));
    if (badgeText) eyebrow.append(element('span', `release-badge ${badgeText === 'BETA' ? 'badge-beta' : 'badge-stable'}`, badgeText));
    copy.append(eyebrow, element('h3', '', release.name || release.tag_name));
    heading.append(copy, element('time', 'release-date', formatDate(release.published_at)));
    return heading;
  }

  function releaseNotesLink(release) {
    return externalLink('Release notes ↗', release.html_url, 'release-notes-link');
  }

  function renderLatest(component, channels) {
    latestSection.replaceChildren();
    latestSection.setAttribute('aria-labelledby', 'latest-heading');
    latestSection.append(element('div', 'section-kicker', 'LATEST STABLE RELEASE'));
    latestSection.append(element('h2', 'release-section-title', 'Ready for production'));

    const release = channels.stable?.[0];
    if (!release) {
      const empty = element('article', 'latest-release-card empty-release');
      empty.append(element('h3', '', 'No stable release found'), element('p', '', 'Check the full GitHub release archive or come back soon.'));
      const link = externalLink('Browse all releases ↗', repoReleases, 'text-link');
      if (link) empty.append(link);
      latestSection.append(empty);
      return;
    }

    const card = element('article', 'latest-release-card');
    const top = element('div', 'latest-card-top');
    const icon = element('div', 'latest-download-icon', component === 'agent' ? 'A' : 'S');
    top.append(icon, releaseHeading(release, 'STABLE'));
    const notes = releaseNotesLink(release);
    if (notes) top.append(notes);
    card.append(top);

    const primary = primaryAsset(release, component);
    const primaryDownload = primary && externalLink(`Download ${assetLabel(primary.name)}`, primary.url, 'button button-primary featured-download-button');
    if (primaryDownload) {
      primaryDownload.append(element('span', 'download-arrow', '↓'));
      card.append(primaryDownload);
    }
    card.append(element('h4', 'asset-section-title', 'All available files'));
    card.append(assetList(release, component, true));
    latestSection.append(card);
  }

  function renderPrevious(component, channels) {
    previousSection.replaceChildren();
    const older = channels.stable?.slice(1) || [];
    const header = element('div', 'release-list-header');
    header.append(element('div', 'section-kicker', 'STABLE HISTORY'), element('h2', 'release-section-title', 'Previous versions'));
    previousSection.append(header);
    if (!older.length) {
      previousSection.append(element('p', 'release-empty-note', 'No previous stable versions listed yet.'));
      return;
    }

    const list = element('div', 'previous-release-list');
    older.forEach((release) => {
      const card = element('details', 'previous-release-card');
      const summary = element('summary', 'previous-release-summary');
      const version = element('span', 'previous-version', `v${release.version}`);
      const name = element('span', 'previous-name', release.name || release.tag_name);
      const date = element('time', 'release-date', formatDate(release.published_at));
      summary.append(version, name, date, element('span', 'expand-icon', '+'));
      card.append(summary);
      const details = element('div', 'previous-release-details');
      details.append(assetList(release, component));
      const notes = releaseNotesLink(release);
      if (notes) details.append(notes);
      card.append(details);
      list.append(card);
    });
    previousSection.append(list);
  }

  function renderBeta(component, channels) {
    betaSection.replaceChildren();
    const header = element('div', 'beta-heading');
    header.append(element('div', 'section-kicker', 'EARLY ACCESS'), element('h2', 'release-section-title', 'Beta builds'), element('p', '', 'Pre-release software may be incomplete or change without notice. Use beta builds for testing, not production fleets.'));
    betaSection.append(header);
    const releases = channels.beta || [];
    if (!releases.length) {
      betaSection.append(element('p', 'release-empty-note', 'No beta builds are currently listed.'));
      return;
    }

    const list = element('div', 'beta-release-list');
    releases.forEach((release) => {
      const card = element('details', 'beta-release-card');
      const summary = element('summary', 'previous-release-summary');
      summary.append(element('span', 'previous-version', `v${release.version}`), element('span', 'previous-name', release.name || release.tag_name), element('time', 'release-date', formatDate(release.published_at)), element('span', 'expand-icon', '+'));
      card.append(summary);
      const details = element('div', 'previous-release-details');
      details.append(assetList(release, component));
      const notes = releaseNotesLink(release);
      if (notes) details.append(notes);
      card.append(details);
      list.append(card);
    });
    betaSection.append(list);
  }

  function renderCatalog() {
    if (!catalog) return;
    const channels = catalog[selectedComponent];
    renderLatest(selectedComponent, channels);
    renderPrevious(selectedComponent, channels);
    renderBeta(selectedComponent, channels);
    status.textContent = catalog.stale
      ? 'GitHub is temporarily unavailable; showing recently cached releases.'
      : `Release metadata refreshed ${formatDate(catalog.fetched_at)} · Downloads served by GitHub.`;
    status.classList.toggle('status-stale', Boolean(catalog.stale));
  }

  async function loadReleases() {
    try {
      const response = await fetch(API_URL, { headers: { Accept: 'application/json' } });
      if (!response.ok) throw new Error(`release metadata request failed (${response.status})`);
      catalog = await response.json();
      if (!catalog.agent || !catalog.server) throw new Error('invalid release metadata');
      renderCatalog();
    } catch {
      status.classList.add('status-error');
      status.replaceChildren(
        element('span', '', 'Could not load release information right now. '),
        externalLink('Browse GitHub Releases ↗', repoReleases, 'text-link')
      );
    }
  }

  setComponent(selectedComponent, false);
  loadReleases();
})();
