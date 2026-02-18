import React, { useEffect, useMemo, useState } from "react";
import axios from "axios";
import useResizableColumns from "./useResizableColumns";

const BALANCE_COLUMNS_STORAGE_KEY = "wm_balance_columns_v1";
const DEFAULT_BALANCE_COLUMN_WIDTHS = {
  id: 70,
  no: 90,
  code: 130,
  description: 260,
  incomeKg: 120,
  sentKg: 120,
  balanceKg: 120,
  reservedKg: 120,
  freeKg: 120,
};

export default function BalanceTable({ token, username, filter, exportToCSV }) {
  const [rows, setRows] = useState([]);
  const [error, setError] = useState("");
  const { columnWidths, handleResizeStart } = useResizableColumns(BALANCE_COLUMNS_STORAGE_KEY, DEFAULT_BALANCE_COLUMN_WIDTHS);

  const headers = {
    Authorization: token,
    ...(username ? { "X-Username": username } : {}),
  };

  useEffect(() => {
    const fetchBalance = async () => {
      try {
        const res = await axios.get("/api/balance", { headers });
        setRows(res.data || []);
        setError("");
      } catch (e) {
        setError("Ошибка загрузки баланса: " + (e.response?.data?.error || e.message));
      }
    };

    fetchBalance();
  }, [token, username]);

  const filteredRows = useMemo(() => {
    const q = (filter || "").toLowerCase().trim();
    if (!q) {
      return rows;
    }
    return rows.filter((row) =>
      String(row.code || "").toLowerCase().includes(q) ||
      String(row.description || "").toLowerCase().includes(q)
    );
  }, [rows, filter]);

  const columns = [
    { key: "id", label: "ID" },
    { key: "no", label: "NO" },
    { key: "code", label: "CODE" },
    { key: "description", label: "DESCRIPTION" },
    { key: "incomeKg", label: "IncomeKG" },
    { key: "sentKg", label: "SentKG" },
    { key: "balanceKg", label: "BalanceKG" },
    { key: "reservedKg", label: "ReservedKG" },
    { key: "freeKg", label: "FreeKG" },
  ];

  return (
    <div className="wm-surface">
      <div className="flex justify-end mb-4">
        <button
          onClick={() => exportToCSV(filteredRows, "balance.csv")}
          className="wm-btn wm-btn-primary"
        >
          Export to CSV
        </button>
      </div>

      {error && <p className="wm-error mb-4">{error}</p>}

      <div className="wm-table-wrap">
        <table className="wm-table">
          <thead>
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className="wm-th relative"
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
            {filteredRows.length === 0 ? (
              <tr>
                <td colSpan="9" className="wm-empty">Баланс не найден</td>
              </tr>
            ) : (
              filteredRows.map((row) => (
                <tr key={`${row.id}-${row.code}`}>
                  <td className="wm-td">{row.id}</td>
                  <td className="wm-td">{row.no}</td>
                  <td className="wm-td">{row.code}</td>
                  <td className="wm-td">{row.description}</td>
                  <td className="wm-td">{row.incomeKg}</td>
                  <td className="wm-td">{row.sentKg}</td>
                  <td className="wm-td">{row.balanceKg}</td>
                  <td className="wm-td">{row.reservedKg}</td>
                  <td className="wm-td">{row.freeKg}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
