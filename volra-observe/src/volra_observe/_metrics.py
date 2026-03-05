"""Core metrics, instrumentation, and public API for volra-observe."""

from __future__ import annotations

import contextlib
import functools
import time
from typing import Any, Callable, Generator

from prometheus_client import (
    CollectorRegistry,
    Counter,
    Histogram,
    start_http_server,
)

from volra_observe.pricing import estimate_cost

# ---------------------------------------------------------------------------
# Prometheus metrics (registered on first init)
# ---------------------------------------------------------------------------

_registry = CollectorRegistry()
_initialized = False

tokens_total = Counter(
    "volra_llm_tokens_total",
    "Total LLM tokens consumed",
    ["model", "type"],
    registry=_registry,
)

cost_total = Counter(
    "volra_llm_cost_dollars_total",
    "Estimated LLM cost in USD",
    ["model"],
    registry=_registry,
)

request_duration = Histogram(
    "volra_llm_request_duration_seconds",
    "LLM request duration",
    ["model", "status"],
    buckets=(0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0),
    registry=_registry,
)

errors_total = Counter(
    "volra_llm_errors_total",
    "Total LLM errors",
    ["model", "error_type"],
    registry=_registry,
)

tool_calls_total = Counter(
    "volra_tool_calls_total",
    "Total tool/function calls",
    ["tool_name"],
    registry=_registry,
)


# ---------------------------------------------------------------------------
# Public API
# ---------------------------------------------------------------------------


def init(port: int = 9101, auto_patch: bool = True) -> None:
    """Initialize volra-observe: start Prometheus server and optionally patch LLM SDKs.

    Args:
        port: Port for the Prometheus HTTP metrics server.
        auto_patch: If True, monkey-patch OpenAI and Anthropic SDKs.
    """
    global _initialized
    if _initialized:
        return
    start_http_server(port, registry=_registry)
    if auto_patch:
        _patch_openai()
        _patch_anthropic()
    _initialized = True


def record(
    model: str,
    input_tokens: int,
    output_tokens: int,
    duration_s: float,
    status: str = "ok",
    tool_names: list[str] | None = None,
) -> None:
    """Record metrics for a single LLM call."""
    tokens_total.labels(model=model, type="input").inc(input_tokens)
    tokens_total.labels(model=model, type="output").inc(output_tokens)
    cost = estimate_cost(model, input_tokens, output_tokens)
    if cost > 0:
        cost_total.labels(model=model).inc(cost)
    request_duration.labels(model=model, status=status).observe(duration_s)
    if tool_names:
        for name in tool_names:
            tool_calls_total.labels(tool_name=name).inc()


def record_error(model: str, error_type: str) -> None:
    """Record an LLM error."""
    errors_total.labels(model=model, error_type=error_type).inc()


def record_tool_call(tool_name: str, count: int = 1) -> None:
    """Record tool/function calls explicitly."""
    tool_calls_total.labels(tool_name=tool_name).inc(count)


def track_llm(model: str = "unknown") -> Callable:
    """Decorator that tracks LLM call metrics.

    Usage::

        @track_llm(model="gpt-4o")
        def call_llm(prompt: str) -> str:
            return openai_client.chat.completions.create(...)
    """

    def decorator(fn: Callable) -> Callable:
        @functools.wraps(fn)
        def wrapper(*args: Any, **kwargs: Any) -> Any:
            start = time.monotonic()
            try:
                result = fn(*args, **kwargs)
                duration = time.monotonic() - start
                input_tok, output_tok = _extract_tokens(result)
                record(model, input_tok, output_tok, duration, status="ok")
                return result
            except Exception as exc:
                duration = time.monotonic() - start
                request_duration.labels(model=model, status="error").observe(duration)
                record_error(model, _classify_error(exc))
                raise

        return wrapper

    return decorator


@contextlib.contextmanager
def llm_context(model: str = "unknown") -> Generator[dict[str, Any], None, None]:
    """Context manager for tracking LLM calls.

    Usage::

        with llm_context(model="claude-sonnet-4-20250514") as ctx:
            response = anthropic_client.messages.create(...)
            ctx["response"] = response  # optional: enables token extraction
    """
    ctx: dict[str, Any] = {}
    start = time.monotonic()
    try:
        yield ctx
        duration = time.monotonic() - start
        response = ctx.get("response")
        input_tok, output_tok = _extract_tokens(response)
        tool_names = ctx.get("tool_names")
        record(model, input_tok, output_tok, duration, status="ok", tool_names=tool_names)
    except Exception as exc:
        duration = time.monotonic() - start
        request_duration.labels(model=model, status="error").observe(duration)
        record_error(model, _classify_error(exc))
        raise


