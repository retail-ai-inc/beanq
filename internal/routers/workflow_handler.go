package routers

import (
	"net/http"

	"github.com/retail-ai-inc/beanq/v4/helper/berror"
	"github.com/retail-ai-inc/beanq/v4/helper/response"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WorkFlow struct {
	workflowCollection *mongo.Collection
}

func NewWorkFlow(collection *mongo.Collection) *WorkFlow {
	return &WorkFlow{workflowCollection: collection}
}

func (t *WorkFlow) List(w http.ResponseWriter, r *http.Request) {
	result, cancel := response.Get()
	defer cancel()

	if t.workflowCollection == nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = "workflow storage is not configured"
		_ = result.Json(w, http.StatusServiceUnavailable)
		return
	}

	page, err := parsePage(r)
	if err != nil {
		writeBadRequest(w, err)
		return
	}
	query := r.URL.Query()

	ctx := r.Context()

	skip := (page.Page - 1) * page.PageSize
	if skip < 0 {
		skip = 0
	}
	opts := options.Find()
	opts.SetSkip(skip)
	opts.SetLimit(page.PageSize)
	opts.SetSort(bson.D{{Key: "CreatedAt", Value: -1}})

	filter := bson.M{}
	if channel := query.Get("channel"); channel != "" {
		filter["Channel"] = channel
	}
	if topic := query.Get("topic"); topic != "" {
		filter["Topic"] = topic
	}
	if status := query.Get("status"); status != "" {
		filter["Status"] = status
	}
	cursor, err := t.workflowCollection.Find(ctx, filter, opts)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var data []bson.M
	if err := cursor.All(ctx, &data); err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	total, err := t.workflowCollection.CountDocuments(ctx, filter)
	if err != nil {
		result.Code = berror.InternalServerErrorCode
		result.Msg = err.Error()
		_ = result.Json(w, http.StatusInternalServerError)
		return
	}
	result.Data = map[string]any{"data": data, "total": newPageMeta(page.Page, page.PageSize, total).TotalPages, "cursor": page.Page}
	_ = result.Json(w, http.StatusOK)
}

func (t *WorkFlow) Delete(w http.ResponseWriter, r *http.Request) {
	res, cancel := response.Get()
	defer cancel()

	if t.workflowCollection == nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = "workflow storage is not configured"
		_ = res.Json(w, http.StatusServiceUnavailable)
		return
	}

	ctx := r.Context()
	id := r.PostFormValue("id")
	if id == "" {
		res.Code = berror.MissParameterCode
		res.Msg = "missing id"
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	nid, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		res.Code = berror.MissParameterCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusBadRequest)
		return
	}

	result, err := t.workflowCollection.DeleteOne(ctx, bson.M{"_id": nid})
	if err != nil {
		res.Code = berror.InternalServerErrorCode
		res.Msg = err.Error()
		_ = res.Json(w, http.StatusInternalServerError)
		return
	}
	if result.DeletedCount == 0 {
		writeAPIError(w, http.StatusNotFound, berror.MissParameterCode, "workflow not found")
		return
	}

	res.Data = result.DeletedCount
	_ = res.Json(w, http.StatusOK)
}
