# UI Architecture

Version: 1.0

---

# Goal

Orders uses a lightweight Server Side Rendering (SSR) user interface.

The UI must:

- be fast;
- work without JavaScript;
- support desktop, tablet and mobile devices;
- have a consistent appearance;
- be PWA Ready;
- remain easy to maintain.

---

# Architecture

Rendering model:

- Server Side Rendering (SSR)
- html/template
- embed.FS

Templates are embedded into the application binary.

Static resources are served by the HTTP server.

---

# UI Stack

## Rendering

- html/template
- embed.FS

---

## CSS

Framework:

- Pico CSS

Reasons:

- Mobile First
- lightweight
- semantic HTML
- responsive
- good browser compatibility

Project styles:

```
web/static/css/orders.css
```

Pico CSS is never modified directly.

---

## Icons

Library:

- Lucide

Format:

- SVG

Icons are stored locally.

---

## JavaScript

Native JavaScript only.

Third-party JavaScript frameworks are not used.

JavaScript is optional and provides only progressive enhancement.

The application remains fully functional without JavaScript.

---

# PWA Ready

The application architecture is prepared for Progressive Web App support.

Current version does not require:

- Service Worker
- Offline mode
- Push notifications

Future versions may add these features without changing the UI architecture.

Reserved files:

```
web/static/manifest.json
web/static/sw.js
```

---

# Project Structure

```
internal/

    app/

        templates/

            layout.html

            login.html

            orders.html

            customers.html

            products.html

            users.html

            components/

                header.html
                sidebar.html
                toolbar.html
                table.html
                dialog.html
                pagination.html

web/

    static/

        css/

            pico.min.css
            orders.css

        js/

        icons/

        images/

        manifest.json

        sw.js
```

---

# Rendering

Every page is rendered through a single function.

```go
func (a *App) Render(
    w http.ResponseWriter,
    page string,
    data any,
)
```

Render is responsible for:

- loading templates;
- executing the layout;
- rendering the requested page.

Handlers never execute templates directly.

---

# Layout

All pages use a common layout.

Layout contains:

- HTML document
- Head
- Navigation
- Main content
- Footer

Only the page content changes.

---

# Components

The interface is built from reusable components.

Core components:

- Button
- Toolbar
- Input
- Select
- Checkbox
- Table
- Dialog
- Pagination
- Menu
- Alert

Components contain no business logic.

Components receive rendering data only.

---

# Forms

General rules:

- labels above controls;
- full width on mobile devices;
- consistent spacing;
- identical buttons.

---

# Tables

Tables are one of the primary UI elements.

Supported features:

- sorting
- filtering
- paging
- responsive layout

All tables use the same appearance.

---

# Navigation

Desktop:

Left:

- Orders
- Customers
- Products
- Users

Right:

- Current user
- Logout

Mobile:

Navigation collapses into a menu button.

---

# Themes

Supported themes:

- Default

Future:

- Dark

Themes affect only CSS.

Templates remain unchanged.

---

# Browser Support

Supported browsers:

- Chrome
- Edge
- Firefox
- Safari

Responsive targets:

- Desktop
- Tablet
- Mobile

The UI follows the Mobile First approach.

---

# Static Assets

All static assets are bundled with the application.

External CDN resources are not used.

Assets include:

- CSS
- JavaScript
- Fonts
- Icons
- Images

The application is fully self-contained.

---

# Design Principles

The UI follows these principles:

- Server Side Rendering
- Mobile First
- Progressive Enhancement
- PWA Ready
- Semantic HTML
- Reusable Components
- Minimal JavaScript
- Local Static Assets
- Consistent Layout

# UI Philosophy

The server is the single source of truth.
HTML is generated on the server.
The browser is responsible only for presentation and user interaction.
Business logic never executes in the browser.