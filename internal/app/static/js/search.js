// Живой поиск списков: подавляет запросы из поля поиска (data-auto-search)
// короче минимальной длины. Отменяются только запросы, инициированные самим
// полем ввода (по мере набора); отправка формы (Enter / кнопка «Найти»)
// работает всегда.
//
// Правило: автопоиск выполняется при длине запроса >= минимальной
// или когда поле поиска очищено полностью. Промежуточные значения
// длиной 1..minLength-1 — только Enter/кнопка.
const DEFAULT_SEARCH_MIN_LENGTH = 3;

document.addEventListener('htmx:beforeRequest', function (event) {
  const elt = event.detail.elt;
  if (!elt || !elt.matches) return;

  const input = elt.matches('input[data-auto-search]') ? elt : null;
  if (!input) return;

  const raw = input.dataset.minLength;
  const minLength = raw ? parseInt(raw, 10) : DEFAULT_SEARCH_MIN_LENGTH;

  const length = input.value.trim().length;
  if (length > 0 && length < minLength) {
    event.preventDefault();
  }
});
