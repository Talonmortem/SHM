import React, { useMemo, useState } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const ARTICLE_SORT_OPTIONS = ["id", "no", "code", "description", "euro", "colli", "kg", "value"];
const ARTICLE_COLUMNS_STORAGE_KEY = "wm_articles_columns_v1";
const DEFAULT_ARTICLE_COLUMN_WIDTHS = {
  id: 90,
  no: 100,
  code: 140,
  description: 260,
  euro: 110,
  colli: 110,
  kg: 110,
  value: 110,
  actions: 150,
};

function createEmptyArticle() {
  return {
    id: "",
    no: "",
    code: "",
    description: "",
    euro: "",
    colli: "",
    kg: "",
    value: "",
  };
}

export default function ArticleTable({ articles, setArticles, filter, token, username, exportToCSV }) {
  const [editingArticleServiceId, setEditingArticleServiceId] = useState(null);
  const [articleForm, setArticleForm] = useState(createEmptyArticle);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");
  const [sortBy, setSortBy] = useState("code");
  const [sortDirection, setSortDirection] = useState("asc");
  const { columnWidths, handleResizeStart } = useResizableColumns(ARTICLE_COLUMNS_STORAGE_KEY, DEFAULT_ARTICLE_COLUMN_WIDTHS);

  const headers = {
    Authorization: token,
    ...(username ? { "X-Username": username } : {}),
  };

  const resetForm = () => {
    setEditingArticleServiceId(null);
    setArticleForm(createEmptyArticle());
    setShowForm(false);
  };

  const openCreateForm = () => {
    setEditingArticleServiceId(null);
    setArticleForm(createEmptyArticle());
    setShowForm(true);
    setError("");
  };

  const handleEditArticle = (article) => {
    setEditingArticleServiceId(article.serviceId ?? article.id);
    setArticleForm({
      id: article.id ?? "",
      no: article.no ?? "",
      code: article.code || "",
      description: article.description || "",
      euro: article.euro ?? "",
      colli: article.colli ?? "",
      kg: article.kg ?? "",
      value: article.value ?? "",
    });
    setShowForm(true);
    setError("");
  };

  const handleDeleteArticle = async (id) => {
    try {
      await axios.delete(`/api/articles/${id}`, { headers });
      setArticles((prev) => (prev || []).filter((article) => (article.serviceId ?? article.id) !== id));
      if (editingArticleServiceId === id) {
        resetForm();
      }
      setError("");
    } catch (e) {
      setError("Ошибка удаления артикула: " + (e.response?.data?.error || e.message));
    }
  };

  const handleSubmitArticle = async (e) => {
    e.preventDefault();

    if (!String(articleForm.id || "").trim()) {
      setError("ID обязателен");
      return;
    }
    if (!articleForm.code.trim()) {
      setError("Код артикула обязателен");
      return;
    }

    try {
      if (editingArticleServiceId) {
        await axios.put(
          `/api/articles/${editingArticleServiceId}`,
          articleForm,
          { headers }
        );
        setArticles((prev) =>
          (prev || []).map((article) =>
            (article.serviceId ?? article.id) === editingArticleServiceId ? { ...article, ...articleForm, serviceId: editingArticleServiceId } : article
          )
        );
      } else {
        const res = await axios.post("/api/articles", articleForm, { headers });
        setArticles((prev) => [...(prev || []), res.data]);
      }

      setError("");
      resetForm();
    } catch (e) {
      const action = editingArticleServiceId ? "сохранения" : "создания";
      setError(`Ошибка ${action} артикула: ` + (e.response?.data?.error || e.message));
    }
  };

  const filteredArticles = useMemo(() => {
    const lowerFilter = filter.toLowerCase();
    const rows = (articles || []).filter((article) => {
      const code = (article.code || "").toString().toLowerCase();
      const description = (article.description || "").toLowerCase();
      return code.includes(lowerFilter) || description.includes(lowerFilter);
    });

    rows.sort((a, b) => {
      const left = a[sortBy];
      const right = b[sortBy];
      const leftNumeric = typeof left === "number" || !Number.isNaN(Number(left));
      const rightNumeric = typeof right === "number" || !Number.isNaN(Number(right));

      if (leftNumeric && rightNumeric) {
        const delta = Number(left) - Number(right);
        return sortDirection === "asc" ? delta : -delta;
      }

      const compare = String(left || "").localeCompare(String(right || ""), "ru", { sensitivity: "base" });
      return sortDirection === "asc" ? compare : -compare;
    });

    return rows;
  }, [articles, filter, sortBy, sortDirection]);

  const toggleSort = (column) => {
    if (!ARTICLE_SORT_OPTIONS.includes(column)) {
      return;
    }
    if (sortBy === column) {
      setSortDirection((prev) => (prev === "asc" ? "desc" : "asc"));
      return;
    }
    setSortBy(column);
    setSortDirection("asc");
  };

  const renderSortMark = (column) => {
    if (sortBy !== column) {
      return "";
    }
    return sortDirection === "asc" ? " ▲" : " ▼";
  };

  const columns = [
    { key: "id", label: "ID", sortable: true },
    { key: "no", label: "NO", sortable: true },
    { key: "code", label: "Code", sortable: true },
    { key: "description", label: "Description", sortable: true },
    { key: "euro", label: "Euro", sortable: true },
    { key: "colli", label: "Colli", sortable: true },
    { key: "kg", label: "KG", sortable: true },
    { key: "value", label: "Value", sortable: true },
    { key: "actions", label: "Actions", sortable: false, isAction: true },
  ];

  return (
    <div className="wm-surface">
      <div className="flex flex-wrap gap-2 items-center justify-between mb-4">
        <div className="flex flex-wrap gap-2 items-center">
          <button
            onClick={() => openCreateForm()}
            className="wm-btn wm-btn-primary"
          >
            Добавить приход
          </button>
        </div>

        <button
          onClick={() => exportToCSV(articles || [], "articles.csv")}
          className="wm-btn wm-btn-primary"
        >
          Export to CSV
        </button>
      </div>

      {error && <p className="wm-error mb-4">{error}</p>}

      {showForm && (
        <form onSubmit={handleSubmitArticle} className="mb-4 p-3 border rounded-xl bg-slate-50">
          <h3 className="text-lg font-semibold mb-3">
            {editingArticleServiceId ? "Редактирование прихода" : "Новый приход"}
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
            <input
              type="number"
              placeholder="ID"
              value={articleForm.id}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, id: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="number"
              placeholder="NO"
              value={articleForm.no}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, no: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Code"
              value={articleForm.code}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, code: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Description"
              value={articleForm.description}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, description: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Euro"
              value={articleForm.euro}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, euro: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Colli"
              value={articleForm.colli}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, colli: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="KG"
              value={articleForm.kg}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, kg: e.target.value }))}
              className="wm-input w-full"
            />
            <input
              type="text"
              placeholder="Value"
              value={articleForm.value}
              onChange={(e) => setArticleForm((prev) => ({ ...prev, value: e.target.value }))}
              className="wm-input w-full"
            />
          </div>

          <div className="flex gap-2 mt-3">
            <button
              type="submit"
              className="wm-btn wm-btn-primary"
            >
              {editingArticleServiceId ? "Сохранить" : "Создать"}
            </button>
            <button
              type="button"
              onClick={resetForm}
              className="wm-btn"
            >
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
                  className={`wm-th relative ${column.sortable ? "cursor-pointer" : ""} ${column.isAction ? "wm-action-cell" : ""}`}
                  style={{ width: columnWidths[column.key] }}
                  onClick={column.sortable ? () => toggleSort(column.key) : undefined}
                >
                  {column.label}
                  {column.sortable ? renderSortMark(column.key) : ""}
                  <div
                    className="wm-col-resize"
                    onMouseDown={(e) => handleResizeStart(column.key, e)}
                  />
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {filteredArticles.length === 0 ? (
              <tr>
                <td colSpan="9" className="wm-empty">Приход не найден</td>
              </tr>
            ) : (
              filteredArticles.map((article) => (
                <tr key={article.serviceId || article.id}>
                  <td className="wm-td" style={{ width: columnWidths.id }}>{article.id}</td>
                  <td className="wm-td" style={{ width: columnWidths.no }}>{article.no}</td>
                  <td className="wm-td" style={{ width: columnWidths.code }}>{article.code}</td>
                  <td className="wm-td" style={{ width: columnWidths.description }}>{article.description}</td>
                  <td className="wm-td" style={{ width: columnWidths.euro }}>{article.euro}</td>
                  <td className="wm-td" style={{ width: columnWidths.colli }}>{article.colli}</td>
                  <td className="wm-td" style={{ width: columnWidths.kg }}>{article.kg}</td>
                  <td className="wm-td" style={{ width: columnWidths.value }}>{article.value}</td>
                  <td className="wm-td wm-action-cell" style={{ width: columnWidths.actions }}>
                    <div className="wm-action-buttons">
                      <button
                        onClick={() => handleEditArticle(article)}
                        className="wm-btn"
                      >
                        Редактировать
                      </button>
                      <button
                        onClick={() => handleDeleteArticle(article.serviceId ?? article.id)}
                        className="wm-btn wm-btn-danger"
                      >
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
