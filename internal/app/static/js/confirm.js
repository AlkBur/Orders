// Единый обработчик подтверждения необратимых действий. Форма с атрибутом
// data-confirm запрашивает подтверждение через window.confirm перед отправкой.
// Используется списками и карточками (например, удаление пользователя).
(function () {
  'use strict';

  document.addEventListener('submit', function (e) {
    var form = e.target.closest('form[data-confirm]');
    if (!form) return;
    var message = form.getAttribute('data-confirm');
    if (!window.confirm(message)) e.preventDefault();
  });
})();
