// Единый обработчик подтверждения необратимых действий. Форма с атрибутом
// data-confirm открывает модальное окно (Bulma) с текстом из атрибута вместо
// системного window.confirm. Публичный контракт data-confirm="..." сохранён:
// механизм используется и в списках, и в карточках (удаление, отправка в 1С).
(function () {
  'use strict';

  var approvedForm = null;
  var pendingForm = null;
  var pendingSubmitter = null;
  var modal = null;
  var messageEl = null;
  var okButton = null;

  function buildModal() {
    if (modal) return;

    modal = document.createElement('div');
    modal.className = 'modal';
    modal.setAttribute('role', 'dialog');
    modal.setAttribute('aria-modal', 'true');
    modal.setAttribute('aria-labelledby', 'confirm-modal-title');
    modal.innerHTML =
      '<div class="modal-background" data-confirm-cancel></div>' +
      '<div class="modal-card">' +
      '<header class="modal-card-head">' +
      '<p class="modal-card-title" id="confirm-modal-title">Подтверждение</p>' +
      '<button type="button" class="delete" aria-label="Закрыть" data-confirm-cancel></button>' +
      '</header>' +
      '<section class="modal-card-body"><p data-confirm-message></p></section>' +
      '<footer class="modal-card-foot is-justify-content-flex-end">' +
      '<button type="button" class="button" data-confirm-cancel>Отмена</button>' +
      '<button type="button" class="button is-primary" data-confirm-ok>ОК</button>' +
      '</footer>' +
      '</div>';

    messageEl = modal.querySelector('[data-confirm-message]');
    okButton = modal.querySelector('[data-confirm-ok]');
    okButton.addEventListener('click', onConfirm);

    var cancels = modal.querySelectorAll('[data-confirm-cancel]');
    for (var i = 0; i < cancels.length; i++) {
      cancels[i].addEventListener('click', cancel);
    }

    document.addEventListener('keydown', function (e) {
      if (e.key === 'Escape' && modal.classList.contains('is-active')) cancel();
    });

    document.body.appendChild(modal);
  }

  function open(form, submitter) {
    buildModal();
    pendingForm = form;
    pendingSubmitter = submitter || null;
    messageEl.textContent = form.getAttribute('data-confirm') || '';
    modal.classList.add('is-active');
    okButton.focus();
  }

  function closeModal() {
    if (modal) modal.classList.remove('is-active');
    pendingForm = null;
    pendingSubmitter = null;
  }

  // Отмена: закрыть без отправки и вернуть фокус на кнопку, которой отправляли.
  function cancel() {
    var submitter = pendingSubmitter;
    closeModal();
    if (submitter && typeof submitter.focus === 'function') submitter.focus();
  }

  function onConfirm() {
    var form = pendingForm;
    var submitter = pendingSubmitter;
    closeModal();
    if (!form) return;

    // Помечаем форму одобренной, чтобы наш же обработчик submit пропустил
    // повторную отправку.
    approvedForm = form;
    if (typeof form.requestSubmit === 'function') {
      form.requestSubmit(submitter && submitter.form === form ? submitter : undefined);
    } else {
      form.submit();
    }
    // Если submit-событие не возникло (например, не прошла HTML5-валидация)
    // или его не обработал наш слушатель, сбрасываем флаг на следующем тике.
    setTimeout(function () {
      if (approvedForm === form) approvedForm = null;
    }, 0);
  }

  document.addEventListener('submit', function (e) {
    var form = e.target.closest('form[data-confirm]');
    if (!form) return;
    if (form === approvedForm) {
      approvedForm = null;
      return;
    }
    e.preventDefault();
    open(form, e.submitter);
  });
})();
