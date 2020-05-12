package tracer

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Bhinneka/golib"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
)

// Middleware for wrap from http inbound (request from client)
func Middleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		tracer := opentracing.GlobalTracer()
		operationName := fmt.Sprintf("%s %s%s", req.Method, req.Host, req.URL.Path)

		var span opentracing.Span
		var ctx context.Context
		if spanCtx, err := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(req.Header)); err != nil {
			span, ctx = opentracing.StartSpanFromContext(req.Context(), operationName)
			ext.SpanKindRPCServer.Set(span)
		} else {
			span = tracer.StartSpan(operationName, ext.RPCServerOption((spanCtx)))
			ctx = opentracing.ContextWithSpan(req.Context(), span)
			ext.SpanKindRPCClient.Set(span)
		}

		body, _ := ioutil.ReadAll(req.Body)
		bodyString := string(golib.MaskJSONPassword(body))
		bodyString = golib.MaskPassword(bodyString)

		isRemoveBody, ok := req.Context().Value("remove-tag-body").(bool)
		if ok {
			if !isRemoveBody {
				span.SetTag("body", bodyString)
			}
		} else {
			span.SetTag("body", bodyString)
		}

		req.Body = ioutil.NopCloser(bytes.NewBuffer(body)) // reuse body

		span.SetTag("http.headers", req.Header)
		ext.HTTPUrl.Set(span, req.Host+req.RequestURI)
		ext.HTTPMethod.Set(span, req.Method)

		span.LogEvent("start_handling_request")

		defer func() {
			span.LogEvent("complete_handling_request")
			span.Finish()
		}()

		h.ServeHTTP(w, req.WithContext(ctx))
	})
}
