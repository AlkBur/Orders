// Живой поиск списков: подавляет запросы из поля поиска (data-auto-search)
// короче минимальной длины. Отменяются только запросы, инициированные самим
// полем ввода (по мере набора); отправка формы (Enter / кнопка «Найти»)
// работает всегда.
const autoSearchMinLength = 3;

document.addEventListener('htmx:beforeRequest', function (event) {
  const elt = event.detail.elt;
  if (!elt || !elt.matches) return;

  const input = elt.matches('input[data-auto-search]') ? elt : null;
  if (!input) return;

  if (input.value.trim().length < autoSearchMinLength) {
    event.preventDefault();
  }
});
