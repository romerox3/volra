"""Tests for core metrics and instrumentation APIs."""

from unittest.mock import MagicMock

from volra_observe._metrics import (
    _classify_error,
    _extract_tokens,
    _registry,
    cost_total,
    errors_total,
    llm_context,
    record,
    record_error,
    record_tool_call,
    request_duration,
    tokens_total,
    tool_calls_total,
    track_llm,
)
from volra_observe.pricing import estimate_cost


def _get_counter_value(counter, labels: dict) -> float:
    """Get the current value of a prometheus counter with given labels."""
    return counter.labels(**labels)._value.get()


def _get_histogram_count(histogram, labels: dict) -> float:
    """Get the observation count of a prometheus histogram."""
    return histogram.labels(**labels)._sum.get()


class TestRecord:
    def test_record_increments_tokens(self):
        before_input = _get_counter_value(tokens_total, {"model": "test-model", "type": "input"})
        before_output = _get_counter_value(tokens_total, {"model": "test-model", "type": "output"})
        record("test-model", input_tokens=100, output_tokens=50, duration_s=1.0)
        after_input = _get_counter_value(tokens_total, {"model": "test-model", "type": "input"})
        after_output = _get_counter_value(tokens_total, {"model": "test-model", "type": "output"})
        assert after_input - before_input == 100
        assert after_output - before_output == 50

    def test_record_increments_cost(self):
        before = _get_counter_value(cost_total, {"model": "gpt-4o"})
        record("gpt-4o", input_tokens=1000, output_tokens=1000, duration_s=1.0)
        after = _get_counter_value(cost_total, {"model": "gpt-4o"})
        expected = estimate_cost("gpt-4o", 1000, 1000)
        assert abs((after - before) - expected) < 1e-9

    def test_record_unknown_model_no_cost(self):
        before = _get_counter_value(cost_total, {"model": "mystery-llm"})
        record("mystery-llm", input_tokens=500, output_tokens=500, duration_s=0.5)
        after = _get_counter_value(cost_total, {"model": "mystery-llm"})
        assert after == before  # no cost increment for unknown models

    def test_record_with_tool_names(self):
        before = _get_counter_value(tool_calls_total, {"tool_name": "get_weather"})
        record("gpt-4o", 100, 50, 1.0, tool_names=["get_weather"])
        after = _get_counter_value(tool_calls_total, {"tool_name": "get_weather"})
        assert after - before == 1


class TestRecordError:
    def test_record_error(self):
        before = _get_counter_value(errors_total, {"model": "gpt-4o", "error_type": "rate_limit"})
        record_error("gpt-4o", "rate_limit")
        after = _get_counter_value(errors_total, {"model": "gpt-4o", "error_type": "rate_limit"})
        assert after - before == 1


class TestRecordToolCall:
    def test_record_tool_call(self):
        before = _get_counter_value(tool_calls_total, {"tool_name": "search_db"})
        record_tool_call("search_db", count=3)
        after = _get_counter_value(tool_calls_total, {"tool_name": "search_db"})
        assert after - before == 3


class TestExtractTokens:
    def test_openai_response(self):
        response = MagicMock()
        response.usage.prompt_tokens = 150
        response.usage.completion_tokens = 80
        assert _extract_tokens(response) == (150, 80)

    def test_anthropic_response(self):
        response = MagicMock()
        response.usage.prompt_tokens = 0
        response.usage.completion_tokens = 0
        response.usage.input_tokens = 200
        response.usage.output_tokens = 100
        assert _extract_tokens(response) == (200, 100)

    def test_none_response(self):
        assert _extract_tokens(None) == (0, 0)

    def test_no_usage(self):
        response = MagicMock(spec=[])  # no usage attribute
        assert _extract_tokens(response) == (0, 0)


class TestClassifyError:
    def test_rate_limit(self):
        exc = type("RateLimitError", (Exception,), {})()
        assert _classify_error(exc) == "rate_limit"

    def test_timeout(self):
        exc = type("TimeoutError", (Exception,), {})()
        assert _classify_error(exc) == "timeout"

    def test_auth(self):
        exc = type("AuthenticationError", (Exception,), {})()
        assert _classify_error(exc) == "auth_error"

    def test_connection(self):
        exc = type("ConnectionError", (Exception,), {})()
        assert _classify_error(exc) == "connection_error"

    def test_generic(self):
        exc = ValueError("something")
        assert _classify_error(exc) == "api_error"


class TestTrackLLMDecorator:
    def test_successful_call(self):
        response = MagicMock()
        response.usage.prompt_tokens = 100
        response.usage.completion_tokens = 50

        @track_llm(model="decorator-test")
        def my_llm_call():
            return response

        before = _get_counter_value(tokens_total, {"model": "decorator-test", "type": "input"})
        result = my_llm_call()
        after = _get_counter_value(tokens_total, {"model": "decorator-test", "type": "input"})
        assert result is response
        assert after - before == 100

    def test_failed_call(self):
        @track_llm(model="decorator-fail")
        def my_failing_call():
            raise type("RateLimitError", (Exception,), {})("rate limited")

        before = _get_counter_value(errors_total, {"model": "decorator-fail", "error_type": "rate_limit"})
        try:
            my_failing_call()
        except Exception:
            pass
        after = _get_counter_value(errors_total, {"model": "decorator-fail", "error_type": "rate_limit"})
        assert after - before == 1


class TestLLMContext:
    def test_successful_context(self):
        response = MagicMock()
        response.usage.prompt_tokens = 200
        response.usage.completion_tokens = 100

        before = _get_counter_value(tokens_total, {"model": "ctx-test", "type": "input"})
        with llm_context(model="ctx-test") as ctx:
            ctx["response"] = response
        after = _get_counter_value(tokens_total, {"model": "ctx-test", "type": "input"})
        assert after - before == 200

    def test_failed_context(self):
        before = _get_counter_value(errors_total, {"model": "ctx-fail", "error_type": "timeout"})
        try:
            with llm_context(model="ctx-fail"):
                raise type("TimeoutError", (Exception,), {})("timed out")
        except Exception:
            pass
        after = _get_counter_value(errors_total, {"model": "ctx-fail", "error_type": "timeout"})
        assert after - before == 1

    def test_context_with_tool_names(self):
        response = MagicMock()
        response.usage.prompt_tokens = 50
        response.usage.completion_tokens = 30

        before = _get_counter_value(tool_calls_total, {"tool_name": "ctx_tool"})
        with llm_context(model="ctx-tool-test") as ctx:
            ctx["response"] = response
            ctx["tool_names"] = ["ctx_tool"]
        after = _get_counter_value(tool_calls_total, {"tool_name": "ctx_tool"})
        assert after - before == 1
