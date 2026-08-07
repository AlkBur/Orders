(function () {
  'use strict';

  function getModal() {
    return document.getElementById('receipts-modal');
  }

  function openDialog(url) {
    fetch(url, { headers: { 'HX-Request': 'true' } })
      .then(function (resp) {
        if (!resp.ok) throw new Error('Request failed');
        return resp.text();
      })
      .then(function (html) {
        var current = getModal();
        if (current) current.outerHTML = html;
        var modal = getModal();
        if (modal) modal.classList.add('is-active');
      })
      .catch(function () {
        var modal = getModal();
        if (!modal) return;
        var body = modal.querySelector('.modal-card-body');
        if (body) body.innerHTML = '<p class="receipts-modal-error">Не удалось загрузить содержимое.</p>';
        modal.classList.add('is-active');
      });
  }

  function closeDialog() {
    var modal = getModal();
    if (modal) modal.classList.remove('is-active');
  }

  document.addEventListener('click', function (e) {
    var disabled = e.target.closest('a[aria-disabled="true"]');
    if (disabled) {
      e.preventDefault();
      return;
    }

    if (e.target.closest('.receipts-modal-background') || e.target.closest('[data-modal-close]')) {
      closeDialog();
      return;
    }

    var opener = e.target.closest('[data-dialog-url]');
    if (!opener) return;
    e.preventDefault();
    openDialog(opener.getAttribute('data-dialog-url'));
  });

  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') closeDialog();
  });

  document.addEventListener('submit', function (e) {
    var form = e.target.closest('form[data-confirm]');
    if (!form) return;
    var message = form.getAttribute('data-confirm');
    if (!window.confirm(message)) e.preventDefault();
  });
})();