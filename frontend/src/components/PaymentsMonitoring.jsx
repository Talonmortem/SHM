import React, { useState, useEffect, useCallback, useMemo } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const PAYMENTS_COLUMNS_STORAGE_KEY = "wm_payments_columns_v1";
const DEFAULT_PAYMENTS_COLUMN_WIDTHS = {
  select: 52,
  date: 180,
  method: 140,
  amount: 120,
  comment: 260,
  actions: 150,
};

function formatAmount(n) {
  return Number(n || 0).toLocaleString(undefined, { maximumFractionDigits: 2 });
}

function parseDateValue(value) {
  if (!value) return null;
  const normalized = value.includes("T") ? value : value.replace(" ", "T");
  const dt = new Date(normalized);
  if (Number.isNaN(dt.getTime())) return null;
  return dt;
}

function formatDateTime(value) {
  const dt = parseDateValue(value);
  if (!dt) return value || "";
  return dt.toLocaleString();
}

function pad2(v) {
  return String(v).padStart(2, "0");
}

function toDateTimeLocalValue(value) {
  const dt = parseDateValue(value);
  if (!dt) return "";
  return `${dt.getFullYear()}-${pad2(dt.getMonth() + 1)}-${pad2(dt.getDate())}T${pad2(dt.getHours())}:${pad2(dt.getMinutes())}`;
}

function nowDateTimeLocal() {
  return toDateTimeLocalValue(new Date().toISOString());
}

