# volra-observe

Framework-agnostic LLM observability for [Volra](https://github.com/romerox3/volra) agents.

## Install

```bash
pip install volra-observe
```

## Quick Start

```python
import volra_observe

# Auto-patches OpenAI and Anthropic SDKs, starts Prometheus server on :9101
volra_observe.init(port=9101)
```

## Metrics Exposed

| Metric | Type | Labels |
|--------|------|--------|
| `volra_llm_tokens_total` | Counter | model, type (input/output) |
| `volra_llm_cost_dollars_total` | Counter | model |
| `volra_llm_request_duration_seconds` | Histogram | model, status |
| `volra_llm_errors_total` | Counter | model, error_type |
| `volra_tool_calls_total` | Counter | tool_name |

## Manual Instrumentation

```python
from volra_observe import track_llm, llm_context

# Decorator
@track_llm(model="gpt-4o")
def call_llm(prompt):
    return client.chat.completions.create(model="gpt-4o", messages=[...])

# Context manager
with llm_context(model="claude-sonnet-4-20250514") as ctx:
    response = anthropic.messages.create(...)
    ctx["response"] = response
```
