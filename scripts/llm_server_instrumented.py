"""Wrapper around llama_cpp.server that adds /metrics with Prometheus
HTTP metrics (latency, request rate, errors) and LLM-specific token
counters extracted from chat/completion responses."""
import json
from prometheus_client import Counter, Histogram
from prometheus_fastapi_instrumentator import Instrumentator
from starlette.middleware.base import BaseHTTPMiddleware

import llama_cpp.server.app as _srv_app
import llama_cpp.server.__main__ as _srv_main

# ---- LLM-specific metrics --------------------------------------------------
PROMPT_TOKENS = Counter(
    "llm_prompt_tokens_total", "Cumulative prompt tokens", ["model", "endpoint"]
)
COMPLETION_TOKENS = Counter(
    "llm_completion_tokens_total", "Cumulative completion tokens", ["model", "endpoint"]
)
TOTAL_TOKENS = Counter(
    "llm_total_tokens_total", "Cumulative total tokens (prompt + completion)", ["model", "endpoint"]
)
REQUESTS = Counter(
    "llm_requests_total", "LLM API requests", ["model", "endpoint", "status"]
)
LATENCY = Histogram(
    "llm_request_duration_seconds", "End-to-end request latency",
    ["model", "endpoint"],
    buckets=(0.1, 0.5, 1, 2, 5, 10, 20, 30, 60, 120, 300),
)


class TokenMetricsMiddleware(BaseHTTPMiddleware):
    """Parses chat/completion JSON responses to extract token usage."""

    LLM_PATHS = ("/v1/chat/completions", "/v1/completions")

    async def dispatch(self, request, call_next):
        import time
        start = time.perf_counter()
        response = await call_next(request)
        if request.url.path not in self.LLM_PATHS:
            return response

        # Read & rebuild streaming body
        body = b""
        async for chunk in response.body_iterator:
            body += chunk
        from starlette.responses import Response
        new_resp = Response(
            content=body, status_code=response.status_code,
            headers=dict(response.headers), media_type=response.media_type,
        )

        duration = time.perf_counter() - start
        endpoint = request.url.path
        model = "unknown"
        try:
            data = json.loads(body)
            model = data.get("model", model)
            usage = data.get("usage") or {}
            if usage:
                PROMPT_TOKENS.labels(model, endpoint).inc(usage.get("prompt_tokens", 0))
                COMPLETION_TOKENS.labels(model, endpoint).inc(usage.get("completion_tokens", 0))
                TOTAL_TOKENS.labels(model, endpoint).inc(usage.get("total_tokens", 0))
        except Exception:
            pass

        REQUESTS.labels(model, endpoint, str(response.status_code)).inc()
        LATENCY.labels(model, endpoint).observe(duration)
        return new_resp


# ---- Monkey-patch create_app to install instrumentation --------------------
_orig_create_app = _srv_app.create_app

def _patched_create_app(*args, **kwargs):
    app = _orig_create_app(*args, **kwargs)
    app.add_middleware(TokenMetricsMiddleware)
    Instrumentator(
        should_group_status_codes=False,
        should_ignore_untemplated=True,
    ).instrument(app).expose(app, endpoint="/metrics", include_in_schema=False)
    return app

_srv_app.create_app = _patched_create_app
_srv_main.create_app = _patched_create_app

if __name__ == "__main__":
    _srv_main.main()
