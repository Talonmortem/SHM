import React, { useState, useEffect, useCallback, useMemo } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const PRODUCT_COLUMNS_STORAGE_KEY = "wm_products_columns_v1";

const DEFAULT_PRODUCT_COLUMN_WIDTHS = {
  status: 160,
  pick: 72,
  name: 200,
  article: 100,
  video: 150,
  cursEvro: 100,
  priceEvro: 100,
  weight: 100,
  sumEvro: 100,
  sumRub: 100,
  skidka: 100,
  summaRubSoSkidkoj: 150,
  count: 100,
  onePrice: 100,
  actions: 150,
};

function createEmptyProduct() {
  return {
    id: null,
    status: 1,
    name: "",
    articlesInProduct: [],
    video: "",
    description: "",
    skidka: "0",
    weight: "0",
    count: 0,
    summaRubSoSkidkoj: "0",
    onePrice: "0",
  };
}

function getArticlesInProduct(value) {
  return Array.isArray(value) ? value : [];
}

export default function ProductTable({ products, articles = [], setProducts, setArticles, filter, statusFilter, setFilter, token, username, exportToCSV, setOrders }) {
  const [editingProductId, setEditingProductId] = useState(null);
  const [newProduct, setNewProduct] = useState(createEmptyProduct);
  const [showAddForm, setShowAddForm] = useState(false);
  const [selectedProducts, setSelectedProducts] = useState([]);
  const [showOrderModal, setShowOrderModal] = useState(false);
  const [newOrderName, setNewOrderName] = useState("");
  const [newOrderDescription, setNewOrderDescription] = useState("");
  const [payments, setPayments] = useState([{ method: "", amount: "", comment: "" }]);
  const [paymentMethods, setPaymentMethods] = useState([]);
  const [showConfirmStep, setShowConfirmStep] = useState(false);
  const [lastClickedIndex, setLastClickedIndex] = useState(null);
  const { columnWidths, setColumnWidths, handleResizeStart } = useResizableColumns(PRODUCT_COLUMNS_STORAGE_KEY, DEFAULT_PRODUCT_COLUMN_WIDTHS);
  const [articleSearchByIndex, setArticleSearchByIndex] = useState({});

  const headers = { Authorization: token, "X-Username": username };

  const statusOptions = [
    { value: 1, label: "На продаже" },
    { value: 2, label: "Забронировано" },
    { value: 3, label: "Продано" },
  ];

  const statusBadgeByValue = {
    1: { label: "На продаже", className: "wm-status-sale" },
    2: { label: "Забронировано", className: "wm-status-reserved" },
    3: { label: "Продано", className: "wm-status-sold" },
  };

  useEffect(() => {
    console.log("Products:", products);
  }, [products]);

  useEffect(() => {
    const fetchPaymentMethods = async () => {
      try {
        const response = await axios.get("/api/payment_methods", { headers });
        setPaymentMethods(response.data.map((method) => ({ value: method.method, label: method.method })));
      } catch (error) {
        console.error("Error fetching payment methods:", error);
        alert("Error fetching payment methods: " + (error.response?.data?.error || "Unknown error"));
      }
    };
    fetchPaymentMethods();
  }, [token, username]);

  useEffect(() => {
    if (!showAddForm || editingProductId) {
      return;
    }

    let cancelled = false;

    const fillGeneratedName = async () => {
      try {
        const response = await axios.get("/api/products/generate-name", {
          headers: { Authorization: token, "X-Username": username },
        });
        const generatedName = (response.data?.name || "").trim();
        if (!generatedName || cancelled) {
          return;
        }
        setNewProduct((prev) => (prev.name ? prev : { ...prev, name: generatedName }));
      } catch (error) {
        console.error("Error generating product name:", error);
      }
    };

    fillGeneratedName();

    return () => {
      cancelled = true;
    };
  }, [showAddForm, editingProductId, token, username]);

  const resetProductForm = useCallback(() => {
    setShowAddForm(false);
    setEditingProductId(null);
    setNewProduct(createEmptyProduct());
    setArticleSearchByIndex({});
  }, []);

  const handleEditProduct = useCallback((product) => {
    setEditingProductId(product.id);
    setNewProduct({ ...product, articlesInProduct: [...getArticlesInProduct(product?.articlesInProduct)] });
    setArticleSearchByIndex({});
    setShowAddForm(true);
  }, []);

  const refreshArticles = useCallback(async () => {
    if (typeof setArticles !== "function") {
      return;
    }
    const response = await axios.get("/api/articles", { headers });
    setArticles(response.data || []);
  }, [headers, setArticles]);

  const handleSaveProduct = useCallback(async () => {
    try {
      console.log("Saving product:", newProduct);
      const response = await axios.put(`/api/products/${newProduct.id}`, newProduct, { headers });
      setProducts(products.map((p) => (p.id === newProduct.id ? { ...response.data } : p)));
      if (typeof setOrders === "function") {
        const ordersResponse = await axios.get("/api/orders", { headers });
        setOrders(ordersResponse.data || []);
      }
      await refreshArticles();
      resetProductForm();
    } catch (error) {
      console.error("Error updating product:", error);
      alert("Error updating product: " + (error.response?.data?.error || "Unknown error"));
    }
  }, [newProduct, products, setProducts, headers, refreshArticles, resetProductForm, setOrders]);

  const handleAddProduct = useCallback(async () => {
    try {
      console.log("Adding product:", newProduct);
      const response = await axios.post("/api/products", newProduct, { headers });
      setProducts([...products, response.data]);
      await refreshArticles();
      resetProductForm();
    } catch (error) {
      console.error("Error adding product:", error);
      alert("Error adding product: " + (error.response?.data?.error || "Unknown error"));
    }
  }, [newProduct, products, setProducts, headers, refreshArticles, resetProductForm]);

  const handleDeleteProduct = useCallback(async (id) => {
    try {
      console.log("Deleting product:", id);
      await axios.delete(`/api/products/${id}`, { headers });
      setProducts(products.filter((p) => p.id !== id));
      await refreshArticles();
    } catch (error) {
      console.error("Error deleting product:", error);
      alert("Error deleting product: " + (error.response?.data?.error || "Unknown error"));
    }
  }, [products, setProducts, headers, refreshArticles]);

  const handleCreateOrder = useCallback(async () => {
    try {
      console.log("Creating order with products:", selectedProducts, "Name:", newOrderName, "Description:", newOrderDescription, "Payments:", payments);
      const formattedPayments = payments
        .filter((p) => p.method && p.amount)
        .map((p) => ({ method: p.method, amount: parseFloat(p.amount), comment: p.comment }));
      const response = await axios.post(
        "/api/orders",
        {
          components: selectedProducts.map((id) => products.find((p) => p.id === id)),
          quantity: selectedProducts.length,
          status: 0,
          name: newOrderName,
          description: newOrderDescription,
          payments: formattedPayments,
        },
        { headers }
      );
      setOrders((prev) => [...prev, response.data]);
      setSelectedProducts([]);
      setNewOrderName("");
      setNewOrderDescription("");
      setPayments([{ method: "", amount: "", comment: "" }]);
      setShowOrderModal(false);
      setShowConfirmStep(false);
      setLastClickedIndex(null);
      setProducts(products.map((p) => (selectedProducts.includes(p.id) ? { ...p, status: 2 } : p)));
    } catch (error) {
      console.error("Error creating order:", error);
      alert("Error creating order: " + (error.response?.data?.error || "Unknown error"));
    }
  }, [selectedProducts, newOrderName, newOrderDescription, payments, products, setOrders, headers, setProducts]);

  const filteredProducts = useMemo(() => {
    console.log("Raw products:", products);
    const lowerFilter = filter.toLowerCase();
    const filtered = (products || []).filter(
      (p) =>
        (statusFilter === "" || p.status?.toString() === statusFilter) &&
        (lowerFilter === "" ||
          p.name?.toLowerCase().includes(lowerFilter) ||
          p.description?.toLowerCase().includes(lowerFilter) ||
          p.skidka?.toLowerCase().includes(lowerFilter) ||
          p.summaRubSoSkidkoj?.toLowerCase().includes(lowerFilter) ||
          p.weight?.toString().includes(lowerFilter) ||
          p.count?.toString().includes(lowerFilter) ||
          p.onePrice?.toLowerCase().includes(lowerFilter) ||
          statusOptions.find((opt) => opt.value === p.status)?.label.toLowerCase().includes(lowerFilter) ||
          getArticlesInProduct(p.articlesInProduct).some((a) =>
            a.article.toString().includes(lowerFilter) ||
            String((articles || []).find((item) => Number(item.serviceId || item.id) === Number(a.article))?.code || "").toLowerCase().includes(lowerFilter) ||
            a.cursEvro.includes(lowerFilter) ||
            a.priceEvro.includes(lowerFilter) ||
            a.weight.includes(lowerFilter) ||
            a.sumEvro.includes(lowerFilter) ||
            a.sumRub.includes(lowerFilter)
          ))
    ).sort((a, b) => a.id - b.id);
    console.log("Filtered Products:", filtered);
    return filtered;
  }, [products, statusFilter, filter, statusOptions, articles]);

  const handleRowClick = useCallback((productId, index, e) => {
    const product = products.find((p) => p.id === productId);
    if (product.status !== 1) return;
    if (e.shiftKey && lastClickedIndex !== null) {
      const start = Math.min(lastClickedIndex, index);
      const end = Math.max(lastClickedIndex, index);
      const range = filteredProducts.slice(start, end + 1)
        .filter((p) => p.status === 1)
        .map((p) => p.id);
      setSelectedProducts([...new Set([...selectedProducts, ...range])]);
    } else {
      setSelectedProducts((prev) =>
        prev.includes(productId)
          ? prev.filter((id) => id !== productId)
          : [...prev, productId]
      );
      setLastClickedIndex(index);
    }
  }, [selectedProducts, lastClickedIndex, filteredProducts, products]);

  const handlePaymentChange = useCallback((index, field, value) => {
    const updatedPayments = [...payments];
    updatedPayments[index] = { ...updatedPayments[index], [field]: value };
    setPayments(updatedPayments);
  }, [payments]);

  const addPayment = useCallback(() => {
    setPayments([...payments, { method: "", amount: "", comment: "" }]);
  }, [payments]);

  const removePayment = useCallback((index) => {
    if (payments.length > 1) {
      setPayments(payments.filter((_, i) => i !== index));
    }
  }, [payments]);

  const buildProductTotals = useCallback((updatedArticles, skidkaValue) => {
    const totalWeight = updatedArticles.reduce((sum, a) => sum + (parseFloat(a.weight) || 0), 0);
    const totalSumRub = updatedArticles.reduce((sum, a) => sum + (parseFloat(a.sumRub) || 0), 0);
    const skidkaPercent = parseFloat((skidkaValue || "").toString().replace("%", "")) / 100 || 0;
    const summaRubSoSkidkoj = (totalSumRub * (1 - skidkaPercent)).toFixed(2);
    const onePriceProduct = totalWeight !== 0 ? (parseFloat(summaRubSoSkidkoj) / totalWeight).toFixed(2) : "0.00";
    return { totalWeight: totalWeight.toFixed(2), summaRubSoSkidkoj, onePriceProduct };
  }, []);

  const updateArticle = (index, field, value) => {
    const updatedArticles = [...getArticlesInProduct(newProduct.articlesInProduct)];
    updatedArticles[index] = { ...updatedArticles[index], [field]: value };

    const a = updatedArticles[index] || {};
    const priceEvro = parseFloat(a.priceEvro) || 0;
    const cursEvro = parseFloat(a.cursEvro) || 0;
    const weight = parseFloat(a.weight) || 0;

    // Validate inputs
    if (priceEvro < 0 || cursEvro < 0) {
      alert("Values cannot be negative");
      return;
    }

    const sumEvro = (priceEvro * weight).toFixed(2);
    const sumRub = (parseFloat(sumEvro) * cursEvro).toFixed(2);

    updatedArticles[index] = {
      ...updatedArticles[index],
      sumEvro,
      sumRub,
    };
    const { totalWeight, summaRubSoSkidkoj, onePriceProduct } = buildProductTotals(updatedArticles, newProduct.skidka);

    setNewProduct({
      ...newProduct,
      articlesInProduct: updatedArticles,
      weight: totalWeight,
      summaRubSoSkidkoj,
      onePrice: onePriceProduct,
    });
  };

  const handleArticleSelect = useCallback((index, rawArticleServiceId) => {
    const articleServiceID = parseInt(String(rawArticleServiceId), 10);
    if (Number.isNaN(articleServiceID)) {
      return;
    }
    const selected = (articles || []).find((a) => Number(a.serviceId) === articleServiceID);
    if (!selected) {
      updateArticle(index, "article", articleServiceID);
      return;
    }

    const updatedArticles = [...getArticlesInProduct(newProduct.articlesInProduct)];
    const current = updatedArticles[index] || {};
    const nextArticle = {
      ...current,
      article: articleServiceID,
      cursEvro: String(selected.euro ?? current.cursEvro ?? "0"),
      priceEvro: String(selected.value ?? current.priceEvro ?? "0"),
      weight: String(selected.kg ?? current.weight ?? "0"),
    };

    const priceEvro = parseFloat(nextArticle.priceEvro) || 0;
    const cursEvro = parseFloat(nextArticle.cursEvro) || 0;
    const weight = parseFloat(nextArticle.weight) || 0;
    const sumEvro = (priceEvro * weight).toFixed(2);
    const sumRub = (parseFloat(sumEvro) * cursEvro).toFixed(2);

    updatedArticles[index] = {
      ...nextArticle,
      sumEvro,
      sumRub,
    };

    const { totalWeight, summaRubSoSkidkoj, onePriceProduct } = buildProductTotals(updatedArticles, newProduct.skidka);
    setNewProduct({
      ...newProduct,
      articlesInProduct: updatedArticles,
      weight: totalWeight,
      summaRubSoSkidkoj,
      onePrice: onePriceProduct,
    });
  }, [articles, buildProductTotals, newProduct, updateArticle]);

  const addArticle = () => {
    setNewProduct({
      ...newProduct,
      articlesInProduct: [
        ...getArticlesInProduct(newProduct.articlesInProduct),
        {
          id: null,
          article: 0,
          cursEvro: "0",
          priceEvro: "0",
          weight: "0",
          sumEvro: "0",
          sumRub: "0",
        },
      ],
    });
  };

  const removeArticle = (index) => {
    const updatedArticles = getArticlesInProduct(newProduct.articlesInProduct).filter((_, i) => i !== index);
    const { totalWeight, summaRubSoSkidkoj, onePriceProduct } = buildProductTotals(updatedArticles, newProduct.skidka);
    setArticleSearchByIndex((prev) => {
      const next = {};
      Object.keys(prev).forEach((k) => {
        const oldIdx = Number(k);
        if (oldIdx < index) next[oldIdx] = prev[oldIdx];
        if (oldIdx > index) next[oldIdx - 1] = prev[oldIdx];
      });
      return next;
    });

    setNewProduct({
      ...newProduct,
      articlesInProduct: updatedArticles,
      weight: totalWeight,
      summaRubSoSkidkoj,
      onePrice: onePriceProduct,
    });
  };

  const getFilteredArticles = useCallback((searchValue) => {
    const normalized = (searchValue || "").toLowerCase().trim();
    if (!normalized) {
      return articles || [];
    }
    return (articles || []).filter((a) => {
      const code = String(a.code ?? "");
      const description = (a.description || "").toLowerCase();
      return code.includes(normalized) || description.includes(normalized);
    });
  }, [articles]);

  const articlesByServiceId = useMemo(() => {
    const map = new Map();
    (articles || []).forEach((a) => {
      const key = Number(a.serviceId ?? a.id);
      if (!Number.isNaN(key)) {
        map.set(key, a);
      }
    });
    return map;
  }, [articles]);

  const columns = [
    { key: "name", label: "Name", width: columnWidths.name },
    { key: "article", label: "Article", width: columnWidths.article },
    { key: "video", label: "Video", width: columnWidths.video },
    { key: "cursEvro", label: "CursEvro", width: columnWidths.cursEvro },
    { key: "priceEvro", label: "PriceEvro", width: columnWidths.priceEvro },
    { key: "weight", label: "Weight", width: columnWidths.weight },
    { key: "sumEvro", label: "SumEvro", width: columnWidths.sumEvro },
    { key: "sumRub", label: "SumRub", width: columnWidths.sumRub },
    { key: "skidka", label: "Skidka", width: columnWidths.skidka },
    { key: "summaRubSoSkidkoj", label: "SummaRubSoSkidkoj", width: columnWidths.summaRubSoSkidkoj },
    { key: "count", label: "Count", width: columnWidths.count },
    { key: "onePrice", label: "OnePrice", width: columnWidths.onePrice },
    { key: "actions", label: "Actions", width: columnWidths.actions },
  ];

  // Flatten products and articles for table display
  const tableRows = useMemo(() => {
    const rows = [];
    filteredProducts.forEach((product) => {
      getArticlesInProduct(product.articlesInProduct).forEach((article, index) => {
        const linkedArticle = articlesByServiceId.get(Number(article.article));
        rows.push({
          productId: product.id,
          isProductRow: index === 0,
          status: product.status,
          name: index === 0 ? product.name : "",
          article: linkedArticle?.code ?? article.article,
          video: index === 0 ? product.video : "",
          cursEvro: article.cursEvro,
          priceEvro: article.priceEvro,
          weight: article.weight,
          sumEvro: article.sumEvro,
          sumRub: article.sumRub,
          skidka: index === 0 ? product.skidka : "",
          summaRubSoSkidkoj: index === 0 ? product.summaRubSoSkidkoj : "",
          count: index === 0 ? product.count : "",
          onePrice: index === 0 ? product.onePrice : "",
        });
      });
    });
    return rows;
  }, [filteredProducts, articlesByServiceId]);

  return (
    <div className="wm-surface">
      <div className="wm-toolbar">
        <div className="flex gap-2">
          <span className="px-3 py-1 bg-blue-50 text-blue-700 rounded border border-blue-200">
            Selected: {selectedProducts.length}
          </span>
          <button
            onClick={() => setShowOrderModal(true)}
            className="wm-btn wm-btn-primary"
            disabled={selectedProducts.length === 0}
          >
            Create Order from Selected
          </button>
          <button
            onClick={() => {
              setEditingProductId(null);
              setNewProduct(createEmptyProduct());
              setArticleSearchByIndex({});
              setShowAddForm(true);
            }}
            className="wm-btn wm-btn-primary"
          >
            Add Product
          </button>
          <button
            onClick={() =>
              exportToCSV(
                (products || []).map((p) => ({
                  id: p.id,
                  status: statusOptions.find((opt) => opt.value === p.status)?.label || "Unknown",
                  name: p.name,
                  video: p.video,
                  description: p.description,
                  weight: p.weight,
                  count: p.count,
                  skidka: p.skidka,
                  summaRubSoSkidkoj: p.summaRubSoSkidkoj,
                  onePrice: p.onePrice,
                  articlesInProduct: JSON.stringify(p.articlesInProduct),
                })),
                "products.csv"
              )
            }
            className="wm-btn wm-btn-primary"
          >
            Export to CSV
          </button>
          <button
            onClick={() => {
              setFilter("");
              setColumnWidths(DEFAULT_PRODUCT_COLUMN_WIDTHS);
              window.localStorage.removeItem(PRODUCT_COLUMNS_STORAGE_KEY);
            }}
            className="wm-btn"
          >
            Reset table view
          </button>
        </div>
      </div>

      {showAddForm && (
        <div
          className="fixed inset-0 bg-slate-900/30 backdrop-blur-[2px] flex items-center justify-center z-50"
          onClick={(e) => {
            if (e.target === e.currentTarget) {
              resetProductForm();
            }
          }}
        >
          <div className="bg-white p-6 rounded-xl shadow-xl w-full max-w-4xl max-h-[85vh] overflow-y-auto">
            <h2 className="text-xl font-bold mb-4">{editingProductId ? "Edit Product" : "Add New Product"}</h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm mb-1">Name</label>
                <input
                  type="text"
                  value={newProduct.name}
                  onChange={(e) => setNewProduct({ ...newProduct, name: e.target.value })}
                  className="wm-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">Video</label>
                <input
                  type="text"
                  value={newProduct.video}
                  onChange={(e) => setNewProduct({ ...newProduct, video: e.target.value })}
                  className="wm-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">Description</label>
                <input
                  type="text"
                  value={newProduct.description}
                  onChange={(e) => setNewProduct({ ...newProduct, description: e.target.value })}
                  className="wm-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">Status</label>
                <select
                  value={newProduct.status}
                  onChange={(e) => setNewProduct({ ...newProduct, status: parseInt(e.target.value) })}
                  className="wm-select w-full"
                >
                  {statusOptions.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm mb-1">Skidka (%)</label>
                <input
                  type="text"
                  value={newProduct.skidka}
                  onChange={(e) => {
                    const value = e.target.value;
                    if (value.match(/^\d*\.?\d*%?$/)) {
                      const updatedArticles = [...getArticlesInProduct(newProduct.articlesInProduct)];
                      const { totalWeight, summaRubSoSkidkoj, onePriceProduct } = buildProductTotals(updatedArticles, value);
                      setNewProduct({
                        ...newProduct,
                        skidka: value,
                        weight: totalWeight,
                        summaRubSoSkidkoj,
                        onePrice: onePriceProduct,
                      });
                    }
                  }}
                  className="wm-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">SummaRubSoSkidkoj (calc)</label>
                <input
                  type="text"
                  value={newProduct.summaRubSoSkidkoj}
                  readOnly
                  className="wm-input w-full bg-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">Weight (calc)</label>
                <input
                  type="text"
                  value={newProduct.weight}
                  readOnly
                  className="wm-input w-full bg-gray-100"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">Count (manual)</label>
                <input
                  type="number"
                  value={newProduct.count}
                  onChange={(e) => setNewProduct({ ...newProduct, count: parseInt(e.target.value, 10) || 0 })}
                  className="wm-input w-full"
                />
              </div>
              <div>
                <label className="block text-sm mb-1">OnePrice (calc)</label>
                <input
                  type="text"
                  value={newProduct.onePrice}
                  readOnly
                  className="wm-input w-full bg-gray-100"
                />
              </div>
            </div>
            <div className="mt-4">
              <h3 className="text-lg font-bold mb-2">Articles in Product</h3>
              {getArticlesInProduct(newProduct.articlesInProduct).map((article, index) => (
                <div key={index} className="border p-2 mb-2 rounded bg-white">
                  <div className="grid grid-cols-1 md:grid-cols-4 gap-2">
                    <div>
                      <label className="block text-sm mb-1">Article filter</label>
                      <input
                        type="text"
                        value={articleSearchByIndex[index] || ""}
                        onChange={(e) => setArticleSearchByIndex((prev) => ({ ...prev, [index]: e.target.value }))}
                        placeholder="Code or description..."
                        className="wm-input w-full"
                      />
                    </div>
                    <div>
                      <label className="block text-sm mb-1">Article (code + description)</label>
                      <select
                        value={article.article || ""}
                        onChange={(e) => handleArticleSelect(index, e.target.value)}
                        className="wm-select w-full"
                      >
                        <option value="">Select article...</option>
        {getFilteredArticles(articleSearchByIndex[index]).map((a) => (
                          <option key={a.serviceId || a.id} value={a.serviceId || a.id}>
                            {a.code} - {a.description} (stock kg: {a.kg ?? 0})
                          </option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-sm mb-1">CursEvro</label>
                      <input
                        type="text"
                        value={article.cursEvro}
                        onChange={(e) => updateArticle(index, "cursEvro", e.target.value)}
                        className="wm-input w-full"
                      />
                    </div>
                    <div>
                      <label className="block text-sm mb-1">PriceEvro</label>
                      <input
                        type="text"
                        value={article.priceEvro}
                        onChange={(e) => updateArticle(index, "priceEvro", e.target.value)}
                        className="wm-input w-full"
                      />
                    </div>
                    <div>
                      <label className="block text-sm mb-1">Weight</label>
                      <input
                        type="text"
                        value={article.weight}
                        onChange={(e) => updateArticle(index, "weight", e.target.value)}
                        className="wm-input w-full"
                      />
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 mt-6">
                        Available kg now: {articles.find((a) => Number(a.serviceId || a.id) === Number(article.article))?.kg ?? 0}
                      </p>
                    </div>
                    <div>
                      <label className="block text-sm mb-1">SumEvro (calc)</label>
                      <input type="text" value={article.sumEvro} readOnly className="w-full p-1 border rounded bg-gray-100" />
                    </div>
                    <div>
                      <label className="block text-sm mb-1">SumRub (calc)</label>
                      <input type="text" value={article.sumRub} readOnly className="w-full p-1 border rounded bg-gray-100" />
                    </div>
                  </div>
                  <button
                    onClick={() => removeArticle(index)}
                    className="wm-btn wm-btn-danger"
                  >
                    Remove Article
                  </button>
                </div>
              ))}
              <button
                onClick={addArticle}
                className="wm-btn wm-btn-primary"
              >
                Add Article
              </button>
            </div>
            <div className="flex justify-end gap-2 mt-4">
              <button
                onClick={editingProductId ? handleSaveProduct : handleAddProduct}
                className="wm-btn wm-btn-primary"
              >
                {editingProductId ? "Save Product" : "Add Product"}
              </button>
              <button
                onClick={resetProductForm}
                className="wm-btn"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {showOrderModal && (
        <div className="fixed inset-0 bg-slate-900/30 backdrop-blur-[2px] flex items-center justify-center z-50">
          <div className="bg-white p-6 rounded-xl shadow-xl w-full max-w-2xl">
            <h2 className="text-xl font-bold mb-4">{showConfirmStep ? "Confirm Order" : "Create New Order"}</h2>
            {!showConfirmStep ? (
              <>
                <div className="mb-4">
                  <label className="block text-sm mb-1">Name</label>
                  <input
                    type="text"
                    value={newOrderName}
                    onChange={(e) => setNewOrderName(e.target.value)}
                    className="wm-input w-full"
                  />
                </div>
                <div className="mb-4">
                  <label className="block text-sm mb-1">Order Description</label>
                  <input
                    type="text"
                    value={newOrderDescription}
                    onChange={(e) => setNewOrderDescription(e.target.value)}
                    className="wm-input w-full"
                  />
                </div>
                <div className="mb-4">
                  <label className="block text-sm mb-1">Payments</label>
                  {payments.map((payment, index) => (
                    <div key={index} className="flex mb-2 space-x-2">
                      <select
                        value={payment.method}
                        onChange={(e) => handlePaymentChange(index, "method", e.target.value)}
                        className="wm-select w-1/3"
                      >
                        <option value="">Select Payment Method</option>
                        {paymentMethods.map((option) => (
                          <option key={option.value} value={option.value}>
                            {option.label}
                          </option>
                        ))}
                      </select>
                      <input
                        type="number"
                        placeholder="Amount"
                        value={payment.amount}
                        onChange={(e) => handlePaymentChange(index, "amount", e.target.value)}
                        className="wm-input w-1/3"
                      />
                      <input
                        type="text"
                        placeholder="Comment"
                        value={payment.comment}
                        onChange={(e) => handlePaymentChange(index, "comment", e.target.value)}
                        className="wm-input w-1/3"
                      />
                      {payments.length > 1 && (
                        <button
                          onClick={() => removePayment(index)}
                          className="wm-btn wm-btn-danger"
                        >
                          Remove
                        </button>
                      )}
                    </div>
                  ))}
                  <button
                    onClick={addPayment}
                    className="wm-btn wm-btn-primary"
                  >
                    Add Payment
                  </button>
                </div>
                <div className="mb-4">
                  <label className="block text-sm mb-1">Selected Products</label>
                  <div className="max-h-64 overflow-y-auto border rounded p-2">
                    {products.filter((p) => selectedProducts.includes(p.id)).map((product) => (
                      <div key={product.id} className="p-1 bg-blue-100">
                        <span>{product.name}</span>
                      </div>
                    ))}
                  </div>
                  <p className="text-sm text-gray-500 mt-2">Click rows in table to select products. Hold Shift to select a range.</p>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => {
                      if (selectedProducts.length > 0) {
                        setShowConfirmStep(true);
                      } else {
                        alert("Please select at least one product.");
                      }
                    }}
                    className="wm-btn wm-btn-primary"
                  >
                    Next
                  </button>
                  <button
                    onClick={() => {
                      setShowOrderModal(false);
                      setSelectedProducts([]);
                      setNewOrderName("");
                      setNewOrderDescription("");
                      setPayments([{ method: "", amount: "", comment: "" }]);
                      setShowConfirmStep(false);
                      setLastClickedIndex(null);
                    }}
                    className="wm-btn"
                  >
                    Cancel
                  </button>
                </div>
              </>
            ) : (
              <>
                <div className="mb-4">
                  <h3 className="font-bold">Order Summary</h3>
                  <p><strong>Name:</strong> {newOrderName || "None"}</p>
                  <p><strong>Description:</strong> {newOrderDescription || "None"}</p>
                  <p><strong>Payments:</strong></p>
                  <ul className="list-disc pl-5">
                    {payments
                      .filter((p) => p.method && p.amount)
                      .map((p, i) => (
                        <li key={i}>{`${p.method}: ${p.amount} (Comment: ${p.comment || "None"})`}</li>
                      ))}
                  </ul>
                  <p><strong>Selected Products:</strong></p>
                  <ul className="list-disc pl-5">
                    {selectedProducts.map((id) => {
                      const product = products.find((p) => p.id === id);
                      return <li key={id}>{product?.name || "Unknown"}</li>;
                    })}
                  </ul>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={handleCreateOrder}
                    className="wm-btn wm-btn-primary"
                  >
                    Confirm Order
                  </button>
                  <button
                    onClick={() => setShowConfirmStep(false)}
                    className="wm-btn"
                  >
                    Back
                  </button>
                  <button
                    onClick={() => {
                      setShowOrderModal(false);
                      setSelectedProducts([]);
                      setNewOrderName("");
                      setNewOrderDescription("");
                      setPayments([{ method: "", amount: "", comment: "" }]);
                      setShowConfirmStep(false);
                      setLastClickedIndex(null);
                    }}
                    className="wm-btn"
                  >
                    Cancel
                  </button>
                </div>
              </>
            )}
          </div>
        </div>
      )}

      <div className="wm-table-wrap max-h-[500px]">
        <table className="wm-table text-[12px]">
          <thead>
            <tr>
              <th className="wm-th relative" style={{ width: columnWidths.status }}>
                Status
                <div
                  className="wm-col-resize"
                  onMouseDown={(e) => handleResizeStart("status", e)}
                />
              </th>
              <th className="wm-th relative" style={{ width: columnWidths.pick }}>
                Pick
                <div
                  className="wm-col-resize"
                  onMouseDown={(e) => handleResizeStart("pick", e)}
                />
              </th>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`wm-th relative text-[12px] ${column.key === "actions" ? "wm-action-cell" : ""}`}
                  style={{ width: column.width }}
                >
                  {column.label}
                  <div
                    className="wm-col-resize"
                    onMouseDown={(e) => handleResizeStart(column.key, e)}
                  />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {tableRows.map((row, index) => (
              <tr
                key={`${row.productId}-${index}`}
                className={selectedProducts.includes(row.productId) ? "bg-blue-50" : ""}
                onClick={(e) => handleRowClick(row.productId, index, e)}
                style={{ cursor: row.status === 1 ? "pointer" : "default" }}
              >
                <td className="wm-td">
                  {row.isProductRow && (() => {
                    const badge = statusBadgeByValue[row.status] || {
                      label: "Неизвестно",
                      className: "wm-status-new",
                    };
                    return (
                      <span className={`wm-status-badge ${badge.className}`}>
                        {badge.label}
                      </span>
                    );
                  })()}
                </td>
                <td className="wm-td">
                  {row.isProductRow && (
                    <input
                      type="checkbox"
                      checked={selectedProducts.includes(row.productId)}
                      disabled={row.status !== 1}
                      onClick={(e) => e.stopPropagation()}
                      onChange={(e) => handleRowClick(row.productId, index, e)}
                    />
                  )}
                </td>
                <td className="wm-td">{row.name}</td>
                <td className="wm-td">{row.article}</td>
                <td className="wm-td">
                  {row.video && (
                    <a
                      href={row.video}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-blue-500"
                      onClick={(e) => e.stopPropagation()}
                    >
                      {row.video}
                    </a>
                  )}
                </td>
                <td className="wm-td">{row.cursEvro}</td>
                <td className="wm-td">{row.priceEvro}</td>
                <td className="wm-td">{row.weight}</td>
                <td className="wm-td">{row.sumEvro}</td>
                <td className="wm-td">{row.sumRub}</td>
                <td className="wm-td">{row.skidka}</td>
                <td className="wm-td">{row.summaRubSoSkidkoj}</td>
                <td className="wm-td">{row.count}</td>
                <td className="wm-td">{row.onePrice}</td>
                <td className="wm-td wm-action-cell">
                  {row.isProductRow && (
                    <div className="wm-action-buttons">
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleEditProduct(products.find((p) => p.id === row.productId));
                        }}
                        className="wm-btn"
                      >
                        Edit
                      </button>
                      <button
                        onClick={(e) => {
                          e.stopPropagation();
                          handleDeleteProduct(row.productId);
                        }}
                        className="wm-btn wm-btn-danger"
                      >
                        Delete
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