export default function PaymentsMonitoring({ token, exportToCSV, filter = "", dateFrom = "", dateTo = "" }) {
  const [methods, setMethods] = useState([]);
  const [dataByMethod, setDataByMethod] = useState({});
  const [selectedByMethod, setSelectedByMethod] = useState({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [viewMode, setViewMode] = useState("group"); // "group" or "list"
  const [allPayments, setAllPayments] = useState([]); // For list view
  const [selectedPayments, setSelectedPayments] = useState(new Set()); // For list view

  // Add/Edit modal
  const [showModal, setShowModal] = useState(false);
  const [isEdit, setIsEdit] = useState(false);
  const [form, setForm] = useState({
      id: null,
      date: nowDateTimeLocal(),
      method: "",
      amount: "",
      comment: "",
  });
  const { columnWidths, handleResizeStart } = useResizableColumns(
    PAYMENTS_COLUMNS_STORAGE_KEY,
    DEFAULT_PAYMENTS_COLUMN_WIDTHS
  );

  const headers = { Authorization: token };
  const normalizedFilter = (filter || "").toLowerCase().trim();
  const columns = [
    { key: "select", label: "" },
    { key: "date", label: "Date" },
    { key: "method", label: "Method" },
    { key: "amount", label: "Amount" },
    { key: "comment", label: "Comment" },
    { key: "actions", label: "Actions", isAction: true },
  ];

  // Load methods and payments
  const reloadPayments = useCallback(async () => {
    try {
      setLoading(true);
      setError("");

      const pmRes = await axios.get("/api/payment_methods", { headers });
      const pm = Array.isArray(pmRes.data) ? pmRes.data : [];
      setMethods(pm);

      const reqs = pm.map((m) =>
        axios
          .get("/api/payments_monitoring", {
            headers,
            params: {
              method: m.method,
              date_from: dateFrom || undefined,
              date_to: dateTo || undefined,
            },
          })
          .then((r) => [m.method, Array.isArray(r.data) ? r.data : []])
      );

      const entries = await Promise.all(reqs);
      const nextData = {};
      const nextSelected = {};
      const allRows = [];
      for (const [method, rows] of entries) {
        nextData[method] = rows;
        nextSelected[method] = new Set();
        allRows.push(...rows);
      }
      setDataByMethod(nextData);
      setSelectedByMethod(nextSelected);
      setAllPayments(
        allRows.sort((a, b) => {
          const left = parseDateValue(a.date)?.getTime() || 0;
          const right = parseDateValue(b.date)?.getTime() || 0;
          return right - left;
        })
      );
      setSelectedPayments(new Set());
    } catch (e) {
      setError(e?.response?.data?.error || e.message || "Failed to load payments");
    } finally {
      setLoading(false);
    }
  }, [token, dateFrom, dateTo]);

  useEffect(() => {
    reloadPayments();
  }, [reloadPayments]);

  // Selection handlers for grouped view
  const toggleRow = (method, id) => {
    setSelectedByMethod((prev) => {
      const copy = { ...prev };
      const set = new Set(copy[method] || []);
      set.has(id) ? set.delete(id) : set.add(id);
      copy[method] = set;
      return copy;
    });
  };

  const matchesGlobalFilter = useCallback(
    (row) => {
      if (!normalizedFilter) return true;
      const date = formatDateTime(row.date).toLowerCase();
      const searchText = `${date} ${row.method || ""} ${row.comment || ""} ${row.amount ?? ""}`.toLowerCase();
      return searchText.includes(normalizedFilter);
    },
    [normalizedFilter]
  );

  const getRowsByMethod = useCallback(
    (method) => (dataByMethod[method] || []).filter(matchesGlobalFilter),
    [dataByMethod, matchesGlobalFilter]
  );

  const filteredAllPayments = useMemo(() => allPayments.filter(matchesGlobalFilter), [allPayments, matchesGlobalFilter]);

  const toggleAllInMethod = (method, rows) => {
    setSelectedByMethod((prev) => {
      const copy = { ...prev };
      const set = new Set(copy[method] || []);
      const allSelected = rows.length > 0 && rows.every((r) => set.has(r.id));
      copy[method] = allSelected ? new Set() : new Set(rows.map((r) => r.id));
      return copy;
    });
  };

  const clearSelection = (method) => {
    setSelectedByMethod((prev) => ({ ...prev, [method]: new Set() }));
  };

  // Selection handlers for list view
  const toggleRowList = (id) => {
    setSelectedPayments((prev) => {
      const copy = new Set(prev);
      copy.has(id) ? copy.delete(id) : copy.add(id);
      return copy;
    });
  };

  const toggleAllList = () => {
    const allSelected = filteredAllPayments.length > 0 && filteredAllPayments.every((r) => selectedPayments.has(r.id));
    setSelectedPayments(allSelected ? new Set() : new Set(filteredAllPayments.map((r) => r.id)));
  };

  const clearSelectionList = () => {
    setSelectedPayments(new Set());
  };

  const totalsForMethod = useCallback(
    (method, rows) => {
      const selected = selectedByMethod[method] || new Set();
      let sum = 0;
      let count = 0;
      for (const row of rows) {
        if (selected.has(row.id)) {
          count += 1;
          sum += Number(row.amount ?? 0);
        }
      }
      return { sum, count, total: sum };
    },
    [selectedByMethod]
  );

  const totalsForList = useCallback(() => {
    let sum = 0;
    let count = 0;
    for (const row of filteredAllPayments) {
      if (selectedPayments.has(row.id)) {
        count += 1;
        sum += Number(row.amount ?? 0);
      }
    }
    return { sum, count, total: sum };
  }, [filteredAllPayments, selectedPayments]);

  const totalAmount = useCallback((rows) => {
    return rows.reduce((sum, row) => sum + Number(row.amount ?? 0), 0);
  }, []);

  // Add/Edit handlers
  const openAddForm = (method = "") => {
    setIsEdit(false);
    setForm({
      id: null,
      date: "",
      method: method,
      amount: "",
      comment: "",
    });
    setShowModal(true);
  };

  const openEditForm = (method, id) => {
    const row = (dataByMethod[method] || allPayments).find((r) => r.id === id);
    if (!row) return;
    setIsEdit(true);
    setForm({
      id: row.id,
      date: toDateTimeLocalValue(row.date),
      method: row.method,
      amount: row.amount ?? "",
      comment: row.comment || "",
    });
    setShowModal(true);
  };

  const handleSave = async () => {
    try {
      if (!form.method) return alert("No method defined");

      const payload = {
        date: form.date || nowDateTimeLocal(),
        method: form.method,
        amount: Number(form.amount || 0),
        comment: form.comment || "",
      };

      if (isEdit && form.id != null) {
        await axios.put(`/api/payments/${form.id}`, payload, { headers });
      } else {
        await axios.post("/api/payments", payload, { headers });
      }
      setShowModal(false);
      await reloadPayments();
    } catch (e) {
      alert("Failed to save: " + (e?.response?.data?.error || e.message));
    }
  };

  const handleDeletePayment = async (id) => {
    if (!window.confirm("Delete this payment?")) return;
    try {
      await axios.delete(`/api/payments/${id}`, { headers });
      await reloadPayments();
    } catch (e) {
      alert("Failed to delete: " + (e?.response?.data?.error || e.message));
    }
  };

  // Render
  if (loading) return <p>Loading…</p>;

  return (
    <div className="wm-surface">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-bold">Payments Monitoring</h2>
        <div className="flex gap-2">
          <button
            onClick={() => setViewMode("group")}
            className={`wm-btn ${viewMode === "group" ? "wm-btn-primary" : ""}`}
          >
            Group by
          </button>
          <button
            onClick={() => setViewMode("list")}
            className={`wm-btn ${viewMode === "list" ? "wm-btn-primary" : ""}`}
          >
            List
          </button>
          <button
            onClick={() => exportToCSV(filteredAllPayments, "payments_all.csv")}
            className="wm-btn wm-btn-primary"
          >
            Export all CSV
          </button>
        </div>
      </div>

      {/* Date filters */}
      {error && <p className="wm-error mb-4">{error}</p>}

      {viewMode === "group" ? (
        // Grouped view
        methods.map(({ id, method }) => {
          const rows = getRowsByMethod(method);
          const selected = selectedByMethod[method] || new Set();
          const totals = totalsForMethod(method, rows);
          const allSelected = rows.length > 0 && rows.every((r) => selected.has(r.id));

          return (
            <div key={id || method} className="wm-table-wrap mb-6">
              <div className="flex items-center justify-between px-4 py-3 bg-gray-50">
                <div className="font-semibold">Method: {method}</div>
                <div className="flex gap-2">
                  <button
                    onClick={() => toggleAllInMethod(method, rows)}
                    className="wm-btn"
                  >
                    {allSelected ? "Unselect all" : "Select all"}
                  </button>
                  <button
                    onClick={() => clearSelection(method)}
                    className="wm-btn"
                  >
                    Clear
                  </button>
                  <button
                    onClick={() => exportToCSV(rows, `payments_${method}.csv`)}
                    className="wm-btn wm-btn-primary"
                  >
                    Export CSV
                  </button>
                  {selected.size > 0 && (
                    <button
                      onClick={() => {
                        const onlySelected = rows.filter((r) => selected.has(r.id));
                        exportToCSV(onlySelected, `payments_${method}_selected.csv`);
                      }}
                      className="wm-btn wm-btn-primary"
                    >
                      Export selected
                    </button>
                  )}
                  <button
                    onClick={() => openAddForm(method)}
                    className="wm-btn wm-btn-primary"
                  >
                    Add
                  </button>
                </div>
              </div>

              <table className="wm-table">
                <thead>
                  <tr>
                    {columns.map((column) => (
                      <th
                        key={column.key}
                        className={`wm-th relative ${column.isAction ? "wm-action-cell" : ""}`}
                        style={{ width: columnWidths[column.key] }}
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
                  {rows.length === 0 ? (
                    <tr>
                      <td colSpan="6" className="wm-empty">
                        No payments
                      </td>
                    </tr>
                  ) : (
                    rows.map((p) => {
                      const isSelected = selected.has(p.id);
                      return (
                        <tr key={p.id} className={isSelected ? "bg-blue-50" : ""}>
                          <td className="wm-td text-center" style={{ width: columnWidths.select }}>
                            <input
                              type="checkbox"
                              checked={isSelected}
                              onChange={() => toggleRow(method, p.id)}
                              onClick={(e) => e.stopPropagation()}
                            />
                          </td>
                          <td className="wm-td" style={{ width: columnWidths.date }}>{formatDateTime(p.date)}</td>
                          <td className="wm-td" style={{ width: columnWidths.method }}>{p.method}</td>
                          <td className={`wm-td text-right ${p.amount > 0 ? "text-green-600" : "text-red-600"}`} style={{ width: columnWidths.amount }}>
                            {formatAmount(p.amount)}
                          </td>
                          <td className="wm-td" style={{ width: columnWidths.comment }}>{p.comment}</td>
                          <td className="wm-td wm-action-cell" style={{ width: columnWidths.actions }}>
                            <div className="wm-action-buttons">
                              <button onClick={() => openEditForm(method, p.id)} className="wm-btn">
                                Edit
                              </button>
                              <button onClick={() => handleDeletePayment(p.id)} className="wm-btn wm-btn-danger">
                                Delete
                              </button>
                            </div>
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
                <tfoot>
                  <tr>
                    <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.select }}></td>
                    <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.date }}></td>
                    <td className="wm-td bg-gray-50 font-semibold text-right" style={{ width: columnWidths.method }}>Total:</td>
                    <td className="wm-td bg-gray-50 font-semibold text-right" style={{ width: columnWidths.amount }}>
                      {formatAmount(totalAmount(rows))}
                    </td>
                    <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.comment }}></td>
                    <td className="wm-td wm-action-cell bg-gray-50 font-semibold" style={{ width: columnWidths.actions }}></td>
                  </tr>
                </tfoot>
              </table>

              {totals.count > 0 && (
                <div className="px-4 py-3 border-t bg-white flex gap-6">
                  <div>Selected: {totals.count}</div>
                  <div>Total: {formatAmount(totals.total)}</div>
                </div>
              )}
            </div>
          );
        })
      ) : (
        // List view
        <div className="wm-table-wrap mb-6">
          <div className="flex items-center justify-between px-4 py-3 bg-gray-50">
            <div className="font-semibold">All Payments</div>
            <div className="flex gap-2">
              <button
                onClick={toggleAllList}
                className="wm-btn"
              >
                {filteredAllPayments.length > 0 && filteredAllPayments.every((r) => selectedPayments.has(r.id)) ? "Unselect all" : "Select all"}
              </button>
              <button
                onClick={clearSelectionList}
                className="wm-btn"
              >
                Clear
              </button>
              <button
                onClick={() => exportToCSV(filteredAllPayments, "payments_list.csv")}
                className="wm-btn wm-btn-primary"
              >
                Export CSV
              </button>
              {selectedPayments.size > 0 && (
                <button
                  onClick={() => {
                    const onlySelected = filteredAllPayments.filter((r) => selectedPayments.has(r.id));
                    exportToCSV(onlySelected, "payments_list_selected.csv");
                  }}
                  className="wm-btn wm-btn-primary"
                >
                  Export selected
                </button>
              )}
              <button
                onClick={() => openAddForm()}
                className="wm-btn wm-btn-primary"
              >
                Add
              </button>
            </div>
          </div>

          <table className="wm-table">
            <thead>
              <tr>
                {columns.map((column) => (
                  <th
                    key={column.key}
                    className={`wm-th relative ${column.isAction ? "wm-action-cell" : ""}`}
                    style={{ width: columnWidths[column.key] }}
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
              {filteredAllPayments.length === 0 ? (
                <tr>
                  <td colSpan="6" className="wm-empty">
                    No payments
                  </td>
                </tr>
              ) : (
                filteredAllPayments.map((p) => {
                  const isSelected = selectedPayments.has(p.id);
                  return (
                    <tr key={p.id} className={isSelected ? "bg-blue-50" : ""}>
                      <td className="wm-td text-center" style={{ width: columnWidths.select }}>
                        <input
                          type="checkbox"
                          checked={isSelected}
                          onChange={() => toggleRowList(p.id)}
                          onClick={(e) => e.stopPropagation()}
                        />
                      </td>
                      <td className="wm-td" style={{ width: columnWidths.date }}>{formatDateTime(p.date)}</td>
                      <td className="wm-td" style={{ width: columnWidths.method }}>{p.method}</td>
                      <td className={`wm-td text-right ${p.amount > 0 ? "text-green-600" : "text-red-600"}`} style={{ width: columnWidths.amount }}>
                        {formatAmount(p.amount)}
                      </td>
                      <td className="wm-td" style={{ width: columnWidths.comment }}>{p.comment}</td>
                      <td className="wm-td wm-action-cell" style={{ width: columnWidths.actions }}>
                        <div className="wm-action-buttons">
                          <button onClick={() => openEditForm(p.method, p.id)} className="wm-btn">
                            Edit
                          </button>
                          <button onClick={() => handleDeletePayment(p.id)} className="wm-btn wm-btn-danger">
                            Delete
                          </button>
                        </div>
                      </td>
                    </tr>
                  );
                })
              )}
            </tbody>
            <tfoot>
              <tr>
                <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.select }}></td>
                <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.date }}></td>
                <td className="wm-td bg-gray-50 font-semibold text-right" style={{ width: columnWidths.method }}>Total:</td>
                <td className="wm-td bg-gray-50 font-semibold text-right" style={{ width: columnWidths.amount }}>
                  {formatAmount(totalAmount(filteredAllPayments))}
                </td>
                <td className="wm-td bg-gray-50 font-semibold" style={{ width: columnWidths.comment }}></td>
                <td className="wm-td wm-action-cell bg-gray-50 font-semibold" style={{ width: columnWidths.actions }}></td>
              </tr>
            </tfoot>
          </table>

          {selectedPayments.size > 0 && (
            <div className="px-4 py-3 border-t bg-white flex gap-6">
              <div>Selected: {totalsForList().count}</div>
              <div>Total: {formatAmount(totalsForList().total)}</div>
            </div>
          )}
        </div>
      )}

      {showModal && (
        <div className="fixed inset-0 bg-slate-900/30 backdrop-blur-[2px] flex items-center justify-center z-50">
          <div className="bg-white p-6 rounded-xl shadow-xl w-full max-w-md">
            <div className="space-y-3">
              <input
                type="datetime-local"
                value={form.date || nowDateTimeLocal()}
                onChange={(e) => setForm({ ...form, date: e.target.value })}
                className="wm-input w-full"
              />
              <select
                value={form.method}
                onChange={(e) => setForm({ ...form, method: e.target.value })}
                className="wm-select w-full"
                disabled={isEdit}
              >
                <option value="">Select method</option>
                {methods.map(({ method }) => (
                  <option key={method} value={method}>
                    {method}
                  </option>
                ))}
              </select>
              <input
                type="number"
                value={form.amount}
                onChange={(e) => setForm({ ...form, amount: e.target.value })}
                className="wm-input w-full text-right"
                placeholder="Amount"
              />
              <textarea
                value={form.comment}
                onChange={(e) => setForm({ ...form, comment: e.target.value })}
                className="wm-textarea w-full"
                rows="3"
                placeholder="Comment"
              />
            </div>
            <div className="flex justify-end gap-2 mt-5">
              <button onClick={() => setShowModal(false)} className="wm-btn">
                Cancel
              </button>
              <button onClick={handleSave} className="wm-btn wm-btn-primary">
                Save
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
