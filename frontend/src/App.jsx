import React, { useEffect, useMemo, useState } from "react";
import axios from "axios";
import Papa from "papaparse";
import ProductTable from "./components/ProductTable";
import OrderTable from "./components/OrderTable";
import PaymentsMonitoring from "./components/PaymentsMonitoring";
import ArticleTable from "./components/ArticleTable";
import ClientsTable from "./components/ClientsTable";
import ShippingTable from "./components/ShippingTable";
import BalanceTable from "./components/BalanceTable";

const FILTER_PRESETS_STORAGE_KEY = "wm_filter_presets_v1";

const VIEW_META = {
  products: { title: "Товары", subtitle: "Каталог и складские остатки" },
  orders: { title: "Заказы", subtitle: "Сборка и отгрузка" },
  payments: { title: "Платежи", subtitle: "Мониторинг оплат" },
  articles: { title: "Приход", subtitle: "Поставки и складской приход" },
  balance: { title: "Баланс", subtitle: "Остатки по артикулам и статусам" },
  clients: { title: "Клиенты", subtitle: "База клиентов и контактов" },
  shipping: { title: "Отправки", subtitle: "Календарь и контроль отгрузок" },
};

function readStoredFilterPresets() {
  if (typeof window === "undefined") {
    return {};
  }
  try {
    const parsed = JSON.parse(window.localStorage.getItem(FILTER_PRESETS_STORAGE_KEY) || "{}");
    if (!parsed || typeof parsed !== "object") {
      return {};
    }
    return parsed;
  } catch {
    return {};
  }
}

