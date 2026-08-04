// Обработка структурированных ошибок платформы. JSON-ответы Infrastructure
// (400/413/429) htmx не заменяет на HTML по умолчанию, поэтому alert рисуется
// клиентом рядом с формой, помеченной атрибутом data-response-errors.
//
// Единое место создания alert клиентом: структура повторяет серверный
// шаблон components/alert.html (один текст или «Исправьте ошибки» + список).
// Если в ответе нет поля errors — ничего не делаем: серверные HTML-ответы
// (200) рисуют alert сами.
//
// Используется только для HTMX-запросов: обычный ответ с HTML (200) обрабатывает
// htmx, а ошибки сети/валидации (WebSocket, form-data) — вне этой зоны.

window.Errors = (function () {
  // renderAlert собирает DOM alert'а из структурированного ответа.
  function renderAlert(data) {
    var alert = document.createElement('div');
    alert.className = 'alert alert--error';
    alert.setAttribute('role', 'alert');
    alert.setAttribute('data-response-error', '');

    var errors = Array.isArray(data.errors) ? data.errors : [];
    if (errors.length === 1) {
      alert.textContent = errors[0];
      return alert;
    }

    var title = document.createElement('p');
    title.className = 'alert-title';
    title.textContent = 'Исправьте ошибки:';

    var list = document.createElement('ul');
    list.className = 'alert-list';
    errors.forEach(function (message) {
      var item = document.createElement('li');
      item.textContent = message;
      list.appendChild(item);
    });

    alert.appendChild(title);
    alert.appendChild(list);
    return alert;
  }

  // show заменяет прежний alert клиента и вставляет новый перед формой.
  function show(data, elt) {
    var container = elt && elt.parentElement;
    if (!container) return;

    var old = container.querySelector('.alert[data-response-error]');
    if (old) old.remove();

    var alert = renderAlert(data);
    if (alert) container.insertBefore(alert, elt);
  }

  document.addEventListener('htmx:afterRequest', function (event) {
    var elt = event.detail.elt;
    if (!elt || !elt.hasAttribute || !elt.hasAttribute('data-response-errors')) return;

    var data;
    try {
      data = JSON.parse(event.detail.xhr.responseText);
    } catch (err) {
      return;
    }
    if (!data || !Array.isArray(data.errors) || data.errors.length === 0) return;

    show(data, elt);
  });

  return { renderAlert: renderAlert, show: show };
})();
