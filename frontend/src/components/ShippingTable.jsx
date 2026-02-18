import React, { useMemo, useState, useEffect } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const SHIPPING_COLUMNS_STORAGE_KEY = "wm_shipping_columns_v1";
const SHIPPING_NOTES_COLUMNS_STORAGE_KEY = "wm_shipping_notes_columns_v1";
const DEFAULT_SHIPPING_COLUMN_WIDTHS = {
  ship_date: 128,
  city: 130,
  full_name: 200,
  phone: 150,
  passport_inn: 170,
  tk: 130,
  places: 120,
  price: 110,
  weight: 110,
  actions: 150,
};
const DEFAULT_SHIPPING_NOTES_COLUMN_WIDTHS = {
  ship_date: 140,
  note: 360,
  actions: 150,
};

function createEmptyShipment(defaultDate = "") {
  return {
    ship_date: defaultDate,
    city: "",
    full_name: "",
    phone: "",
    passport_inn: "",
    tk: "",
    places: "",
    price: "",
    weight: "",
  };
}

function normalizeNumber(value) {
  if (value === null || value === undefined || value === "") return 0;
  const cleaned = String(value).replace(/\s/g, "").replace(",", ".");
  const n = Number(cleaned);
  return Number.isNaN(n) ? 0 : n;
}