export default function App() {
  const [token, setToken] = useState(localStorage.getItem("token") || "");
  const [view, setView] = useState("login");
  const [username, setUsername] = useState(localStorage.getItem("username") || "");
  const [password, setPassword] = useState("");
  const [products, setProducts] = useState([]);
  const [orders, setOrders] = useState([]);
  const [articles, setArticles] = useState([]);
  const [clients, setClients] = useState([]);
  const [shipments, setShipments] = useState([]);
  const [productFilter, setProductFilter] = useState("");
  const [orderFilter, setOrderFilter] = useState("");
  const [articleFilter, setArticleFilter] = useState("");
  const [clientFilter, setClientFilter] = useState("");
  const [shippingFilter, setShippingFilter] = useState("");
  const [paymentsFilter, setPaymentsFilter] = useState("");
  const [balanceFilter, setBalanceFilter] = useState("");
  const [productStatusFilter, setProductStatusFilter] = useState("");
  const [orderStatusFilter, setOrderStatusFilter] = useState("");
  const [paymentsDateFrom, setPaymentsDateFrom] = useState("");
  const [paymentsDateTo, setPaymentsDateTo] = useState("");
  const [shippingDateFrom, setShippingDateFrom] = useState("");
  const [shippingDateTo, setShippingDateTo] = useState("");
  const [selectedPresetName, setSelectedPresetName] = useState("");
  const [filterPresetsByView, setFilterPresetsByView] = useState(readStoredFilterPresets);
  const [error, setError] = useState("");

  const authHeaders = useMemo(() => {
    const headers = { Authorization: token };
    if (username) {
      headers["X-Username"] = username;
    }
    return headers;
  }, [token, username]);

  useEffect(() => {
    if (!token) {
      setView("login");
      return;
    }

    setView("products");
    fetchProducts();
    fetchOrders();
    fetchArticles();
    fetchClients();
    fetchShipments();
  }, [token]);

  const fetchProducts = async () => {
    try {
      const res = await axios.get("/api/products", { headers: authHeaders });
      setProducts(res.data || []);
    } catch (e) {
      setError("Ошибка загрузки товаров: " + (e.response?.data?.error || e.message));
    }
  };

  const fetchOrders = async () => {
    try {
      const res = await axios.get("/api/orders", { headers: authHeaders });
      setOrders(res.data || []);
    } catch (e) {
      setError("Ошибка загрузки заказов: " + (e.response?.data?.error || e.message));
    }
  };

  const fetchArticles = async () => {
    try {
      const res = await axios.get("/api/articles", { headers: authHeaders });
      setArticles(res.data || []);
    } catch (e) {
      setError("Ошибка загрузки артикулов: " + (e.response?.data?.error || e.message));
    }
  };

  const fetchClients = async () => {
    try {
      const res = await axios.get("/api/clients", { headers: authHeaders });
      setClients(res.data || []);
    } catch (e) {
      setError("Ошибка загрузки клиентов: " + (e.response?.data?.error || e.message));
    }
  };

  const fetchShipments = async () => {
    try {
      const res = await axios.get("/api/shipments", { headers: authHeaders });
      setShipments(res.data || []);
    } catch (e) {
      setError("Ошибка загрузки отправок: " + (e.response?.data?.error || e.message));
    }
  };

  const handleLogin = async (e) => {
    e.preventDefault();
    try {
      const res = await axios.post("/login", { username, password });
      const nextToken = res.data.token;
      const nextUsername = res.data.username;

      setToken(nextToken);
      setUsername(nextUsername);
      localStorage.setItem("token", nextToken);
      localStorage.setItem("username", nextUsername);
      setPassword("");
      setError("");
    } catch (e) {
      setError("Ошибка входа: " + (e.response?.data?.error || "Unknown error"));
    }
  };

  const handleLogout = () => {
    setToken("");
    setUsername("");
    localStorage.removeItem("token");
    localStorage.removeItem("username");
    setView("login");
    setError("");
  };

  const exportToCSV = (data, filename) => {
    const csv = Papa.unparse(data);
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    const link = document.createElement("a");
    link.href = URL.createObjectURL(blob);
    link.setAttribute("download", filename);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  const inSaleCount = useMemo(() => products.filter((p) => p.status === 1).length, [products]);
  const reservedCount = useMemo(() => products.filter((p) => p.status === 2).length, [products]);
  const soldCount = useMemo(() => products.filter((p) => p.status === 3).length, [products]);
  const globalFilterByView = {
    products: {
      value: productFilter,
      setValue: setProductFilter,
      placeholder: "Глобальный фильтр по товарам",
    },
    orders: {
      value: orderFilter,
      setValue: setOrderFilter,
      placeholder: "Глобальный фильтр по заказам",
    },
    payments: {
      value: paymentsFilter,
      setValue: setPaymentsFilter,
      placeholder: "Глобальный фильтр по платежам",
    },
    articles: {
      value: articleFilter,
      setValue: setArticleFilter,
      placeholder: "Глобальный фильтр по приходу",
    },
    balance: {
      value: balanceFilter,
      setValue: setBalanceFilter,
      placeholder: "Глобальный фильтр по балансу",
    },
    clients: {
      value: clientFilter,
      setValue: setClientFilter,
      placeholder: "Глобальный фильтр по клиентам",
    },
    shipping: {
      value: shippingFilter,
      setValue: setShippingFilter,
      placeholder: "Глобальный фильтр по отправкам",
    },
  };
  const resetActiveGlobalFilters = () => {
    if (view === "products") {
      setProductFilter("");
      setProductStatusFilter("");
      return;
    }
    if (view === "orders") {
      setOrderFilter("");
      setOrderStatusFilter("");
      return;
    }
    if (view === "payments") {
      setPaymentsFilter("");
      setPaymentsDateFrom("");
      setPaymentsDateTo("");
      return;
    }
    if (view === "articles") {
      setArticleFilter("");
      return;
    }
    if (view === "clients") {
      setClientFilter("");
      return;
    }
    if (view === "balance") {
      setBalanceFilter("");
      return;
    }
    if (view === "shipping") {
      setShippingFilter("");
      setShippingDateFrom("");
      setShippingDateTo("");
    }
  };

  const buildFiltersForView = (targetView) => {
    if (targetView === "products") {
      return { query: productFilter, status: productStatusFilter };
    }
    if (targetView === "orders") {
      return { query: orderFilter, status: orderStatusFilter };
    }
    if (targetView === "payments") {
      return { query: paymentsFilter, dateFrom: paymentsDateFrom, dateTo: paymentsDateTo };
    }
    if (targetView === "articles") {
      return { query: articleFilter };
    }
    if (targetView === "clients") {
      return { query: clientFilter };
    }
    if (targetView === "balance") {
      return { query: balanceFilter };
    }
    if (targetView === "shipping") {
      return { query: shippingFilter, dateFrom: shippingDateFrom, dateTo: shippingDateTo };
    }
    return {};
  };

  const applyPresetFilters = (targetView, filters = {}) => {
    if (targetView === "products") {
      setProductFilter(filters.query || "");
      setProductStatusFilter(filters.status || "");
      return;
    }
    if (targetView === "orders") {
      setOrderFilter(filters.query || "");
      setOrderStatusFilter(filters.status || "");
      return;
    }
    if (targetView === "payments") {
      setPaymentsFilter(filters.query || "");
      setPaymentsDateFrom(filters.dateFrom || "");
      setPaymentsDateTo(filters.dateTo || "");
      return;
    }
    if (targetView === "articles") {
      setArticleFilter(filters.query || "");
      return;
    }
    if (targetView === "clients") {
      setClientFilter(filters.query || "");
      return;
    }
    if (targetView === "balance") {
      setBalanceFilter(filters.query || "");
      return;
    }
    if (targetView === "shipping") {
      setShippingFilter(filters.query || "");
      setShippingDateFrom(filters.dateFrom || filters.date || "");
      setShippingDateTo(filters.dateTo || filters.date || "");
    }
  };

  const persistFilterPresets = (nextPresets) => {
    setFilterPresetsByView(nextPresets);
    window.localStorage.setItem(FILTER_PRESETS_STORAGE_KEY, JSON.stringify(nextPresets));
  };

  const saveActiveFilterPreset = () => {
    const rawName = window.prompt("Название пресета:");
    const name = rawName ? rawName.trim() : "";
    if (!name) {
      return;
    }
    const preset = { name, filters: buildFiltersForView(view) };
    const current = filterPresetsByView[view] || [];
    const nextByView = {
      ...filterPresetsByView,
      [view]: [...current.filter((item) => item.name !== name), preset],
    };
    persistFilterPresets(nextByView);
    setSelectedPresetName(name);
  };

  const deleteActiveFilterPreset = () => {
    if (!selectedPresetName) {
      return;
    }
    const current = filterPresetsByView[view] || [];
    const nextByView = {
      ...filterPresetsByView,
      [view]: current.filter((item) => item.name !== selectedPresetName),
    };
    persistFilterPresets(nextByView);
    setSelectedPresetName("");
  };

  useEffect(() => {
    setSelectedPresetName("");
  }, [view]);

  if (view === "login") {
    return (
      <div className="wm-login-page">
        <div className="wm-login-card fade-in-scale">
          <p className="wm-kicker">Second-Hand WMS</p>
          <h1>Панель склада</h1>
          <p className="wm-login-subtitle">Рабочее место для учета товара, заказов и платежей.</p>

          {error && <p className="wm-error">{error}</p>}

          <form className="wm-login-form" onSubmit={handleLogin}>
            <input
              type="text"
              placeholder="Логин"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              autoComplete="username"
            />
            <input
              type="password"
              placeholder="Пароль"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete="current-password"
            />
            <button type="submit" className="wm-btn wm-btn-primary">
              Войти
            </button>
          </form>
        </div>
      </div>
    );
  }

  const meta = VIEW_META[view] || VIEW_META.products;
  const activeGlobalFilter = globalFilterByView[view];
  const presetsForView = filterPresetsByView[view] || [];

  return (
    <div className="wm-app-shell">
      <header className="wm-topbar fade-up">
        <div>
          <p className="wm-kicker">Second-Hand Warehouse</p>
          <h1>Система управления складом</h1>
          <p className="wm-subtitle">{meta.subtitle}</p>
        </div>
        <div className="wm-userbox">
          <span>Пользователь: {username || "unknown"}</span>
          <button onClick={handleLogout} className="wm-btn wm-btn-danger">
            Выйти
          </button>
        </div>
      </header>

      <section className="wm-stats fade-up delay-1">
        <article>
          <span>Товары</span>
          <strong>{products.length}</strong>
        </article>
        <article>
          <span>На продаже</span>
          <strong>{inSaleCount}</strong>
        </article>
        <article>
          <span>Забронировано</span>
          <strong>{reservedCount}</strong>
        </article>
        <article>
          <span>Продано</span>
          <strong>{soldCount}</strong>
        </article>
        <article>
          <span>Заказы</span>
          <strong>{orders.length}</strong>
        </article>
        <article>
          <span>Приход</span>
          <strong>{articles.length}</strong>
        </article>
        <article>
          <span>Клиенты</span>
          <strong>{clients.length}</strong>
        </article>
        <article>
          <span>Отправки</span>
          <strong>{shipments.length}</strong>
        </article>
      </section>

      <nav className="wm-tabs fade-up delay-2" aria-label="Разделы">
        <button onClick={() => setView("products")} className={view === "products" ? "is-active" : ""}>
          Товары
        </button>
        <button onClick={() => setView("orders")} className={view === "orders" ? "is-active" : ""}>
          Заказы
        </button>
        <button onClick={() => setView("payments")} className={view === "payments" ? "is-active" : ""}>
          Платежи
        </button>
        <button onClick={() => setView("articles")} className={view === "articles" ? "is-active" : ""}>
          Приход
        </button>
        <button onClick={() => setView("balance")} className={view === "balance" ? "is-active" : ""}>
          Баланс
        </button>
        <button onClick={() => setView("clients")} className={view === "clients" ? "is-active" : ""}>
          Клиенты
        </button>
        <button onClick={() => setView("shipping")} className={view === "shipping" ? "is-active" : ""}>
          Отправки
        </button>
      </nav>

      {error && <p className="wm-error fade-up delay-2">{error}</p>}

      <main className="wm-panel fade-up delay-3">
        <div className="wm-panel-head">
          <h2>{meta.title}</h2>
          {activeGlobalFilter && (
            <div className="wm-global-filters">
              <input
                type="text"
                className="wm-input w-full md:w-96"
                placeholder={activeGlobalFilter.placeholder}
                value={activeGlobalFilter.value}
                onChange={(e) => activeGlobalFilter.setValue(e.target.value)}
              />
              {view === "products" && (
                <select
                  className="wm-select"
                  value={productStatusFilter}
                  onChange={(e) => setProductStatusFilter(e.target.value)}
                >
                  <option value="">Все статусы товара</option>
                  <option value="1">На продаже</option>
                  <option value="2">Забронировано</option>
                  <option value="3">Продано</option>
                </select>
              )}
              {view === "orders" && (
                <select
                  className="wm-select"
                  value={orderStatusFilter}
                  onChange={(e) => setOrderStatusFilter(e.target.value)}
                >
                  <option value="">Все статусы заказа</option>
                  <option value="0">Новый</option>
                  <option value="1">Готов к отправке</option>
                  <option value="2">Отправлен</option>
                </select>
              )}
              {view === "payments" && (
                <>
                  <input
                    type="date"
                    className="wm-input"
                    value={paymentsDateFrom}
                    onChange={(e) => setPaymentsDateFrom(e.target.value)}
                  />
                  <input
                    type="date"
                    className="wm-input"
                    value={paymentsDateTo}
                    onChange={(e) => setPaymentsDateTo(e.target.value)}
                  />
                </>
              )}
              {view === "shipping" && (
                <>
                  <input
                    type="date"
                    className="wm-input"
                    value={shippingDateFrom}
                    onChange={(e) => setShippingDateFrom(e.target.value)}
                  />
                  <input
                    type="date"
                    className="wm-input"
                    value={shippingDateTo}
                    onChange={(e) => setShippingDateTo(e.target.value)}
                  />
                </>
              )}
              <button className="wm-btn" onClick={resetActiveGlobalFilters}>
                Сбросить
              </button>
              <select
                className="wm-select"
                value={selectedPresetName}
                onChange={(e) => {
                  const name = e.target.value;
                  setSelectedPresetName(name);
                  const preset = presetsForView.find((item) => item.name === name);
                  if (preset) {
                    applyPresetFilters(view, preset.filters);
                  }
                }}
              >
                <option value="">Пресеты фильтров</option>
                {presetsForView.map((preset) => (
                  <option key={preset.name} value={preset.name}>
                    {preset.name}
                  </option>
                ))}
              </select>
              <button className="wm-btn" onClick={saveActiveFilterPreset}>
                Сохранить пресет
              </button>
              <button
                className="wm-btn wm-btn-danger"
                onClick={deleteActiveFilterPreset}
                disabled={!selectedPresetName}
              >
                Удалить пресет
              </button>
            </div>
          )}
        </div>

        {view === "products" && (
          <ProductTable
            products={products}
            articles={articles}
            setProducts={setProducts}
            setArticles={setArticles}
            setOrders={setOrders}
            filter={productFilter}
            statusFilter={productStatusFilter}
            setFilter={setProductFilter}
            token={token}
            username={username}
            exportToCSV={exportToCSV}
          />
        )}

        {view === "orders" && (
          <OrderTable
            orders={orders || []}
            products={products}
            setOrders={setOrders}
            setShipments={setShipments}
            filter={orderFilter}
            statusFilter={orderStatusFilter}
            setFilter={setOrderFilter}
            token={token}
            username={username}
            exportToCSV={exportToCSV}
          />
        )}

        {view === "payments" && (
          <PaymentsMonitoring
            token={token}
            username={username}
            exportToCSV={exportToCSV}
            filter={paymentsFilter}
            dateFrom={paymentsDateFrom}
            dateTo={paymentsDateTo}
          />
        )}

        {view === "articles" && (
          <ArticleTable
            articles={articles}
            setArticles={setArticles}
            filter={articleFilter}
            token={token}
            username={username}
            exportToCSV={exportToCSV}
          />
        )}

        {view === "clients" && (
          <ClientsTable
            clients={clients}
            setClients={setClients}
            filter={clientFilter}
            token={token}
            username={username}
            exportToCSV={exportToCSV}
          />
        )}

        {view === "balance" && (
          <BalanceTable
            token={token}
            username={username}
            filter={balanceFilter}
            exportToCSV={exportToCSV}
          />
        )}

        {view === "shipping" && (
          <ShippingTable
            shipments={shipments}
            setShipments={setShipments}
            filter={shippingFilter}
            dateFrom={shippingDateFrom}
            dateTo={shippingDateTo}
            token={token}
            username={username}
            exportToCSV={exportToCSV}
          />
        )}
      </main>
    </div>
  );
}
