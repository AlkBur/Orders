// Панель расширенного отбора списка чеков.
//
// Состояние значений (даты, суммы, организация, контрагент, статус)
// хранится на сервере и приходит в URL; панель только перерисовывается
// сервером. Alpine здесь отвечает за три локальные вещи:
//   - раскрытие/сворачивание панели (open);
//   - зависимые пикеры «организация → контрагент»: список контрагентов
//     сужается по выбранной организации;
//   - поисковый picker контрагента (модальное окно): поиск по названию
//     выполняется в уже загруженном справочнике, без запросов к серверу.
//
// Начальные данные сериализованы на сервере в атрибут data-filter
// (см. app/receipt_filter.go): JS только разбирает их, а не владеет
// источником истины.
document.addEventListener('alpine:init', () => {
  Alpine.data('receiptsFilter', (el) => ({
    open: false,
    orgs: [],
    customers: [],
    orgID: 0,
    custID: 0,

    pickerOpen: false,
    pickerQuery: '',

    init() {
      const raw = el ? el.dataset.filter : '';
      if (raw) {
        try {
          const data = JSON.parse(raw);
          this.open = !!data.open;
          this.orgID = data.orgId || 0;
          this.custID = data.customerId || 0;
          this.orgs = data.orgs || [];
          this.customers = data.customers || [];
        } catch (e) {
          // Некорректный payload — панель работает с пустыми справочниками.
        }
      }
    },

    // visibleCustomers — контрагенты, доступные при текущей организации:
    // без выбранной организации показываются все.
    get visibleCustomers() {
      if (!this.orgID) return this.customers;
      return this.customers.filter((c) => c.organization_id === this.orgID);
    },

    // onOrgChange — после смены организации контрагент, принадлежащий
    // другой организации, сбрасывается на «Не выбран». Сервер дублирует
    // эту проверку (validateReceiptFilterPair).
    onOrgChange() {
      if (!this.custID) return;
      const stillValid = this.visibleCustomers.some((c) => c.id === this.custID);
      if (!stillValid) this.custID = 0;
    },

    // selectedCustomerName — отображаемое имя выбранного контрагента.
    // Значение вычисляется из справочника, чтобы не зависеть от порядка
    // серверной отрисовки после применения фильтра.
    get selectedCustomerName() {
      if (!this.custID) return 'Не выбран';
      const c = this.customers.find((c) => c.id === this.custID);
      return c ? c.name : 'Не выбран';
    },

    // filteredPickerCustomers — контрагенты picker: доступные при текущей
    // организации (visibleCustomers), отфильтрованные по запросу поиска.
    // Поиск регистронезависимый, по подстроке названия.
    get filteredPickerCustomers() {
      const query = this.pickerQuery.trim().toLowerCase();
      if (!query) return this.visibleCustomers;
      return this.visibleCustomers.filter((c) =>
        (c.name || '').toLowerCase().includes(query)
      );
    },

    // openCustomerPicker — открыть picker контрагента. Список всегда
    // ограничен выбранной организацией; поисковый запрос сбрасывается.
    openCustomerPicker() {
      this.pickerQuery = '';
      this.pickerOpen = true;
    },

    // closeCustomerPicker — закрыть picker без изменения отбора.
    closeCustomerPicker() {
      this.pickerOpen = false;
    },

    // selectCustomer — применить выбор контрагента. Если организация ещё
    // не выбрана, она автоматически заполняется организацией контрагента,
    // чтобы серверная пара «орг → контрагент» оставалась валидной.
    selectCustomer(c) {
      this.custID = c.id;
      if (!this.orgID && c.organization_id) {
        this.orgID = c.organization_id;
      }
      this.pickerQuery = '';
      this.pickerOpen = false;
    },

    // clearCustomer — сбросить контрагента. Организация не меняется.
    clearCustomer() {
      this.custID = 0;
    },
  }));
});