export default function ShippingTable({ shipments, setShipments, filter, dateFilter, token, username, exportToCSV }) {
  const [editingShipmentId, setEditingShipmentId] = useState(null);
  const [shipmentForm, setShipmentForm] = useState(createEmptyShipment());
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");
  const [clients, setClients] = useState([]);
  const [showSuggestions, setShowSuggestions] = useState(false);
  const [activeField, setActiveField] = useState(null);

  const [notes, setNotes] = useState([]);
  const [noteForm, setNoteForm] = useState({ id: null, ship_date: "", note: "" });
  const [showNoteForm, setShowNoteForm] = useState(false);
  const [noteError, setNoteError] = useState("");
  const { columnWidths: shipmentColumnWidths, handleResizeStart: handleShipmentResizeStart } = useResizableColumns(
    SHIPPING_COLUMNS_STORAGE_KEY,
    DEFAULT_SHIPPING_COLUMN_WIDTHS
  );
  const { columnWidths: notesColumnWidths, handleResizeStart: handleNotesResizeStart } = useResizableColumns(
    SHIPPING_NOTES_COLUMNS_STORAGE_KEY,
    DEFAULT_SHIPPING_NOTES_COLUMN_WIDTHS
  );

  const headers = {
    Authorization: token,
    ...(username ? { "X-Username": username } : {}),
  };

  useEffect(() => {
    const fetchClients = async () => {
      try {
        const res = await axios.get("/api/clients", { headers });
        setClients(res.data || []);
      } catch {
        setClients([]);
      }
    };
    fetchClients();
  }, [token, username]);

  useEffect(() => {
    if (!dateFilter) {
      setNotes([]);
      return;
    }

    const fetchNotes = async () => {
      try {
        const res = await axios.get("/api/shipment_notes", {
          headers,
          params: { date: dateFilter },
        });
        setNotes(res.data || []);
        setNoteError("");
      } catch (e) {
        setNoteError("Ошибка загрузки заметок: " + (e.response?.data?.error || e.message));
      }
    };

    fetchNotes();
  }, [dateFilter, token, username]);

  const resetForm = () => {
    setEditingShipmentId(null);
    setShipmentForm(createEmptyShipment(dateFilter));
    setShowForm(false);
    setShowSuggestions(false);
    setActiveField(null);
  };

  const openCreateForm = () => {
    setEditingShipmentId(null);
    setShipmentForm(createEmptyShipment(dateFilter));
    setShowForm(true);
    setError("");
    setShowSuggestions(false);
    setActiveField(null);
  };

  const handleEditShipment = (shipment) => {
    setEditingShipmentId(shipment.id);
    setShipmentForm({
      ship_date: shipment.ship_date || "",
      city: shipment.city || "",
      full_name: shipment.full_name || "",
      phone: shipment.phone || "",
      passport_inn: shipment.passport_inn || "",
      tk: shipment.tk || "",
      places: shipment.places ?? "",
      price: shipment.price ?? "",
      weight: shipment.weight ?? "",
    });
    setShowForm(true);
    setError("");
    setShowSuggestions(false);
    setActiveField(null);
  };

  const applyClientToForm = (client, nameOverride) => {
    setShipmentForm((prev) => ({
      ...prev,
      city: client.city || "",
      full_name: nameOverride || client.full_name || "",
      phone: client.phone || "",
      passport_inn: client.passport_number || "",
      tk: client.tk || "",
    }));
  };

  const handleDeleteShipment = async (id) => {
    try {
      await axios.delete(`/api/shipments/${id}`, { headers });
      setShipments((prev) => (prev || []).filter((shipment) => shipment.id !== id));
      if (editingShipmentId === id) {
        resetForm();
      }
      setError("");
    } catch (e) {
      setError("Ошибка удаления отправки: " + (e.response?.data?.error || e.message));
    }
  };

  const handleSubmitShipment = async (e) => {
    e.preventDefault();

    if (!shipmentForm.ship_date) {
      setError("Дата обязательна");
      return;
    }
    if (!shipmentForm.full_name.trim()) {
      setError("ФИО обязательно");
      return;
    }

    const payload = {
      ...shipmentForm,
      places: Math.trunc(normalizeNumber(shipmentForm.places)),
      price: normalizeNumber(shipmentForm.price),
      weight: normalizeNumber(shipmentForm.weight),
    };

    try {
      if (editingShipmentId) {
        await axios.put(`/api/shipments/${editingShipmentId}`, payload, { headers });
        setShipments((prev) =>
          (prev || []).map((shipment) =>
            shipment.id === editingShipmentId ? { ...shipment, ...payload } : shipment
          )
        );
      } else {
        const res = await axios.post("/api/shipments", payload, { headers });
        setShipments((prev) => [res.data, ...(prev || [])]);
      }

      setError("");
      resetForm();
    } catch (e) {
      const action = editingShipmentId ? "сохранения" : "создания";
      setError(`Ошибка ${action} отправки: ` + (e.response?.data?.error || e.message));
    }
  };

  const clientSuggestionsByName = useMemo(() => {
    const needle = shipmentForm.full_name.toLowerCase().trim();
    if (!needle) return [];
    const rows = (clients || []).filter((c) =>
      String(c.full_name || "").toLowerCase().includes(needle)
    );
    return rows.slice(0, 8);
  }, [clients, shipmentForm.full_name]);

  const clientSuggestionsByPhone = useMemo(() => {
    const needle = shipmentForm.phone.toLowerCase().trim();
    if (!needle) return [];
    const rows = (clients || []).filter((c) =>
      String(c.phone || "").toLowerCase().includes(needle)
    );
    return rows.slice(0, 8);
  }, [clients, shipmentForm.phone]);

  const clientSuggestionsByPassport = useMemo(() => {
    const needle = shipmentForm.passport_inn.toLowerCase().trim();
    if (!needle) return [];
    const rows = (clients || []).filter((c) =>
      String(c.passport_number || "").toLowerCase().includes(needle)
    );
    return rows.slice(0, 8);
  }, [clients, shipmentForm.passport_inn]);

  const handleFullNameChange = (value) => {
    setShipmentForm((prev) => ({ ...prev, full_name: value }));
    setShowSuggestions(true);
    setActiveField("full_name");
    const exact = (clients || []).find(
      (c) => String(c.full_name || "").toLowerCase().trim() === value.toLowerCase().trim()
    );
    if (exact) {
      applyClientToForm(exact, exact.full_name);
      setShowSuggestions(false);
    }
  };

  const handlePhoneChange = (value) => {
    setShipmentForm((prev) => ({ ...prev, phone: value }));
    setShowSuggestions(true);
    setActiveField("phone");
    const exact = (clients || []).find(
      (c) => String(c.phone || "").toLowerCase().trim() === value.toLowerCase().trim()
    );
    if (exact) {
      applyClientToForm(exact, exact.full_name);
      setShowSuggestions(false);
    }
  };

  const handlePassportChange = (value) => {
    setShipmentForm((prev) => ({ ...prev, passport_inn: value }));
    setShowSuggestions(true);
    setActiveField("passport_inn");
    const exact = (clients || []).find(
      (c) => String(c.passport_number || "").toLowerCase().trim() === value.toLowerCase().trim()
    );
    if (exact) {
      applyClientToForm(exact, exact.full_name);
      setShowSuggestions(false);
    }
  };

  const activeSuggestions = useMemo(() => {
    if (!activeField) return [];
    if (activeField === "full_name") return clientSuggestionsByName;
    if (activeField === "phone") return clientSuggestionsByPhone;
    if (activeField === "passport_inn") return clientSuggestionsByPassport;
    return [];
  }, [activeField, clientSuggestionsByName, clientSuggestionsByPhone, clientSuggestionsByPassport]);

  const renderSuggestion = (client) => {
    if (activeField === "phone") {
      return (
        <>
          <div className="font-medium">{client.phone || "Без телефона"}</div>
          <div className="text-xs text-gray-500">
            {client.full_name || "Без ФИО"} · {client.city || "Без города"}
          </div>
        </>
      );
    }
    if (activeField === "passport_inn") {
      return (
        <>
          <div className="font-medium">{client.passport_number || "Без номера"}</div>
          <div className="text-xs text-gray-500">
            {client.full_name || "Без ФИО"} · {client.city || "Без города"}
          </div>
        </>
      );
    }
    return (
      <>
        <div className="font-medium">{client.full_name}</div>
        <div className="text-xs text-gray-500">
          {client.city || "Без города"} · {client.phone || "Без телефона"}
        </div>
      </>
    );
  };

  const filteredShipments = useMemo(() => {
    const lowerFilter = filter.toLowerCase().trim();
    return (shipments || []).filter((shipment) => {
      if (dateFilter && shipment.ship_date !== dateFilter) {
        return false;
      }
      if (!lowerFilter) {
        return true;
      }
      return [
        shipment.city,
        shipment.full_name,
        shipment.phone,
        shipment.passport_inn,
        shipment.tk,
        shipment.places,
        shipment.price,
        shipment.weight,
      ]
        .map((v) => String(v || "").toLowerCase())
        .some((v) => v.includes(lowerFilter));
    });
  }, [shipments, filter, dateFilter]);

  const openAddNote = () => {
    setNoteForm({ id: null, ship_date: dateFilter || "", note: "" });
    setShowNoteForm(true);
    setNoteError("");
  };

  const openEditNote = (note) => {
    setNoteForm({ id: note.id, ship_date: note.ship_date || "", note: note.note || "" });
    setShowNoteForm(true);
    setNoteError("");
  };

  const handleSaveNote = async (e) => {
    e.preventDefault();
    if (!noteForm.ship_date) {
      setNoteError("Дата обязательна");
      return;
    }
    if (!noteForm.note.trim()) {
      setNoteError("Заметка обязательна");
      return;
    }

    try {
      if (noteForm.id) {
        await axios.put(`/api/shipment_notes/${noteForm.id}`, noteForm, { headers });
      } else {
        await axios.post("/api/shipment_notes", noteForm, { headers });
      }
      setShowNoteForm(false);
      setNoteError("");

      const res = await axios.get("/api/shipment_notes", {
        headers,
        params: { date: noteForm.ship_date },
      });
      setNotes(res.data || []);
    } catch (e) {
      setNoteError("Ошибка сохранения заметки: " + (e.response?.data?.error || e.message));
    }
  };

  const handleDeleteNote = async (id) => {
    try {
      await axios.delete(`/api/shipment_notes/${id}`, { headers });
      setNotes((prev) => (prev || []).filter((note) => note.id !== id));
      setNoteError("");
    } catch (e) {
      setNoteError("Ошибка удаления заметки: " + (e.response?.data?.error || e.message));
    }
  };

  const shipmentColumns = [
    { key: "ship_date", label: "Дата" },
    { key: "city", label: "Город" },
    { key: "full_name", label: "ФИО" },
    { key: "phone", label: "Номер тел." },
    { key: "passport_inn", label: "Номер паспорта/ИНН" },
    { key: "tk", label: "ТК" },
    { key: "places", label: "Кол-во мест" },
    { key: "price", label: "Цена" },
    { key: "weight", label: "Вес" },
    { key: "actions", label: "Действия", isAction: true },
  ];

  const notesColumns = [
    { key: "ship_date", label: "Дата" },
    { key: "note", label: "Заметка" },
    { key: "actions", label: "Действия", isAction: true },
  ];

  return (
    <div className="wm-surface">
      <div className="flex flex-wrap gap-2 items-center justify-between mb-4">
        <div className="flex flex-wrap gap-2 items-center">
          <button onClick={openCreateForm} className="wm-btn wm-btn-primary">
            Добавить отправку
          </button>
        </div>

        <button
          onClick={() => exportToCSV(filteredShipments || [], "shipments.csv")}
          className="wm-btn wm-btn-primary"
        >
          Export to CSV
        </button>
      </div>

      {error && <p className="wm-error mb-4">{error}</p>}

      {showForm && (
        <form onSubmit={handleSubmitShipment} className="mb-4 p-3 border rounded-xl bg-slate-50">
          <h3 className="text-lg font-semibold mb-3">
            {editingShipmentId ? "Редактирование отправки" : "Новая отправка"}
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <input
              type="date"
              value={shipmentForm.ship_date}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, ship_date: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="ГОРОД"
              value={shipmentForm.city}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, city: e.target.value }))}
              className="wm-input w-full"
            />
            <div className="relative">
              <input
                type="text"
                placeholder="ФИО *"
                value={shipmentForm.full_name}
                onChange={(e) => handleFullNameChange(e.target.value)}
                onFocus={() => {
                  setActiveField("full_name");
                  setShowSuggestions(true);
                }}
                onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                className="wm-input w-full"
              />
              {showSuggestions && activeField === "full_name" && activeSuggestions.length > 0 && (
                <div className="absolute z-10 mt-1 w-full max-h-48 overflow-auto border rounded bg-white shadow">
                  {activeSuggestions.map((client) => (
                    <button
                      key={client.id}
                      type="button"
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => {
                        applyClientToForm(client);
                        setShowSuggestions(false);
                      }}
                      className="w-full text-left px-3 py-2 hover:bg-slate-100"
                    >
                      {renderSuggestion(client)}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="relative">
              <input
                type="text"
                placeholder="Номер тел."
                value={shipmentForm.phone}
                onChange={(e) => handlePhoneChange(e.target.value)}
                onFocus={() => {
                  setActiveField("phone");
                  setShowSuggestions(true);
                }}
                onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                className="wm-input w-full"
              />
              {showSuggestions && activeField === "phone" && activeSuggestions.length > 0 && (
                <div className="absolute z-10 mt-1 w-full max-h-48 overflow-auto border rounded bg-white shadow">
                  {activeSuggestions.map((client) => (
                    <button
                      key={client.id}
                      type="button"
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => {
                        applyClientToForm(client);
                        setShowSuggestions(false);
                      }}
                      className="w-full text-left px-3 py-2 hover:bg-slate-100"
                    >
                      {renderSuggestion(client)}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <div className="relative">
              <input
                type="text"
                placeholder="Номер паспорта/ИНН"
                value={shipmentForm.passport_inn}
                onChange={(e) => handlePassportChange(e.target.value)}
                onFocus={() => {
                  setActiveField("passport_inn");
                  setShowSuggestions(true);
                }}
                onBlur={() => setTimeout(() => setShowSuggestions(false), 150)}
                className="wm-input w-full"
              />
              {showSuggestions && activeField === "passport_inn" && activeSuggestions.length > 0 && (
                <div className="absolute z-10 mt-1 w-full max-h-48 overflow-auto border rounded bg-white shadow">
                  {activeSuggestions.map((client) => (
                    <button
                      key={client.id}
                      type="button"
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => {
                        applyClientToForm(client);
                        setShowSuggestions(false);
                      }}
                      className="w-full text-left px-3 py-2 hover:bg-slate-100"
                    >
                      {renderSuggestion(client)}
                    </button>
                  ))}
                </div>
              )}
            </div>
            <input
              type="text"
              placeholder="ТК"
              value={shipmentForm.tk}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, tk: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Кол-во мест"
              value={shipmentForm.places}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, places: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Цена"
              value={shipmentForm.price}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, price: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Вес"
              value={shipmentForm.weight}
              onChange={(e) => setShipmentForm((prev) => ({ ...prev, weight: e.target.value }))}
              className="wm-input w-full"
            />
          </div>

          <div className="flex gap-2 mt-3">
            <button type="submit" className="wm-btn wm-btn-primary">
              {editingShipmentId ? "Сохранить" : "Создать"}
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
              {shipmentColumns.map((column) => (
                <th
                  key={column.key}
                  className={`wm-th relative ${column.isAction ? "wm-action-cell" : ""}`}
                  style={{ width: shipmentColumnWidths[column.key] }}
                >
                  {column.label}
                  <div
                    className="wm-col-resize"
                    onMouseDown={(e) => handleShipmentResizeStart(column.key, e)}
                  />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filteredShipments.length === 0 ? (
              <tr>
                <td colSpan="10" className="wm-empty">Отправки не найдены</td>
              </tr>
            ) : (
              filteredShipments.map((shipment) => (
                <tr key={shipment.id}>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.ship_date }}>{shipment.ship_date}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.city }}>{shipment.city}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.full_name }}>{shipment.full_name}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.phone }}>{shipment.phone}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.passport_inn }}>{shipment.passport_inn}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.tk }}>{shipment.tk}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.places }}>{shipment.places}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.price }}>{shipment.price}</td>
                  <td className="wm-td" style={{ width: shipmentColumnWidths.weight }}>{shipment.weight}</td>
                  <td className="wm-td wm-action-cell" style={{ width: shipmentColumnWidths.actions }}>
                    <div className="wm-action-buttons">
                      <button onClick={() => handleEditShipment(shipment)} className="wm-btn">
                        Редактировать
                      </button>
                      <button onClick={() => handleDeleteShipment(shipment.id)} className="wm-btn wm-btn-danger">
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

      <div className="mt-6">
        <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
          <h3 className="text-lg font-semibold">Заметки по дню</h3>
          <button onClick={openAddNote} className="wm-btn wm-btn-primary" disabled={!dateFilter}>
            Добавить заметку
          </button>
        </div>

        {!dateFilter && (
          <p className="wm-muted">Выберите дату, чтобы увидеть и добавить заметки.</p>
        )}

        {noteError && <p className="wm-error mb-3">{noteError}</p>}

        {showNoteForm && (
          <form onSubmit={handleSaveNote} className="mb-3 p-3 border rounded-xl bg-slate-50">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
              <input
                type="date"
                value={noteForm.ship_date}
                onChange={(e) => setNoteForm((prev) => ({ ...prev, ship_date: e.target.value }))}
                className="wm-input w-full"
              />
              <input
                type="text"
                placeholder="Заметка"
                value={noteForm.note}
                onChange={(e) => setNoteForm((prev) => ({ ...prev, note: e.target.value }))}
                className="wm-input w-full"
              />
            </div>
            <div className="flex gap-2 mt-3">
              <button type="submit" className="wm-btn wm-btn-primary">
                {noteForm.id ? "Сохранить" : "Создать"}
              </button>
              <button type="button" onClick={() => setShowNoteForm(false)} className="wm-btn">
                Отмена
              </button>
            </div>
          </form>
        )}

        {dateFilter && (
          <div className="wm-table-wrap">
            <table className="wm-table">
              <thead>
                <tr>
                  {notesColumns.map((column) => (
                    <th
                      key={column.key}
                      className={`wm-th relative ${column.isAction ? "wm-action-cell" : ""}`}
                      style={{ width: notesColumnWidths[column.key] }}
                    >
                      {column.label}
                      <div
                        className="wm-col-resize"
                        onMouseDown={(e) => handleNotesResizeStart(column.key, e)}
                      />
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {notes.length === 0 ? (
                  <tr>
                    <td colSpan="3" className="wm-empty">Заметок нет</td>
                  </tr>
                ) : (
                  notes.map((note) => (
                    <tr key={note.id}>
                      <td className="wm-td" style={{ width: notesColumnWidths.ship_date }}>{note.ship_date}</td>
                      <td className="wm-td" style={{ width: notesColumnWidths.note }}>{note.note}</td>
                      <td className="wm-td wm-action-cell" style={{ width: notesColumnWidths.actions }}>
                        <div className="wm-action-buttons">
                          <button onClick={() => openEditNote(note)} className="wm-btn">
                            Редактировать
                          </button>
                          <button onClick={() => handleDeleteNote(note.id)} className="wm-btn wm-btn-danger">
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
        )}
      </div>
    </div>
  );
}