# ---------------------------------------------------------------------------
# Token extraction helpers
# ---------------------------------------------------------------------------


def _extract_tokens(response: Any) -> tuple[int, int]:
    """Extract input/output token counts from an LLM response object.

    Supports OpenAI and Anthropic response formats.
    Returns (0, 0) if extraction fails.
    """
    if response is None:
        return 0, 0

    # OpenAI: response.usage.prompt_tokens / completion_tokens
    usage = getattr(response, "usage", None)
    if usage is not None:
        prompt = getattr(usage, "prompt_tokens", 0) or 0
        completion = getattr(usage, "completion_tokens", 0) or 0
        if prompt or completion:
            return prompt, completion
        # Anthropic: response.usage.input_tokens / output_tokens
        input_tok = getattr(usage, "input_tokens", 0) or 0
        output_tok = getattr(usage, "output_tokens", 0) or 0
        return input_tok, output_tok

    return 0, 0


def _classify_error(exc: Exception) -> str:
    """Classify an exception into an error type label."""
    name = type(exc).__name__.lower()
    if "ratelimit" in name or "rate_limit" in name:
        return "rate_limit"
    if "timeout" in name:
        return "timeout"
    if "authentication" in name or "permission" in name:
        return "auth_error"
    if "connection" in name:
        return "connection_error"
    return "api_error"


# ---------------------------------------------------------------------------
# SDK auto-patching
# ---------------------------------------------------------------------------


def _patch_openai() -> None:
    """Monkey-patch OpenAI Python SDK to instrument LLM calls."""
    try:
        from openai.resources.chat import completions as chat_mod

        _original_create = chat_mod.Completions.create

        @functools.wraps(_original_create)
        def _patched_create(self: Any, *args: Any, **kwargs: Any) -> Any:
            model = kwargs.get("model", "unknown")
            start = time.monotonic()
            try:
                result = _original_create(self, *args, **kwargs)
                duration = time.monotonic() - start
                input_tok, output_tok = _extract_tokens(result)
                tool_names = _extract_tool_calls_openai(result)
                record(model, input_tok, output_tok, duration, status="ok", tool_names=tool_names)
                return result
            except Exception as exc:
                duration = time.monotonic() - start
                request_duration.labels(model=model, status="error").observe(duration)
                record_error(model, _classify_error(exc))
                raise

        chat_mod.Completions.create = _patched_create  # type: ignore[assignment]
    except (ImportError, AttributeError):
        pass  # openai not installed — skip silently


def _patch_anthropic() -> None:
    """Monkey-patch Anthropic Python SDK to instrument LLM calls."""
    try:
        from anthropic.resources import messages as msg_mod

        _original_create = msg_mod.Messages.create

        @functools.wraps(_original_create)
        def _patched_create(self: Any, *args: Any, **kwargs: Any) -> Any:
            model = kwargs.get("model", "unknown")
            start = time.monotonic()
            try:
                result = _original_create(self, *args, **kwargs)
                duration = time.monotonic() - start
                input_tok, output_tok = _extract_tokens(result)
                tool_names = _extract_tool_calls_anthropic(result)
                record(model, input_tok, output_tok, duration, status="ok", tool_names=tool_names)
                return result
            except Exception as exc:
                duration = time.monotonic() - start
                request_duration.labels(model=model, status="error").observe(duration)
                record_error(model, _classify_error(exc))
                raise

        msg_mod.Messages.create = _patched_create  # type: ignore[assignment]
    except (ImportError, AttributeError):
        pass  # anthropic not installed — skip silently


def _extract_tool_calls_openai(response: Any) -> list[str] | None:
    """Extract tool call names from an OpenAI response."""
    choices = getattr(response, "choices", None)
    if not choices:
        return None
    msg = getattr(choices[0], "message", None)
    if not msg:
        return None
    tool_calls = getattr(msg, "tool_calls", None)
    if not tool_calls:
        return None
    return [tc.function.name for tc in tool_calls if hasattr(tc, "function")]


def _extract_tool_calls_anthropic(response: Any) -> list[str] | None:
    """Extract tool call names from an Anthropic response."""
    content = getattr(response, "content", None)
    if not content:
        return None
    names = []
    for block in content:
        if getattr(block, "type", None) == "tool_use":
            name = getattr(block, "name", None)
            if name:
                names.append(name)
    return names or None
