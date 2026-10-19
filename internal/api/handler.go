package api

import (
	"net/http"

	"github.com/csrrmrvll/bearly/internal/accounts"
	"github.com/csrrmrvll/bearly/internal/auth/sessions"
	"github.com/csrrmrvll/bearly/internal/httpx"
	"github.com/csrrmrvll/bearly/internal/logging"
	"github.com/csrrmrvll/bearly/internal/orders"
	"github.com/csrrmrvll/bearly/internal/storefront"
)

type integrationOrderResponse struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	TotalCents int64  `json:"total_cents"`
	CreatedAt  string `json:"created_at"`
}

type orderItemResponse struct {
	ProductID   int64  `json:"product_id"`
	ProductName string `json:"product_name"`
	Quantity    int64  `json:"quantity"`
	PriceCents  int64  `json:"price_cents"`
}

type productResponse struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ImagePath   string `json:"image_path"`
	PriceCents  int64  `json:"price_cents"`
}

type orderResponse struct {
	ID         int64  `json:"id"`
	Status     string `json:"status"`
	TotalCents int64  `json:"total_cents"`
	CreatedAt  string `json:"created_at"`
}

type Handler struct {
	accountStore      *accounts.Store
	orderStore        *orders.Store
	productStore      *storefront.Store
	apiStore          *Store
	logger            *logging.Logger
	maxProductResults int64
}

func NewHandler(accountStore *accounts.Store, orderStore *orders.Store, productStore *storefront.Store, apiStore *Store, logger *logging.Logger, maxProductResults int) *Handler {
	return &Handler{
		accountStore: accountStore, orderStore: orderStore, productStore: productStore, apiStore: apiStore,
		logger: logger, maxProductResults: int64(maxProductResults),
	}
}

func (handler *Handler) AccountOrders(responseWriter http.ResponseWriter, request *http.Request) {
	current, ok := handler.requireAuthentication(responseWriter, request)
	if !ok {
		return
	}
	orders, err := handler.orderStore.ListForUser(request.Context(), current.User.ID)
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	ordersResponse := make([]integrationOrderResponse, 0, len(orders))
	for _, order := range orders {
		ordersResponse = append(ordersResponse, integrationOrderResponse{
			ID: order.ID, Status: order.Status, TotalCents: order.TotalCents, CreatedAt: order.CreatedAt,
		})
	}
	httpx.RespondWithJSON(responseWriter, http.StatusOK, map[string]any{"orders": ordersResponse})
}

func (handler *Handler) Order(responseWriter http.ResponseWriter, request *http.Request) {
	current, ok := handler.requireAuthentication(responseWriter, request)
	if !ok {
		return
	}
	orderID, valid := httpx.ParseSafeInteger(request.PathValue("id"))
	if !valid {
		httpx.RespondWithJSON(responseWriter, http.StatusNotFound, map[string]string{"error": "Order not found"})
		return
	}
	order, found, err := handler.orderStore.FindByID(request.Context(), orderID)
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	if !found || order.UserID != current.User.ID {
		httpx.RespondWithJSON(responseWriter, http.StatusNotFound, map[string]string{"error": "Order not found"})
		return
	}
	items, err := handler.orderStore.ListItems(request.Context(), order.ID)
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	itemResponses := make([]orderItemResponse, 0, len(items))
	for _, item := range items {
		itemResponses = append(itemResponses, orderItemResponse{ProductID: item.ProductID, ProductName: item.ProductName, Quantity: item.Quantity, PriceCents: item.PriceCents})
	}
	orderResponse := orderResponse{ID: order.ID, Status: order.Status, TotalCents: order.TotalCents, CreatedAt: order.CreatedAt}
	httpx.RespondWithJSON(responseWriter, http.StatusOK, map[string]any{"order": orderResponse, "items": itemResponses})
}

func (handler *Handler) Products(responseWriter http.ResponseWriter, request *http.Request) {
	products, err := handler.productStore.ListAllProducts(request.Context())
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	responseWriter.Header().Set("Access-Control-Allow-Origin", "*")
	productsResponse := make([]productResponse, 0, len(products))
	for _, product := range products {
		if product.IsActive == false {
			continue
		}
		productsResponse = append(productsResponse, productResponse{
			ID:          product.ID,
			Name:        product.Name,
			Description: product.Description,
			ImagePath:   product.ImagePath,
			PriceCents:  product.PriceCents,
		})
	}
	httpx.RespondWithJSON(responseWriter, http.StatusOK, map[string]any{"products": productsResponse})
}

func (handler *Handler) ProductPreflight(responseWriter http.ResponseWriter, _ *http.Request) {
	responseWriter.Header().Set("Access-Control-Allow-Origin", "*")
	responseWriter.Header().Set("Access-Control-Allow-Methods", http.MethodGet)
	responseWriter.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) WarehouseOrders(responseWriter http.ResponseWriter, request *http.Request) {
	apiKey, found, err := handler.apiStore.FindKey(request.Context(), request.Header.Get("X-API-Key"))
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	if !found {
		httpx.RespondWithError(responseWriter, http.StatusUnauthorized, "Invalid API key")
		return
	}
	if apiKey.Scope != "orders:read" {
		httpx.RespondWithError(responseWriter, http.StatusForbidden, "API key scope is not allowed")
		return
	}
	orders, err := handler.orderStore.ListAll(request.Context())
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return
	}
	responses := make([]integrationOrderResponse, 0, len(orders))
	for _, order := range orders {
		responses = append(responses, integrationOrderResponse{
			ID: order.ID, Status: order.Status, TotalCents: order.TotalCents, CreatedAt: order.CreatedAt,
		})
	}
	httpx.RespondWithJSON(responseWriter, http.StatusOK, map[string]any{
		"integration": "Warehouse Fulfillment Integration",
		"orders":      responses,
	})
}

func (handler *Handler) requireAuthentication(responseWriter http.ResponseWriter, request *http.Request) (accounts.CurrentSession, bool) {
	current, found, err := sessions.Current(request, handler.accountStore)
	if err != nil {
		handler.internalError(responseWriter, request, err)
		return accounts.CurrentSession{}, false
	}
	if !found {
		httpx.RespondWithJSON(responseWriter, http.StatusUnauthorized, map[string]string{"error": "Authentication required"})
		return accounts.CurrentSession{}, false
	}
	return current, true
}

func (handler *Handler) internalError(responseWriter http.ResponseWriter, request *http.Request, err error) {
	_ = handler.logger.Event("unhandled_error", map[string]any{"method": request.Method, "path": request.URL.Path, "message": err.Error()})
	httpx.RespondWithError(responseWriter, http.StatusInternalServerError, err.Error())
}
