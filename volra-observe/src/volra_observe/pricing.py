"""Embedded LLM pricing table (USD per 1K tokens)."""

COST_PER_1K_TOKENS: dict[str, dict[str, float]] = {
    # OpenAI
    "gpt-4o": {"input": 0.005, "output": 0.015},
    "gpt-4o-mini": {"input": 0.00015, "output": 0.0006},
    "gpt-4-turbo": {"input": 0.01, "output": 0.03},
    "gpt-4": {"input": 0.03, "output": 0.06},
    "gpt-3.5-turbo": {"input": 0.0005, "output": 0.0015},
    "o1": {"input": 0.015, "output": 0.06},
    "o1-mini": {"input": 0.003, "output": 0.012},
    "o3-mini": {"input": 0.0011, "output": 0.0044},
    # Anthropic
    "claude-sonnet-4-20250514": {"input": 0.003, "output": 0.015},
    "claude-haiku-4-5-20251001": {"input": 0.0008, "output": 0.004},
    "claude-opus-4-20250514": {"input": 0.015, "output": 0.075},
    "claude-3-5-sonnet-20241022": {"input": 0.003, "output": 0.015},
    "claude-3-haiku-20240307": {"input": 0.00025, "output": 0.00125},
}


def estimate_cost(model: str, input_tokens: int, output_tokens: int) -> float:
    """Estimate cost in USD for a given model and token counts.

    Returns 0.0 for unknown models (no error — graceful degradation).
    """
    pricing = COST_PER_1K_TOKENS.get(model)
    if pricing is None:
        return 0.0
    return (input_tokens * pricing["input"] + output_tokens * pricing["output"]) / 1000
