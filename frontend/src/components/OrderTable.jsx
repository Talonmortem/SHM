import React, { useState, useEffect, useMemo, useCallback } from 'react';
import axios from 'axios';
import useResizableColumns from "./useResizableColumns";

const ORDER_COLUMNS_STORAGE_KEY = 'wm_orders_columns_v1';

const DEFAULT_ORDER_COLUMN_WIDTHS = {
  id: 50,
  status: 100,
  name: 150,
  description: 200,
  components: 300,
  payments: 200,
  debt: 100,
  actions: 100,
};

const ORDER_STATUS_BADGE_CLASS = {
  0: "wm-status-new",
  1: "wm-status-ready",
  2: "wm-status-shipped",
};

export default function OrderTable({ orders, setOrders, token, exportToCSV, products = [], filter = '', statusFilter = '', setFilter = () => {} }) {
  const [editingOrderId, setEditingOrderId] = useState(null);
  const [newOrder, setNewOrder] = useState({
    id: null,
    components: [],
    quantity: 0,
    status: 0,
    name: '',
    description: '',
    payments: [{ method: '', amount: '', comment: '' }],
    debt: 0,
  });
  const [showAddForm, setShowAddForm] = useState(false);
  const [selectedProducts, setSelectedProducts] = useState([]);
  const [showOrderModal, setShowOrderModal] = useState(false);
  const [showConfirmStep, setShowConfirmStep] = useState(false);
  const [showEditProducts, setShowEditProducts] = useState(false);
  const [originalSelectedProducts, setOriginalSelectedProducts] = useState([]);
  const [removedProductIds, setRemovedProductIds] = useState([]);
  const [paymentMethods, setPaymentMethods] = useState([]);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false); // Added for action loading state
  const { columnWidths, setColumnWidths, handleResizeStart } = useResizableColumns(ORDER_COLUMNS_STORAGE_KEY, DEFAULT_ORDER_COLUMN_WIDTHS);

  const statusOptions = [
    { value: 0, label: 'Новый' },
    { value: 1, label: 'Готов к отправке' },
    { value: 2, label: 'Отправлен' },
  ];

  const productStatusByValue = {
    1: 'На продаже',
    2: 'Забронировано',
    3: 'Продано',
  };

  const fetchPaymentMethods = useCallback(async () => {
    try {
      setLoading(true);
      setError('');
      const response = await axios.get('/api/payment_methods', {
        headers: { Authorization: token },
      });
      setPaymentMethods(response.data.map((method) => ({ value: method.method, label: method.method })));
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to fetch payment methods');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    fetchPaymentMethods();
  }, [fetchPaymentMethods]);

  const validateForm = () => {
    const effectiveSelectedProducts = selectedProducts.filter((id) => !removedProductIds.includes(id));
    if (!newOrder.name.trim()) {
      setError('Order name is required');
      return false;
    }
    const invalidPayment = newOrder.payments.some(
      (p) => p.method && (!p.amount || parseFloat(p.amount) <= 0)
    );
    if (invalidPayment) {
      setError('All payments must have a valid method and positive amount');
      return false;
    }
    if (effectiveSelectedProducts.length === 0) {
      setError('At least one product must be selected');
      return false;
    }
    // Validate that all selectedProducts have corresponding products
    const missingProducts = effectiveSelectedProducts.filter((id) => !products.find((p) => p.id === id));
    if (missingProducts.length > 0) {
      setError(`Some selected products are not available: ${missingProducts.join(', ')}`);
      return false;
    }
    return true;
  };

  const handleEditOrder = (order) => {
    const orderProductIDs = order.components?.map((p) => p.id) || [];
    setEditingOrderId(order.id);
    setNewOrder({
      ...order,
      components: order.components || [],
      name: order.name || '',
      payments: order.payments?.length > 0
        ? order.payments.map((p) => ({ id: p.id, method: p.method, amount: p.amount.toString(), comment: p.comment || '' }))
        : [{ method: '', amount: '', comment: '' }],
      debt: order.debt || 0,
    });
    setOriginalSelectedProducts(orderProductIDs);
    setSelectedProducts(orderProductIDs);
    setRemovedProductIds([]);
    setShowEditProducts(false);
    setShowOrderModal(true);
    setShowAddForm(true);
  };

  const handleSaveOrder = async () => {
    if (!validateForm()) return;
    try {
      setSubmitting(true);
      setError('');
      const formattedPayments = newOrder.payments
        .filter((p) => p.method && p.amount)
        .map((p) => ({ id: p.id || 0, method: p.method, amount: parseFloat(p.amount), comment: p.comment }));
      const effectiveSelectedProducts = selectedProducts.filter((id) => !removedProductIds.includes(id));
      const updatedOrder = {
        ...newOrder,
        components: effectiveSelectedProducts.map((id) => products.find((p) => p.id === id)).filter(Boolean), // Ensure no undefined
        quantity: effectiveSelectedProducts.length,
        name: newOrder.name,
        payments: formattedPayments,
        debt: newOrder.debt,
      };
      const response = await axios.put(`/api/orders/${newOrder.id}`, updatedOrder, {
        headers: { Authorization: token },
      });
      const serverOrder = response?.data?.order;
      setOrders(orders.map((o) => (o.id === newOrder.id ? (serverOrder || { ...o, ...updatedOrder }) : o)));
      resetForm();
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to update order');
    } finally {
      setSubmitting(false);
    }
  };

  const handleAddOrder = async () => {
    if (!validateForm()) return;
    try {
      setSubmitting(true);
      setError('');
      console.log('Creating order with products:', selectedProducts, 'Name:', newOrder.name, 'Description:', newOrder.description, 'Payments:', newOrder.payments);
      const formattedPayments = newOrder.payments
        .filter((p) => p.method && p.amount)
        .map((p) => ({ method: p.method, amount: parseFloat(p.amount), comment: p.comment }));
      const orderData = {
        components: selectedProducts.map((id) => products.find((p) => p.id === id)).filter(Boolean), // Ensure no undefined
        quantity: selectedProducts.length,
        status: newOrder.status,
        name: newOrder.name,
        description: newOrder.description,
        payments: formattedPayments,
        debt: newOrder.debt,
      };
      const response = await axios.post('/api/orders', orderData, {
        headers: { Authorization: token },
      });
      setOrders([...orders, response.data]);
      resetForm();
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to add order');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDeleteOrder = async (id) => {
    try {
      setSubmitting(true);
      setError('');
      await axios.delete(`/api/orders/${id}`, {
        headers: { Authorization: token },
      });
      setOrders(orders.filter((o) => o.id !== id));
    } catch (error) {
      setError(error.response?.data?.error || 'Failed to delete order');
    } finally {
      setSubmitting(false);
    }
  };

  const resetForm = () => {
    setEditingOrderId(null);
    setShowOrderModal(false);
    setShowAddForm(false);
    setShowConfirmStep(false);
    setShowEditProducts(false);
    setOriginalSelectedProducts([]);
    setRemovedProductIds([]);
    setSelectedProducts([]);
    setNewOrder({
      id: null,
      components: [],
      quantity: 0,
      status: 0,
      name: '',
      description: '',
      payments: [{ method: '', amount: '', comment: '' }],
      debt: 0,
    });
  };

  const handlePaymentChange = (index, field, value) => {
    const updatedPayments = [...newOrder.payments];
    updatedPayments[index] = { ...updatedPayments[index], [field]: value };
    setNewOrder({ ...newOrder, payments: updatedPayments });
  };

  const addPayment = () => {
    setNewOrder({
      ...newOrder,
      payments: [...newOrder.payments, { method: '', amount: '', comment: '' }],
    });
  };

  const removePayment = (index) => {
    if (newOrder.payments.length > 1) {
      setNewOrder({
        ...newOrder,
        payments: newOrder.payments.filter((_, i) => i !== index),
      });
    }
  };

  const handleProductSelection = useCallback((productId, shouldSelect) => {
    if (shouldSelect) {
      setSelectedProducts((prev) => (prev.includes(productId) ? prev : [...prev, productId]));
      setRemovedProductIds((prev) => prev.filter((id) => id !== productId));
      return;
    }

    const wasInOriginalOrder = originalSelectedProducts.includes(productId);
    if (editingOrderId && wasInOriginalOrder) {
      setRemovedProductIds((prev) => (prev.includes(productId) ? prev : [...prev, productId]));
      return;
    }

    setSelectedProducts((prev) => prev.filter((id) => id !== productId));
    setRemovedProductIds((prev) => prev.filter((id) => id !== productId));
  }, [editingOrderId, originalSelectedProducts]);

  const getEffectiveSelectedProducts = useCallback(() => (
    selectedProducts.filter((id) => !removedProductIds.includes(id))
  ), [selectedProducts, removedProductIds]);

  const getProductStatusLabel = useCallback((status) => productStatusByValue[status] || 'Unknown', [productStatusByValue]);

  const renderProductDetails = useCallback((id) => {
    const product = products.find((p) => p.id === id);
    const isRemovedPending = removedProductIds.includes(id);
    return (
      <div
        key={id}
        className={`border rounded p-2 bg-white text-sm ${isRemovedPending ? 'opacity-70 border-amber-300 bg-amber-50' : ''}`}
      >
        <div className="font-medium">
          {product?.name || 'Unknown'} {isRemovedPending && <span className="text-amber-700">(Removed, pending confirm)</span>}
        </div>
        <div className="text-gray-600">
          ID: {id} | Status: {getProductStatusLabel(product?.status)}
        </div>
        <div className="text-gray-600">
          Qty: {product?.count ?? 0} | One Price: {product?.onePrice || 0} | Total: {product?.summaRubSoSkidkoj || 0}
        </div>
        {product?.description && <div className="text-gray-500">{product.description}</div>}
      </div>
    );
  }, [products, removedProductIds, getProductStatusLabel]);

  const filteredOrders = useMemo(() => {
    const lowerFilter = filter.toLowerCase();
    return (orders || []).filter(
      (o) =>
        (statusFilter === '' || o.status?.toString() === statusFilter) &&
        (lowerFilter === '' ||
          o.id?.toString().includes(lowerFilter) ||
          (o.name || '').toLowerCase().includes(lowerFilter) ||
          (o.description || '').toLowerCase().includes(lowerFilter) ||
          statusOptions.find((opt) => opt.value === o.status)?.label.toLowerCase().includes(lowerFilter) ||
          (o.components || []).some((p) => (p.name || '').toLowerCase().includes(lowerFilter)) ||
          (o.payments || []).some((p) => `${p.method || ''} ${p.comment || ''}`.toLowerCase().includes(lowerFilter)))
    );
  }, [orders, statusFilter, filter]);

  const columns = [
    { key: 'id', label: 'ID', width: columnWidths.id },
    { key: 'status', label: 'Status', width: columnWidths.status },
    { key: 'name', label: 'Name', width: columnWidths.name },
    { key: 'description', label: 'Description', width: columnWidths.description },
    { key: 'components', label: 'Components', width: columnWidths.components },
    { key: 'payments', label: 'Payments', width: columnWidths.payments },
    { key: 'debt', label: 'Debt', width: columnWidths.debt },
    { key: 'actions', label: 'Actions', width: columnWidths.actions },
  ];

  if (loading || !products) {
    return <p className="text-gray-600">Loading...</p>;
  }

  return (
    <div className="wm-surface">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-bold">Order Management</h2>
        <div className="flex gap-2">
          <button
            onClick={() => {
              setShowAddForm(true);
              setShowOrderModal(true);
              setShowEditProducts(true);
              setOriginalSelectedProducts([]);
              setRemovedProductIds([]);
              setNewOrder({
                id: null,
                components: [],
                quantity: 0,
                status: 0,
                name: '',
                description: '',
                payments: [{ method: '', amount: '', comment: '' }],
                debt: 0,
              });
              setSelectedProducts([]);
            }}
            className="wm-btn wm-btn-primary"
            disabled={submitting || products.length === 0}
          >
            Add Order
          </button>
          <button
            onClick={() =>
              exportToCSV(
                (orders || []).map((o) => ({
                  id: o.id,
                  status: statusOptions.find((opt) => opt.value === o.status)?.label || 'Unknown',
                  name: o.name || '',
                  description: o.description,
                  components: o.components?.map((p) => p.name).join(', ') || '',
                  payments: o.payments?.map((p) => `${p.method}: ${p.amount}`).join(', ') || '',
                  debt: o.debt,
                })),
                'orders.csv'
              )
            }
            className="wm-btn wm-btn-primary"
            disabled={submitting}
          >
            Export to CSV
          </button>
          <button
            onClick={() => {
              setFilter('');
              setColumnWidths(DEFAULT_ORDER_COLUMN_WIDTHS);
              window.localStorage.removeItem(ORDER_COLUMNS_STORAGE_KEY);
            }}
            className="wm-btn"
            disabled={submitting}
          >
            Reset table view
          </button>
        </div>
      </div>

      {error && (
        <div className="mb-4 p-4 bg-red-100 text-red-700 rounded flex justify-between items-center">
          <span>{error}</span>
          <button onClick={() => setError('')} className="wm-btn wm-btn-danger">
            ×
          </button>
        </div>
      )}

      {showOrderModal && (
        <div
          className="fixed inset-0 bg-slate-900/30 backdrop-blur-[2px] flex items-center justify-center z-50"
          onClick={(e) => {
            if (e.target === e.currentTarget) resetForm();
          }}
        >
          <div className="bg-white p-6 rounded-xl shadow-xl w-full max-w-2xl max-h-[80vh] overflow-y-auto">
            <h3 className="text-lg font-bold mb-4">
              {showConfirmStep ? 'Confirm Order' : editingOrderId ? 'Edit Order' : 'Create New Order'}
            </h3>
            {!showConfirmStep ? (
              <div className="space-y-4">
                <div>
                  <label className="block text-sm mb-1">Order Name</label>
                  <input
                    type="text"
                    value={newOrder.name}
                    onChange={(e) => setNewOrder({ ...newOrder, name: e.target.value })}
                    className="wm-input w-full"
                  />
                </div>
                <div>
                  <label className="block text-sm mb-1">Order Description</label>
                  <input
                    type="text"
                    value={newOrder.description}
                    onChange={(e) => setNewOrder({ ...newOrder, description: e.target.value })}
                    className="wm-input w-full"
                  />
                </div>
                <div>
                  <label className="block text-sm mb-1">Status</label>
                  <select
                    value={newOrder.status}
                    onChange={(e) => setNewOrder({ ...newOrder, status: parseInt(e.target.value) })}
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
                  <label className="block text-sm mb-1">Debt</label>
                  <input
                    type="number"
                    value={newOrder.debt}
                    onChange={(e) => setNewOrder({ ...newOrder, debt: parseFloat(e.target.value) || 0 })}
                    className="wm-input w-full"
                    placeholder="Debt Amount"
                  />
                </div>
                <div>
                  <label className="block text-sm mb-1">Payments</label>
                  {newOrder.payments.map((payment, index) => (
                    <div key={index} className="flex gap-2 mb-2">
                      <select
                        value={payment.method}
                        onChange={(e) => handlePaymentChange(index, 'method', e.target.value)}
                        className="wm-select w-1/3"
                        disabled={submitting}
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
                        onChange={(e) => handlePaymentChange(index, 'amount', e.target.value)}
                        className="wm-input w-1/3"
                        disabled={submitting}
                      />
                      <input
                        type="text"
                        placeholder="Comment"
                        value={payment.comment}
                        onChange={(e) => handlePaymentChange(index, 'comment', e.target.value)}
                        className="wm-input w-1/3"
                        disabled={submitting}
                      />
                      {newOrder.payments.length > 1 && (
                        <button
                          onClick={() => removePayment(index)}
                          className="wm-btn wm-btn-danger"
                          disabled={submitting}
                        >
                          Remove
                        </button>
                      )}
                    </div>
                  ))}
                  <button
                    onClick={addPayment}
                    className="wm-btn wm-btn-primary"
                    disabled={submitting}
                  >
                    Add Payment
                  </button>
                </div>
                <div>
                  <label className="block text-sm mb-1">Selected Products</label>
                  {editingOrderId && !showEditProducts ? (
                    <div className="border rounded p-3 bg-slate-50">
                      {selectedProducts.length === 0 ? (
                        <p className="text-gray-500">No products selected</p>
                      ) : (
                        <div className="space-y-2 mb-3">
                          {selectedProducts.map((id) => renderProductDetails(id))}
                        </div>
                      )}
                      <button
                        type="button"
                        onClick={() => setShowEditProducts(true)}
                        className="wm-btn wm-btn-primary"
                        disabled={submitting}
                      >
                        Edit Products
                      </button>
                    </div>
                  ) : (
                    <>
                      <div className="max-h-64 overflow-y-auto border rounded p-2">
                        {products.length === 0 ? (
                          <p className="text-gray-500">No products available</p>
                        ) : (
                          products
                            .filter((p) => p.status === 1 || selectedProducts.includes(p.id))
                            .map((product) => {
                              const isSelected = selectedProducts.includes(product.id);
                              const isRemovedPending = removedProductIds.includes(product.id);
                              const canAdd = product.status === 1 && !isSelected;
                              const canRestore = isSelected && isRemovedPending;
                              const canRemove = isSelected && !isRemovedPending;
                              return (
                                <div
                                  key={product.id}
                                  className={`flex items-center justify-between p-2 rounded ${
                                    isRemovedPending ? 'bg-amber-50' : isSelected ? 'bg-blue-50' : 'hover:bg-gray-50'
                                  }`}
                                >
                                  <span>
                                    {product.name}
                                    {isRemovedPending && <span className="text-amber-700 ml-2">(Removed pending)</span>}
                                  </span>
                                  {canAdd ? (
                                    <button
                                      type="button"
                                      onClick={() => handleProductSelection(product.id, true)}
                                      className="wm-btn wm-btn-primary"
                                      disabled={submitting}
                                    >
                                      Add
                                    </button>
                                  ) : canRestore ? (
                                    <button
                                      type="button"
                                      onClick={() => handleProductSelection(product.id, true)}
                                      className="wm-btn wm-btn-primary"
                                      disabled={submitting}
                                    >
                                      Restore
                                    </button>
                                  ) : (
                                    <button
                                      type="button"
                                      onClick={() => handleProductSelection(product.id, false)}
                                      className="wm-btn wm-btn-danger"
                                      disabled={submitting || !canRemove}
                                    >
                                      Remove
                                    </button>
                                  )}
                                </div>
                              );
                            })
                        )}
                      </div>
                      {editingOrderId && (
                        <div className="mt-2 flex justify-end">
                          <button
                            type="button"
                            onClick={() => setShowEditProducts(false)}
                            className="wm-btn"
                            disabled={submitting}
                          >
                            Done
                          </button>
                        </div>
                      )}
                      <p className="text-sm text-gray-500 mt-2">
                        Manage products with Add/Remove buttons. Removed products are marked and will be deleted only after Confirm Order.
                      </p>
                    </>
                  )}
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={() => {
                      if (getEffectiveSelectedProducts().length > 0) {
                        setShowConfirmStep(true);
                      } else {
                        setError('Please select at least one product.');
                      }
                    }}
                    className="wm-btn wm-btn-primary"
                    disabled={submitting}
                  >
                    Next
                  </button>
                  <button
                    onClick={resetForm}
                    className="wm-btn"
                    disabled={submitting}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            ) : (
              <div className="space-y-4">
                <div>
                  <h3 className="font-bold">Order Summary</h3>
                  <p>
                    <strong>Name:</strong> {newOrder.name || 'None'}
                  </p>
                  <p>
                    <strong>Description:</strong> {newOrder.description || 'None'}
                  </p>
                  <p>
                    <strong>Status:</strong>{' '}
                    {statusOptions.find((opt) => opt.value === newOrder.status)?.label || 'Unknown'}
                  </p>
                  <p>
                    <strong>Payments:</strong>
                  </p>
                  <ul className="list-disc pl-5">
                    {newOrder.payments
                      .filter((p) => p.method && p.amount)
                      .map((p, i) => (
                        <li key={i}>{`${p.method}: ${p.amount}${p.comment ? ` (${p.comment})` : ''}`}</li>
                      ))}
                  </ul>
                  <p>
                    <strong>Selected Products:</strong>
                  </p>
                  <ul className="list-disc pl-5">
                    {selectedProducts.map((id) => {
                      const product = products.find((p) => p.id === id);
                      const isRemovedPending = removedProductIds.includes(id);
                      return (
                        <li key={id}>
                          {product?.name || 'Unknown'} (ID: {id}, Status: {productStatusByValue[product?.status] || 'Unknown'}, Qty: {product?.count ?? 0}, One: {product?.onePrice || 0}, Total: {product?.summaRubSoSkidkoj || 0}{isRemovedPending ? ', Removed pending confirm' : ''})
                        </li>
                      );
                    })}
                  </ul>
                  <p>
                    <strong>Debt:</strong> {newOrder.debt || 0}
                  </p>
                </div>
                <div className="flex justify-end gap-2">
                  <button
                    onClick={editingOrderId ? handleSaveOrder : handleAddOrder}
                    className="wm-btn wm-btn-primary"
                    disabled={submitting}
                  >
                    {submitting ? 'Processing...' : 'Confirm Order'}
                  </button>
                  <button
                    onClick={() => setShowConfirmStep(false)}
                    className="wm-btn"
                    disabled={submitting}
                  >
                    Back
                  </button>
                  <button
                    onClick={resetForm}
                    className="wm-btn"
                    disabled={submitting}
                  >
                    Cancel
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      <div className="wm-table-wrap relative max-h-[500px]">
        <table className="wm-table text-sm">
          <thead>
            <tr>
              {columns.map((column) => (
                <th
                  key={column.key}
                  className={`wm-th relative font-semibold ${column.key === "actions" ? "wm-action-cell" : ""}`}
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
            {filteredOrders.length === 0 ? (
              <tr>
                <td colSpan={columns.length} className="wm-empty">
                  No orders found
                </td>
              </tr>
            ) : (
              filteredOrders.map((order) => (
                <tr key={order.id}>
                  <td className="wm-td">{order.id}</td>
                  <td className="wm-td">
                    <span className={`wm-status-badge ${ORDER_STATUS_BADGE_CLASS[order.status] || "wm-status-new"}`}>
                      {statusOptions.find((opt) => opt.value === order.status)?.label || 'Unknown'}
                    </span>
                  </td>
                  <td className="wm-td">{order.name || 'None'}</td>
                  <td className="wm-td">{order.description}</td>
                  <td className="wm-td">
                    {order.components?.map((p, index) => (
                      <div key={index}>{p.name}</div>
                    )) || ''}
                  </td>
                  <td className="wm-td">
                    {order.payments?.map((p, index) => (
                      <div key={index}>{p.method}: {p.amount}</div>
                    )) || ''}
                  </td>
                  <td className="wm-td">{order.debt || 0}</td>
                  <td className="wm-td wm-action-cell">
                    <div className="wm-action-buttons">
                    <button
                      onClick={() => handleEditOrder(order)}
                      className="wm-btn"
                      disabled={submitting}
                    >
                      Edit
                    </button>
                    <button
                      onClick={() => handleDeleteOrder(order.id)}
                      className="wm-btn wm-btn-danger"
                      disabled={submitting}
                    >
                      Delete
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
