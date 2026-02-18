import React, { useMemo, useState } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const CLIENTS_COLUMNS_STORAGE_KEY = "wm_clients_columns_v1";
const DEFAULT_CLIENTS_COLUMN_WIDTHS = {
  city: 140,
  full_name: 220,
  phone: 150,
  passport_number: 180,
  tk: 140,
  comment: 260,
  actions: 150,
};

function createEmptyClient() {
  return {
    city: "",
    full_name: "",
    phone: "",
    passport_number: "",
    tk: "",
    comment: "",
  };
}

export default function ClientsTable({ clients, setClients, filter, token, username, exportToCSV }) {
  const [editingClientId, setEditingClientId] = useState(null);
  const [clientForm, setClientForm] = useState(createEmptyClient);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");
  const { columnWidths, handleResizeStart } = useResizableColumns(CLIENTS_COLUMNS_STORAGE_KEY, DEFAULT_CLIENTS_COLUMN_WIDTHS);

  const headers = {
    Authorization: token,
    ...(username ? { "X-Username": username } : {}),
  };

  const resetForm = () => {
    setEditingClientId(null);
    setClientForm(createEmptyClient());
    setShowForm(false);
  };

  const openCreateForm = () => {
    setEditingClientId(null);
    setClientForm(createEmptyClient());
    setShowForm(true);
    setError("");
  };

  const handleEditClient = (client) => {
    setEditingClientId(client.id);
    setClientForm({
      city: client.city || "",
      full_name: client.full_name || "",
      phone: client.phone || "",
      passport_number: client.passport_number || "",
      tk: client.tk || "",
      comment: client.comment || "",
    });
    setShowForm(true);
    setError("");
  };

  const handleDeleteClient = async (id) => {
    try {
      await axios.delete(`/api/clients/${id}`, { headers });
      setClients((prev) => (prev || []).filter((client) => client.id !== id));
      if (editingClientId === id) {
        resetForm();
      }
      setError("");
    } catch (e) {
      setError("Ошибка удаления клиента: " + (e.response?.data?.error || e.message));
    }
  };

  const handleSubmitClient = async (e) => {
    e.preventDefault();

    if (!clientForm.full_name.trim()) {
      setError("ФИО обязательно");
      return;
    }

    try {
      if (editingClientId) {
        await axios.put(`/api/clients/${editingClientId}`, clientForm, { headers });
        setClients((prev) =>
          (prev || []).map((client) =>
            client.id === editingClientId ? { ...client, ...clientForm } : client
          )
        );
      } else {
        const res = await axios.post("/api/clients", clientForm, { headers });
        setClients((prev) => [res.data, ...(prev || [])]);
      }

      setError("");
      resetForm();
    } catch (e) {
      const action = editingClientId ? "сохранения" : "создания";
      setError(`Ошибка ${action} клиента: ` + (e.response?.data?.error || e.message));
    }
  };

  const filteredClients = useMemo(() => {
    const lowerFilter = filter.toLowerCase().trim();
    if (!lowerFilter) {
      return clients || [];
    }

    return (clients || []).filter((client) =>
      [
        client.city,
        client.full_name,
        client.phone,
        client.passport_number,
        client.tk,
        client.comment,
      ]
        .map((v) => String(v || "").toLowerCase())
        .some((v) => v.includes(lowerFilter))
    );
  }, [clients, filter]);

  const columns = [
    { key: "city", label: "ГОРОД" },
    { key: "full_name", label: "ФИО" },
    { key: "phone", label: "ТЕЛ" },
    { key: "passport_number", label: "Номер паспорта" },
    { key: "tk", label: "ТК" },
    { key: "comment", label: "Комментарий" },
    { key: "actions", label: "Действия", isAction: true },
  ];

  return (
    <div className="wm-surface">
      <div className="flex flex-wrap gap-2 items-center justify-between mb-4">
        <div className="flex flex-wrap gap-2 items-center">
          <button onClick={openCreateForm} className="wm-btn wm-btn-primary">
            Добавить клиента
          </button>
        </div>

        <button
          onClick={() => exportToCSV(clients || [], "clients.csv")}
          className="wm-btn wm-btn-primary"
        >
          Export to CSV
        </button>
      </div>

      {error && <p className="wm-error mb-4">{error}</p>}

      {showForm && (
        <form onSubmit={handleSubmitClient} className="mb-4 p-3 border rounded-xl bg-slate-50">
          <h3 className="text-lg font-semibold mb-3">
            {editingClientId ? "Редактирование клиента" : "Новый клиент"}
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <input
              type="text"
              placeholder="ГОРОД"
              value={clientForm.city}
              onChange={(e) => setClientForm((prev) => ({ ...prev, city: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="ФИО *"
              value={clientForm.full_name}
              onChange={(e) => setClientForm((prev) => ({ ...prev, full_name: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="ТЕЛ"
              value={clientForm.phone}
              onChange={(e) => setClientForm((prev) => ({ ...prev, phone: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Номер паспорта"
              value={clientForm.passport_number}
              onChange={(e) => setClientForm((prev) => ({ ...prev, passport_number: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="ТК"
              value={clientForm.tk}
              onChange={(e) => setClientForm((prev) => ({ ...prev, tk: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Комментарий"
              value={clientForm.comment}
              onChange={(e) => setClientForm((prev) => ({ ...prev, comment: e.target.value }))}
              className="wm-input w-full"
            />
          </div>

          <div className="flex gap-2 mt-3">
            <button type="submit" className="wm-btn wm-btn-primary">
              {editingClientId ? "Сохранить" : "Создать"}
            </button>
            <button type="button" onClick={resetForm} className="wm-btn">
              Отмена
            </button>
          </div>
        </form>
      )}

      <div className="wm-table-wrap">
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
            {filteredClients.length === 0 ? (
              <tr>
                <td colSpan="7" className="wm-empty">Клиенты не найдены</td>
              </tr>
            ) : (
              filteredClients.map((client) => (
                <tr key={client.id}>
                  <td className="wm-td" style={{ width: columnWidths.city }}>{client.city}</td>
                  <td className="wm-td" style={{ width: columnWidths.full_name }}>{client.full_name}</td>
                  <td className="wm-td" style={{ width: columnWidths.phone }}>{client.phone}</td>
                  <td className="wm-td" style={{ width: columnWidths.passport_number }}>{client.passport_number}</td>
                  <td className="wm-td" style={{ width: columnWidths.tk }}>{client.tk}</td>
                  <td className="wm-td" style={{ width: columnWidths.comment }}>{client.comment}</td>
                  <td className="wm-td wm-action-cell" style={{ width: columnWidths.actions }}>
                    <div className="wm-action-buttons">
                      <button onClick={() => handleEditClient(client)} className="wm-btn">
                        Редактировать
                      </button>
                      <button onClick={() => handleDeleteClient(client.id)} className="wm-btn wm-btn-danger">
                        Удалить
                      </button>
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
