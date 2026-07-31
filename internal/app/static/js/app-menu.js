(function () {
  var root = document.documentElement;
  root.classList.add('js');

  var menus = document.querySelectorAll('[data-app-menu]');
  for (var i = 0; i < menus.length; i++) {
    (function (menu) {
      var toggle = menu.querySelector('[data-app-menu-toggle]');
      var panel = menu.querySelector('[data-app-menu-panel]');

      function close() {
        menu.classList.remove('is-open');
        if (toggle) {
          toggle.setAttribute('aria-expanded', 'false');
        }
      }

      if (toggle) {
        toggle.addEventListener('click', function (e) {
          e.stopPropagation();
          var open = menu.classList.toggle('is-open');
          toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
        });
      }

      document.addEventListener('click', function (e) {
        if (!menu.contains(e.target)) {
          close();
        }
      });

      document.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') {
          close();
        }
      });

      if (panel) {
        panel.addEventListener('click', close);
      }
    })(menus[i]);
  }
})();
