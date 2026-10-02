(() => {
  const menuButton = document.querySelector('.menu-toggle');
  const nav = document.querySelector('#primary-nav');
  const toast = document.querySelector('.toast');
  let toastTimer;

  menuButton?.addEventListener('click', () => {
    const isOpen = menuButton.getAttribute('aria-expanded') === 'true';
    menuButton.setAttribute('aria-expanded', String(!isOpen));
    nav?.classList.toggle('open', !isOpen);
  });

  nav?.querySelectorAll('a').forEach((link) => {
    link.addEventListener('click', () => {
      menuButton?.setAttribute('aria-expanded', 'false');
      nav.classList.remove('open');
    });
  });

  document.querySelectorAll('[data-copy]').forEach((button) => {
    button.addEventListener('click', async () => {
      const command = button.getAttribute('data-copy') || '';
      try {
        await navigator.clipboard.writeText(command);
        button.textContent = 'Copied';
        if (toast) {
          toast.textContent = 'Install command copied to clipboard';
          toast.classList.add('show');
          window.clearTimeout(toastTimer);
          toastTimer = window.setTimeout(() => toast.classList.remove('show'), 2200);
        }
      } catch {
        button.textContent = 'Select command';
      }
      window.setTimeout(() => { button.textContent = 'Copy'; }, 1800);
    });
  });

  const year = document.querySelector('#year');
  if (year) year.textContent = String(new Date().getFullYear());
})();